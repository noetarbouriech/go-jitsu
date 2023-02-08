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

var normalStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("63")).
	Padding(2).
	Align(lipgloss.Center).
	Width(12).
	Height(8).
	Margin(0, 2)

var selectedStyle = lipgloss.NewStyle().
	Bold(true).
	BorderStyle(lipgloss.ThickBorder()).
	BorderForeground(lipgloss.Color("255")).
	Padding(2).
	Align(lipgloss.Center).
	Width(13).
	Height(9).
	Margin(0, 2)

type model struct {
	cursor   int
	deck     []card
	selected map[int]struct{}
}

type card struct {
	value  string
	symbol string
	color  string
}

var (
	symbols = []string{"❄️", "💧", "🔥"}
	colors  = []string{"#863ba4", "#58a7e5", "#eab942", "#93c47d"}
)

func buildCard() card {
	rand.Seed(time.Now().UnixNano())
	val := strconv.Itoa(rand.Intn(12-1+1) + 1)
	sym := symbols[rand.Intn(len(symbols))]
	col := colors[rand.Intn(len(colors))]

	card := card{value: val, symbol: sym, color: col}
	return card
}

func initialModel() model {
	deck := []card{buildCard(), buildCard(), buildCard(), buildCard(), buildCard()}

	return model{
		deck:     deck,
		cursor:   0,
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

	for i, choice := range m.deck {
		normalStyle.BorderForeground(lipgloss.Color(choice.color))
		if i == m.cursor {
			cards = append(cards, selectedStyle.Render(choice.value+choice.symbol))
		} else {
			cards = append(cards, normalStyle.Render(choice.value+choice.symbol))
		}
	}

	s += lipgloss.JoinHorizontal(lipgloss.Center, cards...)

	return lipgloss.Place(220, 100, lipgloss.Center, lipgloss.Bottom, s)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
