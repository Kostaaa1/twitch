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
	Total float64
	Err   error
	Done  bool
}

type unit struct {
	label     string
	err       error
	byteCount float64
	total     float64
	estimated time.Time     // estimated time for finish (based on total)
	startTime time.Time     //
	elapsed   time.Duration // how much time passed since start
	done      bool
}

type UnitProvider interface {
	GetID() string
}

type Model struct {
	ctx       context.Context
	cancel    context.CancelFunc
	spinner   spinner.Model
	width     int
	program   *tea.Program
	units     []*unit
	exiting   bool
	doneCount int
	C         chan Message
}

var (
	colorMuted = lipgloss.Color("#8B8B8B")
	mutedStyle = lipgloss.NewStyle().Foreground(colorMuted)
)

type spinnerOpts func(m *Model)

func WithCancelFunc(cancel context.CancelFunc) spinnerOpts {
	return func(m *Model) {
		m.cancel = cancel
	}
}

func New(ctx context.Context, opts ...spinnerOpts) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		units:     make([]*unit, 0),
		ctx:       ctx,
		spinner:   s,
		doneCount: 0,
		C:         make(chan Message),
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
			m.updateModelUnit(msg)
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

func (m *Model) updateModelUnit(msg Message) {
	found := false

	for i := 0; i < len(m.units); i++ {
		unit := m.units[i]
		if unit.label == msg.ID {
			unit.byteCount += float64(msg.Bytes)
			if unit.startTime.IsZero() {
				unit.startTime = time.Now()
			}
			unit.done = msg.Done
			if msg.Err != nil {
				unit.err = msg.Err
				unit.done = true
			}
			if msg.Done {
				m.doneCount++
			}
			found = true
		}
	}

	if !found {
		newunit := &unit{
			label:     msg.ID,
			err:       msg.Err,
			byteCount: 0,
			elapsed:   0,
			startTime: time.Time{},
			estimated: time.Time{},
			total:     0,
			done:      false,
		}
		if msg.Err != nil {
			m.doneCount++
			newunit.done = true
		}
		m.units = append(m.units, newunit)
	}
}

func (m Model) View() string {
	var str strings.Builder
	for _, unit := range m.units {
		str.WriteString(m.printUnit(unit))
	}

	style := lipgloss.NewStyle()
	if m.width > 0 {
		style = style.Width(m.width)
	}

	return style.Render(str.String())
}

func (m *Model) Run() {
	m.program = tea.NewProgram(m)
	if _, err := m.program.Run(); err != nil {
		fmt.Printf("Error starting program: %v\n", err)
		// panic(err)
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

	close(m.C)
	m.cancel()
	return m, tea.Quit
}

func (m *Model) updateTime() {
	for _, unit := range m.units {
		if !unit.done && unit.byteCount > 0 {
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

	return fmt.Sprintf(
		"%.1f %s . %.2f MB/s . %s",
		totalConverted,
		sizeUnits[i],
		downloadSpeedMBs(total, elapsed),
		elapsed.Truncate(time.Second),
	)
}

func (m Model) printUnit(u *unit) string {
	if m.width == 0 {
		return ""
	}

	label := m.spinner.Style.Render(u.label)

	var str strings.Builder
	if u.err != nil {
		str.WriteString("❌ ")
		str.WriteString(label)
		str.WriteString("\n")
		str.WriteString(mutedStyle.PaddingLeft(3).Render(u.err.Error()))
		str.WriteString("\n")
		return str.String()
	}

	progress := progressMsg(u.byteCount, u.elapsed)

	if u.done {
		str.WriteString("✅ ")
		str.WriteString(label)
		str.WriteString("\n")
		str.WriteString(mutedStyle.PaddingLeft(3).Render(progress))
		str.WriteString("\n")
	} else {
		str.WriteString(" ")
		str.WriteString(m.spinner.View())
		str.WriteString(label)
		str.WriteString("\n")
		str.WriteString(mutedStyle.PaddingLeft(3).Render(progress))
		str.WriteString("\n")
	}

	return str.String()
}
