package main

import (
	"log"
	"twitch-tui/internal/config"
	"twitch-tui/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	cfg := config.Load()

	model := tui.New(cfg)
	program := tea.NewProgram(&model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		log.Fatal(err)
	}
}
