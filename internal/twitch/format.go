package twitch

import (
	"fmt"
	"strings"
	"twitch-tui/internal/extentions/api"

	"github.com/gempir/go-twitch-irc/v4"
)

func (s *Service) formatMessage(msg twitch.PrivateMessage) ChatMessage {
	flare := resolveFlare(msg)

	taggedUsers, taggedColors := extractTags(msg.Message, s.randomColor)

	nameColor := msg.User.Color
	if nameColor == "" {
		nameColor = s.randomColor()
	}

	highlight, prepend := resolveHighlight(msg, nameColor, s)

	return ChatMessage{
		Time:         msg.Time,
		User:         msg.User.Name,
		Content:      msg.Message,
		Flare:        flare,
		NameColor:    nameColor,
		TaggedUsers:  taggedUsers,
		TaggedColors: taggedColors,
		Bits:         msg.Bits,
		Highlight:    highlight,
		Prepend:      prepend,
	}
}

func (s *Service) formatUserNotice(msg twitch.UserNoticeMessage) (ChatMessage, bool) {
	switch msg.MsgID {
	case "sub", "resub", "subgift", "anonsubgift", "submysterygift",
		"giftpaidupgrade", "anongiftpaidupgrade", "shoutout-received", "shoutout-sent":
	default:
		return ChatMessage{}, false
	}

	nameColor := msg.User.Color
	if nameColor == "" {
		nameColor = s.randomColor()
	}

	content := msg.SystemMsg
	var highlight string
	if msg.Message != "" {
		content += ": " + msg.Message
		highlight = s.randomColor()
	}

	return ChatMessage{
		Time:      msg.Time,
		User:      "SYSTEM",
		Flare:     "SYSTEM",
		Content:   content,
		NameColor: nameColor,
		Highlight: highlight,
	}, true
}

func resolveFlare(msg twitch.PrivateMessage) string {
	if msg.CustomRewardID != "" {
		return "REDEEM"
	}

	_, isMod := msg.User.Badges["moderator"]
	_, isBroadcaster := msg.User.Badges["broadcaster"]
	_, isVip := msg.User.Badges["vip"]

	switch {
	case isBroadcaster, isMod:
		return "MOD"
	case isVip:
		return "VIP"
	default:
		return ""
	}
}

func resolveHighlight(msg twitch.PrivateMessage, nameColor string, s *Service) (highlight, prepend string) {
	switch {
	case msg.Bits > 0:
		prepend = fmt.Sprintf("- Cheer%d -", msg.Bits)
		highlight = s.randomColor()
		if s.bitsApi.Enable && s.bitsApi.Endpoint != "" && msg.Bits >= s.bitsApi.BitsAmount {
			api.SendBitsNotification(s.bitsApi.Endpoint, msg.User.Name, msg.Message, nameColor)
		}
	case msg.FirstMessage:
		prepend = "- First -"
	case msg.Tags["msg-id"] == "highlighted-message":
		highlight = s.randomColor()
	}
	return
}

func extractTags(message string, colorFn func() string) ([]string, map[string]string) {
	var taggedUsers []string
	taggedColors := make(map[string]string)
	seen := make(map[string]bool)

	for _, word := range strings.Fields(message) {
		if !strings.HasPrefix(word, "@") {
			continue
		}
		user := strings.TrimRight(strings.TrimPrefix(word, "@"), ",.:;!?")
		if user != "" && !seen[user] {
			seen[user] = true
			taggedUsers = append(taggedUsers, user)
			taggedColors[user] = colorFn()
		}
	}

	return taggedUsers, taggedColors
}
