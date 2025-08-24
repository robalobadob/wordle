package main

import (
	"database/sql"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
	GamesPlayed  int       `json:"gamesPlayed"`
	Wins         int       `json:"wins"`
	Streak       int       `json:"streak"`
}

type SignupReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
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

func hashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost) // cost=10
	return string(b), err
}
func checkPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func createUser(db *sql.DB, username, pw string) (*User, error) {
	username = normalizeUsername(username)
	if err := validateSignup(username, pw); err != nil {
		return nil, err
	}
	var exists int
	_ = db.QueryRow(`SELECT 1 FROM users WHERE lower(username)=lower(?)`, username).Scan(&exists)
	if exists == 1 {
		return nil, errors.New("username taken")
	}
	h, err := hashPassword(pw)
	if err != nil {
		return nil, err
	}
	u := &User{
		ID:           genID(),
		Username:     username,
		PasswordHash: h,
		CreatedAt:    time.Now().UTC(),
	}
	_, err = db.Exec(`INSERT INTO users (id, username, password_hash, created_at) VALUES (?,?,?,?)`,
		u.ID, u.Username, u.PasswordHash, u.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return u, nil
}

func findUserByUsername(db *sql.DB, username string) (*User, error) {
	row := db.QueryRow(`SELECT id, username, password_hash, created_at, games_played, wins, streak
	                    FROM users WHERE lower(username)=lower(?)`, username)
	return scanUser(row)
}
func findUserByID(db *sql.DB, id string) (*User, error) {
	row := db.QueryRow(`SELECT id, username, password_hash, created_at, games_played, wins, streak
	                    FROM users WHERE id=?`, id)
	return scanUser(row)
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var created string
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &created, &u.GamesPlayed, &u.Wins, &u.Streak); err != nil {
		return nil, err
	}
	t, _ := time.Parse(time.RFC3339, created)
	u.CreatedAt = t
	return &u, nil
}

func signJWT(id, username string) (string, time.Time, error) {
	days := envInt("JWT_EXPIRES_DAYS", 14)
	secret := []byte(os.Getenv("JWT_SECRET"))
	exp := time.Now().Add(time.Duration(days) * 24 * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":       id,
		"username": username,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	})
	ss, err := token.SignedString(secret)
	return ss, exp, err
}

func setAuthCookie(w http.ResponseWriter, token string, exp time.Time) {
	name := envStr("COOKIE_NAME", "wordle_token")
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

func clearAuthCookie(w http.ResponseWriter) {
	name := envStr("COOKIE_NAME", "wordle_token")
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
	if c, err := r.Cookie(envStr("COOKIE_NAME", "wordle_token")); err == nil {
		return c.Value
	}
	return ""
}
