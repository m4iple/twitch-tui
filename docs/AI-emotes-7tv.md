# 7TV Emote Support Implementation Plan

## Overview
Implement 7TV emote display for Twitch TUI similar to the browser extension - load emotes on channel join, cache them locally, and replace emote codes with images in chat messages.

## Key APIs

### 7TV API Endpoints (v3)
- **Global emotes**: `GET https://7tv.io/v3/emote-sets/global`
- **Channel emotes**: `GET https://7tv.io/v3/users/twitch/{twitch_id}`
- **CDN format**: `https://cdn.7tv.app/emote/{emote_id}/{1x|2x|3x|4x}.webp`

### API Response Structure
```json
{
  "id": "user_id",
  "emote_set": {
    "emotes": [
      {
        "id": "emote_id",
        "name": "emote_code",
        "animated": true,
        "data": {
          "host": {
            "url": "//cdn.7tv.app/emote/emote_id"
          }
        }
      }
    ]
  }
}
```

## Implementation Steps

### 1. Create `internal/emotes/` Package

#### `seven_tv.go` - 7TV API Client
- HTTP client for 7TV API
- Fetch global emotes list
- Fetch channel emotes by Twitch user ID
- Parse responses into Go structs

#### `cache.go` - Local Filesystem Cache
- Cache location: `~/.cache/twitch-tui/emotes/`
- Store emote metadata as JSON (code -> ID mapping, animated flag)
- Store emote images as webp files
- Cache expiration: 24 hours
- Functions:
  - `LoadCachedEmotes(channel string) ([]Emote, error)`
  - `SaveEmote(emote Emote, data []byte) error`
  - `GetEmoteImage(emoteID string, size string) ([]byte, error)`
  - `IsCacheValid(channel string) bool`

#### `manager.go` - Emote Manager
- Coordinates API fetching and caching
- Maintains in-memory emote map: `map[string]Emote` (code -> emote)
- Async loading to avoid blocking UI
- Functions:
  - `LoadChannelEmotes(channelID string) error` - async fetch
  - `GetEmote(code string) (*Emote, bool)`
  - `HasEmote(code string) bool`

### 2. Extend Data Models

#### `internal/twitch/twitch.go`
Extend `ChatMessage` struct:
```go
type ChatMessage struct {
    Time        time.Time
    User        string
    Flare       string
    Content     string
    TaggedUser  string
    Highlight   string
    NameColor   string
    TaggedColor string
    Bits        int
    Emotes      []EmotePosition  // NEW: detected emotes in message
}

type EmotePosition struct {
    Code  string // emote code (e.g., "EZ")
    Start int    // start index in Content
    End   int    // end index in Content
}
```

Update `formatMessage()` to detect emotes during message parsing.

### 3. Update Message Formatting

#### `internal/tui/message_format.go`

Modify `formatMessage()` function:
1. Split message content by spaces (words)
2. For each word, check if it's an emote code (via emote manager)
3. Build formatted output:
   - Regular text: apply existing styling
   - Emote codes: replace with rendered image using `go-termimg`
4. Handle text wrapping carefully:
   - Emotes typically render as 2-3 character cells
   - Calculate width properly with `lipgloss.Width()`

Example pseudo-code:
```go
func (m Model) formatMessageWithEmotes(msg twitch.ChatMessage) string {
    words := strings.Fields(msg.Content)
    var parts []string
    
    for _, word := range words {
        if emote, ok := m.emoteManager.GetEmote(word); ok {
            // Render emote image
            img := renderEmote(emote)
            parts = append(parts, img)
        } else {
            parts = append(parts, word)
        }
    }
    
    content := strings.Join(parts, " ")
    // ... rest of formatting
}
```

### 4. Integrate with TUI Model

#### `internal/tui/model.go`

Add to `Model` struct:
```go
type Model struct {
    state         appState
    twitch        *twitch.Service
    config        config.Config
    messages      []twitch.ChatMessage
    viewport      viewport.Model
    textInput     textinput.Model
    width         int
    height        int
    ready         bool
    filter        string
    rng           *rand.Rand
    emoteManager  *emotes.Manager  // NEW
}
```

Update `Init()`:
- Initialize emote manager with cache directory
- If joining channel on startup, trigger emote load

Update channel switch flow:
1. User joins channel
2. Fetch Twitch user ID for channel
3. Call `emoteManager.LoadChannelEmotes(twitchID)`
4. Continue rendering messages (emotes appear as they load)

### 5. Configuration

#### `internal/config/config.go`

Add to `Config` struct:
```go
type Config struct {
    Twitch TwitchConfig
    Theme  ThemeConfig
    Emotes EmoteConfig  // NEW
}

type EmoteConfig struct {
    Enabled  bool   `toml:"enabled"`
    Provider string `toml:"provider"` // "7tv" or "all"
    Size     string `toml:"size"`     // "1x", "2x", "3x", "4x"
    CacheDir string `toml:"cache_dir"`
}
```

Default config values:
```toml
[emotes]
enabled = true
provider = "7tv"
size = "2x"
cache_dir = "~/.cache/twitch-tui/emotes"
```

### 6. Image Rendering

Use `github.com/blacktop/go-termimg` for terminal image display:

```go
import "github.com/blacktop/go-termimg"

func renderEmote(emote Emote, size string) string {
    img, err := termimg.Open(emote.GetImagePath(size))
    if err != nil {
        return emote.Code  // fallback to text
    }
    
    widget := termimg.NewImageWidget(img)
    widget.SetSize(2, 1)  // 2 cols x 1 row (adjust as needed)
    rendered, _ := widget.Render()
    return rendered
}
```

## Dependencies

Add to `go.mod`:
```bash
go get github.com/blacktop/go-termimg
```

## Open Questions

1. **Twitch User ID Resolution**: The 7TV API requires a Twitch user ID, not username. Need to implement username -> ID lookup (likely via Twitch API or Helix).

2. **Image Size**: Default to 2x (64px), but make configurable. Larger looks better but takes more terminal space.

3. **Multiple Providers**: Start with 7TV only, but structure code to easily add BTTV and FFZ later.

4. **Cache Duration**: Refresh emotes on every channel join, but keep images cached for 7 days.

5. **Animation Support**: 7TV has animated emotes (GIF/WebP). Terminal support varies - may need to convert to static or skip animation.

## File Structure

```
internal/
├── emotes/
│   ├── seven_tv.go      # 7TV API client
│   ├── cache.go         # Filesystem cache
│   ├── manager.go       # Emote manager
│   └── types.go         # Shared types
├── twitch/
│   └── twitch.go        # Update ChatMessage struct
├── tui/
│   ├── model.go         # Add emoteManager field
│   └── message_format.go # Update formatting
└── config/
    └── config.go        # Add emote config
```

## Testing Plan

1. Test API fetching with known channels
2. Test cache read/write
3. Test emote detection in messages
4. Test image rendering in terminal
5. Test fallback when emotes fail to load

## Future Enhancements

- [ ] BTTV emote support
- [ ] FFZ emote support
- [ ] Twitch native emotes
- [ ] Emote picker UI (`:emotes` command)
- [ ] Animated emote support (where terminal supports it)
- [ ] Custom emote size per-user setting
