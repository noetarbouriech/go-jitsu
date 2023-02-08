package ui

import (
	"math/rand"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
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
	termWidth  int
	termHeight int
	cursor     int
	deck       []card
	selected   card
	cardPlayed card
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

func initialModel(pty ssh.Pty) model {
	deck := []card{buildCard(), buildCard(), buildCard(), buildCard(), buildCard()}

	return model{
		termWidth:  pty.Window.Width,
		termHeight: pty.Window.Height,
		cursor:     0,
		deck:       deck,
		selected:   card{value: "", symbol: "", color: ""},
		cardPlayed: card{
			value:  "",
			symbol: "",
			color:  "",
		},
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
			// show card on the middle of the screen
			m.cardPlayed = m.deck[m.cursor]
			// remove card from deck
			m.deck = append(m.deck[:m.cursor], m.deck[m.cursor+1:]...)
			// put cursor back to 0
			m.cursor = 0
			// TODO: add new card to deck
		}
	}

	return m, nil
}

func (m model) View() string {
	var s string
	var cards []string
	if m.cardPlayed.value != "" {
		selectedStyle.BorderForeground(lipgloss.Color(m.cardPlayed.color))
		s += "Card played:\n\n"
		var playedCards = []string{}
		playedCards = append(playedCards, selectedStyle.Render(m.cardPlayed.value+m.cardPlayed.symbol))
		playedCards = append(playedCards, selectedStyle.Render(m.cardPlayed.value+m.cardPlayed.symbol))

		s += lipgloss.JoinHorizontal(lipgloss.Center, playedCards...)

	}
	s += "\nWhat card to play?\nq"

	for i, choice := range m.deck {
		selectedStyle.BorderForeground(lipgloss.Color(255))
		normalStyle.BorderForeground(lipgloss.Color(choice.color))

		if i == m.cursor {
			cards = append(cards, selectedStyle.Render(choice.value+choice.symbol))
		} else {
			cards = append(cards, normalStyle.Render(choice.value+choice.symbol))
		}
	}

	s += lipgloss.JoinHorizontal(lipgloss.Center, cards...)
	return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, s)
}

func TeaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	pty, _, _ := s.Pty()
	return initialModel(pty), []tea.ProgramOption{tea.WithAltScreen()}
}
