package twitch

import (
	"errors"
	"strings"
	"twitch-tui/internal/config"

	"github.com/gempir/go-twitch-irc/v4"
)

func (t *Service) Connect() {
	t.startSession()
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

	t.client.OnConnect(func() {
		t.SysChan <- "Connected to #" + t.CurrentChannel

		if !t.Authenticated {
			t.SysChan <- "Not logged in — user and channel IDs will not be fetched. Use :login to authenticate."
			return
		}

		if t.ChannelID == "" {
			if err := t.FetchChannelID(); err != nil {
				t.SysChan <- "Channel ID lookup failed: " + err.Error()
			} else {
				if err := config.UpdateChannelID(t.ChannelID); err != nil {
					t.SysChan <- "Failed to save channel ID: " + err.Error()
				}
				if err := config.UpdateUserID(t.UserID); err != nil {
					t.SysChan <- "Failed to save user ID: " + err.Error()
				}
			}
		}

		if t.UserID == "" {
			if err := t.FetchUserID(); err != nil {
				t.SysChan <- "User ID lookup failed: " + err.Error()
			} else if err := config.UpdateUserID(t.UserID); err != nil {
				t.SysChan <- "Failed to save user ID: " + err.Error()
			}
		}
	})

	t.client.OnNoticeMessage(func(message twitch.NoticeMessage) {
		if strings.Contains(message.Message, "Login authentication failed") {
			t.SysChan <- "Auth failed. Attempting auto-refresh..."
			t.refresh()
		}
	})

	t.client.Join(t.CurrentChannel)

	go func() {
		if err := t.client.Connect(); err != nil {
			t.SysChan <- "Connection error: " + err.Error()
		}
	}()
}

func (t *Service) SwitchChannel(newName string) error {
	if t.client == nil {
		return errors.New("client not initialized")
	}

	t.client.Depart(t.CurrentChannel)
	t.CurrentChannel = newName
	t.ChannelID = ""
	t.client.Join(t.CurrentChannel)

	if err := t.FetchChannelID(); err != nil {
		t.SysChan <- "Channel ID lookup failed: " + err.Error()
	} else if err := config.UpdateChannelID(t.ChannelID); err != nil {
		t.SysChan <- "Failed to save channel ID: " + err.Error()
	}

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

	t.login()
	t.startSession()
	return nil
}

func (t *Service) login() {
	if !strings.HasPrefix(t.token, "oauth:") {
		t.token = "oauth:" + t.token
	}

	t.client = twitch.NewClient(t.User, t.token)
	t.Authenticated = true
}
