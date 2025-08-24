package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	"github.com/robalobadob/wordle/apps/go-server/internal/game"
	"github.com/robalobadob/wordle/apps/go-server/internal/store"
	"github.com/robalobadob/wordle/apps/go-server/internal/words"

)

type Server struct {
	r     *chi.Mux
	store store.Store
}

func New(st store.Store) *Server {
	s := &Server{r: chi.NewRouter(), store: st}

	// --- middlewares ---
	s.r.Use(chimw.RequestID)
	s.r.Use(chimw.RealIP)
	s.r.Use(chimw.Recoverer)
	s.r.Use(chimw.Timeout(10 * time.Second))
	s.r.Use(jsonContentType)
	s.r.Use(corsAllowAll) // dev‑friendly CORS; tighten later for prod

	// --- routes ---
	s.r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"wordle-go","endpoints":["/health","POST /game/new","POST /game/guess"]}`))
	})

	s.r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	s.r.Post("/game/new", s.handleNewGame)
	s.r.Post("/game/guess", s.handleGuess)

	// JSON 404s so mistakes are obvious in dev
	s.r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not_found","path":"`+r.URL.Path+`"}`, http.StatusNotFound)
	})

	s.r.Get("/debug/words", func(w http.ResponseWriter, r *http.Request) {
		a, g := words.Stats()
		_ = json.NewEncoder(w).Encode(map[string]int{
			"answers": a,
			"allowed": g,
		})
	})

	return s
}

func (s *Server) Start(addr string) error { return http.ListenAndServe(addr, s.r) }

// --- middleware ---

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func corsAllowAll(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			// Preflight OK
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- handlers ---

type newGameReq struct {
	Mode   string `json:"mode"`   // "normal" | "cheat" (cheat ignored for now)
	Answer string `json:"answer"` // optional fixed answer for testing
}

type newGameRes struct {
	GameID string `json:"gameId"`
}

func (s *Server) handleNewGame(w http.ResponseWriter, r *http.Request) {
	var req newGameReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	g := game.New(req.Answer) // random default inside game.New
	if err := s.store.Save(r.Context(), g); err != nil {
		log.Error().Err(err).Msg("save game")
		http.Error(w, `{"error":"save_failed"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(newGameRes{GameID: g.ID})
}

type guessReq struct {
	GameID string `json:"gameId"`
	Guess  string `json:"guess"`
}

type guessRes struct {
	Marks []game.Mark `json:"marks"`
	State string      `json:"state"` // "playing" | "won" | "lost"
}

func (s *Server) handleGuess(w http.ResponseWriter, r *http.Request) {
	var req guessReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad_json"}`, http.StatusBadRequest)
		return
	}
	g, err := s.store.Get(r.Context(), req.GameID)
	if err != nil {
		http.Error(w, `{"error":"not_found"}`, http.StatusNotFound)
		return
	}
	marks, state, err := g.ApplyGuess(req.Guess)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	if err := s.store.Save(r.Context(), g); err != nil {
		http.Error(w, `{"error":"save_failed"}`, http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(guessRes{Marks: marks, State: state})
}
