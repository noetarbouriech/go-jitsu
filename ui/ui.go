package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var style = lipgloss.NewStyle().
	Bold(true).
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")).
	Padding(2).
	Align(lipgloss.Center).
	Width(12).
	Height(8)

type model struct {
	cursor   int
	deck     []card
	selected map[int]struct{}
}

type card struct {
	value  int
	symbol string
	color  int
}

var symbols = []string{"❄️", "💧", "🔥"}
var colors = []int{1, 100, 200, 255}

func buildCard() card {
	rand.Seed(time.Now().UnixNano())
	val := rand.Intn(12-1+1) + 1
	sym := symbols[rand.Intn(len(symbols))]
	col := colors[rand.Intn(len(colors))]

	card := card{value: val, symbol: sym, color: col}
	return card
}

func initialModel() model {

	deck := []card{buildCard(), buildCard(), buildCard(), buildCard(), buildCard()}

	return model{
		deck: deck,

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "left", "h":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "l":
			if m.cursor < len(m.deck)-1 {
				m.cursor++
			}
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	var s string
	var cards []string
	s = "What card to play?\n\n"

	for _, choice := range m.deck {
		choiceString := strconv.Itoa(choice.value) + choice.symbol

		cards = append(cards, style.Render(choiceString))
		// s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	s += lipgloss.JoinHorizontal(lipgloss.Center, cards...)
	s += "\nPress q to quit.\n"

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
