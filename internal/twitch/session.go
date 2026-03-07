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

// set up the message listenn and on connect fetch the channel id when possible
func (t *Service) startSession() {
	t.client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		t.logRaw(message.Raw)
		t.MsgChan <- t.formatMessage(message)
	})

	t.client.OnUserNoticeMessage(func(message twitch.UserNoticeMessage) {
		if msg, ok := t.formatUserNotice(message); ok {
			t.MsgChan <- msg
		}
	})

	t.client.OnConnect(func() {
		t.SysChan <- "Connected to #" + t.CurrentChannel

		// if we are not logged dont even try to fetch the ids
		if !t.Authenticated {
			t.SysChan <- "Not logged in — user and channel IDs will not be fetched. Use :login to authenticate."
			return
		}

		// fetch the channel user id
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

		// fetch the logged in user id
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

	t.client.Join(t.CurrentChannel) // join the channel

	go func() {
		if err := t.client.Connect(); err != nil {
			t.SysChan <- "Connection error: " + err.Error()
		}
	}()
}

// exit current channel and join the new one - also fetch our beloved ids
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

// post text input to twitch
func (t *Service) Say(message string) {
	t.client.Say(t.CurrentChannel, message)
}

// exported login used in the login command
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
