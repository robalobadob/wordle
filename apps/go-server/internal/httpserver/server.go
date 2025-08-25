// apps/go-server/internal/httpserver/server.go
//
// HTTP server wiring for the Wordle backend.
// Responsibilities:
//   - Router + middleware (JSON, CORS, timeouts, panic recovery, request IDs).
//   - Public endpoints: "/", "/health".
//   - Game endpoints (optional auth): POST /game/new, POST /game/guess.
//   - Daily Challenge endpoints (optional auth): mounted under /daily.
//   - Auth + profile/stat endpoints (require auth): /auth/*, /stats/me, /games/mine.
//   - JWT + cookie handling, anonymous session cookie, user CRUD helpers.
//   - Database persistence for games and user stats.
//
// Notes:
//   - CORS is origin‑aware and credentials‑enabled (so cookies work).
//   - Optional auth decorates requests with user context when a valid token is present;
//     routes can still run for guests.
//   - Require‑auth middleware enforces presence and validity of a JWT.

package httpserver

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/robalobadob/wordle/apps/go-server/internal/game"
	"github.com/robalobadob/wordle/apps/go-server/internal/store"
	"github.com/robalobadob/wordle/apps/go-server/internal/words"
)

// Server bundles router, in-memory game store, and DB handle.
type Server struct {
	r     *chi.Mux
	store store.Store
	db    *sql.DB
}

// New constructs a Server, installs middleware, and registers routes.
func New(st store.Store, db *sql.DB) *Server {
	s := &Server{r: chi.NewRouter(), store: st, db: db}

	// --- middleware ---
	s.r.Use(chimw.RequestID)                 // add X-Request-ID
	s.r.Use(chimw.RealIP)                    // set RemoteAddr from X-Forwarded-For etc.
	s.r.Use(chimw.Recoverer)                 // recover from panics
	s.r.Use(chimw.Timeout(10 * time.Second)) // bound handler time
	s.r.Use(jsonContentType)                 // default JSON responses
	s.r.Use(corsFromEnv)                     // credentials-friendly CORS

	// --- diagnostics ---
	s.r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"wordle-go","endpoints":["/health","POST /game/new","POST /game/guess","/auth/*"]}`))
	})
	s.r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// Game endpoints — OPTIONAL AUTH (guests can play)
	s.r.With(s.withOptionalAuth()).Post("/game/new", s.handleNewGame)
	s.r.With(s.withOptionalAuth()).Post("/game/guess", s.handleGuess)

	// Daily Challenge — OPTIONAL AUTH (guests can play; progress persisted on win)
	s.mountDaily(s.r.With(s.withOptionalAuth()))

	// Auth + profile/stats (require auth)
	s.mountAuthRoutes()

	// JSON 404 for easier debugging
	s.r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not_found","path":"`+r.URL.Path+`"}`, http.StatusNotFound)
	})

	// Debug: word list counts
	s.r.Get("/debug/words", func(w http.ResponseWriter, r *http.Request) {
		a, g := words.Stats()
		_ = json.NewEncoder(w).Encode(map[string]int{"answers": a, "allowed": g})
	})

	return s
}

// Start begins serving HTTP on addr.
func (s *Server) Start(addr string) error { return http.ListenAndServe(addr, s.r) }

// Router exposes the internal router (useful for tests).
func (s *Server) Router() chi.Router { return s.r }

// ----------------------------- middleware ----------------------------------

// jsonContentType sets a default JSON Content-Type header on all responses.
func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

// corsFromEnv enables credentialed CORS for a single origin.
// Uses CLIENT_ORIGIN env var; defaults to http://localhost:5173.
func corsFromEnv(next http.Handler) http.Handler {
	origin := os.Getenv("CLIENT_ORIGIN")
	if origin == "" {
		origin = "http://localhost:5173"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ------------------------------ GAME ---------------------------------------

// newGameReq/Res payloads for POST /game/new.
type newGameReq struct {
	Mode   string `json:"mode"`   // "normal" | "cheat" (cheat currently ignored)
	Answer string `json:"answer"` // optional fixed answer (testing)
}
type newGameRes struct {
	GameID string `json:"gameId"`
}

// handleNewGame creates a new in-memory game and persists a DB "owner" row
// (either user_id or anonymous_id) for history/stats.
func (s *Server) handleNewGame(w http.ResponseWriter, r *http.Request) {
	var req newGameReq
	_ = json.NewDecoder(r.Body).Decode(&req)

	// Create game (random answer by default if req.Answer is empty)
	g := game.New(req.Answer)
	if err := s.store.Save(r.Context(), g); err != nil {
		log.Error().Err(err).Msg("save game")
		http.Error(w, `{"error":"save_failed"}`, http.StatusInternalServerError)
		return
	}

	// Persist owner row; do NOT store answer in DB unless schema requires it
	now := time.Now().UTC().Format(time.RFC3339)
	if me, _ := r.Context().Value(ctxUserKey{}).(*authUser); me != nil {
		_, err := s.db.Exec(`INSERT INTO games (id, user_id, answer, started_at, status, guesses)
		                     VALUES (?,?,?,?,?,0)`, g.ID, me.ID, "", now, "playing")
		if err != nil {
			log.Warn().Err(err).Str("gameId", g.ID).Msg("insert user game row")
		}
	} else {
		anon := s.ensureAnonID(w, r)
		_, err := s.db.Exec(`INSERT INTO games (id, anonymous_id, answer, started_at, status, guesses)
		                     VALUES (?,?,?,?,?,0)`, g.ID, anon, "", now, "playing")
		if err != nil {
			log.Warn().Err(err).Str("gameId", g.ID).Msg("insert anon game row")
		}
	}

	_ = json.NewEncoder(w).Encode(newGameRes{GameID: g.ID})
}

// guessReq/Res payloads for POST /game/guess.
type guessReq struct {
	GameID string `json:"gameId"`
	Guess  string `json:"guess"`
}
type guessRes struct {
	Marks []game.Mark `json:"marks"`
	State string      `json:"state"` // "playing" | "won" | "lost"
}

// handleGuess applies a guess to an in-memory game, persists progress,
// and (if finished) updates user stats in a best-effort transaction.
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

	// Persist counters/history (best effort, non-fatal if it fails)
	me, _ := r.Context().Value(ctxUserKey{}).(*authUser)
	ownerClause := `anonymous_id=?`
	ownerArg := any(s.ensureAnonID(w, r))
	if me != nil {
		ownerClause = `user_id=?`
		ownerArg = any(me.ID)
	}

	tx, _ := s.db.Begin()
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE games SET guesses = guesses + 1 WHERE id=? AND `+ownerClause, g.ID, ownerArg); err != nil {
		log.Warn().Err(err).Msg("update guesses")
	}

	if state == "won" || state == "lost" {
		if _, err := tx.Exec(`UPDATE games SET status=?, finished_at=? WHERE id=? AND `+ownerClause,
			state, time.Now().UTC().Format(time.RFC3339), g.ID, ownerArg); err != nil {
			log.Warn().Err(err).Msg("finish game")
		}
		if me != nil {
			if err := s.bumpStats(tx, me.ID, state == "won"); err != nil {
				log.Warn().Err(err).Str("user", me.ID).Msg("bump stats")
			}
		}
	}
	_ = tx.Commit()

	_ = json.NewEncoder(w).Encode(guessRes{Marks: marks, State: state})
}

// ------------------------------- AUTH --------------------------------------

// Request payloads for signup/login.
type signupReq struct{ Username, Password string }
type loginReq struct{ Username, Password string }

// authUser is placed into request context by auth middleware.
type authUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

// mountAuthRoutes registers authentication + gated routes (/auth/*, /stats/me, /games/mine).
func (s *Server) mountAuthRoutes() {
	s.r.Post("/auth/signup", s.handleSignup)
	s.r.Post("/auth/login", s.handleLogin)
	s.r.Post("/auth/logout", s.handleLogout)

	// Current user (gated)
	s.r.With(s.requireAuth()).Get("/auth/me", func(w http.ResponseWriter, r *http.Request) {
		me, _ := r.Context().Value(ctxUserKey{}).(*authUser)
		if me == nil {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(me)
	})

	// Stats (gated)
	s.r.With(s.requireAuth()).Get("/stats/me", func(w http.ResponseWriter, r *http.Request) {
		me, _ := r.Context().Value(ctxUserKey{}).(*authUser)
		if me == nil {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		u, err := s.findUserByID(me.ID)
		if err != nil {
			http.Error(w, `{"error":"not_found"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          u.ID,
			"gamesPlayed": u.GamesPlayed,
			"wins":        u.Wins,
			"streak":      u.Streak,
		})
	})

	// Recent games (gated)
	s.r.With(s.requireAuth()).Get("/games/mine", func(w http.ResponseWriter, r *http.Request) {
		me, _ := r.Context().Value(ctxUserKey{}).(*authUser)
		if me == nil {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		rows, err := s.db.Query(`SELECT id, status, guesses, started_at, COALESCE(finished_at,'')
		                         FROM games WHERE user_id=? ORDER BY started_at DESC LIMIT 50`, me.ID)
		if err != nil {
			http.Error(w, `{"error":"db_error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type gameRow struct {
			ID         string `json:"id"`
			Status     string `json:"status"`
			Guesses    int    `json:"guesses"`
			StartedAt  string `json:"startedAt"`
			FinishedAt string `json:"finishedAt,omitempty"`
		}
		out := []gameRow{}
		for rows.Next() {
			var gr gameRow
			if err := rows.Scan(&gr.ID, &gr.Status, &gr.Guesses, &gr.StartedAt, &gr.FinishedAt); err == nil {
				if gr.FinishedAt == "" {
					gr.FinishedAt = ""
				}
				out = append(out, gr)
			}
		}
		_ = json.NewEncoder(w).Encode(out)
	})
}

// handleSignup creates a new user, signs a JWT, sets auth cookie, and claims anon history.
func (s *Server) handleSignup(w http.ResponseWriter, r *http.Request) {
	var body signupReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
		return
	}
	u, err := s.createUser(body.Username, body.Password)
	if err != nil {
		if err.Error() == "username taken" {
			http.Error(w, `{"error":"Username taken"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	tok, exp, err := s.signJWT(u.ID, u.Username)
	if err != nil {
		http.Error(w, `{"error":"sign_failed"}`, http.StatusInternalServerError)
		return
	}
	s.setAuthCookie(w, tok, exp)
	// Attach any anonymous games to the new account
	s.claimAnonGames(s.ensureAnonID(w, r), u.ID)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": u.ID, "username": u.Username, "createdAt": u.CreatedAt})
}

// handleLogin authenticates user, sets cookie, and claims anon history.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body loginReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
		return
	}
	u, err := s.findUserByUsername(strings.TrimSpace(body.Username))
	if err != nil || !checkPassword(u.PasswordHash, body.Password) {
		http.Error(w, `{"error":"Invalid username or password"}`, http.StatusUnauthorized)
		return
	}
	tok, exp, err := s.signJWT(u.ID, u.Username)
	if err != nil {
		http.Error(w, `{"error":"sign_failed"}`, http.StatusInternalServerError)
		return
	}
	s.setAuthCookie(w, tok, exp)
	s.claimAnonGames(s.ensureAnonID(w, r), u.ID)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": u.ID, "username": u.Username})
}

// handleLogout clears the auth cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearAuthCookie(w)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// --------------------------- optional auth ---------------------------------

// withOptionalAuth decorates requests with user context if a valid JWT is present.
// It never 401s; used for routes where guests are allowed.
func (s *Server) withOptionalAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if tok := bearerOrCookie(r); tok != "" {
				claims := jwt.MapClaims{}
				if t, err := jwt.ParseWithClaims(tok, claims, func(t *jwt.Token) (interface{}, error) {
					return []byte(getEnv("JWT_SECRET", "dev_secret_change_me")), nil
				}); err == nil && t.Valid {
					if id, _ := claims["id"].(string); id != "" {
						if u, err := s.findUserByID(id); err == nil {
							ctx := context.WithValue(r.Context(), ctxUserKey{}, &authUser{ID: u.ID, Username: u.Username})
							r = r.WithContext(ctx)
						}
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

const anonCookieName = "wordle_anon"

// ensureAnonID returns an existing anon cookie or sets a new one.
// Used to associate guest games with a stable identifier.
func (s *Server) ensureAnonID(w http.ResponseWriter, r *http.Request) string {
	if c, err := r.Cookie(anonCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	id := genID()
	http.SetCookie(w, &http.Cookie{
		Name:     anonCookieName,
		Value:    id,
		Path:     "/",
		HttpOnly: true,
		Secure:   os.Getenv("NODE_ENV") == "production",
		SameSite: func() http.SameSite {
			if os.Getenv("NODE_ENV") == "production" {
				return http.SameSiteNoneMode
			}
			return http.SameSiteLaxMode
		}(),
		Expires: time.Now().Add(180 * 24 * time.Hour),
	})
	return id
}

// claimAnonGames transfers any anonymous games to a user account after auth.
func (s *Server) claimAnonGames(anonID, userID string) {
	if anonID == "" || userID == "" {
		return
	}
	if _, err := s.db.Exec(`UPDATE games SET user_id=?, anonymous_id=NULL WHERE anonymous_id=?`, userID, anonID); err != nil {
		log.Warn().Err(err).Msg("claim anon games")
	}
}

// ------------------------ auth helpers & users -----------------------------

// userRow matches the users table shape.
type userRow struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	GamesPlayed  int
	Wins         int
	Streak       int
}

// createUser validates input, checks uniqueness, hashes password, and inserts a new user.
func (s *Server) createUser(username, pw string) (*userRow, error) {
	username = normalizeUsername(username)
	if err := validateSignup(username, pw); err != nil {
		return nil, err
	}
	var exists int
	_ = s.db.QueryRow(`SELECT 1 FROM users WHERE lower(username)=lower(?)`, username).Scan(&exists)
	if exists == 1 {
		return nil, errors.New("username taken")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	id := genID()
	if _, err := s.db.Exec(`INSERT INTO users (id, username, password_hash, created_at) VALUES (?,?,?,?)`,
		id, username, string(h), now); err != nil {
		return nil, err
	}
	return &userRow{ID: id, Username: username, PasswordHash: string(h), CreatedAt: mustParse(now)}, nil
}

// findUserByUsername/ID load a user row or return an error if missing.
func (s *Server) findUserByUsername(username string) (*userRow, error) {
	row := s.db.QueryRow(`SELECT id, username, password_hash, created_at, games_played, wins, streak
	                      FROM users WHERE lower(username)=lower(?)`, username)
	return scanUser(row)
}
func (s *Server) findUserByID(id string) (*userRow, error) {
	row := s.db.QueryRow(`SELECT id, username, password_hash, created_at, games_played, wins, streak
	                      FROM users WHERE id=?`, id)
	return scanUser(row)
}

// scanUser converts a *sql.Row into a userRow.
func scanUser(row *sql.Row) (*userRow, error) {
	var u userRow
	var created string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &created, &u.GamesPlayed, &u.Wins, &u.Streak); err != nil {
		return nil, err
	}
	u.CreatedAt = mustParse(created)
	return &u, nil
}

// mustParse parses RFC3339 timestamps; on error returns zero time.
func mustParse(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// checkPassword is a bcrypt verifier.
func checkPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// normalizeUsername trims whitespace; adjust here if you want stricter rules.
func normalizeUsername(u string) string {
	return strings.TrimSpace(u)
}

// validateSignup enforces basic username/password rules.
func validateSignup(u, p string) error {
	if len(u) < 3 || len(u) > 24 {
		return errors.New("username must be 3–24 chars")
	}
	for _, r := range u {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return errors.New("username: letters, numbers, underscore only")
		}
	}
	if len(p) < 8 || len(p) > 100 {
		return errors.New("password must be 8–100 chars")
	}
	return nil
}

// genID creates a 22‑char URL‑safe, crypto‑random identifier (no padding).
func genID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	s := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b[:])
	if len(s) > 22 {
		return s[:22]
	}
	return s
}

// bumpStats increments games played; updates wins and streak based on result (within tx).
func (s *Server) bumpStats(tx *sql.Tx, userID string, won bool) error {
	var gp, wins, streak int
	row := tx.QueryRow(`SELECT games_played, wins, streak FROM users WHERE id=?`, userID)
	if err := row.Scan(&gp, &wins, &streak); err != nil {
		return err
	}
	gp++
	if won {
		wins++
		streak++
	} else {
		streak = 0
	}
	_, err := tx.Exec(`UPDATE users SET games_played=?, wins=?, streak=? WHERE id=?`, gp, wins, streak, userID)
	return err
}

// ------------------------------ JWT & cookies ------------------------------

// signJWT creates an HS256 JWT with id/username and a configurable expiry (JWT_EXPIRES_DAYS; default 14).
func (s *Server) signJWT(id, username string) (string, time.Time, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev_secret_change_me"
	}
	days := 14
	if v := os.Getenv("JWT_EXPIRES_DAYS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			days = n
		}
	}
	exp := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       id,
		"username": username,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	})
	ss, err := t.SignedString([]byte(secret))
	return ss, exp, err
}

// setAuthCookie writes the auth token cookie with appropriate security attributes.
func (s *Server) setAuthCookie(w http.ResponseWriter, token string, exp time.Time) {
	name := getEnv("COOKIE_NAME", "wordle_token")
	secure := os.Getenv("NODE_ENV") == "production"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode // required for third‑party contexts when Secure
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  exp,
	})
}

// clearAuthCookie deletes the auth token cookie.
func (s *Server) clearAuthCookie(w http.ResponseWriter) {
	name := getEnv("COOKIE_NAME", "wordle_token")
	secure := os.Getenv("NODE_ENV") == "production"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		MaxAge:   -1,
	})
}

// bearerOrCookie extracts a bearer token from Authorization header or auth cookie.
func bearerOrCookie(r *http.Request) string {
	// Authorization: Bearer <token>
	if a := r.Header.Get("Authorization"); strings.HasPrefix(strings.ToLower(a), "bearer ") {
		return strings.TrimSpace(a[7:])
	}
	if c, err := r.Cookie(getEnv("COOKIE_NAME", "wordle_token")); err == nil {
		return c.Value
	}
	return ""
}

// ---------------------------- auth middleware ------------------------------

// ctxUserKey is the context key type for storing authUser.
type ctxUserKey struct{}

// requireAuth enforces a valid JWT and injects authUser into request context.
func (s *Server) requireAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := bearerOrCookie(r)
			if tokenStr == "" {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(getEnv("JWT_SECRET", "dev_secret_change_me")), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				return
			}
			id, _ := claims["id"].(string)
			username, _ := claims["username"].(string)
			if id == "" || username == "" {
				http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				return
			}
			// Ensure user still exists
			if _, err := s.findUserByID(id); err != nil {
				http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserKey{}, &authUser{ID: id, Username: username})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ------------------------------- small util --------------------------------

// getEnv returns the value of k or def if unset/empty.
func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
