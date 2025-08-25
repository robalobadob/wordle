// apps/go-server/go.mod
//
// Go module definition for the Wordle Go server.
//
// Module path:
//   github.com/robalobadob/wordle/apps/go-server
//
// Go version:
//   Built for Go 1.22.
//
// Direct dependencies:
//   • github.com/go-chi/chi/v5 v5.0.12
//       - Lightweight HTTP router used for defining API routes.
//   • github.com/golang-jwt/jwt/v5 v5.2.1
//       - JWT implementation for authentication (sign/verify tokens).
//   • github.com/joho/godotenv v1.5.1
//       - Loads environment variables from `.env` files in development.
//   • github.com/mattn/go-sqlite3 v1.14.22
//       - SQLite3 driver for database access.
//   • github.com/rs/zerolog v1.33.0
//       - Structured, leveled logging with JSON output.
//   • golang.org/x/crypto v0.26.0
//       - Crypto utilities (bcrypt, HMAC, etc.), used in auth & daily mode.
//
// Indirect dependencies (transitive):
//   • github.com/mattn/go-colorable v0.1.13
//       - Provides cross-platform colorized terminal output (used by zerolog).
//   • github.com/mattn/go-isatty v0.0.19
//       - Detects if output is a terminal (isatty check).
//   • golang.org/x/sys v0.23.0
//       - Low-level system call utilities, pulled by crypto/logging deps.

module github.com/robalobadob/wordle/apps/go-server

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.12
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/joho/godotenv v1.5.1
	github.com/mattn/go-sqlite3 v1.14.22
	github.com/rs/zerolog v1.33.0
	golang.org/x/crypto v0.26.0
)

require (
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.19    // indirect
	golang.org/x/sys v0.23.0              // indirect
)
