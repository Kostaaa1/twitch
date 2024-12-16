package spinner

import (
	"fmt"
	"strings"
	"time"

	"github.com/Kostaaa1/twitch/internal/bytecount"
	"github.com/Kostaaa1/twitch/internal/config"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error

type quitMsg struct{}

type ChannelMessage struct {
	Text   string
	Bytes  int64
	Error  error
	IsDone bool
}

type unit struct {
	Title       string
	TotalBytes  float64
	StartTime   time.Time
	ElapsedTime time.Duration
	IsDone      bool
	Err         error
}

type UnitProvider interface {
	GetTitle() string
	GetError() error
}

type model struct {
	units     []unit
	spinner   spinner.Model
	err       error
	width     int
	doneCount int
	program   *tea.Program
	ProgChan  chan ChannelMessage
}

var (
	spinnerMap = map[string]spinner.Spinner{
		"dot": {
			Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
			FPS:    time.Second / 10,
		},
		"meter":     spinner.Meter,
		"line":      spinner.Line,
		"pulse":     spinner.Pulse,
		"ellipsis":  spinner.Ellipsis,
		"jump":      spinner.Jump,
		"points":    spinner.Points,
		"globe":     spinner.Globe,
		"hamburger": spinner.Hamburger,
		"minidot":   spinner.MiniDot,
		"monkey":    spinner.Monkey,
		"moon":      spinner.Moon,
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

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.waitForMsg())
}

func (m *model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		return <-m.ProgChan
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.units) == m.doneCount {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case quitMsg:
		return m, tea.Quit

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

	case ChannelMessage:
		for i := range m.units {
			if m.units[i].Title == msg.Text {
				if msg.Error != nil {
					m.units[i].Err = msg.Error
				}

				if m.units[i].StartTime.IsZero() {
					m.units[i].StartTime = time.Now()
				}
				m.units[i].TotalBytes += float64(msg.Bytes)

				if msg.IsDone {
					m.units[i].IsDone = true
					m.doneCount++
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
	for i := range m.units {
		if !m.units[i].IsDone && m.units[i].TotalBytes > 0 {
			m.units[i].ElapsedTime = time.Since(m.units[i].StartTime)
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
	for i := 0; i < len(m.units); i++ {
		str.WriteString(m.wrapText(m.constructStateMessage(m.units[i])))
		str.WriteString("\n")
	}
	return str.String()
}

func (m model) wrapText(text string) string {
	return lipgloss.NewStyle().Width(m.width - 5).Render(text)
}

func (m model) constructStateMessage(s unit) string {
	if s.Err != nil {
		return constructErrorMessage(s.Err)
	}

	var str strings.Builder

	message := m.getProgressMsg(s.TotalBytes, s.ElapsedTime)
	if s.IsDone {
		str.WriteString(constructSuccessMessage(s.Title, message))
	} else {
		str.WriteString(fmt.Sprintf(" %s %s %s", m.spinner.View(), s.Title, message))
	}

	return str.String()
}

func constructSuccessMessage(text, message string) string {
	return fmt.Sprintf("✅ %s %s", text, message)
}

func constructErrorMessage(err error) string {
	return fmt.Sprintf("❌ %s", err.Error())
}

func initModel(cfg config.Downloader) model {
	s := spinner.New()
	s.Spinner = validateSpinnerModel(cfg.SpinnerModel)
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{
		spinner: s,
	}
}

func New[T UnitProvider](units []T, spinnerModel string) *model {
	progChan := make(chan ChannelMessage, len(units))
	su := make([]unit, len(units))

	doneCount := 0
	for i, u := range units {
		err := u.GetError()
		su[i] = unit{
			Title:       u.GetTitle(),
			TotalBytes:  0,
			ElapsedTime: 0,
			IsDone:      false,
			Err:         err,
		}
		if err != nil {
			doneCount++
		}

	}

	s := spinner.New()
	s.Spinner = validateSpinnerModel(spinnerModel)
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return &model{
		units:     su,
		spinner:   s,
		doneCount: doneCount,
		ProgChan:  progChan,
	}
}

func (m *model) Run() {
	m.program = tea.NewProgram(m)
	if _, err := m.program.Run(); err != nil {
		fmt.Printf("Error starting programram: %v", err)
		panic(err)
	}
}

func (m model) Close() {
	m.program.Send(quitMsg{})
}
