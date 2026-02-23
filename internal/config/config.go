package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Twitch struct {
	Channel    string `toml:"channel"`
	User       string `toml:"user"`
	Oauth      string `toml:"oauth"`
	Refresh    string `toml:"refresh"`
	RefreshApi string `toml:"refresh_api"`
	UserID     string `toml:"user_id"`
	ChannelID  string `toml:"channel_id"`
	ClientID   string `toml:"client_id"`
}

type Theme struct {
	Crust     string `toml:"crust"`
	Mantle    string `toml:"mantle"`
	Base      string `toml:"base"`
	Surface0  string `toml:"surface0"`
	Surface1  string `toml:"surface1"`
	Surface2  string `toml:"surface2"`
	Overlay0  string `toml:"overlay0"`
	Overlay1  string `toml:"overlay1"`
	Overlay2  string `toml:"overlay2"`
	Subtext0  string `toml:"subtext0"`
	Subtext1  string `toml:"subtext1"`
	Text      string `toml:"text"`
	Lavender  string `toml:"lavender"`
	Blue      string `toml:"blue"`
	Sapphire  string `toml:"sapphire"`
	Sky       string `toml:"sky"`
	Teal      string `toml:"teal"`
	Green     string `toml:"green"`
	Yellow    string `toml:"yellow"`
	Peach     string `toml:"peach"`
	Maroon    string `toml:"maroon"`
	Red       string `toml:"red"`
	Mauve     string `toml:"mauve"`
	Pink      string `toml:"pink"`
	Flamingo  string `toml:"flamingo"`
	Rosewater string `toml:"rosewater"`
}

type Style struct {
	DateFormat string `toml:"date_format"`
}

type BitsApi struct {
	Enable     bool   `toml:"enable"`
	BitsAmount int    `toml:"bits_amount"`
	Endpoint   string `toml:"endpoint"`
}

type Config struct {
	Twitch  Twitch  `toml:"twitch"`
	Theme   Theme   `toml:"theme"`
	Style   Style   `toml:"style"`
	BitsApi BitsApi `toml:"bits_api"`
}

func Load() Config {
	cfg := defaultConfig()
	configPath, err := getConfigPath()
	if err != nil {
		return cfg
	}
	if err := loadConfigFile(configPath, &cfg); err != nil {
		return cfg
	}

	return cfg
}

func defaultConfig() Config {
	return Config{
		Twitch:  defaultTwitch(),
		Theme:   defaultTheme(),
		Style:   defaultStyle(),
		BitsApi: defaultBitsApi(),
	}
}

func defaultTwitch() Twitch {
	return Twitch{
		Channel:    "",
		User:       "",
		Oauth:      "",
		Refresh:    "",
		RefreshApi: defaultRefreshAPI,
		UserID:     "",
		ChannelID:  "",
		ClientID:   "",
	}
}

func defaultTheme() Theme {
	// Catppuccin Frappe
	return Theme{
		Crust:     "#232634",
		Mantle:    "#292c3c",
		Base:      "#303446",
		Surface0:  "#414559",
		Surface1:  "#51576d",
		Surface2:  "#626880",
		Overlay0:  "#737994",
		Overlay1:  "#838ba7",
		Overlay2:  "#949cbb",
		Subtext0:  "#a5adce",
		Subtext1:  "#949cbb",
		Text:      "#c6d0f5",
		Lavender:  "#babbf1",
		Blue:      "#8caaee",
		Sapphire:  "#85c1dc",
		Sky:       "#99d1db",
		Teal:      "#81c8be",
		Green:     "#a6d189",
		Yellow:    "#e5c890",
		Peach:     "#ef9f76",
		Maroon:    "#ea999c",
		Red:       "#e78284",
		Mauve:     "#ca9ee6",
		Pink:      "#f4b8e4",
		Flamingo:  "#eebebe",
		Rosewater: "#f2d5cf",
	}
}

func defaultStyle() Style {
	return Style{
		DateFormat: "15:04:05",
	}
}

func defaultBitsApi() BitsApi {
	return BitsApi{
		Enable:     false,
		BitsAmount: 0,
		Endpoint:   "",
	}
}

func UpdateTokens(newOauth, newRefresh string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	cfg, err := readConfigFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file: %v", err)
		}
		cfg = defaultConfig()
	}

	cfg.Twitch.Oauth = newOauth
	cfg.Twitch.Refresh = newRefresh
	if cfg.Twitch.RefreshApi == "" {
		cfg.Twitch.RefreshApi = defaultRefreshAPI
	}

	if err := writeConfigFile(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func UpdateLogin(user, oauth string) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	cfg, err := readConfigFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file: %v", err)
		}
		cfg = defaultConfig()
	}

	cfg.Twitch.User = user
	cfg.Twitch.Oauth = oauth
	if cfg.Twitch.RefreshApi == "" {
		cfg.Twitch.RefreshApi = defaultRefreshAPI
	}

	if err := writeConfigFile(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func UpdateConfig(cfg Config) error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	if cfg.Twitch.RefreshApi == "" {
		cfg.Twitch.RefreshApi = defaultRefreshAPI
	}

	if err := writeConfigFile(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

const defaultRefreshAPI = "https://twitchtokengenerator.com/api/refresh/"
const configFileName = "config.toml"
const appDir = "twitch-tui"

func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %v", err)
	}

	appConfigDir := filepath.Join(configDir, appDir)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(appConfigDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	return filepath.Join(appConfigDir, configFileName), nil
}

func loadConfigFile(path string, cfg *Config) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := toml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %v", err)
	}

	if cfg.Twitch.RefreshApi == "" {
		cfg.Twitch.RefreshApi = defaultRefreshAPI
	}

	return nil
}

func readConfigFile(path string) (Config, error) {
	cfg := defaultConfig()
	if err := loadConfigFile(path, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func writeConfigFile(path string, cfg Config) error {
	var buf bytes.Buffer
	encoder := toml.NewEncoder(&buf)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config file: %v", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
