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

// theme converted from hex to lipgloss
type ThemeStyles struct {
	Base      lipgloss.Style
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
	stateView                  // Normal mode - hjkl navigation, i for insert, : for command
	stateInputChat             // Insert mode - typing chat messages
	stateInputCommand          // Command mode - typing commands
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

	state := stateInputChannel    // starting state is Channel select
	if cfg.Twitch.Channel != "" { // when a channel already in cfg switch to view state
		state = stateView
		ti.Blur()
	}

	return Model{
		state:     state,
		config:    cfg,
		textInput: ti,
		twitch:    twitch.New(cfg),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		textinput.Blink,
		waitForSystemMsg(m.twitch.SysChan), // listen to the system messages
		tea.Tick(time.Second, func(_ time.Time) tea.Msg { return tickMsg{} }), // add tick messages - updating the time correctly
	}

	if m.state == stateView {
		cmds = append(cmds, m.connectCmd(), waitForChatMsg(m.twitch.MsgChan)) // listen to the twitch chat messages
	}

	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg: // tick update
		return m, tea.Tick(time.Second, func(_ time.Time) tea.Msg { return tickMsg{} })

	case systemMsg: // print system message
		m.handleScroll(formatSystemMessage(string(msg)))
		return m, waitForSystemMsg(m.twitch.SysChan)

	case twitch.ChatMessage: // print chat message
		m.handleScroll(msg)
		return m, waitForChatMsg(m.twitch.MsgChan)

	case tea.KeyMsg: // user key press
		return m.handleKey(msg)

	case tea.WindowSizeMsg: // update view on window resize
		m.updateViewport(msg)
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	// print the shortcut commands in the inout and the header (CTRL + J => :join)
	var tiCmd, vpCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.updateInputState()

	return m, tea.Batch(tiCmd, vpCmd)
}

// handle keypresses that are not in the textbox (shortcuts)
func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+q":
		return m, tea.Quit

	case "ctrl+c":
		if m.state == stateView || m.state == stateInputChat {
			m.setCommand(":config ")
		}
		return m, nil

	case "ctrl+f":
		if m.state == stateView || m.state == stateInputChat {
			m.setCommand(":find ")
		}
		return m, nil

	case "ctrl+j":
		if m.state == stateView || m.state == stateInputChat {
			m.setCommand(":join ")
		}
		return m, nil

	case "enter":
		return m.handleEnter()

	case "i": // switch to input state
		if m.state == stateView {
			m.state = stateInputChat
			m.textInput.Focus()
			m.textInput.SetValue("")
			return m, nil
		}

	case ":": // switch to command state
		if m.state == stateView {
			m.setCommand(":")
			return m, nil
		}

	case "esc": // switch form input / command to view state
		if m.state == stateInputChat || m.state == stateInputCommand {
			m.state = stateView
			m.textInput.Blur()
			m.textInput.Reset()
			return m, nil
		}
	}

	var tiCmd, vpCmd tea.Cmd
	m.textInput, tiCmd = m.textInput.Update(msg)
	m.updateInputState()

	if m.state == stateView {
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

// set the input as focused, add the prefill, set the ui state
func (m *Model) setCommand(prefix string) {
	m.textInput.Focus()
	m.textInput.SetValue(prefix)
	m.textInput.SetCursor(len(prefix))
	m.state = stateInputCommand
}

// send chat / command
func (m *Model) handleEnter() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textInput.Value())

	if strings.HasPrefix(input, ":") {
		m.textInput.Reset()
		m.state = stateView
		m.textInput.Blur()
		return m, m.executeCommand(input)
	}

	switch m.state {
	case stateInputChannel: // change the channel and switch to view mode - not on the command but the the starting state
		if input == "" {
			return m, nil
		}
		m.twitch.CurrentChannel = input
		m.twitch.ChannelID = ""
		m.config.Twitch.Channel = input
		m.config.Twitch.ChannelID = ""
		m.config.Twitch.UserID = m.twitch.UserID
		if err := config.UpdateConfig(m.config); err != nil {
			m.handleScroll(formatSystemMessage("Failed to save config: " + err.Error()))
		}
		m.textInput.Reset()
		m.textInput.Placeholder = "Send a message..."
		m.state = stateView
		m.textInput.Blur()
		return m, tea.Batch(m.connectCmd(), waitForChatMsg(m.twitch.MsgChan))

	case stateInputChat, stateInputCommand:
		if input != "" {
			m.textInput.Reset()
			m.state = stateView
			m.textInput.Blur()
			return m, m.sendMsgCmd(input)
		}
	}

	return m, nil
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

// handle the auto scroll down - on message - but disable when user scrolls up
func (m *Model) handleScroll(msg twitch.ChatMessage) {
	atBottom := m.viewport.AtBottom()
	m.messages = append(m.messages, msg)
	m.viewport.SetContent(m.buildContent())
	if atBottom {
		m.viewport.GotoBottom()
	}
}

// build the message view
func (m Model) buildContent() string {
	var sb strings.Builder
	// go through all the messages and apply the filter - when avaiable - and then print them
	for _, msg := range m.messages {
		if m.filter != "" && !strings.Contains(strings.ToLower(msg.Content), strings.ToLower(m.filter)) {
			continue
		}
		sb.WriteString(m.formatMessage(msg))
	}
	return sb.String()
}

// handle the auto scroll down - on viewport change - but disable when user scrolls up
func (m *Model) refreshViewport() {
	atBottom := m.viewport.AtBottom()
	m.viewport.SetContent(m.buildContent())
	if atBottom {
		m.viewport.GotoBottom()
	}
}

// switch the ui state depending on the input - : = command
func (m *Model) updateInputState() {
	if m.state == stateInputChat && strings.HasPrefix(strings.TrimSpace(m.textInput.Value()), ":") {
		m.state = stateInputCommand
		return
	}

	if m.state == stateInputCommand && !strings.HasPrefix(strings.TrimSpace(m.textInput.Value()), ":") {
		m.state = stateInputChat
	}
}

// init twitch connection
func (m *Model) connectCmd() tea.Cmd {
	return func() tea.Msg {
		m.twitch.Connect()
		return nil
	}
}

// init twitch channel switching
func (m *Model) switchChannelCmd(channel string) tea.Cmd {
	return func() tea.Msg {
		if err := m.twitch.SwitchChannel(channel); err != nil {
			return systemMsg("Join failed: " + err.Error())
		}
		return nil
	}
}

// send twicht chat message
func (m *Model) sendMsgCmd(content string) tea.Cmd {
	return func() tea.Msg {
		m.twitch.Say(content)
		return twitch.ChatMessage{
			Time:    time.Now(),
			User:    m.config.Twitch.User,
			Flare:   "TUI",
			Content: content,
		}
	}
}

// on window size change recalculate the viewport size
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

	m.refreshViewport()
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

// convert cfg theme to lipgloss Theme
func (m Model) getStyles() ThemeStyles {
	return ThemeStyles{
		Base:      lipgloss.NewStyle().Foreground(lipgloss.Color(m.config.Theme.Base)),
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
