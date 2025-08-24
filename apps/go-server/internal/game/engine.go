package game

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
)

const (
	defaultRows = 6
	defaultCols = 5
)

func New(withAnswer string) *Game {
	ans := withAnswer
	if ans == "" {
		// TODO: swap with shared word list picker; for now a fixed answer helps dev.
		ans = "crane"
	}
	return &Game{
		ID:      randomID(),
		Answer:  strings.ToLower(ans),
		Rows:    defaultRows,
		Cols:    defaultCols,
		Guesses: []string{},
	}
}

func (g *Game) ApplyGuess(guess string) ([]Mark, string, error) {
	if g.Finished {
		return nil, g.state(), errors.New("game finished")
	}
	guess = strings.ToLower(strings.TrimSpace(guess))
	if len(guess) != g.Cols || !isAlpha(guess) {
		return nil, g.state(), errors.New("invalid guess")
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

func (g *Game) state() string {
	if g.Finished {
		if g.Won { return "won" }
		return "lost"
	}
	return "playing"
}

func scoreGuess(answer, guess string) []Mark {
	// Two‑pass algorithm (classic Wordle):
	// 1) Mark exact hits and count remaining letters in answer.
	// 2) For non‑hits, mark present if count > 0, else miss.
	n := len(guess)
	res := make([]Mark, n)
	answerRunes := []rune(answer)
	guessRunes := []rune(guess)

	// counts by letter (a-z)
	var counts [26]int
	for i := 0; i < n; i++ {
		if guessRunes[i] == answerRunes[i] {
			res[i] = MarkHit
		} else {
			counts[idx(answerRunes[i])]++
		}
	}
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

func idx(r rune) int { return int(r - 'a') }

func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

func allHit(m []Mark) bool {
	for _, x := range m {
		if x != MarkHit { return false }
	}
	return true
}

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
