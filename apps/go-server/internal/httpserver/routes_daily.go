// apps/go-server/internal/httpserver/routes_daily.go
//
// HTTP routes for the "Daily Challenge" mode.
// Exposes three endpoints under /daily:
//   - POST /daily/new         → start a daily game (creates or reuses session)
//   - POST /daily/guess       → submit a guess for today’s daily game
//   - GET  /daily/leaderboard → fetch top 20 results for today (or a given date)
//
// Each user can play once per day (enforced by DB + in-memory session).
// Sessions are held in memory for active play and persisted to DB on win.
// Deterministic word selection is based on date + salt.

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

// dailyServer wraps dependencies for /daily endpoints.
type dailyServer struct {
	srv      *Server
	store    *daily.Store
	salt     string
	sessions map[string]*dailySession // active sessions keyed by userID|date
	mu       sync.Mutex               // guards sessions
}

// dailySession holds transient in-memory state for an in-progress daily game.
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

// mountDaily registers all /daily routes.
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

// dateKeyNow returns today's date key, deterministic word index, and answer.
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

// userIDWithAnon returns the authenticated user ID if logged in,
// otherwise ensures an anonymous ID via Server.ensureAnonID.
func (d *dailyServer) userIDWithAnon(w http.ResponseWriter, r *http.Request) (string, bool) {
	if me, _ := r.Context().Value(ctxUserKey{}).(*authUser); me != nil {
		return me.ID, true
	}
	return d.srv.ensureAnonID(w, r), true
}

// -----------------------------------------------------------------------------
// /daily/new

// newRes is returned by /daily/new.
type newRes struct {
	GameID string `json:"gameId"`
	Date   string `json:"date"`
	Played bool   `json:"played"`
}

// handleNew creates or reuses a daily session for the current date.
// - If user already has a DB row for today → return Played=true.
// - Otherwise create/reuse an in-memory session and return GameID.
func (d *dailyServer) handleNew(w http.ResponseWriter, r *http.Request) {
	uid, ok := d.userIDWithAnon(w, r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	date, idx, answer := d.dateKeyNow()

	// Check if already played (persisted in DB).
	if played, err := d.store.AlreadyPlayed(r.Context(), uid, date); err == nil && played {
		_ = json.NewEncoder(w).Encode(newRes{GameID: "", Date: date, Played: true})
		return
	}

	// Reuse or create session in memory.
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

// -----------------------------------------------------------------------------
// /daily/guess

// dailyGuessReq is the request payload for /daily/guess.
type dailyGuessReq struct {
	GameID string `json:"gameId"`
	Word   string `json:"word"`
}

// dailyGuessRes is the response payload for /daily/guess.
type dailyGuessRes struct {
	Marks   []int  `json:"marks"`  // per-letter: 0=miss, 1=present, 2=hit
	State   string `json:"state"`  // in_progress | won | locked
	Guesses int    `json:"guesses"`
}

// handleGuess validates and applies a guess for today's daily session.
// - Ensures valid GameID and word.
// - Rejects if no session or session finished.
// - Validates against allowed word list.
// - Scores guess using words.Score.
// - Updates session state; persists result to DB if won.
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

	// Find session.
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

	// Validate word.
	if _, ok := words.Allowed()[p.Word]; !ok {
		http.Error(w, "word not allowed", http.StatusBadRequest)
		return
	}

	// Score guess.
	marks := words.Score(p.Word, sess.Answer)

	// Update in-memory session.
	d.mu.Lock()
	sess.Guesses++
	won := allHits(marks)
	if won {
		sess.Finished = true
	}
	d.mu.Unlock()

	// Persist and return.
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

// allHits reports true if every mark == 2 (hit).
func allHits(m []int) bool {
	for _, v := range m {
		if v != 2 {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------
// /daily/leaderboard

// lbRes is returned by /daily/leaderboard.
type lbRes struct {
	Date string        `json:"date"`
	Top  []daily.LBRow `json:"top"`
}

// handleLeaderboard returns the leaderboard for the given date (default today).
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
