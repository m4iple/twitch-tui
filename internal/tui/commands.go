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

	def, ok := commandRegistry()[name]
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

func commandRegistry() map[string]commandDef {
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
			Name:    "quit",
			Aliases: []string{"q"},
			Usage:   ":quit",
			Handle:  handleQuitCommand,
		},
	}

	registry := make(map[string]commandDef, len(commands)*2)
	for _, cmd := range commands {
		registry[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			registry[alias] = cmd
		}
	}

	return registry
}

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
		m.config.Twitch.Channel = channel
		m.filter = ""
		m.messages = nil
		m.refreshViewport()
		if err := config.UpdateConfig(m.config); err != nil {
			m.handleScroll(formatSystemMessage(fmt.Sprintf("Failed to save config: %v", err)))
		}
		return tea.Batch(m.connectCmd(), waitForChatMsg(m.twitch.MsgChan)), nil
	}

	if err := m.twitch.SwitchChannel(channel); err != nil {
		return nil, fmt.Errorf("join failed: %v", err)
	}

	m.config.Twitch.Channel = channel
	m.filter = ""
	m.messages = nil
	m.refreshViewport()
	if err := config.UpdateConfig(m.config); err != nil {
		m.handleScroll(formatSystemMessage(fmt.Sprintf("Failed to save config: %v", err)))
	}
	return nil, nil
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
