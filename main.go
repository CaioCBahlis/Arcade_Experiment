package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tetris_agents/internal/ui"
)

func main() {
	if _, err := tea.NewProgram(ui.New(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
