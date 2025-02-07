package root

import (
	"fmt"
	"os"
	"strings"

	"github.com/Kostaaa1/twitch/cli/chat/view/chat"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var mainColor = lipgloss.Color("#8839ef")
var docStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(mainColor)

type item struct {
	title, desc string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list   list.Model
	twitch *twitch.TWClient
	cfg    *config.Data
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if msg.String() == "enter" {
			item := m.list.SelectedItem().FilterValue()
			if strings.HasPrefix(item, "Chats") {
				chat.Open(m.twitch, m.cfg)
			}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.list.View())
}

func Open(twitch *twitch.TWClient, cfg *config.Data) {
	var items = []list.Item{
		item{title: "Chats", desc: "Open chats."},
		item{title: "???", desc: "Add list."},
	}
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(mainColor).BorderLeftForeground(mainColor)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(mainColor).BorderLeftForeground(mainColor)
	d.Styles.FilterMatch = d.Styles.FilterMatch.Underline(false)

	m := model{
		list:   list.New(items, d, 0, 0),
		twitch: twitch,
		cfg:    cfg,
	}
	m.list.Title = " Twitch "
	m.list.Styles.Title = lipgloss.NewStyle().Background(mainColor).Bold(true)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
