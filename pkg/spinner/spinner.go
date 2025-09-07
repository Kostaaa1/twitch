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

type Message struct {
	Text    string
	Message string
	Bytes   int64
	Error   error
	IsDone  bool
}

type unit struct {
	Title   string
	Message string
	Err     error

	TotalBytes  float64
	StartTime   time.Time
	ElapsedTime time.Duration

	IsDone bool
}

type UnitProvider interface {
	GetTitle() string
	GetError() error
}

type Model struct {
	cancelFunc context.CancelFunc
	units      []unit
	spinner    spinner.Model
	err        error
	width      int
	doneCount  int
	program    *tea.Program
	C          chan Message
}

var (
	units      = []string{"B", "KB", "MB", "GB", "TB"}
	spinnerMap = map[string]spinner.Spinner{
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
		"dot": {
			Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
			FPS:    time.Second / 10,
		},
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.waitForMsg())
}

func (m *Model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		return <-m.C
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case Message:
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

func (m *Model) exit() {
	for i := 0; i < len(m.units); i++ {
		m.units[i].IsDone = true
	}
	m.cancelFunc()
}

func (m *Model) updateTime() {
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

func (m Model) View() string {
	if m.err != nil {
		return m.err.Error()
	}

	var str strings.Builder

	for i := 0; i < len(m.units); i++ {
		fullMsg := m.formatMessage(m.units[i])
		str.WriteString(fullMsg)
		str.WriteString("\n")
	}

	return str.String()
}

func wrapText(s string, limit int) string {
	if limit <= 0 || len(s) < limit {
		return s
	}

	indexes := []int{}
	for i := 0; i < len(s); i++ {
		if i > 0 && i%limit == 0 {
			indexes = append(indexes, i)
		}
	}

	parts := []string{}
	for i, index := range indexes {
		if len(parts) == 0 {
			parts = append(parts, s[0:index])
		}
		if len(indexes)-1 == i {
			parts = append(parts, s[index:])
			break
		}
		parts = append(parts, s[index:indexes[i+1]])
	}

	return strings.Join(parts, "\n")
}

func (m Model) formatMessage(u unit) string {
	if u.Err != nil {
		return wrapText(errorMsg(u.Err), m.width-4)
	}

	var str strings.Builder

	progress := getProgress(u.TotalBytes, u.ElapsedTime)
	if u.IsDone {
		str.WriteString(wrapText(successMsg(u.Title, progress), m.width-4))
	} else {
		parts := []string{
			m.spinner.View(),
			strings.Join([]string{wrapText(u.Title, m.width-4), progress, u.Message}, " "),
		}
		str.WriteString(strings.Join(parts, " "))
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

// Unit can be whatever satisfies UnitProvider interface
func New[T UnitProvider](units []T, spinnerModel string, cancelFunc context.CancelFunc) *Model {
	// TODO: CHANGE STRUCTURE, WE NEED O(1) ACCESS TO UNITS
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

	return &Model{
		cancelFunc: cancelFunc,
		units:      su,
		spinner:    s,
		doneCount:  doneCount,
		C:          make(chan Message, len(units)),
	}
}

// func (m *Model) ProgressChan() chan Message {
// 	return m.ch
// }

func (m *Model) Run() {
	m.program = tea.NewProgram(m)
	if _, err := m.program.Run(); err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		panic(err)
	}
}

func (m Model) Quit() {
	if m.program != nil {
		m.program.Quit()
	}
}
