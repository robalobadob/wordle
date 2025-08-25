// apps/go-server/internal/words/daily_exports.go
//
// Provides word lists and scoring utilities for the Daily Challenge mode.
// Wraps assets.AnswersList/AllowedList and exposes:
//   - Answers(): canonical list of valid answers
//   - Allowed(): set of all valid guesses (answers ⊆ allowed)
//   - Score():   Wordle-style evaluation (miss=0, present=1, hit=2)
//
// Notes:
//   • Data is lazily initialized once via sync.Once, reading from embedded files.
//   • Answers and Allowed are lowercase for consistency.
//   • Score implements the two-pass Wordle algorithm.

package words

import (
	"sync"

	"github.com/robalobadob/wordle/apps/go-server/assets"
)

var (
	dailyOnce    sync.Once          // ensures initDaily runs once
	dailyAnswers []string           // list of valid answers
	dailyAllowed map[string]struct{} // set of allowed guesses
	dailyInitErr error              // init error, if any
)

// initDaily loads answer and allowed word lists into memory.
// Called once on first access.
func initDaily() {
	dailyAllowed = make(map[string]struct{})

	// Load canonical answer list
	ans, err := assets.AnswersList()
	if err != nil {
		dailyInitErr = err
		return
	}
	dailyAnswers = ans

	// Load allowed guess list
	all, err := assets.AllowedList()
	if err != nil {
		dailyInitErr = err
		return
	}

	// Build guess set: include both allowed + answers
	for _, w := range all {
		dailyAllowed[w] = struct{}{}
	}
	for _, w := range dailyAnswers {
		dailyAllowed[w] = struct{}{}
	}
}

// Answers returns the canonical answer list (all lowercase).
func Answers() []string {
	dailyOnce.Do(initDaily)
	return dailyAnswers
}

// Allowed returns the allowed guess set (all lowercase).
// Answers are always included for safety.
func Allowed() map[string]struct{} {
	dailyOnce.Do(initDaily)
	return dailyAllowed
}

// Score compares guess vs. answer and returns a slice of ints:
//   0 = miss (letter not in answer)
//   1 = present (letter in answer, wrong position)
//   2 = hit (letter in correct position)
//
// Implements the standard two-pass Wordle scoring:
//   Pass 1: mark exact matches (hits) and count remaining letters.
//   Pass 2: for non-hits, mark present if unused letters remain.
func Score(guess, answer string) []int {
	n := len(answer)
	out := make([]int, n)
	if len(guess) != n {
		return out
	}

	// Pass 1: hits and frequency counts
	freq := make(map[byte]int, n)
	for i := 0; i < n; i++ {
		if guess[i] == answer[i] {
			out[i] = 2 // hit
		} else {
			freq[answer[i]]++
		}
	}

	// Pass 2: mark presents where applicable
	for i := 0; i < n; i++ {
		if out[i] == 2 {
			continue
		}
		c := guess[i]
		if freq[c] > 0 {
			out[i] = 1 // present
			freq[c]--
		}
	}
	return out
}
