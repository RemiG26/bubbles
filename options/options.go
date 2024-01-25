package options

import (
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	lastID int
	idMtx  sync.Mutex
)

// Return the next ID we should use on the Model.
func nextID() int {
	idMtx.Lock()
	defer idMtx.Unlock()
	lastID++
	return lastID
}

// New returns a new filepicker model with default styling and key bindings.
func New() Model {
	return Model{
		id:            nextID(),
		Options:       []string{},
		Cursor:        ">",
		selected:      0,
		AutoHeight:    true,
		Height:        0,
		max:           0,
		min:           0,
		selectedStack: newStack(),
		minStack:      newStack(),
		maxStack:      newStack(),
		KeyMap:        DefaultKeyMap(),
		Styles:        DefaultStyles(),
	}
}

type errorMsg struct {
	err error
}

const (
	marginBottom  = 5
	fileSizeWidth = 8
	paddingLeft   = 2
)

// KeyMap defines key bindings for each user action.
type KeyMap struct {
	Down   key.Binding
	Up     key.Binding
	Select key.Binding
}

// DefaultKeyMap defines the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Down:   key.NewBinding(key.WithKeys("j", "down", "ctrl+n"), key.WithHelp("j", "down")),
		Up:     key.NewBinding(key.WithKeys("k", "up", "ctrl+p"), key.WithHelp("k", "up")),
		Select: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	}
}

// Styles defines the possible customizations for styles in the file picker.
type Styles struct {
	DisabledCursor lipgloss.Style
	Cursor         lipgloss.Style
	Option         lipgloss.Style
	Selected       lipgloss.Style
	EmptyDirectory lipgloss.Style
}

// DefaultStyles defines the default styling for the file picker.
func DefaultStyles() Styles {
	return DefaultStylesWithRenderer(lipgloss.DefaultRenderer())
}

// DefaultStylesWithRenderer defines the default styling for the file picker,
// with a given Lip Gloss renderer.
func DefaultStylesWithRenderer(r *lipgloss.Renderer) Styles {
	return Styles{
		DisabledCursor: r.NewStyle().Foreground(lipgloss.Color("247")),
		Cursor:         r.NewStyle().Foreground(lipgloss.Color("212")),
		Option:         r.NewStyle(),
		Selected:       r.NewStyle().Foreground(lipgloss.Color("212")).Bold(true),
		EmptyDirectory: r.NewStyle().Foreground(lipgloss.Color("240")).PaddingLeft(paddingLeft).SetString("Bummer. No Options Provided."),
	}
}

// Model represents a file picker.
type Model struct {
	id int

	Options []string

	KeyMap KeyMap

	selected      int
	selectedStack stack

	min      int
	max      int
	maxStack stack
	minStack stack

	Height     int
	AutoHeight bool

	Cursor string
	Styles Styles
}

type stack struct {
	Push   func(int)
	Pop    func() int
	Length func() int
}

func newStack() stack {
	slice := make([]int, 0)
	return stack{
		Push: func(i int) {
			slice = append(slice, i)
		},
		Pop: func() int {
			res := slice[len(slice)-1]
			slice = slice[:len(slice)-1]
			return res
		},
		Length: func() int {
			return len(slice)
		},
	}
}

func (m Model) pushView() {
	m.minStack.Push(m.min)
	m.maxStack.Push(m.max)
	m.selectedStack.Push(m.selected)
}

func (m Model) popView() (int, int, int) {
	return m.selectedStack.Pop(), m.minStack.Pop(), m.maxStack.Pop()
}

// Init initializes the file picker model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles user interactions within the file picker model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if m.AutoHeight {
			m.Height = msg.Height - marginBottom
		}
		m.max = m.Height - 1
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.KeyMap.Down):
			m.selected++
			if m.selected >= len(m.Options) {
				m.selected = len(m.Options) - 1
			}
			if m.selected > m.max {
				m.min++
				m.max++
			}
		case key.Matches(msg, m.KeyMap.Up):
			m.selected--
			if m.selected < 0 {
				m.selected = 0
			}
			if m.selected < m.min {
				m.min--
				m.max--
			}
		}
	}
	return m, nil
}

// View returns the view of the file picker.
func (m Model) View() string {
	if len(m.Options) == 0 {
		return m.Styles.EmptyDirectory.String()
	}
	var s strings.Builder

	for i, f := range m.Options {
		if i < m.min {
			continue
		}
		if i > m.max {
			break
		}

		name := f

		if m.selected == i {
			selected := fmt.Sprintf(" %s", name)
			s.WriteString(m.Styles.Cursor.Render(m.Cursor) + m.Styles.Selected.Render(selected))
			s.WriteRune('\n')
			continue
		}

		style := m.Styles.Option

		fileName := style.Render(name)
		s.WriteString(fmt.Sprintf("  %s", fileName))
		s.WriteRune('\n')
	}

	return s.String()
}

// DidSelectFile returns whether a user has selected a file (on this msg).
func (m Model) DidSelectOption(msg tea.Msg) (bool, string) {
	didSelect, option := m.didSelectOption(msg)
	if didSelect {
		return true, option
	}
	return false, ""
}

func (m Model) didSelectOption(msg tea.Msg) (bool, string) {
	if len(m.Options) == 0 {
		return false, ""
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If the msg does not match the Select keymap then this could not have been a selection.
		if !key.Matches(msg, m.KeyMap.Select) {
			return false, ""
		}

		// The key press was a selection, let's confirm whether the current file could
		// be selected or used for navigating deeper into the stack.
		f := m.Options[m.selected]

		return true, f

		// If the msg was not a KeyMsg, then the file could not have been selected this iteration.
		// Only a KeyMsg can select a file.
	default:
		return false, ""
	}
	return false, ""
}
