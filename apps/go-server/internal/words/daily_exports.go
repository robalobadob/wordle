// apps/go-server/internal/words/daily_exports.go
package words

import (
	"sync"

	"github.com/robalobadob/wordle/apps/go-server/assets"
)

var (
	dailyOnce    sync.Once
	dailyAnswers []string
	dailyAllowed map[string]struct{}
	dailyInitErr error
)

func initDaily() {
	dailyAllowed = make(map[string]struct{})

	ans, err := assets.AnswersList()
	if err != nil {
		dailyInitErr = err
		return
	}
	dailyAnswers = ans

	all, err := assets.AllowedList()
	if err != nil {
		dailyInitErr = err
		return
	}
	// Build the set. Include answers in allowed for safety.
	for _, w := range all {
		dailyAllowed[w] = struct{}{}
	}
	for _, w := range dailyAnswers {
		dailyAllowed[w] = struct{}{}
	}
}

// Answers returns the canonical answer list (lowercase).
func Answers() []string {
	dailyOnce.Do(initDaily)
	return dailyAnswers
}

// Allowed returns the allowed guess set (lowercase).
func Allowed() map[string]struct{} {
	dailyOnce.Do(initDaily)
	return dailyAllowed
}

// Score computes 0=miss, 1=present, 2=hit for guess vs answer.
func Score(guess, answer string) []int {
	n := len(answer)
	out := make([]int, n)
	if len(guess) != n {
		return out
	}
	// hits pass + freq of remaining
	freq := make(map[byte]int, n)
	for i := 0; i < n; i++ {
		if guess[i] == answer[i] {
			out[i] = 2
		} else {
			freq[answer[i]]++
		}
	}
	// present pass
	for i := 0; i < n; i++ {
		if out[i] == 2 {
			continue
		}
		c := guess[i]
		if freq[c] > 0 {
			out[i] = 1
			freq[c]--
		}
	}
	return out
}
