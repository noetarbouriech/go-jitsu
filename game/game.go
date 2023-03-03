package game

import (
	"errors"
	"fmt"
	"log"
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

type player struct {
	cardPlayed chan card
}

type Room struct {
	players map[ssh.Session]player
}

var room Room

func GameMiddleware() wish.Middleware {
	if len(room.players) == 0 {
		room = Room{
			players: make(map[ssh.Session]player),
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
		if len(room.players) < 2 {
			room.players[s] = player{make(chan card)}
			fmt.Println("new player connected")
		} else {
			wish.Println(s, lipgloss.NewStyle().BorderStyle(lipgloss.DoubleBorder()).Padding(1).Align(lipgloss.Center).Render("\nToo many players online 💩\n"))
			s.Close()
			return nil
		}
		m, _ := TeaHandler(s, room)
		return newProg(m, tea.WithInput(s), tea.WithOutput(s), tea.WithAltScreen())
	}
	return bm.MiddlewareWithProgramHandler(teaHandler, termenv.ANSI256)
}

type model struct {
	termWidth         int
	termHeight        int
	session           ssh.Session
	room              Room
	cursor            int
	deck              []card
	cardPlayedByMe    card
	cardPlayedByOther card
	cardsWonByMe      map[string][]string
	cardsWonByOther   map[string][]string
}

type card struct {
	value  string
	symbol string
	color  string
}

var (
	symbols = []string{"🧊", "💧", "🔥"}
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
		termWidth:         pty.Window.Width,
		termHeight:        pty.Window.Height,
		session:           s,
		room:              r,
		cursor:            0,
		deck:              deck,
		cardPlayedByMe:    card{value: "", symbol: "", color: ""},
		cardPlayedByOther: card{value: "", symbol: "", color: ""},
		cardsWonByMe:      map[string][]string{},
		cardsWonByOther:   map[string][]string{},
	}
}

func (m model) Init() tea.Cmd {
	m.cardsWonByMe = make(map[string][]string)
	m.cardsWonByOther = make(map[string][]string)
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			// reset cards played
			for sessionPlayer := range room.players {
				wish.Println(sessionPlayer, "Goodbye 🚪👋")
				sessionPlayer.Close()
			}
			room = Room{
				players: map[ssh.Session]player{},
			}

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
			if len(m.room.players) < 2 {
				return m, nil
			}

			// show card on the middle of the screen
			m.cardPlayedByMe = m.deck[m.cursor]

			// print card played by player
			fmt.Println(m.cardPlayedByMe)

			// add card to channel
			go m.playCard(m.cardPlayedByMe)

			// get other player card
			otherPlayer, err := m.getOtherPlayer()
			if err != nil {
				log.Fatal(err)
			}
			m.cardPlayedByOther = <-otherPlayer.cardPlayed

			// card duel
			winner := cardDuel(m.cardPlayedByMe, m.cardPlayedByOther)
			if winner.value == "" {
				fmt.Println("draw")
			}
			if m.cardPlayedByMe == winner {
				fmt.Println(m.session.User() + " has won")
				m.cardsWonByMe[m.cardPlayedByMe.symbol] = append(m.cardsWonByMe[m.cardPlayedByMe.symbol], m.cardPlayedByMe.color)
			} else {
				fmt.Println("other player has won")
				m.cardsWonByOther[m.cardPlayedByOther.symbol] = append(m.cardsWonByOther[m.cardPlayedByOther.symbol], m.cardPlayedByOther.color)
			}
			fmt.Println(m.session.User(), m.cardsWonByMe)

			// remove card from deck
			m.deck = append(m.deck[:m.cursor], m.deck[m.cursor+1:]...)

			// put cursor back to 0
			m.cursor = 0

			// add new card to deck
			m.deck = append(m.deck, buildCard())
		}
	}

	return m, nil
}

func (m model) playCard(cardPlayed card) {
	m.room.players[m.session].cardPlayed <- cardPlayed
}

func cardDuel(c1 card, c2 card) card {
	if c1.symbol == c2.symbol {
		if c1.value == c2.value {
			return card{}
		} else if c1.value > c2.value {
			return c1
		}
		return c2
	}

	if c1.symbol == "🧊" {
		if c2.symbol == "💧" {
			return c1
		}
		return c2
	}
	if c1.symbol == "💧" {
		if c2.symbol == "🧊" {
			return c2
		}
		return c1
	}
	if c1.symbol == "🔥" {
		if c2.symbol == "💧" {
			return c2
		}
		return c1
	}

	return card{}
}

func (m model) getOtherPlayer() (player, error) {
	for _, p := range m.room.players {
		if m.room.players[m.session] != p {
			return p, nil
		}
	}
	return player{}, errors.New("no player found... 😿")
}

func (m model) View() string {
	var s string
	var cards []string

	if len(m.room.players) == 1 {
		return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, "⏳ Waiting for another player")
	}
	for _, c := range m.cardsWonByMe["💧"] {
		s += lipgloss.NewStyle().Background(lipgloss.Color(c)).Render("💧\n")
	}
	for _, c := range m.cardsWonByMe["🧊"] {
		s += lipgloss.NewStyle().Background(lipgloss.Color(c)).Render("🧊\n")
	}
	for _, c := range m.cardsWonByMe["🔥"] {
		s += lipgloss.NewStyle().Background(lipgloss.Color(c)).Render("🔥\n")
	}
	normalStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(2).
		Align(lipgloss.Center).
		Width(12).
		Height(8).
		Margin(0, 2)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(lipgloss.Color("255")).
		Padding(2).
		Align(lipgloss.Center).
		Width(13).
		Height(9).
		Margin(0, 2)
	if m.cardPlayedByMe.value != "" {
		selectedStyle.BorderForeground(lipgloss.Color(m.cardPlayedByMe.color))
		s += "Card played:\n\n"
		// display current card and other player card
		playedCards := []string{
			normalStyle.BorderForeground(lipgloss.Color(m.cardPlayedByMe.color)).
				Render(m.cardPlayedByMe.value + m.cardPlayedByMe.symbol),
			normalStyle.BorderForeground(lipgloss.Color(m.cardPlayedByOther.color)).
				Render(m.cardPlayedByOther.value + m.cardPlayedByOther.symbol),
		}
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
