// apps/go-server/internal/game/types.go
//
// Core type definitions for the Wordle game engine.
// Defines:
//   - Mark: per-letter result of a guess (hit/present/miss).
//   - Game: state for a single in-progress or finished game.

package game

// Mark represents the evaluation result for a single letter in a guess.
// Possible values:
//   - "hit":    letter is correct and in the correct position.
//   - "present": letter exists in the answer but in a different position.
//   - "miss":   letter does not exist in the answer at all.
type Mark string

const (
	MarkHit    Mark = "hit"
	MarkPresent     = "present"
	MarkMiss        = "miss"
)

// Game holds the state of a single Wordle game session.
type Game struct {
	ID       string   // Unique game identifier (random hex string).
	Answer   string   // The solution word (always lowercase).
	Rows     int      // Maximum number of guesses allowed (typically 6).
	Cols     int      // Number of letters per word (typically 5).
	Guesses  []string // List of guesses made so far (lowercased).
	Finished bool     // True once the game is over (won or lost).
	Won      bool     // True if the game was finished with a win.
}
