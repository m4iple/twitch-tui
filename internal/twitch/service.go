package twitch

import (
	"math/rand"
	"os"
	"time"
	"twitch-tui/internal/config"

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

	theme         config.Theme
	bitsApi       config.BitsApi
	EmotesEnabled bool

	logFile *os.File
}

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

		theme:         cfg.Theme,
		bitsApi:       cfg.Api.Bits,
		EmotesEnabled: cfg.Emotes.Enable,
	}

	if s.token != "" {
		s.login()
	}

	s.initLogger(cfg)

	return s
}

func (s *Service) UpdateConfig(cfg config.Config) {
	s.theme = cfg.Theme
	s.bitsApi = cfg.Api.Bits
	s.EmotesEnabled = cfg.Emotes.Enable
}

func (t *Service) randomColor() string {
	palette := []string{
		t.theme.Lavender,
		t.theme.Blue,
		t.theme.Sapphire,
		t.theme.Sky,
		t.theme.Teal,
		t.theme.Green,
		t.theme.Yellow,
		t.theme.Peach,
		t.theme.Maroon,
		t.theme.Red,
		t.theme.Mauve,
		t.theme.Pink,
		t.theme.Flamingo,
		t.theme.Rosewater,
	}
	return palette[rand.Intn(len(palette))]
}
