package spinner

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/bytecount"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/Kostaaa1/twitch/pkg/twitch"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error

type Spinner struct {
	text        string
	totalBytes  float64
	startTime   time.Time
	elapsedTime time.Duration
	isDone      bool
	err         error
}

type model struct {
	state        []Spinner
	progressChan chan twitch.ProgresbarChanData
	spinner      spinner.Model
	err          error
	width        int
}

var (
	spinnerMap = map[string]spinner.Spinner{
		"meter":    spinner.Meter,
		"dot":      spinner.Dot,
		"line":     spinner.Line,
		"pulse":    spinner.Pulse,
		"ellipsis": spinner.Ellipsis,
		"jump":     spinner.Jump,
		"points":   spinner.Points,
	}
)

func validateSpinnerModel(model string) spinner.Spinner {
	_, ok := spinnerMap[model]
	if ok {
		return spinnerMap[model]
	} else {
		return spinnerMap["dot"]
	}
}

func initialModel(units []twitch.MediaUnit, progChan chan twitch.ProgresbarChanData, cfg config.Downloader) model {
	s := spinner.New()
	s.Spinner = validateSpinnerModel(cfg.SpinnerModel)
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		spinner:      s,
		state:        initSpinner(units),
		progressChan: progChan,
	}
}

func initSpinner(units []twitch.MediaUnit) []Spinner {
	var state []Spinner
	for _, unit := range units {
		displayPath := ""
		if f, ok := unit.W.(*os.File); ok && f != nil {
			displayPath = f.Name()
		}
		state = append(state, Spinner{
			text:        displayPath,
			totalBytes:  0,
			elapsedTime: 0,
			isDone:      false,
			err:         unit.Error,
		})
	}
	return state
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.waitForMsg())
}

type chanMsg struct {
	twitch.ProgresbarChanData
}

func (m *model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		return chanMsg{ProgresbarChanData: <-m.progressChan}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		default:
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, nil

	case chanMsg:
		for i := range m.state {
			if m.state[i].text == msg.Text {
				if msg.Error != nil {
					m.state[i].err = msg.Error
				}
				if m.state[i].startTime.IsZero() {
					m.state[i].startTime = time.Now()
				}
				m.state[i].totalBytes += float64(msg.Bytes)

				if msg.IsDone {
					m.state[i].isDone = true
				}
				break
			}
		}

		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, tea.Batch(cmd, m.waitForMsg())

	default:
		m.updateTime()
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, tea.Batch(cmd, m.waitForMsg())
	}
}

func (m *model) updateTime() {
	for i := range m.state {
		if !m.state[i].isDone && m.state[i].totalBytes > 0 {
			m.state[i].elapsedTime = time.Since(m.state[i].startTime)
		}
	}
}

func downloadSpeedKBs(totalBytes float64, elapsedTime time.Duration) float64 {
	elapsedSeconds := elapsedTime.Seconds()
	if elapsedSeconds == 0 {
		return 0
	}
	bytesPerSecond := totalBytes / elapsedSeconds
	kilobytesPerSecond := bytesPerSecond / (1024 * 1024)
	return kilobytesPerSecond
}

func (m *model) getProgressMsg(total float64, elapsed time.Duration) string {
	b := bytecount.ConvertBytes(total)
	speed := downloadSpeedKBs(total, elapsed)
	msg := fmt.Sprintf("(%.1f %s | %.2f Mb/s) [%s]", b.Total, b.Unit, speed, elapsed.Truncate(time.Second))
	return msg
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	var str strings.Builder

	for i := 0; i < len(m.state); i++ {
		msg := m.constructStateMessage(m.state[i])
		if i == 0 {
			msg = strings.Split(msg, "\n")[1]
		}
		str.WriteString(msg)
	}
	return str.String()
}

func (m model) wrapText(text string) string {
	return lipgloss.NewStyle().Width(m.width).Render(text)
}

func (m model) constructStateMessage(s Spinner) string {
	if s.err != nil {
		return m.wrapText(constructErrorMessage(s.err))
	}

	var str strings.Builder
	message := m.getProgressMsg(s.totalBytes, s.elapsedTime)
	if s.isDone {
		str.WriteString(m.wrapText(constructSuccessMessage(s.text, message)))
	} else {
		wrappedText := m.wrapText(fmt.Sprintf("%s %s", s.text, message))
		str.WriteString(fmt.Sprintf("\n %s %s", m.spinner.View(), wrappedText))
	}

	return str.String()
}

func constructSuccessMessage(text, message string) string {
	return fmt.Sprintf(" \n✅ %s %s", text, message)
}

func constructErrorMessage(err error) string {
	return fmt.Sprintf(" \n❌ %s \n", err.Error())
}

func New(units []twitch.MediaUnit, progressChan chan twitch.ProgresbarChanData, cfg config.Downloader) {
	p := tea.NewProgram(initialModel(units, progressChan, cfg))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting program: %v", err)
	}
}
