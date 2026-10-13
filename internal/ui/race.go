package ui

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tetris_agents/internal/agent"
	"tetris_agents/internal/game"
)

const (
	// raceInterval is the delay between inputs at the start of a race.
	raceInterval = 30 * time.Millisecond
	// raceMinInterval is the fastest the race ever runs.
	raceMinInterval = 10 * time.Millisecond
	// roundPieces is how many pieces the leader places per round; the race
	// speeds up by raceSpeedup at the start of each new round.
	roundPieces = 25
	raceSpeedup = 0.93
)

type racer struct {
	info agent.Info
	g    *game.Game
	ag   agent.Agent
}

func (r *racer) placed() int { return r.g.Spawned - 1 }

func (r *racer) finished() bool { return r.g.Over }

// Race runs every agent on the same piece sequence, side by side.
type Race struct {
	racers []*racer
	paused bool
	gen    int
	w, h   int // terminal size; zero until the first WindowSizeMsg
}

// Rows of text around each board (name above; score, lines and status
// below) and shared footer lines under all boards.
const (
	raceRowChrome = 1 + 3
	raceFooter    = 2 + 1
)

// perRow is how many boards fit side by side at scale s, with one space
// between boards. Boards that do not fit wrap onto further rows.
func (r Race) perRow(s int) int {
	n := len(r.racers)
	if r.w <= 0 {
		return n
	}
	return max(1, min(n, (r.w+1)/(boardChars(s)+1)))
}

// scale picks the largest board scale at which all racers fit on screen.
func (r Race) scale() int {
	n := len(r.racers)
	return fitScale(r.w, r.h, func(s int) (int, int) {
		per := r.perRow(s)
		rows := (n + per - 1) / per
		bc := boardChars(s)
		return per*bc + per - 1, rows*(bc+raceRowChrome) + raceFooter
	})
}

// NewRace starts a race between all available agents. Every agent's game
// uses the same seed, so they receive identical pieces.
func NewRace() Race {
	seed := time.Now().UnixNano()
	r := Race{gen: nextGen()}
	for _, info := range agent.All() {
		r.racers = append(r.racers, &racer{info: info, g: game.New(seed), ag: info.New()})
	}
	return r
}

func (r Race) done() bool {
	for _, rc := range r.racers {
		if !rc.finished() {
			return false
		}
	}
	return true
}

// round is 1 at the start and advances every roundPieces pieces placed by the
// furthest-along racer.
func (r Race) round() int {
	most := 0
	for _, rc := range r.racers {
		most = max(most, rc.placed())
	}
	return most/roundPieces + 1
}

// interval is the current delay between inputs, shrinking each round.
func (r Race) interval() time.Duration {
	d := float64(raceInterval) * math.Pow(raceSpeedup, float64(r.round()-1))
	return max(time.Duration(d), raceMinInterval)
}

func (r Race) tick() tea.Cmd {
	gen := r.gen
	return tea.Tick(r.interval(), func(time.Time) tea.Msg { return tickMsg(gen) })
}

func (r Race) Init() tea.Cmd { return r.tick() }

func (r Race) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		r.w, r.h = msg.Width, msg.Height
		return r, nil

	case tickMsg:
		if int(msg) != r.gen || r.paused || r.done() {
			return r, nil
		}
		for _, rc := range r.racers {
			if rc.finished() {
				continue
			}
			if !agent.Apply(rc.g, rc.ag.Act(rc.g)) {
				rc.g.HardDrop()
			}
		}
		if r.done() {
			return r, nil
		}
		return r, r.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return r, tea.Quit
		case "esc":
			return r, func() tea.Msg { return backMsg{} }
		case "r":
			n := NewRace()
			n.w, n.h = r.w, r.h
			return n, n.tick()
		case "p":
			if r.done() {
				return r, nil
			}
			r.paused = !r.paused
			r.gen = nextGen()
			if r.paused {
				return r, nil
			}
			return r, r.tick()
		}
	}
	return r, nil
}

// ranking returns the racers ordered best first: by score, then lines.
func (r Race) ranking() []*racer {
	out := append([]*racer(nil), r.racers...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].g.Score != out[j].g.Score {
			return out[i].g.Score > out[j].g.Score
		}
		return out[i].g.Lines > out[j].g.Lines
	})
	return out
}

var leaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))

func (r Race) View() string {
	rank := r.ranking()
	place := map[*racer]int{}
	for i, rc := range rank {
		place[rc] = i + 1
	}
	done := r.done()
	scale := r.scale()
	colStyle := lipgloss.NewStyle().Width(boardChars(scale))

	var cols []string
	for _, rc := range r.racers {
		head := fmt.Sprintf("#%d %s", place[rc], rc.info.Name)
		if place[rc] == 1 {
			head = leaderStyle.Render(head)
		} else {
			head = titleStyle.Render(head)
		}
		status := dimStyle.Render("racing")
		switch {
		case rc.g.Over:
			status = overStyle.Render("TOPPED OUT")
		}
		col := lipgloss.JoinVertical(lipgloss.Left,
			head,
			renderBoard(rc.g, false, scale),
			fmt.Sprintf("Score %d", rc.g.Score),
			fmt.Sprintf("Lines %d  Piece %d", rc.g.Lines, rc.placed()),
			status,
		)
		cols = append(cols, colStyle.Render(col))
	}

	var foot string
	switch {
	case done:
		w := rank[0]
		foot = leaderStyle.Render(fmt.Sprintf("Winner: %s with %d points", w.info.Name, w.g.Score)) +
			"\n" + dimStyle.Render("r: new race · esc: menu · q: quit")
	case r.paused:
		foot = titleStyle.Render("PAUSED") + dimStyle.Render("  p: resume · esc: menu")
	default:
		speed := float64(raceInterval) / float64(r.interval())
		foot = dimStyle.Render(fmt.Sprintf("Round %d · %.1fx speed · until everyone tops out · p: pause · r: restart · esc: menu", r.round(), speed))
	}
	// Boards that do not fit side by side wrap onto further rows.
	per := r.perRow(scale)
	var rows []string
	for i := 0; i < len(cols); i += per {
		var line []string
		for j := i; j < min(i+per, len(cols)); j++ {
			if j > i {
				line = append(line, " ")
			}
			line = append(line, cols[j])
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, line...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...) + "\n" + strings.TrimRight(foot, "\n") + "\n"
}
