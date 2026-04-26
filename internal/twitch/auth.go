package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"twitch-tui/internal/config"

	"github.com/gempir/go-twitch-irc/v4"
)

const (
	twitchDeviceCodeURL = "https://id.twitch.tv/oauth2/device"
	twitchTokenURL      = "https://id.twitch.tv/oauth2/token"
	twitchScopes        = "chat:read chat:edit"
)

type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Message      string `json:"message"`
}

// login user
func (t *Service) login() {
	if !strings.HasPrefix(t.token, "oauth:") {
		t.token = "oauth:" + t.token
	}

	t.client = twitch.NewClient(t.User, t.token)
	t.Authenticated = true
}

// perform first-party Twitch OAuth using the device code flow.
func (t *Service) loginWithDeviceCode(clientID string) error {
	if clientID == "" {
		return errors.New("missing client ID")
	}

	if t.client != nil {
		t.client.Disconnect()
	}

	device, err := t.startDeviceCodeFlow(clientID)
	if err != nil {
		return err
	}

	t.SysChan <- fmt.Sprintf("Open %s and enter code: %s", device.VerificationURI, device.UserCode)
	t.SysChan <- "Waiting for Twitch authorization..."

	tokens, err := t.pollDeviceCodeTokens(clientID, device)
	if err != nil {
		return err
	}

	t.token = tokens.AccessToken
	t.refreshToken = tokens.RefreshToken
	t.ClientID = clientID

	userID, login, validatedClientID, err := t.fetchOAuthIdentity()
	if err != nil {
		return err
	}

	t.UserID = userID
	if login != "" {
		t.User = login
	}
	if validatedClientID != "" {
		t.ClientID = validatedClientID
	}

	if err := config.UpdateTokens(t.token, t.refreshToken); err != nil {
		t.SysChan <- fmt.Sprintf("Login succeeded but failed to persist tokens: %v", err)
	}
	if err := config.UpdateLogin(t.User, t.token); err != nil {
		t.SysChan <- fmt.Sprintf("Login succeeded but failed to persist user: %v", err)
	}
	if err := config.UpdateClientID(t.ClientID); err != nil {
		t.SysChan <- fmt.Sprintf("Login succeeded but failed to persist client ID: %v", err)
	}

	t.login()
	t.SysChan <- fmt.Sprintf("OAuth login successful as %s", t.User)
	t.startSession()
	return nil
}

// exported function fetch the user id
func (t *Service) FetchUserID() error {
	id, login, clientID, err := t.fetchOAuthIdentity()
	if err != nil {
		return err
	}

	t.UserID = id

	if t.User == "" && login != "" {
		t.User = login
	}

	if clientID != "" && t.ClientID != clientID {
		t.ClientID = clientID
		_ = config.UpdateClientID(clientID)
	}

	return nil
}

// exported function to fetch the channel id
func (t *Service) FetchChannelID() error {
	if t.CurrentChannel == "" {
		return errors.New("channel name is required to fetch channel ID")
	}

	userID, login, clientID, err := t.fetchOAuthIdentity()
	if err != nil {
		return err
	}

	if t.UserID == "" {
		t.UserID = userID
	}
	if t.User == "" && login != "" {
		t.User = login
	}

	if clientID != "" && t.ClientID != clientID {
		t.ClientID = clientID
		_ = config.UpdateClientID(clientID)
	}

	id, err := t.fetchHelixUserIDByLogin(t.CurrentChannel, clientID)
	if err != nil {
		return err
	}

	t.ChannelID = id
	return nil
}

// we call to the twitch oauth api to log our user in, and in the response we ge the client and user id
func (t *Service) fetchOAuthIdentity() (string, string, string, error) {
	accessToken := strings.TrimPrefix(t.token, "oauth:")
	if accessToken == "" {
		t.SysChan <- "OAuth validate failed: missing access token"
		return "", "", "", errors.New("access token is required to validate")
	}

	req, err := http.NewRequest("GET", "https://id.twitch.tv/oauth2/validate", nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Authorization", "OAuth "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.SysChan <- fmt.Sprintf("OAuth validate failed: request error - %v", err)
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := readBodySnippet(resp.Body)
		t.SysChan <- fmt.Sprintf("OAuth validate failed: status=%s body=%s", resp.Status, msg)
		return "", "", "", fmt.Errorf("failed to validate token: %s", resp.Status)
	}

	var result struct {
		ClientID string `json:"client_id"`
		Login    string `json:"login"`
		UserID   string `json:"user_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.SysChan <- fmt.Sprintf("OAuth validate failed: decode error - %v", err)
		return "", "", "", err
	}

	if result.UserID == "" {
		t.SysChan <- "OAuth validate failed: missing user ID"
		return "", "", "", errors.New("token validation did not return user ID")
	}

	t.SysChan <- fmt.Sprintf("OAuth validate ok: id=%s", result.UserID)
	return result.UserID, result.Login, result.ClientID, nil
}

// call another twitch api to get a forgein user id from our client id
func (t *Service) fetchHelixUserIDByLogin(login, clientID string) (string, error) {
	if login == "" {
		t.SysChan <- "Helix lookup failed: empty login"
		return "", errors.New("login is required to fetch user ID")
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", login)
	id, _, err := t.fetchHelixUser(url, fmt.Sprintf("login=%s", login), clientID)
	return id, err
}

// generate a request and handle the response of a twitch helix call
func (t *Service) fetchHelixUser(url, label, clientID string) (string, string, error) {
	if clientID == "" {
		t.SysChan <- "Helix lookup failed: missing client ID"
		return "", "", errors.New("client ID is required to fetch user ID")
	}

	accessToken := strings.TrimPrefix(t.token, "oauth:")
	if accessToken == "" {
		t.SysChan <- "Helix lookup failed: missing access token"
		return "", "", errors.New("access token is required to fetch user ID")
	}

	t.SysChan <- fmt.Sprintf("Helix lookup: %s", label)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Client-ID", clientID)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.SysChan <- fmt.Sprintf("Helix lookup failed: request error - %v", err)
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := readBodySnippet(resp.Body)
		t.SysChan <- fmt.Sprintf("Helix lookup failed: status=%s body=%s", resp.Status, msg)
		return "", "", fmt.Errorf("failed to fetch user ID: %s", resp.Status)
	}

	var result struct {
		Data []struct {
			ID    string `json:"id"`
			Login string `json:"login"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.SysChan <- fmt.Sprintf("Helix lookup failed: decode error - %v", err)
		return "", "", err
	}

	if len(result.Data) == 0 {
		t.SysChan <- "Helix lookup failed: no user found"
		return "", "", errors.New("no user found")
	}

	t.SysChan <- fmt.Sprintf("Helix lookup ok: id=%s", result.Data[0].ID)
	return result.Data[0].ID, result.Data[0].Login, nil
}

// call the refresh token api to get a new oath token if needed
func (t *Service) refresh() error {
	if t.refreshToken == "" {
		t.SysChan <- "Refresh failed: no refresh token available"
		return errors.New("no refresh token available")
	}
	if t.ClientID == "" {
		t.SysChan <- "Refresh failed: missing client ID"
		return errors.New("missing client ID")
	}

	refreshURL := t.api
	if refreshURL == "" {
		refreshURL = twitchTokenURL
	}

	data := url.Values{}
	data.Set("client_id", t.ClientID)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", t.refreshToken)

	resp, err := http.Post(
		refreshURL,
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(data.Encode()),
	)
	if err != nil {
		t.SysChan <- fmt.Sprintf("Refresh failed: token request error - %v", err)
		return fmt.Errorf("token request failed: %v", err)
	}
	defer resp.Body.Close()

	var result tokenResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.SysChan <- fmt.Sprintf("Refresh failed: could not decode response - %v", err)
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if resp.StatusCode != http.StatusOK || result.AccessToken == "" {
		if result.Message == "" {
			result.Message = resp.Status
		}
		t.SysChan <- fmt.Sprintf("Refresh failed: %s", result.Message)
		return fmt.Errorf("refresh failed: %s", result.Message)
	}

	newRefresh := result.RefreshToken
	if newRefresh == "" {
		newRefresh = t.refreshToken
	}

	if err := config.UpdateTokens(result.AccessToken, newRefresh); err != nil {
		t.SysChan <- fmt.Sprintf("Refresh successful but failed to update config.toml: %v", err)
	}

	t.token = result.AccessToken
	t.refreshToken = newRefresh
	if t.client != nil {
		t.client.Disconnect()
	}
	t.login()

	t.SysChan <- "Token refreshed successfully! Reconnecting..."
	t.startSession()
	return nil
}

func (t *Service) startDeviceCodeFlow(clientID string) (deviceCodeResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scopes", twitchScopes)

	resp, err := http.Post(
		twitchDeviceCodeURL,
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(data.Encode()),
	)
	if err != nil {
		t.SysChan <- fmt.Sprintf("OAuth device flow failed: %v", err)
		return deviceCodeResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg := readBodySnippet(resp.Body)
		t.SysChan <- fmt.Sprintf("OAuth device flow failed: status=%s body=%s", resp.Status, msg)
		return deviceCodeResponse{}, fmt.Errorf("failed to start oauth flow: %s", resp.Status)
	}

	var result deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return deviceCodeResponse{}, err
	}

	if result.DeviceCode == "" || result.VerificationURI == "" {
		return deviceCodeResponse{}, errors.New("invalid device code response")
	}
	if result.Interval <= 0 {
		result.Interval = 5
	}

	return result, nil
}

func (t *Service) pollDeviceCodeTokens(clientID string, device deviceCodeResponse) (tokenResponse, error) {
	timeout := time.Duration(device.ExpiresIn) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(time.Duration(device.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return tokenResponse{}, errors.New("oauth authorization timed out")
		case <-ticker.C:
			resp, err := t.exchangeDeviceCode(clientID, device.DeviceCode)
			if err != nil {
				return tokenResponse{}, err
			}

			switch strings.ToLower(resp.Message) {
			case "", "success":
				if resp.AccessToken != "" {
					return resp, nil
				}
			case "authorization_pending":
				continue
			case "slow_down":
				ticker.Reset(time.Duration(device.Interval+2) * time.Second)
				continue
			case "access_denied":
				return tokenResponse{}, errors.New("oauth authorization denied")
			default:
				if resp.AccessToken != "" {
					return resp, nil
				}
				return tokenResponse{}, fmt.Errorf("oauth token exchange failed: %s", resp.Message)
			}
		}
	}
}

func (t *Service) exchangeDeviceCode(clientID, deviceCode string) (tokenResponse, error) {
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("scopes", twitchScopes)
	data.Set("device_code", deviceCode)
	data.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	resp, err := http.Post(
		twitchTokenURL,
		"application/x-www-form-urlencoded",
		bytes.NewBufferString(data.Encode()),
	)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()

	var result tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return tokenResponse{}, err
	}

	if resp.StatusCode == http.StatusOK {
		return result, nil
	}

	if result.Message == "" {
		result.Message = resp.Status
	}
	return result, nil
}

// readBodySnippet reads up to 500 bytes from r and returns it as a trimmed string.
func readBodySnippet(r io.Reader) string {
	body, _ := io.ReadAll(r)
	msg := strings.TrimSpace(string(body))
	if len(msg) > 500 {
		msg = msg[:500] + "..."
	}
	if msg == "" {
		msg = "<empty body>"
	}
	return msg
}
