package twitch

import (
	"log"
	"math/rand"
	"os"
	"time"
	"twitch-tui/internal/config"
	"twitch-tui/internal/extentions/emotes"

	"github.com/gempir/go-twitch-irc/v4"
)

type ChatMessage struct {
	Time         time.Time
	User         string
	Flare        string
	Content      string
	TaggedUsers  []string
	Highlight    string
	Prepend      string
	NameColor    string
	TaggedColors map[string]string
	Bits         int
}

type Service struct {
	client  *twitch.Client
	MsgChan chan ChatMessage
	SysChan chan string

	CurrentChannel string
	User           string
	Authenticated  bool
	token          string
	refreshToken   string
	api            string
	UserID         string
	ChannelID      string
	ClientID       string

	cfg config.Config

	logFile *os.File
}

// init new twitch irc connection. first without an user then - when set log ourself in
func New(cfg config.Config) *Service {
	s := &Service{
		client:  twitch.NewAnonymousClient(),
		MsgChan: make(chan ChatMessage),
		SysChan: make(chan string),

		CurrentChannel: cfg.Twitch.Channel,
		User:           cfg.Twitch.User,
		token:          cfg.Twitch.Oauth,
		refreshToken:   cfg.Twitch.Refresh,
		api:            cfg.Twitch.RefreshApi,
		UserID:         cfg.Twitch.UserID,
		ChannelID:      cfg.Twitch.ChannelID,
		ClientID:       cfg.Twitch.ClientID,

		cfg: cfg,
	}

	if s.token != "" {
		s.login()
	}

	// when we have the twitch channel id and the emotes are enabled cache them
	if s.ChannelID != "" {
		if cfg.Emotes.SevenTv.Enable {
			go func() {
				if err := emotes.Init7tvCache(s.ChannelID); err != nil {
					log.Printf("7tv emote cache: %v", err)
				}
			}()
		}
		if cfg.Emotes.Bttv.Enable {
			go func() {
				if err := emotes.InitBttvCache(s.ChannelID); err != nil {
					log.Printf("bttv emote cache: %v", err)
				}
			}()
		}
		if cfg.Emotes.Ffz.Enable {
			go func() {
				if err := emotes.InitFfzCache(s.ChannelID); err != nil {
					log.Printf("ffz emote cache: %v", err)
				}
			}()
		}
	}

	s.initLogger(cfg) // inti the message logger

	return s
}

func (s *Service) UpdateConfig(cfg config.Config) {
	s.cfg = cfg
}

// gets a random color from the theme
func (t *Service) randomColor() string {
	palette := []string{
		t.cfg.Theme.Lavender,
		t.cfg.Theme.Blue,
		t.cfg.Theme.Sapphire,
		t.cfg.Theme.Sky,
		t.cfg.Theme.Teal,
		t.cfg.Theme.Green,
		t.cfg.Theme.Yellow,
		t.cfg.Theme.Peach,
		t.cfg.Theme.Maroon,
		t.cfg.Theme.Red,
		t.cfg.Theme.Mauve,
		t.cfg.Theme.Pink,
		t.cfg.Theme.Flamingo,
		t.cfg.Theme.Rosewater,
	}
	return palette[rand.Intn(len(palette))]
}
