package twitch

import (
	"os"
	"path/filepath"
	"twitch-tui/internal/config"
)

func (s *Service) initLogger(cfg config.Config) {
	if !cfg.Log.Enable {
		return
	}

	path := cfg.Log.Path
	if path == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return
		}
		path = filepath.Join(configDir, "twitch-tui", "chat.log")
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}

	s.logFile = f
}

func (s *Service) logRaw(raw string) {
	if s.logFile == nil {
		return
	}
	_, _ = s.logFile.WriteString(raw + "\n")
}

func (s *Service) Close() {
	if s.logFile != nil {
		_ = s.logFile.Close()
		s.logFile = nil
	}
}
