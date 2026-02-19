package tui

import (
	"fmt"
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
		}
		flarePart = bracket + flareStyle.Render(msg.Flare) + closeBracket + " "
	}

	var userStr string
	if msg.Flare == "SYSTEM" {
		userStr = styles.Yellow.Render(msg.User)
	} else if msg.NameColor != "" {
		userStr = lipgloss.NewStyle().Foreground(lipgloss.Color(msg.NameColor)).Render(msg.User)
	} else {
		userStr = m.randomFallbackStyle(styles).Render(msg.User)
	}

	contentPart := msg.Content
	if msg.TaggedUser != "" {
		taggedPattern := "@" + msg.TaggedUser
		var taggedStyle lipgloss.Style
		if msg.TaggedColor != "" {
			taggedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(msg.TaggedColor))
		} else {
			taggedStyle = m.randomFallbackStyle(styles)
		}
		contentPart = strings.ReplaceAll(contentPart, taggedPattern, taggedStyle.Render(taggedPattern))
	}

	var bitsPart string
	if msg.Bits > 0 {
		bitsPart = " " + styles.Peach.Render(fmt.Sprintf("Cheer%d", msg.Bits))
	}

	prefixLen := lipgloss.Width(timePart) + lipgloss.Width(flarePart) + lipgloss.Width(userStr) + 2
	availableWidth := m.width - prefixLen - lipgloss.Width(bitsPart)

	if availableWidth < 20 {
		availableWidth = 20
	}

	wrappedContent := m.wrapText(contentPart, availableWidth)
	lines := strings.Split(wrappedContent, "\n")

	var result strings.Builder
	for i, line := range lines {
		if i == 0 {
			result.WriteString(timePart + " " + flarePart + userStr + styles.Text.Render(": ") + styles.Text.Render(line) + bitsPart + "\n")
		} else {
			indent := strings.Repeat(" ", prefixLen+1)
			result.WriteString(indent + styles.Text.Render(line) + "\n")
		}
	}

	return result.String()
}
