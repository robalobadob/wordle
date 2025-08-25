// apps/go-server/internal/words/words.go
//
// Provides word list management for the game engine.
//
// Responsibilities:
//   - Load answer and allowed guess lists from environment-provided files or fall back to embedded defaults.
//   - Maintain sets for quick lookups (answers only, answers∪guesses).
//   - Supply utility functions like RandomAnswer, IsAllowed, IsAnswer, and Stats.
//
// Word Lists:
//   - "answers": canonical solutions (exactly 5 lowercase letters).
//   - "allowed": valid guesses (always includes answers).
//
// Initialization behavior (Init):
//   1. If WORDS_ANSWERS_FILE and WORDS_ALLOWED_FILE are both set,
//      load answers from the first and allowed guesses from the second.
//   2. If only WORDS_ALLOWED_FILE is set,
//      load that file and use it for both answers and allowed guesses.
//   3. If neither is set,
//      fall back to small embedded defaults from `default_small_answers.txt`
//      and `default_small_allowed.txt` (if present).
//
// Environment variables:
//   WORDS_ANSWERS_FILE=/path/to/answers.txt
//   WORDS_ALLOWED_FILE=/path/to/allowed.txt
//
// Constraints:
//   • Words must be 5 alphabetic letters (a–z).
//   • Lists are normalized to lowercase.
//   • Initialization is run once (sync.Once).

package words

import (
	"bufio"
	"crypto/rand"
	_ "embed"
	"errors"
	"math/big"
	"os"
	"strings"
	"sync"
)

// --- embedded tiny defaults (ensures server runs even if no files configured) ---

//go:embed default_small_answers.txt
var embeddedAnswers string

//go:embed default_small_allowed.txt
var embeddedAllowed string

var (
	initOnce   sync.Once
	answers    []string           // canonical answers
	allowedSet map[string]struct{} // answers ∪ guesses
	answersSet map[string]struct{} // answers only
	initialErr error
)

// Init loads word lists exactly once.
// Returns an error if the answers list ends up empty.
func Init() error {
	initOnce.Do(func() {
		var ansList, allowList []string

		answersPath := os.Getenv("WORDS_ANSWERS_FILE")
		allowedPath := os.Getenv("WORDS_ALLOWED_FILE")

		switch {
		// Case 1: both lists provided
		case answersPath != "" && allowedPath != "":
			var err error
			ansList, err = readWordFile(answersPath)
			if err != nil {
				initialErr = err
				return
			}
			allowList, err = readWordFile(allowedPath)
			if err != nil {
				initialErr = err
				return
			}

		// Case 2: only allowed file provided → use for both
		case answersPath == "" && allowedPath != "":
			var err error
			allowList, err = readWordFile(allowedPath)
			if err != nil {
				initialErr = err
				return
			}
			ansList = allowList

		// Case 3: fallback to embedded defaults
		default:
			ansList = normalizeLines(embeddedAnswers)
			if embeddedAllowed != "" {
				allowList = normalizeLines(embeddedAllowed)
			} else {
				allowList = ansList
			}
		}

		answers = ansList
		answersSet = toSet(ansList)

		// Ensure all answers are also marked as allowed
		allowedSet = toSet(append([]string{}, ansList...))
		for _, w := range allowList {
			allowedSet[w] = struct{}{}
		}

		if len(answers) == 0 {
			initialErr = errors.New("words: answers list is empty")
		}
	})
	return initialErr
}

// readWordFile loads one word per line from a file,
// lowercases, trims, and keeps only valid 5-letter alphabetic words.
func readWordFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		w := strings.TrimSpace(strings.ToLower(sc.Text()))
		if len(w) == 5 && isAlpha(w) {
			out = append(out, w)
		}
	}
	return out, sc.Err()
}

// normalizeLines processes an embedded multiline string
// into a slice of valid lowercase 5-letter words.
func normalizeLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		w := strings.TrimSpace(strings.ToLower(line))
		if len(w) == 5 && isAlpha(w) {
			out = append(out, w)
		}
	}
	return out
}

// toSet converts a list of strings into a lookup set.
func toSet(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, w := range list {
		m[w] = struct{}{}
	}
	return m
}

// isAlpha reports whether s is all lowercase ASCII letters.
func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}

// RandomAnswer returns a cryptographically random answer from the answers list.
// If answers are not loaded yet or empty, falls back to "crane".
func RandomAnswer() string {
	if len(answers) == 0 {
		return "crane"
	}
	nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(answers))))
	return answers[nBig.Int64()]
}

// IsAllowed reports whether w is a valid guess (answers ∪ guesses).
func IsAllowed(w string) bool {
	_, ok := allowedSet[strings.ToLower(w)]
	return ok
}

// IsAnswer reports whether w is an answer word.
func IsAnswer(w string) bool {
	_, ok := answersSet[strings.ToLower(w)]
	return ok
}

// Stats returns counts of loaded words: (answers, allowed).
func Stats() (answersCount int, allowedCount int) {
	return len(answers), len(allowedSet)
}
