package tui

import (
	"strings"

	"twitch-tui/internal/twitch"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) formatMessage(msg twitch.ChatMessage) string {
	styles := m.getStyles()

	timePart := styles.Subtext1.Render(msg.Time.Format(m.config.Style.DateFormat))

	var flarePart string
	if msg.Flare != "" {
		bracket := styles.Maroon.Render("[")
		closeBracket := styles.Maroon.Render("]")
		flareStyle := styles.Red
		switch msg.Flare {
		case "VIP":
			flareStyle = styles.Pink
		case "SYSTEM":
			flareStyle = styles.Yellow
		case "TUI":
			flareStyle = styles.Blue
		case "REDEEM":
			flareStyle = styles.Teal
		}
		flarePart = bracket + flareStyle.Render(msg.Flare) + closeBracket + " "
	}

	var userStr string
	if msg.Flare == "SYSTEM" {
		userStr = styles.Yellow.Render(msg.User)
	} else {
		userStr = lipgloss.NewStyle().Foreground(lipgloss.Color(msg.NameColor)).Render(msg.User)
	}

	contentPart := msg.Content
	for _, taggedUser := range msg.TaggedUsers {
		taggedPattern := "@" + taggedUser
		taggedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(msg.TaggedColors[taggedUser]))
		contentPart = strings.ReplaceAll(contentPart, taggedPattern, taggedStyle.Render(taggedPattern))
	}

	prependPart := ""
	if msg.Prepend != "" {
		prependPart = styles.Red.Render(msg.Prepend + " ")
	}

	prefixLen := lipgloss.Width(timePart) + lipgloss.Width(flarePart) + lipgloss.Width(userStr) + lipgloss.Width(prependPart) + 2
	availableWidth := max(m.width-prefixLen, 20)

	wrappedContent := m.wrapText(contentPart, availableWidth)
	lines := strings.Split(wrappedContent, "\n")

	contentStyle := styles.Text
	if msg.Highlight != "" {
		contentStyle = lipgloss.NewStyle().Background(lipgloss.Color(msg.Highlight)).Foreground(lipgloss.Color(m.config.Theme.Base))
	}

	var result strings.Builder
	for i, line := range lines {
		if i == 0 {
			result.WriteString(timePart + " " + flarePart + userStr + styles.Text.Render(": ") + prependPart + contentStyle.Render(line) + "\n")
		} else {
			indent := strings.Repeat(" ", prefixLen+1)
			result.WriteString(indent + contentStyle.Render(line) + "\n")
		}
	}

	return result.String()
}
