package tui

import (
	"strings"
	"time"
	"twitch-tui/internal/config"
	"twitch-tui/internal/twitch"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type appState int
type systemMsg string
type tickMsg struct{}

type ThemeStyles struct {
	Crust     lipgloss.Style
	Mantle    lipgloss.Style
	Base      lipgloss.Style
	Surface0  lipgloss.Style
	Surface1  lipgloss.Style
	Surface2  lipgloss.Style
	Overlay0  lipgloss.Style
	Overlay1  lipgloss.Style
	Overlay2  lipgloss.Style
	Subtext0  lipgloss.Style
	Subtext1  lipgloss.Style
	Text      lipgloss.Style
	Lavender  lipgloss.Style
	Blue      lipgloss.Style
	Sapphire  lipgloss.Style
	Sky       lipgloss.Style
	Teal      lipgloss.Style
	Green     lipgloss.Style
	Yellow    lipgloss.Style
	Peach     lipgloss.Style
	Maroon    lipgloss.Style
	Red       lipgloss.Style
	Mauve     lipgloss.Style
	Pink      lipgloss.Style
	Flamingo  lipgloss.Style
	Rosewater lipgloss.Style
}

const (
	stateInputChannel appState = iota
	stateInputCommand
	stateChat
)

type Model struct {
	state     appState
	twitch    *twitch.Service
	config    config.Config
	messages  []twitch.ChatMessage
	viewport  viewport.Model
	textInput textinput.Model
	width     int
	height    int
	ready     bool
	filter    string
}

func New(cfg config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter channel"
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 30

	state := stateInputChannel
	if cfg.Twitch.Channel != "" {
		state = stateChat
	}

	t := twitch.New(cfg)

	return Model{
		state:     state,
		config:    cfg,
		textInput: ti,
		twitch:    t,
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	cmds = append(cmds, textinput.Blink)
	cmds = append(cmds, waitForSystemMsg(m.twitch.SysChan))
	cmds = append(cmds, tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return tickMsg{}
	}))

	if m.state == stateChat {
		cmds = append(cmds, m.connectCmd())
		cmds = append(cmds, waitForChatMsg(m.twitch.MsgChan))
	}

	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return tickMsg{}
		})
	case systemMsg:
		m.handleScroll(formatSystemMessage(string(msg)))
		return m, waitForSystemMsg(m.twitch.SysChan)
	case twitch.ChatMessage:
		m.handleScroll(msg)
		return m, waitForChatMsg(m.twitch.MsgChan)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+q":
			return m, tea.Quit
		case "ctrl+f":
			if m.state == stateChat || m.state == stateInputCommand {
				m.textInput.SetValue(":find ")
				m.textInput.SetCursor(len(":find "))
				m.state = stateInputCommand
			}
			return m, nil
		case "ctrl+j":
			if m.state == stateChat || m.state == stateInputCommand {
				m.textInput.SetValue(":join ")
				m.textInput.SetCursor(len(":join "))
				m.state = stateInputCommand
			}
			return m, nil
		case "enter":
			input := strings.TrimSpace(m.textInput.Value())
			if strings.HasPrefix(input, ":") {
				m.textInput.Reset()
				m.state = stateChat
				return m, m.executeCommand(input)
			}

			switch m.state {
			case stateInputChannel:
				channel := input
				if channel != "" {
					m.twitch.SwitchChannel(channel)
					m.config.Twitch.Channel = channel
					if err := config.UpdateConfig(m.config); err != nil {
						m.handleScroll(formatSystemMessage("Failed to save config: " + err.Error()))
					}

					m.textInput.Reset()
					m.textInput.Placeholder = "Send a message..."
					m.state = stateChat

					return m, tea.Batch(
						m.connectCmd(),
						waitForChatMsg(m.twitch.MsgChan),
					)
				}
			case stateChat, stateInputCommand:
				if input != "" {
					m.textInput.Reset()
					m.state = stateChat
					return m, m.sendMsgCmd(input)
				}
			}
		}
	case tea.WindowSizeMsg:
		m.updateViewport(msg)
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	var tiCmd, vpCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.updateInputState()

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

func (m *Model) handleScroll(msg twitch.ChatMessage) {
	atBottom := m.viewport.AtBottom()
	m.messages = append(m.messages, msg)
	m.viewport.SetContent(m.buildContent())
	if atBottom {
		m.viewport.GotoBottom()
	}
}

func (m Model) buildContent() string {
	var sb strings.Builder
	for _, msg := range m.messages {
		if m.filter != "" && !strings.Contains(strings.ToLower(msg.Content), strings.ToLower(m.filter)) {
			continue
		}
		sb.WriteString(m.formatMessage(msg))
	}
	return sb.String()
}

func (m *Model) refreshViewport() {
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(m.buildContent())
	if atBottom {
		m.viewport.GotoBottom()
	}
}

func (m *Model) updateInputState() {
	if m.state != stateChat && m.state != stateInputCommand {
		return
	}

	if strings.HasPrefix(m.textInput.Value(), ":") {
		m.state = stateInputCommand
		return
	}

	if m.state == stateInputCommand {
		m.state = stateChat
	}
}

func (m *Model) connectCmd() tea.Cmd {
	return func() tea.Msg {
		m.twitch.Connect()
		return nil
	}
}

func waitForSystemMsg(sub chan string) tea.Cmd {
	return func() tea.Msg {
		return systemMsg(<-sub)
	}
}

func waitForChatMsg(sub chan twitch.ChatMessage) tea.Cmd {
	return func() tea.Msg {
		return <-sub
	}
}

func formatSystemMessage(content string) twitch.ChatMessage {
	return twitch.ChatMessage{
		Time:    time.Now(),
		User:    "System",
		Flare:   "SYSTEM",
		Content: content,
	}
}

func (m *Model) updateViewport(msg tea.WindowSizeMsg) {
	headerHeight := 3
	footerHeight := 1
	vpHeight := msg.Height - headerHeight - footerHeight

	m.width = msg.Width
	m.height = msg.Height
	m.textInput.Width = msg.Width - 4

	if !m.ready {
		m.viewport = viewport.New(msg.Width, vpHeight)
		m.ready = true
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = vpHeight
	}

	m.viewport.GotoBottom()
}

func (m *Model) sendMsgCmd(content string) tea.Cmd {
	return func() tea.Msg {
		if strings.HasPrefix(content, ":") {
			return nil
		} else {
			m.twitch.Say(content)

			return twitch.ChatMessage{
				Time:    time.Now(),
				User:    m.config.Twitch.User,
				Flare:   "TUI",
				Content: content,
			}
		}
	}
}

func (m Model) wrapText(text string, width int) string {
	if width <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		wordWidth := lipgloss.Width(word)
		currentWidth := lipgloss.Width(currentLine.String())

		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentWidth+1+wordWidth <= width {
			currentLine.WriteString(" " + word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}

func (m Model) getStyles() ThemeStyles {
	return ThemeStyles{
		Crust:     lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Crust)),
		Mantle:    lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Mantle)),
		Base:      lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Base)),
		Surface0:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Surface0)),
		Surface1:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Surface1)),
		Surface2:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Surface2)),
		Overlay0:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Overlay0)),
		Overlay1:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Overlay1)),
		Overlay2:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Overlay2)),
		Subtext0:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Subtext0)),
		Subtext1:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Subtext1)),
		Text:      lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Text)),
		Lavender:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Lavender)),
		Blue:      lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Blue)),
		Sapphire:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Sapphire)),
		Sky:       lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Sky)),
		Teal:      lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Teal)),
		Green:     lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Green)),
		Yellow:    lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Yellow)),
		Peach:     lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Peach)),
		Maroon:    lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Maroon)),
		Red:       lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Red)),
		Mauve:     lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Mauve)),
		Pink:      lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Pink)),
		Flamingo:  lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Flamingo)),
		Rosewater: lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Rosewater)),
	}
}
