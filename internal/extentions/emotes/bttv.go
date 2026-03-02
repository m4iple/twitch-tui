package emotes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

type bttvEmote struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

type bttvChannelResponse struct {
	ChannelEmotes []bttvEmote `json:"channelEmotes"`
	SharedEmotes  []bttvEmote `json:"sharedEmotes"`
}

var (
	bttvCache   map[string]string
	bttvCacheMu sync.RWMutex
)

func InitBttvCache(channelID string) error {
	globalResp, err := http.Get("https://api.betterttv.net/3/cached/emotes/global")
	if err != nil {
		return fmt.Errorf("bttv: global request failed: %w", err)
	}
	defer globalResp.Body.Close()

	var globalEmotes []bttvEmote
	if err := json.NewDecoder(globalResp.Body).Decode(&globalEmotes); err != nil {
		return fmt.Errorf("bttv: global decode failed: %w", err)
	}

	channelURL := fmt.Sprintf("https://api.betterttv.net/3/cached/users/twitch/%s", channelID)
	channelResp, err := http.Get(channelURL)
	if err != nil {
		return fmt.Errorf("bttv: channel request failed: %w", err)
	}
	defer channelResp.Body.Close()

	var channelData bttvChannelResponse
	if err := json.NewDecoder(channelResp.Body).Decode(&channelData); err != nil {
		return fmt.Errorf("bttv: channel decode failed: %w", err)
	}

	all := make([]bttvEmote, 0, len(globalEmotes)+len(channelData.ChannelEmotes)+len(channelData.SharedEmotes))
	all = append(all, globalEmotes...)
	all = append(all, channelData.ChannelEmotes...)
	all = append(all, channelData.SharedEmotes...)

	cache := make(map[string]string, len(all))
	for _, e := range all {
		cache[e.Code] = "https://cdn.betterttv.net/emote/" + e.ID + "/3x"
	}

	bttvCacheMu.Lock()
	bttvCache = cache
	bttvCacheMu.Unlock()

	return nil
}

func addBttvEmotesLink(text string, theme string) string {
	bttvCacheMu.RLock()
	cache := bttvCache
	bttvCacheMu.RUnlock()

	if len(cache) == 0 {
		return text
	}

	emoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(theme))

	words := strings.Split(text, " ")
	for i, word := range words {
		if cdnURL, ok := cache[word]; ok {
			styledWord := emoteStyle.Render(word)
			words[i] = fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", cdnURL, styledWord)
		}
	}

	return strings.Join(words, " ")
}
