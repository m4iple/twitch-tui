package emotes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

type sevenTvResponse struct {
	EmoteSet sevenTvEmoteSet `json:"emote_set"`
}

type sevenTvEmoteSet struct {
	Emotes []sevenTvEmoteEntry `json:"emotes"`
}

type sevenTvEmoteEntry struct {
	Name string           `json:"name"`
	Data sevenTvEmoteData `json:"data"`
}

type sevenTvEmoteData struct {
	Host sevenTvHost `json:"host"`
}

type sevenTvHost struct {
	URL string `json:"url"`
}

var (
	sevenTvCache   map[string]string
	sevenTvCacheMu sync.RWMutex
)

// with the channel id we call the 7tv api and chache the channel emotes
func Init7tvCache(channelID string) error {
	url := fmt.Sprintf("https://7tv.io/v3/users/twitch/%s", channelID)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("7tv: request failed: %w", err)
	}
	defer resp.Body.Close()

	var data sevenTvResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("7tv: decode failed: %w", err)
	}

	cache := make(map[string]string, len(data.EmoteSet.Emotes))
	for _, entry := range data.EmoteSet.Emotes {
		cdnURL := "https:" + entry.Data.Host.URL + "/4x.webp" // 4x is the size of the emote
		cache[entry.Name] = cdnURL
	}

	sevenTvCacheMu.Lock()
	sevenTvCache = cache
	sevenTvCacheMu.Unlock()

	return nil
}

// do the sane as the twitch emotes with the cached words
func add7tvEmotesLink(text string, theme string) string {
	sevenTvCacheMu.RLock()
	cache := sevenTvCache
	sevenTvCacheMu.RUnlock()

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
