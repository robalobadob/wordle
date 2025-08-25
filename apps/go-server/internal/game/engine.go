// apps/go-server/internal/game/engine.go
//
// Core game engine for a single Wordle session.
// Responsibilities:
//   - Create new games with deterministic dimensions (6x5).
//   - Validate and apply guesses (length, alphabetic, allowed list).
//   - Score guesses using the classic two‑pass Wordle algorithm.
//   - Track state transitions: playing → won/lost.
//
// Notes:
//   - Answers/allowed lists are provided by the words package.
//   - Mark is an enum defined in this package (MarkHit/MarkPresent/MarkMiss).
//   - randomID() is a compact hex identifier for correlating server state.
//
// Package-level defaults are kept here for clarity.
package game

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/robalobadob/wordle/apps/go-server/internal/words"
)

const (
	defaultRows = 6
	defaultCols = 5
)

// New constructs a new game instance.
// If withAnswer is empty, a random answer is chosen from the words package.
func New(withAnswer string) *Game {
	ans := withAnswer
	if ans == "" {
		ans = words.RandomAnswer()
	}
	return &Game {
		ID:      randomID(),
		Answer:  strings.ToLower(ans),
		Rows:    defaultRows,
		Cols:    defaultCols,
		Guesses: []string{},
	}
}

// ApplyGuess validates and scores a guess, mutating the game state.
// Returns: the per‑letter marks, the new state string ("playing"/"won"/"lost"), or an error.
//
// Validation rules:
//   - Game must not be finished.
//   - Guess must be exactly g.Cols letters and alphabetic a–z.
//   - Guess must be present in the allowed list.
//
// State transitions:
//   - If all tiles are Hit → Finished = true, Won = true.
//   - Else if the number of guesses reaches g.Rows → Finished = true (loss).
func (g *Game) ApplyGuess(guess string) ([]Mark, string, error) {
	if g.Finished {
		return nil, g.state(), errors.New("game finished")
	}
	guess = strings.ToLower(strings.TrimSpace(guess))
	if len(guess) != g.Cols || !isAlpha(guess) {
		return nil, g.state(), errors.New("invalid guess")
	}
	if !words.IsAllowed(guess) {
		return nil, g.state(), errors.New("not in word list")
	}

	marks := scoreGuess(g.Answer, guess)
	g.Guesses = append(g.Guesses, guess)

	if allHit(marks) {
		g.Finished, g.Won = true, true
	} else if len(g.Guesses) >= g.Rows {
		g.Finished = true
	}
	return marks, g.state(), nil
}

// state reports a coarse string representation of the current game state.
func (g *Game) state() string {
	if g.Finished {
		if g.Won {
			return "won"
		}
		return "lost"
	}
	return "playing"
}

// scoreGuess implements the standard Wordle two‑pass scoring algorithm.
//
// Pass 1:
//   - Mark exact matches as Hit.
//   - Count remaining (non‑hit) answer letters by letter index.
//
// Pass 2:
//   - For each non‑hit guess letter: if there is remaining count for that letter,
//     mark Present and decrement the count; otherwise mark Miss.
//
// This ensures correct behavior with repeated letters in both answer and guess.
func scoreGuess(answer, guess string) []Mark {
	n := len(guess)
	res := make([]Mark, n)
	answerRunes := []rune(answer)
	guessRunes := []rune(guess)

	// Letter frequency for the non‑hit positions (a–z).
	var counts [26]int

	// First pass: mark hits and collect counts for remaining answer letters.
	for i := 0; i < n; i++ {
		if guessRunes[i] == answerRunes[i] {
			res[i] = MarkHit
		} else {
			counts[idx(answerRunes[i])]++
		}
	}

	// Second pass: resolve presents/misses for non‑hit tiles.
	for i := 0; i < n; i++ {
		if res[i] == MarkHit {
			continue
		}
		j := idx(guessRunes[i])
		if j >= 0 && j < 26 && counts[j] > 0 {
			res[i] = MarkPresent
			counts[j]--
		} else {
			res[i] = MarkMiss
		}
	}
	return res
}

// idx maps a lowercase ASCII letter rune to 0..25.
// Assumes inputs are validated to a–z elsewhere.
func idx(r rune) int { return int(r - 'a') }

// isAlpha checks that a string consists only of lowercase a–z.
func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// allHit returns true if all marks are MarkHit.
func allHit(m []Mark) bool {
	for _, x := range m {
		if x != MarkHit {
			return false
		}
	}
	return true
}

// randomID returns a compact 16‑hex‑char identifier.
// Collisions are extremely unlikely given crypto/rand entropy.
func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
