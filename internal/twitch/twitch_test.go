package twitch

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"twitch-tui/internal/config"
)

func TestNew_Anonymous(t *testing.T) {
	cfg := config.Config{
		Twitch: config.Twitch{
			User:  "testuser",
			Oauth: "",
		},
	}

	service := New(cfg)

	if service.Authenticated {
		t.Errorf("Expected Authenticated to be false, got true")
	}
}

func TestNew_Authenticated(t *testing.T) {
	cfg := config.Config{
		Twitch: config.Twitch{
			User:  "testuser",
			Oauth: "abc",
		},
	}

	service := New(cfg)

	if !service.Authenticated {
		t.Errorf("Expected Authenticated to be true, got false")
	}
}

func TestRefresh_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// We send back a JSON string that looks just like a real success response
		response := `{"success": true, "token": "new_fake_token", "refresh": "new_fake_refresh"}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	service := &Service{
		api:          server.URL,
		refreshToken: "old_refresh_token",
		SysChan:      make(chan string, 10),
		User:         "testuser",
	}

	err := service.refresh()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if service.token != "oauth:new_fake_token" {
		t.Errorf("Expected token to be 'oauth:new_fake_token', got '%s'", service.token)
	}

	if service.refreshToken != "new_fake_refresh" {
		t.Errorf("Expected refresh token to be 'new_fake_refresh', got '%s'", service.refreshToken)
	}
}
