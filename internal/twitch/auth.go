package twitch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"twitch-tui/internal/config"
)

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

func (t *Service) fetchHelixUserIDByLogin(login, clientID string) (string, error) {
	if login == "" {
		t.SysChan <- "Helix lookup failed: empty login"
		return "", errors.New("login is required to fetch user ID")
	}

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", login)
	id, _, err := t.fetchHelixUser(url, fmt.Sprintf("login=%s", login), clientID)
	return id, err
}

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

	if err := config.UpdateTokens(result.Token, result.Refresh); err != nil {
		t.SysChan <- fmt.Sprintf("Refresh successful but failed to update config.toml: %v", err)
	}

	t.token = result.Token
	t.refreshToken = result.Refresh
	t.login()

	t.SysChan <- "Token refreshed successfully! Reconnecting..."
	t.startSession()
	return nil
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
