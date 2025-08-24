package main

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type authUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type contextKey string

var userCtxKey = contextKey("user")

func requireAuth(db *DBWrap) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := bearerOrCookie(r)
			if tokenStr == "" {
				http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
				return
			}
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(os.Getenv("JWT_SECRET")), nil
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
			if _, err := findUserByID(db.SQL, id); err != nil {
				http.Error(w, `{"error":"Invalid token"}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), userCtxKey, &authUser{ID: id, Username: username})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func currentUser(r *http.Request) (*authUser, error) {
	u, _ := r.Context().Value(userCtxKey).(*authUser)
	if u == nil {
		return nil, errors.New("no user")
	}
	return u, nil
}
