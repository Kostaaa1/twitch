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
	ID    any
	Bytes int64
	Err   error
	Done  bool
}

type Model struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	spinner    spinner.Model
	width      int

	program *tea.Program
	units   map[any]*unit
	exiting bool
	// used to quit/exit the spinner if all units are done - prevents from always checking if all units are done
	doneCount int
	C         chan Message
}

func spinnerUnitsFromSlice[T UnitProvider](units []T) (map[any]*unit, int) {
	doneCount := 0
	su := make(map[any]*unit, len(units))

	for _, u := range units {
		su[u.GetID()] = &unit{
			title: u.GetTitle(),
			err:   u.GetError(),
		}
		if u.GetError() != nil {
			doneCount++
		}
	}

	return su, doneCount
}

// Unit can be whatever satisfies UnitProvider interface.
// TODO: support multiple colors/shapes per unit. Support multiple spinner shapes
func New[T UnitProvider](ctx context.Context, units []T) *Model {
	su, doneCount := spinnerUnitsFromSlice(units)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ctx, cancel := context.WithCancel(ctx)

	return &Model{
		ctx:        ctx,
		cancelFunc: cancel,
		units:      su,
		spinner:    s,
		doneCount:  doneCount,
		C:          make(chan Message, len(units)),
	}
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
	if len(m.units) == m.doneCount {
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
			if unit, ok := m.units[msg.ID]; ok {
				unit.totalBytes += float64(msg.Bytes)
				if unit.startTime.IsZero() {
					unit.startTime = time.Now()
				}

				unit.err = msg.Err

				if msg.Done {
					unit.done = true
					m.doneCount++
				}
			}

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
		str.WriteString(m.formatMessage(unit))
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

func (m *Model) exit() (tea.Model, tea.Cmd) {
	if m.exiting {
		return m, nil
	}
	m.exiting = true

	for _, unit := range m.units {
		unit.done = true
	}

	m.cancelFunc()
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

func (m Model) formatMessage(u *unit) string {
	if m.width == 0 {
		return ""
	}

	if u.err != nil {
		return errorMsg(u.err.Error())
	}

	var str strings.Builder

	progMsg := progressMsg(u.totalBytes, u.elapsed)
	title := wordBreak(u.title, m.width-10)

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

func successMsg(text, message string) string {
	return fmt.Sprintf("✅ %s %s", text, message)
}

func errorMsg(errMsg string) string {
	return fmt.Sprintf("❌ %s", errMsg)
}

func wordBreak(s string, limit int) string {
	if len(s) < limit {
		return s
	}
	p1 := s[:limit]
	p2 := s[limit+1:]
	return fmt.Sprintf("%s\n%s", p1, wordBreak(p2, limit))
}
