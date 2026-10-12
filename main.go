package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"tetris_agents/internal/agent"
	"tetris_agents/internal/ui"
)

func main() {
	go agent.WarmRL() // train the RL agent while the menu is showing
	if _, err := tea.NewProgram(ui.NewApp(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
