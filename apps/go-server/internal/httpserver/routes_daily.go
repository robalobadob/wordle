package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/robalobadob/wordle/apps/go-server/internal/daily"
	"github.com/robalobadob/wordle/apps/go-server/internal/words"
)

type dailyServer struct {
	srv      *Server
	store    *daily.Store
	salt     string
	sessions map[string]*dailySession // key = userID|date
	mu       sync.Mutex
}

type dailySession struct {
	GameID    string
	UserID    string
	Date      string
	WordIndex int
	Answer    string
	Start     time.Time
	Guesses   int
	Finished  bool
}

func (s *Server) mountDaily(r chi.Router) {
	dd := &dailyServer{
		srv:      s,
		store:    daily.NewStore(s.db),
		salt:     getEnv("DAILY_SALT", "local_dev_salt"),
		sessions: make(map[string]*dailySession),
	}
	r.Route("/daily", func(r chi.Router) {
		r.Post("/new", dd.handleNew)
		r.Post("/guess", dd.handleGuess)
		r.Get("/leaderboard", dd.handleLeaderboard)
	})
}

func (d *dailyServer) dateKeyNow() (date string, idx int, answer string) {
	now := time.Now().UTC()
	date = daily.DateKey(now)
	answers := words.Answers()
	if len(answers) == 0 {
		return date, 0, ""
	}
	idx = daily.WordIndex(now, d.salt, len(answers))
	return date, idx, answers[idx]
}

func (d *dailyServer) userIDWithAnon(w http.ResponseWriter, r *http.Request) (string, bool) {
	if me, _ := r.Context().Value(ctxUserKey{}).(*authUser); me != nil {
		return me.ID, true
	}
	return d.srv.ensureAnonID(w, r), true
}

// ---- /daily/new ----

type newRes struct {
	GameID string `json:"gameId"`
	Date   string `json:"date"`
	Played bool   `json:"played"`
}

func (d *dailyServer) handleNew(w http.ResponseWriter, r *http.Request) {
	uid, ok := d.userIDWithAnon(w, r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	date, idx, answer := d.dateKeyNow()

	// already played?
	if played, err := d.store.AlreadyPlayed(r.Context(), uid, date); err == nil && played {
		_ = json.NewEncoder(w).Encode(newRes{GameID: "", Date: date, Played: true})
		return
	}

	// reuse or create session
	key := uid + "|" + date
	d.mu.Lock()
	if sess, ok := d.sessions[key]; ok {
		d.mu.Unlock()
		_ = json.NewEncoder(w).Encode(newRes{GameID: sess.GameID, Date: date, Played: false})
		return
	}
	sess := &dailySession{
		GameID:    genID(),
		UserID:    uid,
		Date:      date,
		WordIndex: idx,
		Answer:    strings.ToLower(answer),
		Start:     time.Now(),
	}
	d.sessions[key] = sess
	d.mu.Unlock()

	_ = json.NewEncoder(w).Encode(newRes{GameID: sess.GameID, Date: date, Played: false})
}

// ---- /daily/guess ----

type dailyGuessReq struct {
	GameID string `json:"gameId"`
	Word   string `json:"word"`
}

type dailyGuessRes struct {
	Marks   []int  `json:"marks"`  // 0 miss, 1 present, 2 hit
	State   string `json:"state"`  // in_progress|won|locked
	Guesses int    `json:"guesses"`
}

func (d *dailyServer) handleGuess(w http.ResponseWriter, r *http.Request) {
	uid, ok := d.userIDWithAnon(w, r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var p dailyGuessReq
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p.Word = strings.ToLower(strings.TrimSpace(p.Word))
	if p.GameID == "" || len(p.Word) != 5 {
		http.Error(w, "invalid", http.StatusBadRequest)
		return
	}

	date, _, _ := d.dateKeyNow()

	// find session
	key := uid + "|" + date
	d.mu.Lock()
	sess, ok := d.sessions[key]
	d.mu.Unlock()
	if !ok || sess.GameID != p.GameID {
		http.Error(w, "no session", http.StatusConflict)
		return
	}
	if sess.Finished {
		_ = json.NewEncoder(w).Encode(dailyGuessRes{Marks: []int{}, State: "locked", Guesses: sess.Guesses})
		return
	}

	// validate & score via words
	if _, ok := words.Allowed()[p.Word]; !ok {
		http.Error(w, "word not allowed", http.StatusBadRequest)
		return
	}
	marks := words.Score(p.Word, sess.Answer)

	// update
	d.mu.Lock()
	sess.Guesses++
	won := allHits(marks)
	if won {
		sess.Finished = true
	}
	d.mu.Unlock()

	// persist on win
	if won {
		elapsed := int(time.Since(sess.Start).Milliseconds())
		_ = d.store.InsertResult(r.Context(), daily.Result{
			UserID: uid, Date: date, WordIndex: sess.WordIndex, Guesses: sess.Guesses, ElapsedMs: elapsed,
		})
		_ = json.NewEncoder(w).Encode(dailyGuessRes{Marks: marks, State: "won", Guesses: sess.Guesses})
		return
	}

	_ = json.NewEncoder(w).Encode(dailyGuessRes{Marks: marks, State: "in_progress", Guesses: sess.Guesses})
}

func allHits(m []int) bool {
	for _, v := range m {
		if v != 2 {
			return false
		}
	}
	return true
}

// ---- /daily/leaderboard ----

type lbRes struct {
	Date string        `json:"date"`
	Top  []daily.LBRow `json:"top"`
}

func (d *dailyServer) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		date, _, _ = d.dateKeyNow()
	}
	rows, err := d.store.Leaderboard(r.Context(), date, 20)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(lbRes{Date: date, Top: rows})
}
