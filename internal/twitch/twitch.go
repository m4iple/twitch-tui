package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"twitch-tui/internal/config"

	"github.com/gempir/go-twitch-irc/v4"
)

type ChatMessage struct {
	Time        time.Time
	User        string
	Flare       string // SYSTEM, ME, MOD, VIP, ...
	Content     string
	TaggedUser  string // when a massage has a @<user>
	Highlight   string // color of the message
	NameColor   string // color of the name in not there use a random
	TaggedColor string // color of the @<user> string
	Bits        int
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
}

func New(cfg config.Config) *Service {
	msgChan := make(chan ChatMessage)
	sysChan := make(chan string)

	client := twitch.NewAnonymousClient()

	s := &Service{
		client:  client,
		MsgChan: msgChan,
		SysChan: sysChan,

		CurrentChannel: cfg.Twitch.Channel,
		User:           cfg.Twitch.User,
		Authenticated:  false,
		token:          cfg.Twitch.Oauth,
		refreshToken:   cfg.Twitch.Refresh,
		api:            cfg.Twitch.RefreshApi,
	}

	if s.token != "" {
		s.login()
	}

	return s
}

func (t *Service) Connect() error {
	t.startSession()
	return nil
}

func (t *Service) startSession() {
	t.client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		t.MsgChan <- t.formatMessage(message)
	})

	t.client.Join(t.CurrentChannel)

	t.client.OnConnect(func() {
		t.SysChan <- "Connected to Twitch IRC."
	})

	t.client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		if strings.Contains(message.Message, "Login authentication failed") {
			t.SysChan <- "Auth failed. Attempting auto-refresh..."
			t.refresh()
		}
	})

	go func() {
		err := t.client.Connect()
		if err != nil {
			t.SysChan <- "Connection error: " + err.Error()
		}
		t.SysChan <- "Joined channel #" + t.CurrentChannel
	}()
}

func (t *Service) SwitchChannel(newName string) error {
	if t.client == nil {
		t.SysChan <- "client not initialized"
		return errors.New("client not initialized")
	}

	t.client.Depart(t.CurrentChannel)
	t.CurrentChannel = newName
	t.client.Join(t.CurrentChannel)
	t.SysChan <- "Switched to channel: " + newName

	return nil
}

func (t *Service) Say(message string) {
	t.client.Say(t.CurrentChannel, message)
}

func (t *Service) Login(user, token, refresh string) error {
	if user == "" {
		return errors.New("missing user")
	}
	if token == "" {
		return errors.New("missing token")
	}
	if refresh == "" {
		return errors.New("missing refresh token")
	}

	if t.client != nil {
		t.client.Disconnect()
	}

	t.User = user
	t.token = token
	t.refreshToken = refresh

	if err := t.login(); err != nil {
		return err
	}

	t.startSession()
	return nil
}

func (t *Service) login() error {
	if !strings.HasPrefix(t.token, "oauth:") {
		t.token = "oauth:" + t.token
	}

	t.client = twitch.NewClient(t.User, t.token)
	t.Authenticated = true

	return nil
}

func (t *Service) refresh() error {
	if t.refreshToken == "" {
		t.SysChan <- "Refresh failed: no refresh token available"
		return errors.New("no refresh token available")
	}

	resp, err := http.Get(t.api + "/" + t.refreshToken)
	if err != nil {
		t.SysChan <- fmt.Sprintf("Refresh failed: API connection error - %v", err)
		return fmt.Errorf("api connection failed: %v", err)
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Token   string `json:"token"`
		Refresh string `json:"refresh"`
		Message string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.SysChan <- fmt.Sprintf("Refresh failed: could not decode response - %v", err)
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if !result.Success {
		t.SysChan <- fmt.Sprintf("Refresh failed: %s", result.Message)
		return fmt.Errorf("refresh failed: %s", result.Message)
	}

	newToken := result.Token

	if err := config.UpdateTokens(newToken, result.Refresh); err != nil {
		t.SysChan <- fmt.Sprintf("Refresh successful but failed to update config.toml: %v", err)
	}

	if !strings.HasPrefix(newToken, "oauth:") {
		newToken = "oauth:" + newToken
	}

	t.token = newToken
	t.refreshToken = result.Refresh

	if err := t.login(); err != nil {
		t.SysChan <- fmt.Sprintf("Refresh successful but login failed: %v", err)
		return err
	}

	t.SysChan <- "Token refreshed successfully! Reconnecting..."
	t.startSession()
	return nil
}

func (s *Service) formatMessage(msg twitch.PrivateMessage) ChatMessage {
	flare := ""
	_, isMod := msg.User.Badges["moderator"]
	_, isBroadcaster := msg.User.Badges["broadcaster"]
	_, isVip := msg.User.Badges["vip"]

	if isBroadcaster || isMod {
		flare = "MOD"
	} else if isVip {
		flare = "VIP"
	}

	var taggedUser string
	words := strings.Fields(msg.Message)
	for _, word := range words {
		if strings.HasPrefix(word, "@") {
			taggedUser = strings.TrimLeft(word, "@")
			taggedUser = strings.TrimRight(taggedUser, ",.:;!?")
			break
		}
	}

	return ChatMessage{
		Time:        msg.Time,
		User:        msg.User.Name,
		Content:     msg.Message,
		Flare:       flare,
		NameColor:   msg.User.Color,
		TaggedUser:  taggedUser,
		TaggedColor: "",
		Bits:        msg.Bits,
	}
}
