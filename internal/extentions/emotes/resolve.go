package emotes

import (
	"twitch-tui/internal/config"

	irc "github.com/gempir/go-twitch-irc/v4"
)

func ResolveEmotes(content string, emotes []*irc.Emote, cfg config.Config) string {
	if len(emotes) > 0 && cfg.Emotes.Twitch.Enable {
		content = addTwitchEmotesLink(content, emotes, cfg.Emotes.Twitch.Color)
	}

	if cfg.Emotes.SevenTv.Enable {
		content = add7tvEmotesLink(content, cfg.Emotes.SevenTv.Color)
	}

	if cfg.Emotes.Bttv.Enable {
		content = addBttvEmotesLink(content, cfg.Emotes.Bttv.Color)
	}

	if cfg.Emotes.Ffz.Enable {
		content = addFfzEmotesLink(content, cfg.Emotes.Ffz.Color)
	}

	return content
}
