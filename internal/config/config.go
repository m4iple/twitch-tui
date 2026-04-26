package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const defaultRefreshAPI = "https://id.twitch.tv/oauth2/token"
const configFileName = "config.toml"
const appDir = "twitch-tui"

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

// stripped down Catppuccin Theme
type Theme struct {
	Base      string `toml:"base"`
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

type Api struct {
	Bits BitsApi `toml:"bits"`
}

type Emotes struct {
	Twitch  TwitchEmotes  `toml:"twitch"`
	SevenTv SevenTvEmotes `toml:"sevenTv"`
	Bttv    BttvEmotes    `toml:"bttv"`
	Ffz     FfzEmotes     `toml:"ffz"`
}

type TwitchEmotes struct {
	Enable bool   `toml:"enable"`
	Color  string `toml:"color"`
}
type SevenTvEmotes struct {
	Enable bool   `toml:"enable"`
	Color  string `toml:"color"`
}
type BttvEmotes struct {
	Enable bool   `toml:"enable"`
	Color  string `toml:"color"`
}
type FfzEmotes struct {
	Enable bool   `toml:"enable"`
	Color  string `toml:"color"`
}

type Log struct {
	Enable bool   `toml:"enable"`
	Path   string `toml:"path"`
}

type Config struct {
	Twitch Twitch `toml:"twitch"`
	Theme  Theme  `toml:"theme"`
	Style  Style  `toml:"style"`
	Api    Api    `toml:"api"`
	Emotes Emotes `toml:"emotes"`
	Log    Log    `toml:"log"`
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
		Twitch: defaultTwitch(),
		Theme:  defaultTheme(),
		Style:  defaultStyle(),
		Api:    defaultApi(),
		Emotes: defaultEmotes(),
		Log:    defaultLog(),
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
		Base:      "#303446",
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

func defaultApi() Api {
	return Api{
		Bits: BitsApi{
			Enable:     false,
			BitsAmount: 0,
			Endpoint:   "",
		},
	}
}

func defaultEmotes() Emotes {
	theme := defaultTheme()
	return Emotes{
		Twitch: TwitchEmotes{
			Enable: true,
			Color:  theme.Blue,
		},
		SevenTv: SevenTvEmotes{
			Enable: false,
			Color:  theme.Sapphire,
		},
		Bttv: BttvEmotes{
			Enable: false,
			Color:  theme.Rosewater,
		},
		Ffz: FfzEmotes{
			Enable: false,
			Color:  theme.Yellow,
		},
	}
}

func defaultLog() Log {
	return Log{
		Enable: false,
		Path:   "",
	}
}

// Upadting the token and refresh token in the config file - on token refresh
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

// Update oauth and user on login
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

// Update the fetched user id
func UpdateUserID(userID string) error {
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

	cfg.Twitch.UserID = userID
	if cfg.Twitch.RefreshApi == "" {
		cfg.Twitch.RefreshApi = defaultRefreshAPI
	}

	if err := writeConfigFile(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Update the fetched client id
func UpdateClientID(clientID string) error {
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

	cfg.Twitch.ClientID = clientID
	if cfg.Twitch.RefreshApi == "" {
		cfg.Twitch.RefreshApi = defaultRefreshAPI
	}

	if err := writeConfigFile(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// Update the fetched channel id
func UpdateChannelID(channelID string) error {
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

	cfg.Twitch.ChannelID = channelID
	if cfg.Twitch.RefreshApi == "" {
		cfg.Twitch.RefreshApi = defaultRefreshAPI
	}

	if err := writeConfigFile(configPath, cfg); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

// completely update / overwrite config
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

// gets / creates the config path - .config/twitch-tui | %appdata%/Roaming/twitch-tui
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

// opens file - decodes toml - updates cfg pointer
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

// loads defautl config then overwrites it with the conf in the file
func readConfigFile(path string) (Config, error) {
	cfg := defaultConfig()
	if err := loadConfigFile(path, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// write config to disk
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
