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

type Server struct {
	r     *chi.Mux
	store store.Store
	db    *sql.DB
}

func New(st store.Store, db *sql.DB) *Server {
	s := &Server{r: chi.NewRouter(), store: st, db: db}

	// --- middleware ---
	s.r.Use(chimw.RequestID)
	s.r.Use(chimw.RealIP)
	s.r.Use(chimw.Recoverer)
	s.r.Use(chimw.Timeout(10 * time.Second))
	s.r.Use(jsonContentType)
	s.r.Use(corsFromEnv) // origin-aware CORS so cookies can work

	// --- routes ---
	s.r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"wordle-go","endpoints":["/health","POST /game/new","POST /game/guess","/auth/*"]}`))
	})

	s.r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// Game endpoints (now require auth)
	s.r.With(s.requireAuth()).Post("/game/new", s.handleNewGame)
	s.r.With(s.requireAuth()).Post("/game/guess", s.handleGuess)

	// Auth endpoints
	s.mountAuthRoutes()

	// JSON 404s so mistakes are obvious in dev
	s.r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not_found","path":"`+r.URL.Path+`"}`, http.StatusNotFound)
	})

	// Debug
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
func (s *Server) Router() chi.Router      { return s.r }

// --- middleware ---

func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

// CORS that allows cookies (credentials). Uses CLIENT_ORIGIN; falls back to http://localhost:5173.
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

// --- GAME handlers ---

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

// --- AUTH: routes + helpers ---

type signupReq struct{ Username, Password string }
type loginReq struct{ Username, Password string }

type authUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func (s *Server) mountAuthRoutes() {
	s.r.Post("/auth/signup", s.handleSignup)
	s.r.Post("/auth/login", s.handleLogin)
	s.r.Post("/auth/logout", s.handleLogout)

	// protected
	s.r.With(s.requireAuth()).Get("/auth/me", func(w http.ResponseWriter, r *http.Request) {
		me, _ := r.Context().Value(ctxUserKey{}).(*authUser)
		if me == nil {
			http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(me)
	})
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
}

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
	_ = json.NewEncoder(w).Encode(map[string]any{"id": u.ID, "username": u.Username, "createdAt": u.CreatedAt})
}

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
	_ = json.NewEncoder(w).Encode(map[string]any{"id": u.ID, "username": u.Username})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearAuthCookie(w)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

// --- optional auth (does not 401) ---
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

// ---- auth helpers (JWT, bcrypt, user store) ----

type userRow struct {
	ID           string
	Username     string
	PasswordHash string
	CreatedAt    time.Time
	GamesPlayed  int
	Wins         int
	Streak       int
}

func (s *Server) createUser(username, pw string) (*userRow, error) {
	username = normalizeUsername(username)
	if err := validateSignup(username, pw); err != nil {
		return nil, err
	}
	// check exists
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

func scanUser(row *sql.Row) (*userRow, error) {
	var u userRow
	var created string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &created, &u.GamesPlayed, &u.Wins, &u.Streak); err != nil {
		return nil, err
	}
	u.CreatedAt = mustParse(created)
	return &u, nil
}

func mustParse(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func checkPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func normalizeUsername(u string) string {
	return strings.TrimSpace(u)
}

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

func genID() string {
	// 22-char URL-safe id (no padding), crypto-random
	var b [16]byte
	_, _ = rand.Read(b[:])
	s := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b[:])
	if len(s) > 22 {
		return s[:22]
	}
	return s
}

// --- JWT & cookies ---

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

func (s *Server) setAuthCookie(w http.ResponseWriter, token string, exp time.Time) {
	name := getEnv("COOKIE_NAME", "wordle_token")
	secure := os.Getenv("NODE_ENV") == "production"
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
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

// --- auth middleware ---

type ctxUserKey struct{}

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
			// ensure user still exists
			if _, err := s.findUserByID(id); err != nil {
				http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserKey{}, &authUser{ID: id, Username: username})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// --- small util ---

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
