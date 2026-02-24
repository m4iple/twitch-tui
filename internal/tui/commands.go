package tui

import (
	"errors"
	"fmt"
	"strings"

	"twitch-tui/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type commandHandler func(m *Model, args []string) (tea.Cmd, error)

type commandDef struct {
	Name    string
	Aliases []string
	Usage   string
	Handle  commandHandler
}

func (m *Model) executeCommand(input string) tea.Cmd {
	name, args, err := parseCommand(input)
	if err != nil {
		m.handleScroll(formatSystemMessage(err.Error()))
		return nil
	}

	def, ok := commandRegistry[name]
	if !ok {
		m.handleScroll(formatSystemMessage(fmt.Sprintf("Unknown command: %s", name)))
		return nil
	}

	cmd, err := def.Handle(m, args)
	if err != nil {
		m.handleScroll(formatSystemMessage(err.Error()))
		return nil
	}

	return cmd
}

var commandRegistry = func() map[string]commandDef {
	commands := []commandDef{
		{
			Name:    "login",
			Aliases: []string{"l"},
			Usage:   ":login <user> <token> <refresh>",
			Handle:  handleLoginCommand,
		},
		{
			Name:    "join",
			Aliases: []string{"j"},
			Usage:   ":join <channel>",
			Handle:  handleJoinCommand,
		},
		{
			Name:    "find",
			Aliases: []string{"f"},
			Usage:   ":find <string>",
			Handle:  handleFindCommand,
		},
		{
			Name:    "config",
			Aliases: []string{"cfg"},
			Usage:   ":config [reload | api enable/disable | emotes enable/disable]",
			Handle:  handleConfigCommand,
		},
		{
			Name:    "quit",
			Aliases: []string{"q"},
			Usage:   ":quit",
			Handle:  handleQuitCommand,
		},
	}

	reg := make(map[string]commandDef, len(commands)*2)
	for _, cmd := range commands {
		reg[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			reg[alias] = cmd
		}
	}
	return reg
}()

func parseCommand(input string) (string, []string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil, errors.New("empty command")
	}
	if !strings.HasPrefix(trimmed, ":") {
		return "", nil, errors.New("not a command")
	}

	fields := strings.Fields(trimmed)
	if len(fields) == 0 {
		return "", nil, errors.New("invalid command")
	}

	name := strings.TrimPrefix(fields[0], ":")
	if name == "" {
		return "", nil, errors.New("missing command name")
	}

	return strings.ToLower(name), fields[1:], nil
}

func handleLoginCommand(m *Model, args []string) (tea.Cmd, error) {
	if len(args) < 3 {
		return nil, errors.New("Usage: :login <user> <token> <refresh>")
	}

	user := args[0]
	token := args[1]
	refresh := args[2]

	if err := m.twitch.Login(user, token, refresh); err != nil {
		return nil, fmt.Errorf("login failed: %v", err)
	}

	m.config.Twitch.User = user
	m.config.Twitch.Oauth = token
	m.config.Twitch.Refresh = refresh
	m.config.Twitch.UserID = m.twitch.UserID
	if err := config.UpdateConfig(m.config); err != nil {
		m.handleScroll(formatSystemMessage(fmt.Sprintf("Failed to save config: %v", err)))
	}

	m.handleScroll(formatSystemMessage("Logged in as " + user))

	return nil, nil
}

func handleJoinCommand(m *Model, args []string) (tea.Cmd, error) {
	if len(args) < 1 {
		return nil, errors.New("Usage: :join <channel>")
	}

	channel := strings.TrimPrefix(args[0], "#")
	if channel == "" {
		return nil, errors.New("Usage: :join <channel>")
	}

	if m.state == stateInputChannel {
		m.textInput.Reset()
		m.textInput.Placeholder = "Send a message..."
		m.state = stateChat
		m.twitch.CurrentChannel = channel
		m.twitch.ChannelID = ""
		m.config.Twitch.Channel = channel
		m.config.Twitch.ChannelID = ""
		m.config.Twitch.UserID = m.twitch.UserID
		m.filter = ""
		m.messages = nil
		m.refreshViewport()
		if err := config.UpdateConfig(m.config); err != nil {
			m.handleScroll(formatSystemMessage(fmt.Sprintf("Failed to save config: %v", err)))
		}
		return tea.Batch(m.connectCmd(), waitForChatMsg(m.twitch.MsgChan)), nil
	}

	m.config.Twitch.Channel = channel
	m.config.Twitch.ChannelID = ""
	m.config.Twitch.UserID = m.twitch.UserID
	m.filter = ""
	m.messages = nil
	m.refreshViewport()
	if err := config.UpdateConfig(m.config); err != nil {
		m.handleScroll(formatSystemMessage(fmt.Sprintf("Failed to save config: %v", err)))
	}
	return m.switchChannelCmd(channel), nil
}

func handleFindCommand(m *Model, args []string) (tea.Cmd, error) {
	if len(args) == 0 {
		m.filter = ""
	} else {
		m.filter = strings.TrimSpace(strings.Join(args, " "))
	}

	m.refreshViewport()
	return nil, nil
}

func handleQuitCommand(m *Model, args []string) (tea.Cmd, error) {
	return tea.Quit, nil
}

func handleConfigCommand(m *Model, args []string) (tea.Cmd, error) {
	if len(args) == 0 {
		help := "Config commands:\n" +
			"  :config reload              — reload config from disk\n" +
			"  :config api enable          — enable bits API\n" +
			"  :config api disable         — disable bits API\n" +
			"  :config emotes enable       — enable emotes\n" +
			"  :config emotes disable      — disable emotes"
		m.handleScroll(formatSystemMessage(help))
		return nil, nil
	}

	switch strings.ToLower(args[0]) {
	case "reload":
		newCfg := config.Load()
		newCfg.Twitch = m.config.Twitch
		m.config = newCfg
		m.twitch.UpdateConfig(newCfg)
		m.handleScroll(formatSystemMessage("Config reloaded"))

	case "api":
		if len(args) < 2 {
			return nil, errors.New("Usage: :config api enable|disable")
		}
		switch strings.ToLower(args[1]) {
		case "enable":
			m.config.Api.Bits.Enable = true
		case "disable":
			m.config.Api.Bits.Enable = false
		default:
			return nil, fmt.Errorf("Unknown option: %s (use enable or disable)", args[1])
		}
		m.twitch.UpdateConfig(m.config)
		if err := config.UpdateConfig(m.config); err != nil {
			m.handleScroll(formatSystemMessage(fmt.Sprintf("Failed to save config: %v", err)))
		}
		m.handleScroll(formatSystemMessage(fmt.Sprintf("Bits API %sd", args[1])))

	case "emotes":
		if len(args) < 2 {
			return nil, errors.New("Usage: :config emotes enable|disable")
		}
		switch strings.ToLower(args[1]) {
		case "enable":
			m.config.Emotes.Enable = true
		case "disable":
			m.config.Emotes.Enable = false
		default:
			return nil, fmt.Errorf("Unknown option: %s (use enable or disable)", args[1])
		}
		m.twitch.UpdateConfig(m.config)
		if err := config.UpdateConfig(m.config); err != nil {
			m.handleScroll(formatSystemMessage(fmt.Sprintf("Failed to save config: %v", err)))
		}
		m.handleScroll(formatSystemMessage(fmt.Sprintf("Emotes %sd", args[1])))

	default:
		return nil, fmt.Errorf("Unknown config subcommand: %s", args[0])
	}

	return nil, nil
}
