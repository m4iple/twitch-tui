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

	var result strings.Builder
	for i, line := range lines {
		var styledLine string
		if msg.Highlight != "" {
			styledLine = m.applyHighlightWithEmotes(line, msg.Highlight)
		} else {
			styledLine = styles.Text.Render(line)
		}

		if i == 0 {
			result.WriteString(timePart + " " + flarePart + userStr + styles.Text.Render(": ") + prependPart + styledLine + "\n")
		} else {
			indent := strings.Repeat(" ", prefixLen+1)
			result.WriteString(indent + styledLine + "\n")
		}
	}

	return result.String()
}

// applyHighlightWithEmotes applies a background highlight while preserving embedded emote
// escape sequences. It handles the issue where emote styling reset codes would interrupt
// the background color by re-applying the background after each reset.
func (m Model) applyHighlightWithEmotes(line string, highlightColor string) string {
	if highlightColor == "" {
		return line
	}

	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(highlightColor)).
		Foreground(lipgloss.Color(m.config.Theme.Base))

	containsEmotes := strings.Contains(line, "\x1b]8;;")

	if !containsEmotes {
		return highlightStyle.Render(line)
	}

	result := highlightStyle.Render(line)

	if strings.HasSuffix(result, "\x1b[0m") {
		content := result[:len(result)-4]

		reapplyStyle := highlightStyle.Render("")

		reapplyCodes := reapplyStyle[:len(reapplyStyle)-4]

		content = strings.ReplaceAll(content, "\x1b[0m", "\x1b[0m"+reapplyCodes)

		result = content + "\x1b[0m"
	}

	return result
}
