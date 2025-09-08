package spinner

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var exitOnce sync.Once

type Message struct {
	ID any
	// Text    string
	Bytes  int64
	Error  error
	IsDone bool
}

type Model struct {
	C          chan Message
	ctx        context.Context
	cancelFunc context.CancelFunc
	spinner    spinner.Model
	width      int
	program    *tea.Program
	units      map[any]*unit
	exiting    bool
	// used to quit/exit the spinner if all units are done - prevents from always checking if all units are done
	doneCount int
}

// Unit can be whatever satisfies UnitProvider interface.
// TODO: support multiple colors/shapes per unit. Support multiple spinner shapes
func New[T UnitProvider](ctx context.Context, units []T, cancelFunc context.CancelFunc) *Model {
	su := make(map[any]*unit, len(units))
	c := make(chan Message, len(units))

	doneCount := 0
	for _, u := range units {
		su[u.GetID()] = &unit{
			title: u.GetTitle(),
			err:   u.GetError(),
		}

		if u.GetError() != nil {
			doneCount++
		}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		ctx:        ctx,
		units:      su,
		spinner:    s,
		doneCount:  doneCount,
		C:          c,
		cancelFunc: cancelFunc,
	}

	return m
}

func (m *Model) waitForMsg() tea.Cmd {
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
			// TODO: handle size conversion to largest possible size. Also needs to mutate download speed, speed size, possibly ETA

			// unit.downloadSize += float64(msg.Bytes)
			// if unit.downloadSize >= 1024 && int(unit.downloadUnitSize) < len(sizeUnits)-1 {
			// 	unit.downloadSize /= 1024
			// 	unit.downloadUnitSize++
			//

			unit.totalBytes += float64(msg.Bytes)

			if unit.startTime.IsZero() {
				unit.startTime = time.Now()
			}

			if msg.IsDone {
				unit.isDone = true
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
		return m, tea.Batch(cmd, m.waitForMsg())
	}
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

	fmt.Println("called exit()")

	for _, unit := range m.units {
		unit.isDone = true
	}

	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	time.Sleep(time.Second * 5)
	close(m.C)

	return m, tea.Quit
}

// Update ticker
func (m *Model) updateTime() {
	for _, unit := range m.units {
		if !unit.isDone && unit.totalBytes > 0 {
			unit.elapsed = time.Since(unit.startTime)
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

	// calculate speed
	// elapsedSeconds := elapsed.Seconds()
	// if elapsedSeconds > 0 {
	// }
	bytesPerSec := total / elapsed.Seconds()

	return fmt.Sprintf("(%.1f %s | %.2f MB/s) [%s]", totalConverted, sizeUnits[i], bytesPerSec, elapsed.Truncate(time.Second))
}

func (m Model) formatMessage(u *unit) string {
	w := m.width - 4
	if u.err != nil {
		return errorMsg(u.err.Error())
	}

	var str strings.Builder

	progMsg := progressMsg(u.totalBytes, u.elapsed)

	if u.isDone {
		str.WriteString(successMsg(u.title, progMsg))
	} else {
		parts := []string{
			m.spinner.View(),
			strings.Join([]string{u.title, progMsg}, " "),
		}
		str.WriteString(strings.Join(parts, " "))
	}

	return wrapText(str.String(), w)
}

func successMsg(text, message string) string {
	return fmt.Sprintf("✅ %s %s", text, message)
}

func errorMsg(errMsg string) string {
	return fmt.Sprintf("❌ %s", errMsg)
}
