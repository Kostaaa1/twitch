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

type Message struct {
	ID    string
	Bytes int64
	Err   error
	Done  bool
}

// TODO:
// 1. Maybe support multiple colors and spinner.Spinner per unit. If that is the case, we need to redesign this and display spinner per unit (not one spinner with multiple units). Then i would display each per unit.
// 2. Maybe implement write method, where we would need to use gob for encoding/decoding struct bytes.

type Model struct {
	ctx     context.Context
	cancel  context.CancelFunc
	spinner spinner.Model
	width   int
	program *tea.Program
	units   []*unit
	exiting bool
	// used to quit/exit the spinner if all units are done - prevents from always checking if all units are done
	doneCount int
	C         chan Message
}

type spinnerOpts func(m *Model)

func WithCancelFunc(cancel context.CancelFunc) spinnerOpts {
	return func(m *Model) {
		m.cancel = cancel
	}
}

func spinnerUnitsSliceFromSlice[T UnitProvider](units []T) ([]*unit, int) {
	doneCount := 0
	su := make([]*unit, len(units))

	for i, u := range units {
		su[i] = &unit{
			title: u.GetID(),
			err:   u.GetError(),
		}
		if u.GetError() != nil {
			doneCount++
		}
	}

	return su, doneCount
}

func New[T UnitProvider](ctx context.Context, units []T, opts ...spinnerOpts) *Model {
	su, doneCount := spinnerUnitsSliceFromSlice(units)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		ctx:       ctx,
		units:     su,
		spinner:   s,
		doneCount: doneCount,
		C:         make(chan Message, len(units)),
	}

	for _, opt := range opts {
		opt(m)
	}

	return m
}

func (m Model) waitForMsg() tea.Cmd {
	return func() tea.Msg {
		return <-m.C
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.waitForMsg())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.doneCount >= len(m.units) {
		return m.exit()
	}

	select {
	case <-m.ctx.Done():
		return m.exit()
	default:
		switch msg := msg.(type) {
		case tea.QuitMsg:
			return m.exit()

		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc", "ctrl+c":
				return m.exit()
			default:
				return m, nil
			}

		case tea.WindowSizeMsg:
			m.width = msg.Width
			return m, nil

		case Message:
			m.updateProgressUnits(msg)
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, tea.Batch(cmd, m.waitForMsg())

		default:
			m.updateTime()
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}
}

func (m Model) View() string {
	var str strings.Builder
	for _, unit := range m.units {
		str.WriteString(m.printUnit(unit))
		str.WriteString("\n")
	}
	return str.String()
}

func (m *Model) Run() {
	m.program = tea.NewProgram(m)
	if _, err := m.program.Run(); err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		panic(err)
	}
}

func (m *Model) updateProgressUnits(msg Message) {
	for i := range m.units {
		if m.units[i].title == msg.ID {
			m.units[i].totalBytes += float64(msg.Bytes)

			if m.units[i].startTime.IsZero() {
				m.units[i].startTime = time.Now()
			}

			m.units[i].done = msg.Done

			if msg.Done || msg.Err != nil {
				m.doneCount++
			}

			if msg.Err != nil {
				m.units[i].err = msg.Err
				m.units[i].done = true
			}
		}
	}
}

func (m *Model) exit() (tea.Model, tea.Cmd) {
	if m.exiting {
		return m, nil
	}
	m.exiting = true

	for _, unit := range m.units {
		unit.done = true
	}

	m.cancel()
	return m, tea.Quit
}

func (m *Model) updateTime() {
	for _, unit := range m.units {
		if !unit.done && unit.totalBytes > 0 {
			unit.elapsed = time.Since(unit.startTime)
		}
	}
}

func downloadSpeedMBs(bytes float64, elapsed time.Duration) float64 {
	elapsedSeconds := elapsed.Seconds()
	if elapsedSeconds == 0 {
		return 0
	}
	bytesPerSecond := bytes / elapsedSeconds
	kilobytesPerSecond := bytesPerSecond / (1024 * 1024)
	return kilobytesPerSecond
}

func progressMsg(total float64, elapsed time.Duration) string {
	totalConverted := total

	i := 0
	for totalConverted >= 1024 && i < len(sizeUnits)-1 {
		totalConverted /= 1024
		i++
	}

	speed := downloadSpeedMBs(total, elapsed)

	return fmt.Sprintf("(%.1f %s | %.2f MB/s) [%s]", totalConverted, sizeUnits[i], speed, elapsed.Truncate(time.Second))
}

func (m Model) printUnit(u *unit) string {
	if m.width == 0 {
		return ""
	}

	if u.err != nil {
		return errorMsg(u.title, u.err.Error())
	}

	var str strings.Builder

	progMsg := progressMsg(u.totalBytes, u.elapsed)
	title := u.title

	if u.done {
		str.WriteString(successMsg(title, progMsg))
	} else {
		parts := []string{
			m.spinner.View(),
			strings.Join([]string{title, progMsg}, " "),
		}
		str.WriteString(strings.Join(parts, " "))
	}

	return str.String()
}

func successMsg(title, message string) string {
	return fmt.Sprintf("✅ %s %s", title, message)
}

func errorMsg(title, errMsg string) string {
	return fmt.Sprintf("❌ %s %s", title, errMsg)
}

func wordBreak(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	p1 := s[:limit]
	p2 := s[limit+1:]
	return fmt.Sprintf("%s\n%s", p1, wordBreak(p2, limit))
}
