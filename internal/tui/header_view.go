package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) headerView() string {
	styles := m.getStyles()

	bracket := styles.Maroon.Render("[")
	closeBracket := styles.Maroon.Render("]")

	timePart := bracket + styles.Maroon.Render(" Time: ") + styles.Subtext1.Render(time.Now().Format(m.config.Style.DateFormat)) + styles.Maroon.Render(" ") + closeBracket
	channelLabel := m.twitch.CurrentChannel
	if m.twitch.ChannelID != "" {
		channelLabel = fmt.Sprintf("%s (%s)", channelLabel, m.twitch.ChannelID)
	}
	userLabel := m.twitch.User
	if m.twitch.UserID != "" {
		userLabel = fmt.Sprintf("%s (%s)", userLabel, m.twitch.UserID)
	}
	channelPart := bracket + styles.Maroon.Render(" Channel: ") + styles.Green.Render(channelLabel) + styles.Maroon.Render(" ") + closeBracket
	userPart := bracket + styles.Maroon.Render(" User: ") + styles.Yellow.Render(userLabel) + styles.Maroon.Render(" ") + closeBracket
	findLabel := "Find"
	if m.filter != "" {
		findLabel = fmt.Sprintf("Find %q", m.filter)
	}
	findPart := bracket + styles.Maroon.Render(" "+findLabel+" ") + closeBracket

	dash := styles.Maroon.Render("─")
	dataLine := timePart + dash + channelPart + dash + userPart + dash + findPart

	dataLineWidth := lipgloss.Width(dataLine)

	remainingSpace := m.width - dataLineWidth
	if remainingSpace < 0 {
		remainingSpace = 0
	}
	separator := styles.Maroon.Render(strings.Repeat("─", remainingSpace))

	return dataLine + separator + "\n"
}
