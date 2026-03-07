package emotes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"
)

type ffzEmoticon struct {
	Name string            `json:"name"`
	URLs map[string]string `json:"urls"`
}

type ffzSet struct {
	Emoticons []ffzEmoticon `json:"emoticons"`
}

type ffzGlobalResponse struct {
	DefaultSets []int             `json:"default_sets"`
	Sets        map[string]ffzSet `json:"sets"`
}

type ffzRoomResponse struct {
	Room struct {
		Set int `json:"set"`
	} `json:"room"`
	Sets map[string]ffzSet `json:"sets"`
}

var (
	ffzCache   map[string]string
	ffzCacheMu sync.RWMutex
)

// with the channel id we call the ffz api and chache the channel emotes
func InitFfzCache(channelID string) error {
	globalResp, err := http.Get("https://api.frankerfacez.com/v1/set/global")
	if err != nil {
		return fmt.Errorf("ffz: global request failed: %w", err)
	}
	defer globalResp.Body.Close()

	var globalData ffzGlobalResponse
	if err := json.NewDecoder(globalResp.Body).Decode(&globalData); err != nil {
		return fmt.Errorf("ffz: global decode failed: %w", err)
	}

	channelURL := fmt.Sprintf("https://api.frankerfacez.com/v1/room/id/%s", channelID)
	channelResp, err := http.Get(channelURL)
	if err != nil {
		return fmt.Errorf("ffz: channel request failed: %w", err)
	}
	defer channelResp.Body.Close()

	var roomData ffzRoomResponse
	if err := json.NewDecoder(channelResp.Body).Decode(&roomData); err != nil {
		return fmt.Errorf("ffz: channel decode failed: %w", err)
	}

	cache := make(map[string]string)

	for _, id := range globalData.DefaultSets {
		key := fmt.Sprintf("%d", id)
		if set, ok := globalData.Sets[key]; ok {
			for _, e := range set.Emoticons {
				if url, ok := e.URLs["2"]; ok { // the ["2"] is the size of the emote
					cache[e.Name] = url
				}
			}
		}
	}

	channelSetKey := fmt.Sprintf("%d", roomData.Room.Set)
	if set, ok := roomData.Sets[channelSetKey]; ok {
		for _, e := range set.Emoticons {
			if url, ok := e.URLs["2"]; ok { // the ["2"] is the size of the emote
				cache[e.Name] = url
			}
		}
	}

	ffzCacheMu.Lock()
	ffzCache = cache
	ffzCacheMu.Unlock()

	return nil
}

// do the sane as the twitch emotes with the cached words
func addFfzEmotesLink(text string, theme string) string {
	ffzCacheMu.RLock()
	cache := ffzCache
	ffzCacheMu.RUnlock()

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
