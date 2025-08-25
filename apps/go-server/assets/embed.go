// apps/go-server/assets/embed.go
//
// Provides access to embedded word lists used by the Wordle server.
// - answers.txt: canonical list of possible solutions
// - allowed.txt: all words allowed as guesses (superset of answers)
//
// Files are embedded at compile time using Go's embed.FS, so no external
// file access is required at runtime.

package assets

import (
	"bufio"
	"embed"
	"strings"
)

// FS holds the embedded file system containing word lists.
//
//go:embed allowed.txt answers.txt
var FS embed.FS

/**
 * readLines opens an embedded text file and returns a slice of normalized words.
 *
 * Behavior:
 *   - Trims leading/trailing whitespace.
 *   - Ignores blank lines.
 *   - Ignores comment lines starting with '#'.
 *   - Converts all words to lowercase for consistency.
 *
 * @param name file path within FS (e.g. "answers.txt")
 * @return slice of words, or error if file cannot be read.
 */
func readLines(name string) ([]string, error) {
	f, err := FS.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		out = append(out, strings.ToLower(s))
	}
	return out, sc.Err()
}

/**
 * AnswersList loads and returns the list of possible solution words.
 *
 * Source: answers.txt
 */
func AnswersList() ([]string, error) {
	return readLines("answers.txt")
}

/**
 * AllowedList loads and returns the list of all allowed guess words.
 *
 * Source: allowed.txt
 */
func AllowedList() ([]string, error) {
	return readLines("allowed.txt")
}
