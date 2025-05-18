package spinner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error

type quitMsg struct{}

type ChannelMessage struct {
	Text    string
	Message string
	Bytes   int64
	Error   error
	IsDone  bool
}

type unit struct {
	Title       string
	Message     string
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
	// ctx       context.Context
	cancelFunc context.CancelFunc
	units      []unit
	spinner    spinner.Model
	err        error
	width      int
	doneCount  int
	program    *tea.Program
	progChan   chan ChannelMessage
}

var (
	units      = []string{"B", "KB", "MB", "GB", "TB"}
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
		return <-m.progChan
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(m.units) == m.doneCount {
		m.exit()
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.QuitMsg:
		m.exit()
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.exit()
			return m, tea.Quit
		default:
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case errMsg:
		m.err = msg
		return m, nil

	case ChannelMessage:
		for i := range m.units {
			if m.units[i].Title == msg.Text {
				if msg.Error != nil {
					m.units[i].Err = msg.Error
				}

				m.units[i].Message = msg.Message

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

func (m *model) exit() {
	for i := 0; i < len(m.units); i++ {
		m.units[i].IsDone = true
	}
	m.cancelFunc()
}

func (m *model) updateTime() {
	for i := range m.units {
		if !m.units[i].IsDone && m.units[i].TotalBytes > 0 {
			m.units[i].ElapsedTime = time.Since(m.units[i].StartTime)
		}
	}
}

func downloadSpeedMBs(bytes float64, elapsedTime time.Duration) float64 {
	elapsedSeconds := elapsedTime.Seconds()
	if elapsedSeconds == 0 {
		return 0
	}
	bytesPerSecond := bytes / elapsedSeconds
	kilobytesPerSecond := bytesPerSecond / (1024 * 1024)
	return kilobytesPerSecond
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	var str strings.Builder
	for i := 0; i < len(m.units); i++ {
		str.WriteString(m.wrapText(m.formatMessage(m.units[i])) + "\n")
	}
	return str.String()
}

func (m model) wrapText(text string) string {
	return lipgloss.NewStyle().Width(m.width - 5).Render(text)
}

func (m model) formatMessage(u unit) string {
	if u.Err != nil {
		return errorMsg(u.Err)
	}

	var str strings.Builder
	progress := getProgress(u.TotalBytes, u.ElapsedTime)
	if u.IsDone {
		str.WriteString(successMsg(u.Title, progress))
	} else {
		str.WriteString(fmt.Sprintf(" %s %s %s %s", m.spinner.View(), u.Title, progress, u.Message))
	}
	return str.String()
}

func getProgress(total float64, elapsed time.Duration) string {
	totalConverted := total
	i := 0
	for totalConverted >= 1024 && i < len(units)-1 {
		totalConverted /= 1024
		i++
	}
	speed := downloadSpeedMBs(total, elapsed)
	msg := fmt.Sprintf("(%.1f %s | %.2f MB/s) [%s]", totalConverted, units[i], speed, elapsed.Truncate(time.Second))
	return msg
}

func successMsg(text, message string) string {
	return fmt.Sprintf("✅ %s %s", text, message)
}

func errorMsg(err error) string {
	return fmt.Sprintf("❌ %s", err.Error())
}

func New[T UnitProvider](units []T, spinnerModel string, cancelFunc context.CancelFunc) *model {
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
		cancelFunc: cancelFunc,
		units:      su,
		spinner:    s,
		doneCount:  doneCount,
		progChan:   progChan,
	}
}

func (m *model) ProgChan() chan ChannelMessage {
	return m.progChan
}

func (m *model) Run() {
	m.program = tea.NewProgram(m)
	if _, err := m.program.Run(); err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		panic(err)
	}
}

func (m model) Quit() {
	if m.program != nil {
		m.program.Quit()
	}
}
