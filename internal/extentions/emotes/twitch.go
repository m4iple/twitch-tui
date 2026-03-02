package emotes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	irc "github.com/gempir/go-twitch-irc/v4"
)

func addTwitchEmotesLink(text string, emotes []*irc.Emote, theme string) string {
	type segment struct {
		start int
		end   int
		id    string
		name  string
	}

	runes := []rune(text)
	var segments []segment

	for _, emote := range emotes {
		for _, pos := range emote.Positions {
			segments = append(segments, segment{
				start: pos.Start,
				end:   pos.End,
				id:    emote.ID,
				name:  string(runes[pos.Start : pos.End+1]),
			})
		}
	}

	sort.Slice(segments, func(i, j int) bool {
		return segments[i].start < segments[j].start
	})

	emoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme))

	var result strings.Builder
	cursor := 0

	for _, seg := range segments {
		result.WriteString(string(runes[cursor:seg.start]))

		cdnURL := fmt.Sprintf(
			"https://static-cdn.jtvnw.net/emoticons/v2/%s/default/dark/3.0",
			seg.id,
		)
		styledName := emoteStyle.Render(seg.name)
		result.WriteString(fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", cdnURL, styledName))

		cursor = seg.end + 1
	}

	result.WriteString(string(runes[cursor:]))

	return result.String()
}
