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

				// m.state[i].CurrentTime = time.Since(m.state[i].StartTime).Seconds()
				// m.state[i].ByteCount.Convert()
				// if m.state[i].CurrentTime > 0 {
				// m.state[i].KBsPerSecond = float64(m.state[i].ByteCount) / (1024.0 * 1024.0) / m.state[i].CurrentTime
				// }

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

func (m *model) getProgressMsg(total float64, elapsed time.Duration) string {
	b := bytecount.ConvertBytes(total)
	downloadMsg := fmt.Sprintf("(%.1f %s) [%s]", b.Total, b.Unit, elapsed.Truncate(time.Second))
	return downloadMsg
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	var str strings.Builder
	for i := 0; i < len(m.state); i++ {
		str.WriteString(m.constructStateMessage(m.state[i]))
	}
	return str.String()
}

func (m model) constructStateMessage(s Spinner) string {
	if s.err != nil {
		return constructErrorMessage(s.text, s.err)
	}
	message := m.getProgressMsg(s.totalBytes, s.elapsedTime)
	if s.isDone {
		return constructSuccessMessage(s.text, message)
	} else {
		return fmt.Sprintf(" %s %s: %s \n", m.spinner.View(), s.text, message)
	}
}

func constructSuccessMessage(text, message string) string {
	return fmt.Sprintf("✅ %s: %s \n", text, message)
}

func constructErrorMessage(text string, err error) string {
	if err == nil {
		return ""
	}
	prefix := "❌ "
	if text != "" {
		prefix += text + ": "
	}
	return fmt.Sprintf("%s%s\n", prefix, err.Error())
}

func New(units []twitch.MediaUnit, progressChan chan twitch.ProgresbarChanData, cfg config.Downloader) {
	p := tea.NewProgram(initialModel(units, progressChan, cfg))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error starting program: %v", err)
	}
}
