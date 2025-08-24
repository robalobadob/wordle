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

// Embed tiny defaults so the server runs even without external files.
// Replace these with your own or load from files via env vars.
//
//go:embed default_small_answers.txt
var embeddedAnswers string

//go:embed default_small_allowed.txt
var embeddedAllowed string

var (
	initOnce   sync.Once
	answers    []string
	allowedSet map[string]struct{} // answers ∪ guesses
	answersSet map[string]struct{} // answers only
	initialErr error
)

// Init loads wordlists once. If env vars are set, it loads those files.
// Otherwise it falls back to the embedded defaults.
//
// Env vars (absolute or relative paths):
//   WORDS_ANSWERS_FILE=/path/to/answers.txt
//   WORDS_ALLOWED_FILE=/path/to/allowed.txt
//
// Behavior:
//   - both set: uses answers.txt for solutions, allowed.txt for guesses
//   - only WORDS_ALLOWED_FILE set: uses that file for BOTH answers and guesses
//   - neither set: uses embedded tiny defaults
func Init() error {
	initOnce.Do(func() {
		var ansList, allowList []string

		answersPath := os.Getenv("WORDS_ANSWERS_FILE")
		allowedPath := os.Getenv("WORDS_ALLOWED_FILE")

		switch {
		// Case 1: both lists provided explicitly
		case answersPath != "" && allowedPath != "":
			var err error
			ansList, err = readWordFile(answersPath)
			if err != nil { initialErr = err; return }
			allowList, err = readWordFile(allowedPath)
			if err != nil { initialErr = err; return }

		// Case 2: single merged list provided
		case answersPath == "" && allowedPath != "":
			var err error
			allowList, err = readWordFile(allowedPath)
			if err != nil { initialErr = err; return }
			// Use the same list for answers & allowed
			ansList = allowList

		// Case 3: fall back to embedded defaults
		default:
			ansList = normalizeLines(embeddedAnswers)
			if embeddedAllowed != "" {
				allowList = normalizeLines(embeddedAllowed)
			} else {
				allowList = ansList // ensure allowed includes answers at minimum
			}
		}

		answers = ansList
		answersSet = toSet(ansList)

		allowedSet = toSet(append([]string{}, ansList...)) // answers are always allowed
		for _, w := range allowList {
			allowedSet[w] = struct{}{}
		}

		if len(answers) == 0 {
			initialErr = errors.New("words: answers list is empty")
		}
	})
	return initialErr
}

func readWordFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil { return nil, err }
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		w := strings.TrimSpace(strings.ToLower(sc.Text()))
		if len(w) == 5 && isAlpha(w) { out = append(out, w) }
	}
	return out, sc.Err()
}

func normalizeLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		w := strings.TrimSpace(strings.ToLower(line))
		if len(w) == 5 && isAlpha(w) { out = append(out, w) }
	}
	return out
}

func toSet(list []string) map[string]struct{} {
	m := make(map[string]struct{}, len(list))
	for _, w := range list { m[w] = struct{}{} }
	return m
}

func isAlpha(s string) bool {
	for _, r := range s {
		if r < 'a' || r > 'z' { return false }
	}
	return true
}

// RandomAnswer returns a cryptographically random answer from the answers list.
func RandomAnswer() string {
	if len(answers) == 0 { return "crane" } // last‑ditch fallback
	nBig, _ := rand.Int(rand.Reader, big.NewInt(int64(len(answers))))
	return answers[nBig.Int64()]
}

func IsAllowed(w string) bool {
	_, ok := allowedSet[strings.ToLower(w)]
	return ok
}

func IsAnswer(w string) bool {
	_, ok := answersSet[strings.ToLower(w)]
	return ok
}

// Stats returns the number of answers and allowed words currently loaded.
func Stats() (answersCount int, allowedCount int) {
	return len(answers), len(allowedSet)
}
