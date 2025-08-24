package game

type Mark string

const (
	MarkHit    Mark = "hit"
	MarkPresent     = "present"
	MarkMiss        = "miss"
)

type Game struct {
	ID       string
	Answer   string
	Rows     int
	Cols     int
	Guesses  []string
	Finished bool
	Won      bool
}
