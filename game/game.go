package game

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
)

// All users connected
var (
	users []ssh.Session
)

type Room struct {
	playerCount int
	player1     ssh.Session
	player2     ssh.Session
}

var room Room

func GameMiddleware() wish.Middleware {
	if room == (Room{}) {
		room = Room{
			playerCount: 0,
			player1:     nil,
			player2:     nil,
		}
		fmt.Println("New room created")
	}
	newProg := func(m tea.Model, opts ...tea.ProgramOption) *tea.Program {
		p := tea.NewProgram(m, opts...)
		go func() {
			for {
				<-time.After(1 * time.Second)
				p.Send(time.Time(time.Now()))
			}
		}()
		return p
	}
	teaHandler := func(s ssh.Session) *tea.Program {
		_, _, active := s.Pty()
		if !active {
			wish.Fatalln(s, "no active terminal, skipping")
			return nil
		}
		if room.playerCount == 0 {
			room.player1 = s
			fmt.Println("new player 1 connected")
		} else if room.playerCount == 1 {
			room.player2 = s
			fmt.Println("new player 2 connected")
		} else {
			wish.Println(s, lipgloss.NewStyle().BorderStyle(lipgloss.DoubleBorder()).Padding(1).Align(lipgloss.Center).Render("\nToo many players online 💩\n"))
			s.Close()
			return nil
		}
		room.playerCount++
		m, _ := TeaHandler(s, room)
		return newProg(m, tea.WithInput(s), tea.WithOutput(s), tea.WithAltScreen())
	}
	return bm.MiddlewareWithProgramHandler(teaHandler, termenv.ANSI256)
}

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
	session    ssh.Session
	room       Room
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

func initialModel(s ssh.Session, r Room) model {
	deck := []card{buildCard(), buildCard(), buildCard(), buildCard(), buildCard()}
	pty, _, _ := s.Pty()

	return model{
		termWidth:  pty.Window.Width,
		termHeight: pty.Window.Height,
		session:    s,
		room:       r,
		cursor:     0,
		deck:       deck,
		selected:   card{value: "", symbol: "", color: ""},
		cardPlayed: card{value: "", symbol: "", color: ""},
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
			if room.player2 == nil {
				room.player1 = nil
				fmt.Println("player 1 disconnected")
			} else {
				room.player2 = nil
				fmt.Println("player 2 disconnected")
			}
			m.session.Close()
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
		playedCards := []string{}
		playedCards = append(playedCards, selectedStyle.Render(m.cardPlayed.value+m.cardPlayed.symbol))
		playedCards = append(playedCards, selectedStyle.Render(m.cardPlayed.value+m.cardPlayed.symbol))

		s += lipgloss.JoinHorizontal(lipgloss.Center, playedCards...)

	}
	s += "\nWhat card to play?\n"

	for i, choice := range m.deck {
		selectedStyle.BorderForeground(lipgloss.Color(rune(255)))
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

func TeaHandler(s ssh.Session, r Room) (tea.Model, []tea.ProgramOption) {
	return initialModel(s, r), []tea.ProgramOption{tea.WithAltScreen()}
}
