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

// --- dependencies we need from words pkg as an interface (so it's testable)
type dailyDeps interface {
	Answers() []string
	Allowed() map[string]struct{}
	Score(guess, answer string) []int // 0=miss,1=present,2=hit
}

// server state for /daily
type dailyServer struct {
	srv      *Server
	store    *daily.Store
	w        dailyDeps
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

// adaptor to words package
type wordsPkg struct{}

func (wordsPkg) Answers() []string                { return words.Answers() }
func (wordsPkg) Allowed() map[string]struct{}     { return words.Allowed() }
func (wordsPkg) Score(g, a string) []int          { return words.Score(g, a) }

// mountDaily registers /daily routes under the given router.
func (s *Server) mountDaily(r chi.Router) {
	dd := &dailyServer{
		srv:      s,
		store:    daily.NewStore(s.db),
		w:        wordsPkg{},
		salt:     getEnv("DAILY_SALT", "local_dev_salt"),
		sessions: make(map[string]*dailySession),
	}

	r.Route("/daily", func(r chi.Router) {
		r.Post("/new", dd.handleNew)
		r.Post("/guess", dd.handleGuess)
		r.Get("/leaderboard", dd.handleLeaderboard)
	})
}

// helpers

func (d *dailyServer) dateKeyNow() (date string, idx int, answer string) {
	now := time.Now().UTC()
	date = daily.DateKey(now)
	ans := d.w.Answers()
	if len(ans) == 0 {
		return date, 0, "" // shouldn't happen
	}
	idx = daily.WordIndex(now, d.salt, len(ans))
	return date, idx, ans[idx]
}

// userIDWithAnon returns the authed user id if present, otherwise creates/returns anon id
func (d *dailyServer) userIDWithAnon(w http.ResponseWriter, r *http.Request) (string, bool) {
	if me, _ := r.Context().Value(ctxUserKey{}).(*authUser); me != nil {
		return me.ID, true
	}
	// guests allowed; we’ll use the anon cookie as user id
	return d.srv.ensureAnonID(w, r), true
}

// ---- /daily/new -----------------------------------------------------------

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

	// deny if already submitted
	if played, err := d.store.AlreadyPlayed(r.Context(), uid, date); err == nil && played {
		_ = json.NewEncoder(w).Encode(newRes{GameID: "", Date: date, Played: true})
		return
	}

	key := uid + "|" + date
	d.mu.Lock()
	if sess, ok := d.sessions[key]; ok {
		d.mu.Unlock()
		_ = json.NewEncoder(w).Encode(newRes{GameID: sess.GameID, Date: date, Played: false})
		return
	}
	sess := &dailySession{
		GameID:    genID(), // from server.go (same package)
		UserID:    uid,
		Date:      date,
		WordIndex: idx,
		Answer:    answer,
		Start:     time.Now(),
	}
	d.sessions[key] = sess
	d.mu.Unlock()

	_ = json.NewEncoder(w).Encode(newRes{GameID: sess.GameID, Date: date, Played: false})
}

// ---- /daily/guess ---------------------------------------------------------

type guessReq struct {
	GameID string `json:"gameId"`
	Word   string `json:"word"`
}

type guessRes struct {
	Marks   []int  `json:"marks"`
	State   string `json:"state"` // in_progress|won|locked
	Guesses int    `json:"guesses"`
}

func (d *dailyServer) handleGuess(w http.ResponseWriter, r *http.Request) {
	uid, ok := d.userIDWithAnon(w, r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var p guessReq
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

	// lookup session
	key := uid + "|" + date
	d.mu.Lock()
	sess, ok := d.sessions[key]
	d.mu.Unlock()
	if !ok || sess.GameID != p.GameID {
		http.Error(w, "no session", http.StatusConflict)
		return
	}
	if sess.Finished {
		_ = json.NewEncoder(w).Encode(guessRes{Marks: []int{}, State: "locked", Guesses: sess.Guesses})
		return
	}

	// validate guess word  (FIX: check map membership correctly)
	if _, ok := d.w.Allowed()[p.Word]; !ok {
		http.Error(w, "word not allowed", http.StatusBadRequest)
		return
	}

	marks := d.w.Score(p.Word, sess.Answer)

	// update counters
	d.mu.Lock()
	sess.Guesses++
	won := allHits(marks)
	if won {
		sess.Finished = true
	}
	d.mu.Unlock()

	// persist result on win
	if won {
		elapsed := int(time.Since(sess.Start).Milliseconds())
		_ = d.store.InsertResult(r.Context(), daily.Result{
			UserID: uid, Date: date, WordIndex: sess.WordIndex, Guesses: sess.Guesses, ElapsedMs: elapsed,
		})
		_ = json.NewEncoder(w).Encode(guessRes{Marks: marks, State: "won", Guesses: sess.Guesses})
		return
	}

	_ = json.NewEncoder(w).Encode(guessRes{Marks: marks, State: "in_progress", Guesses: sess.Guesses})
}

func allHits(m []int) bool {
	for _, v := range m {
		if v != 2 {
			return false
		}
	}
	return true
}

// ---- /daily/leaderboard ---------------------------------------------------

type lbRes struct {
	Date string         `json:"date"`
	Top  []daily.LBRow  `json:"top"`
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
