package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tetris_agents/internal/agent"
	"tetris_agents/internal/game"
)

const (
	// raceInterval is faster than a watch-along game so a race stays short.
	raceInterval = 30 * time.Millisecond
	// raceLength is how many pieces each agent places if it survives.
	raceLength = 200
)

type racer struct {
	info agent.Info
	g    *game.Game
	ag   agent.Agent
}

func (r *racer) placed() int { return r.g.Spawned - 1 }

func (r *racer) finished() bool { return r.g.Over || r.placed() >= raceLength }

// Race runs every agent on the same piece sequence, side by side.
type Race struct {
	racers []*racer
	paused bool
	gen    int
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

func (r Race) tick() tea.Cmd {
	gen := r.gen
	return tea.Tick(raceInterval, func(time.Time) tea.Msg { return tickMsg(gen) })
}

func (r Race) Init() tea.Cmd { return r.tick() }

func (r Race) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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

const racerWidth = 2*game.Width + 2 // board cells plus the border

var (
	colStyle    = lipgloss.NewStyle().Width(racerWidth).MarginRight(2)
	leaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("220"))
)

func (r Race) View() string {
	rank := r.ranking()
	place := map[*racer]int{}
	for i, rc := range rank {
		place[rc] = i + 1
	}
	done := r.done()

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
		case rc.finished():
			status = dimStyle.Render("finished")
		}
		col := lipgloss.JoinVertical(lipgloss.Left,
			head,
			renderBoard(rc.g, false),
			fmt.Sprintf("Score %d", rc.g.Score),
			fmt.Sprintf("Lines %d  Piece %d", rc.g.Lines, min(rc.placed(), raceLength)),
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
		foot = dimStyle.Render(fmt.Sprintf("%d pieces each, or until topped out · p: pause · r: restart · esc: menu", raceLength))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cols...) + "\n" + strings.TrimRight(foot, "\n") + "\n"
}
