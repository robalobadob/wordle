package assets

import (
	"bufio"
	"embed"
	"strings"
)

//go:embed allowed.txt answers.txt
var FS embed.FS

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

func AnswersList() ([]string, error) {
	return readLines("answers.txt")
}

func AllowedList() ([]string, error) {
	return readLines("allowed.txt")
}
