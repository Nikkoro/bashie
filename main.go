package main

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type alias struct {
	name, command string
}

func (a alias) Title() string       { return a.name }
func (a alias) Description() string { return a.command }
func (a alias) FilterValue() string { return a.name }

type menuItem struct {
	title, desc string
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type screen int

const (
	menuScreen screen = iota
	aliasScreen
	bashrcScreen
	fileScreen
)

type model struct {
	menuList  list.Model
	aliasList list.Model
	viewport  viewport.Model
	textInput textinput.Model
	screen    screen
	fileName  string
	width     int
	height    int
	ready     bool
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF75B7")).
			MarginLeft(2)

	appStyle = lipgloss.NewStyle().Margin(1, 2)
)

func initialModel() model {
	menuItems := []list.Item{
		menuItem{title: "📁 View Aliases", desc: "View current aliases"},
		menuItem{title: "📄 View File", desc: "View the selected file"},
		menuItem{title: "⚙️ Select File", desc: "Specify the file to view"},
	}

	menuList := list.New(menuItems, list.NewDefaultDelegate(), 0, 0)
	menuList.Title = ".bashie"
	menuList.SetShowStatusBar(false)
	menuList.Styles.Title = titleStyle

	aliasList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	aliasList.Title = "Aliases"
	aliasList.Styles.Title = titleStyle

	ti := textinput.New()
	ti.Width = 40

	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle().Padding(1, 2)

	return model{
		menuList:  menuList,
		aliasList: aliasList,
		viewport:  vp,
		textInput: ti,
		fileName:  ".zshrc",
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if !m.ready {
			m.ready = true
			m.width = msg.Width
			m.height = msg.Height
			m.menuList.SetSize(msg.Width-4, msg.Height-4)
			m.aliasList.SetSize(msg.Width-4, msg.Height-4)
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 4

			aliases := loadAliases(m.fileName)
			m.aliasList.SetItems(aliases)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.screen != menuScreen {
				m.screen = menuScreen
				return m, nil
			}

		case "enter":
			if m.screen == menuScreen {
				switch m.menuList.Index() {
				case 0:
					m.screen = aliasScreen
				case 1:
					m.screen = bashrcScreen
					m.viewport.SetContent(readFile(m.fileName))
				case 2:
					m.screen = fileScreen
					m.textInput.SetValue(m.fileName)
					m.textInput.Focus()
					return m, textinput.Blink
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	switch m.screen {
	case menuScreen:
		m.menuList, cmd = m.menuList.Update(msg)
		cmds = append(cmds, cmd)
	case aliasScreen:
		m.aliasList, cmd = m.aliasList.Update(msg)
		cmds = append(cmds, cmd)
	case bashrcScreen:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	case fileScreen:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)

		if msg, ok := msg.(tea.KeyMsg); ok && msg.String() == "enter" {
			m.fileName = m.textInput.Value()
			m.screen = menuScreen
			aliases := loadAliases(m.fileName)
			m.aliasList.SetItems(aliases)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	switch m.screen {
	case menuScreen:
		return appStyle.Render(m.menuList.View())
	case aliasScreen:
		return appStyle.Render(m.aliasList.View())
	case bashrcScreen:
		return appStyle.Render(
			titleStyle.Render(m.fileName) + "\n\n" +
				m.viewport.View() + "\n\n" +
				"press q to quit, esc to go back",
		)
	case fileScreen:
		return appStyle.Render(fmt.Sprintf("\n%s\n\n%s\n\n%s",
			titleStyle.Render("Specify File"),
			m.textInput.View(),
			"enter: save  esc: cancel"))
	default:
		return "Loading..."
	}
}

func loadAliases(fileName string) []list.Item {
	content := readFile(fileName)
	var items []list.Item
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "alias") {
			parts := strings.SplitN(line[6:], "=", 2)
			if len(parts) == 2 {
				items = append(items, alias{
					name:    parts[0],
					command: strings.Trim(parts[1], "'\""),
				})
			}
		}
	}
	return items
}

func readFile(fileName string) string {
	usr, _ := user.Current()
	path := usr.HomeDir + "/" + fileName
	if runtime.GOOS == "windows" {
		path = usr.HomeDir + "\\" + fileName
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "Error reading file"
	}
	return string(content)
}

func main() {
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
