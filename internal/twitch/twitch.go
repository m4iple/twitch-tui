package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
	"twitch-tui/internal/config"
	"twitch-tui/internal/extentions/api"

	"github.com/gempir/go-twitch-irc/v4"
)

type ChatMessage struct {
	Time         time.Time
	User         string
	Flare        string            // SYSTEM, ME, MOD, VIP, ...
	Content      string            // message text
	TaggedUsers  []string          // all @<user> mentions in the message
	Highlight    string            // color of the message
	Prepend      string            // label shown before the message content (e.g. "- Cheer100 -")
	NameColor    string            // color of the username
	TaggedColors map[string]string // color per @<user>
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
	clientID       string

	theme   config.Theme
	bitsApi config.BitsApi
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

func (t *Service) FetchUserID() error {
	if t.User == "" {
		return errors.New("username is required to fetch user ID")
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", t.User)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Client-ID", t.clientID)
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch user ID: %s", resp.Status)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		return errors.New("no user found")
	}

	t.UserID = result.Data[0].ID
	return nil
}

func (t *Service) FetchChannelID() error {
	if t.CurrentChannel == "" {
		return errors.New("channel name is required to fetch channel ID")
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", t.CurrentChannel)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Client-ID", t.clientID)
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch channel ID: %s", resp.Status)
	}

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if len(result.Data) == 0 {
		return errors.New("no channel found")
	}

	t.ChannelID = result.Data[0].ID
	return nil
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
		UserID:         cfg.Twitch.UserID,
		ChannelID:      cfg.Twitch.ChannelID,
		clientID:       cfg.Twitch.ClientID,

		theme:   cfg.Theme,
		bitsApi: cfg.BitsApi,
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

	t.client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		if msg, ok := t.formatUserNotice(message); ok {
			t.MsgChan <- msg
		}
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
		if err := t.FetchChannelID(); err != nil {
			t.SysChan <- "Joined channel #" + t.CurrentChannel
		} else {
			t.SysChan <- "Joined channel #" + t.CurrentChannel + " (ID: " + t.ChannelID + ")"
		}
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
	if err := t.FetchChannelID(); err != nil {
		t.SysChan <- "Switched to channel: " + newName
	} else {
		t.SysChan <- "Switched to channel: " + newName + " (ID: " + t.ChannelID + ")"
	}

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

func (s *Service) formatUserNotice(msg twitch.UserNoticeMessage) (ChatMessage, bool) {
	switch msg.MsgID {
	case "sub", "resub", "subgift", "anonsubgift", "submysterygift", "giftpaidupgrade", "anongiftpaidupgrade",
		"shoutout-received", "shoutout-sent":
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
			Bits:      0,
		}, true
	default:
		return ChatMessage{}, false
	}
}

func (s *Service) formatMessage(msg twitch.PrivateMessage) ChatMessage {
	flare := ""
	_, isMod := msg.User.Badges["moderator"]
	_, isBroadcaster := msg.User.Badges["broadcaster"]
	_, isVip := msg.User.Badges["vip"]

	if msg.CustomRewardID != "" {
		flare = "REDEEM"
	} else if isBroadcaster || isMod {
		flare = "MOD"
	} else if isVip {
		flare = "VIP"
	}

	var taggedUsers []string
	taggedColors := make(map[string]string)
	seen := make(map[string]bool)
	for _, word := range strings.Fields(msg.Message) {
		if strings.HasPrefix(word, "@") {
			user := strings.TrimRight(strings.TrimPrefix(word, "@"), ",.:;!?")
			if user != "" && !seen[user] {
				seen[user] = true
				taggedUsers = append(taggedUsers, user)
				taggedColors[user] = s.randomColor()
			}
		}
	}

	nameColor := msg.User.Color
	if nameColor == "" {
		nameColor = s.randomColor()
	}

	var highlight string
	var prepend string
	if msg.Bits > 0 {
		prepend = fmt.Sprintf("- Cheer%d -", msg.Bits)
		highlight = s.randomColor()
		if s.bitsApi.Enable && s.bitsApi.Endpoint != "" {
			if msg.Bits >= s.bitsApi.BitsAmount {
				api.SendBitsNotification(s.bitsApi.Endpoint, msg.User.Name, msg.Message, nameColor)
			}
		}
	} else if msg.FirstMessage {
		prepend = "- First -"
	} else if msg.Tags["msg-id"] == "highlighted-message" {
		highlight = s.randomColor()
	}

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
