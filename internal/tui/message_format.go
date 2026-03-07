package tui

import (
	"strings"
	"time"

	"twitch-tui/internal/twitch"

	"github.com/charmbracelet/lipgloss"
)

// format system message as if it is a chat message
func formatSystemMessage(content string) twitch.ChatMessage {
	return twitch.ChatMessage{
		Time:    time.Now(),
		User:    "System",
		Flare:   "SYSTEM",
		Content: content,
	}
}

func (m Model) formatMessage(msg twitch.ChatMessage) string {
	styles := m.getStyles()

	timePart := styles.Subtext1.Render(msg.Time.Format(m.config.Style.DateFormat))

	// handle flares
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

	// handle the user string and color
	var userStr string
	if msg.Flare == "SYSTEM" {
		userStr = styles.Yellow.Render(msg.User)
	} else {
		userStr = lipgloss.NewStyle().Foreground(lipgloss.Color(msg.NameColor)).Render(msg.User)
	}

	// combine the the @ users and their colors
	contentPart := msg.Content
	for _, taggedUser := range msg.TaggedUsers {
		taggedPattern := "@" + taggedUser
		taggedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(msg.TaggedColors[taggedUser]))
		contentPart = strings.ReplaceAll(contentPart, taggedPattern, taggedStyle.Render(taggedPattern))
	}

	// add style to the pepend text ie. - Cheer100 -
	prependPart := ""
	if msg.Prepend != "" {
		prependPart = styles.Red.Render(msg.Prepend + " ")
	}

	// calcualte the avaiable space for the text
	prefixLen := lipgloss.Width(timePart) + lipgloss.Width(flarePart) + lipgloss.Width(userStr) + lipgloss.Width(prependPart) + 2
	availableWidth := max(m.width-prefixLen, 20)
	// split the text according to the avaiable width
	wrappedContent := m.wrapText(contentPart, availableWidth)
	lines := strings.Split(wrappedContent, "\n")

	var result strings.Builder
	for i, line := range lines {
		var styledLine string
		// apply background styles
		if msg.Highlight != "" {
			styledLine = m.applyHighlightWithEmotes(line, msg.Highlight)
		} else {
			styledLine = styles.Text.Render(line)
		}

		if i == 0 {
			result.WriteString(timePart + " " + flarePart + userStr + styles.Text.Render(": ") + prependPart + styledLine + "\n")
		} else {
			indent := strings.Repeat(" ", prefixLen+1) // print spaces so we start at the same position as the first text part
			result.WriteString(indent + styledLine + "\n")
		}
	}

	return result.String()
}

// handle text warap on too long mesages
func (m Model) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine strings.Builder // empty line

	for _, word := range words {
		wordWidth := lipgloss.Width(word)
		currentWidth := lipgloss.Width(currentLine.String())

		if currentLine.Len() == 0 { // add first wor to the virtual line
			currentLine.WriteString(word)
		} else if currentWidth+1+wordWidth <= width { // the length of the current virtual line + the new word (and space) is sitll in bounds of the viwport widht
			currentLine.WriteString(" " + word)
		} else { // if we need to break the text
			lines = append(lines, currentLine.String()) // add our curent virtual line tho the array
			currentLine.Reset()                         // reset our virtual line
			currentLine.WriteString(word)               // add the current word to the resetted line
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

// applyHighlightWithEmotes applies a background highlight while preserving embedded emote
// escape sequences. It handles the issue where emote styling reset codes would interrupt
// the background color by re-applying the background after each reset.
func (m Model) applyHighlightWithEmotes(line string, highlightColor string) string {
	if highlightColor == "" {
		return line
	}

	// set highlight style
	highlightStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(highlightColor)).
		Foreground(lipgloss.Color(m.config.Theme.Base))

	// check if we have a emote (colored text with link)
	containsEmotes := strings.Contains(line, "\x1b]8;;") // check for hyperlink
	if !containsEmotes {
		return highlightStyle.Render(line) // render highlight
	}

	result := highlightStyle.Render(line) // render highlight

	if strings.HasSuffix(result, "\x1b[0m") { // string ends with the "clear format" bytes
		content := result[:len(result)-4] // remove the bytes

		reapplyStyle := highlightStyle.Render("") // render highlight for an empty string (we get the start / end codes)

		reapplyCodes := reapplyStyle[:len(reapplyStyle)-4] // we reomve the clear code again, but fromt the empty string so we only get the background format start

		content = strings.ReplaceAll(content, "\x1b[0m", "\x1b[0m"+reapplyCodes) // everytime we have an clear code - after an emote apply the background fromat after it

		result = content + "\x1b[0m" // add the clear code back to the string
	}

	return result
}
