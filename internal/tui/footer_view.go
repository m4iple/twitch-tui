package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) footerView() string {
	styles := m.getStyles()

	bracket := styles.Maroon.Render("[")
	closeBracket := styles.Maroon.Render("]")

	// Dynamic label based on state
	var inputLabel string
	switch m.state {
	case stateInputChannel:
		inputLabel = "Channel"
	case stateInputCommand:
		inputLabel = "Command"
	case stateChat:
		inputLabel = "Chat"
	default:
		inputLabel = "Input"
	}

	chatPart := bracket + styles.Maroon.Render(inputLabel) + closeBracket
	countPart := bracket + styles.Yellow.Render(fmt.Sprintf("%d", len(m.textInput.Value()))) + styles.Maroon.Render(" / ") + styles.Yellow.Render("500") + closeBracket

	dash := styles.Maroon.Render("─")
	infoLine := chatPart + dash + countPart

	// Calculate visible width using lipgloss.Width which handles ANSI codes
	infoLineWidth := lipgloss.Width(infoLine)
	remainingSpace := m.width - infoLineWidth
	if remainingSpace < 0 {
		remainingSpace = 0
	}
	separator := styles.Maroon.Render(strings.Repeat("─", remainingSpace))

	infoLineWithSeparator := infoLine + separator

	inputLine := m.textInput.View()

	return infoLineWithSeparator + "\n" + inputLine
}
