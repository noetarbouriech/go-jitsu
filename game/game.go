package game

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/76creates/stickers"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/muesli/termenv"
)

const rules = `
# Rules of Card-Jitsu
Each turn, the player picked a card from a deck of 5 randomly drawn cards. Each card has its own point value and an element.
- 🔥 beats 🧊.
- 🧊 beats 💧.
- 💧 beats 🔥.
- If same element, the highest number win.
- If same element and same number, it is considered a tie.
- You have to get different color of each element OR three different colors of the same element to win the game.

> To leave this page, press **q**
`

type player struct {
	username   string
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
			room.players[s] = player{
				s.User(),
				make(chan card),
			}
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
	flexBox           *stickers.FlexBox
	room              Room
	cursor            int
	deck              []card
	cardPlayedByMe    card
	cardPlayedByOther card
	cardsWonByMe      map[string][]string
	cardsWonByOther   map[string][]string
	winner            string
	helpMenuOpen      bool
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

	m := model{
		termWidth:         pty.Window.Width,
		termHeight:        pty.Window.Height,
		session:           s,
		flexBox:           stickers.NewFlexBox(0, 0),
		room:              r,
		cursor:            0,
		deck:              deck,
		cardPlayedByMe:    card{value: "", symbol: "", color: ""},
		cardPlayedByOther: card{value: "", symbol: "", color: ""},
		cardsWonByMe:      map[string][]string{},
		cardsWonByOther:   map[string][]string{},
		winner:            "",
		helpMenuOpen:      false,
	}

	return m
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
		case "?":
			m.helpMenuOpen = true
		case "ctrl+c", "q":
			if m.helpMenuOpen {
				m.helpMenuOpen = false
				return m, nil
			}
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
			winner := cardDuel(m.cardPlayedByMe, m.cardPlayedByOther, m)
			if winner.value == "" {
				fmt.Println("draw")
			} else if m.cardPlayedByMe == winner {
				fmt.Println(m.session.User() + " has won")
				m.cardsWonByMe[m.cardPlayedByMe.symbol] = append(m.cardsWonByMe[m.cardPlayedByMe.symbol], m.cardPlayedByMe.color)
			} else {
				fmt.Println("other player has won")
				m.cardsWonByOther[m.cardPlayedByOther.symbol] = append(m.cardsWonByOther[m.cardPlayedByOther.symbol], m.cardPlayedByOther.color)
			}
			fmt.Println(m.session.User(), m.cardsWonByMe)

			// Check if one of the player won
			if isWinnedFromHist(m.cardsWonByMe) {
				m.winner = "👑 you"
			}
			if isWinnedFromHist(m.cardsWonByOther) {
				opponent, _ := m.getOtherPlayer()
				m.winner = "😿 " + opponent.username
			}
			fmt.Println(isWinnedFromHist(m.cardsWonByMe))

			// remove card from deck
			m.deck = append(m.deck[:m.cursor], m.deck[m.cursor+1:]...)

			// put cursor back to 0
			m.cursor = 0

			// add new card to deck
			m.deck = append(m.deck, buildCard())
		}
	}

	m.flexBox.SetWidth(m.termWidth)
	m.flexBox.SetHeight(m.termHeight)

	return m, nil
}

func (m model) constructFlexboxUi() {
	histCardStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#000"))
	cardStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(2).
		Align(lipgloss.Center).
		Width(12).
		Height(8)
	selectedStyle := lipgloss.NewStyle().
		BorderForeground(lipgloss.Color("#fff")).
		BorderForeground(lipgloss.Color("63")).
		Padding(2).
		Align(lipgloss.Center).
		Width(12).
		Height(8).
		BorderStyle(lipgloss.DoubleBorder())

	usernamePlayer := m.session.User()
	usernameOther := "Opponent"
	opponent, error := m.getOtherPlayer()
	if error == nil {
		usernameOther = opponent.username
	}

	// Construct hist cards
	myCardHistory := map[string][]string{
		"💧": {"💧"},
		"🧊": {"🧊"},
		"🔥": {"🔥"},
	}

	otherCardHistory := map[string][]string{
		"💧": {"💧"},
		"🧊": {"🧊"},
		"🔥": {"🔥"},
	}

	for _, c := range m.cardsWonByMe["💧"] {
		myCardHistory["💧"] = append(myCardHistory["💧"], histCardStyle.Background(lipgloss.Color(c)).Render("  "))
	}
	for _, c := range m.cardsWonByMe["🧊"] {
		myCardHistory["🧊"] = append(myCardHistory["🧊"], histCardStyle.Background(lipgloss.Color(c)).Render("  "))
	}
	for _, c := range m.cardsWonByMe["🔥"] {
		myCardHistory["🔥"] = append(myCardHistory["🔥"], histCardStyle.Background(lipgloss.Color(c)).Render("  "))
	}
	for _, c := range m.cardsWonByOther["💧"] {
		otherCardHistory["💧"] = append(otherCardHistory["💧"], histCardStyle.Background(lipgloss.Color(c)).Render("  "))
	}
	for _, c := range m.cardsWonByOther["🧊"] {
		otherCardHistory["🧊"] = append(otherCardHistory["🧊"], histCardStyle.Background(lipgloss.Color(c)).Render("  "))
	}
	for _, c := range m.cardsWonByOther["🔥"] {
		otherCardHistory["🔥"] = append(otherCardHistory["🔥"], histCardStyle.Background(lipgloss.Color(c)).Render("  "))
	}

	rows := []*stickers.FlexBoxRow{
		m.flexBox.NewRow().AddCells(
			[]*stickers.FlexBoxCell{
				stickers.NewFlexBoxCell(1, 3).
					SetStyle(lipgloss.NewStyle().MarginRight(5).Align(lipgloss.Center, lipgloss.Top)).
					SetContent(fmt.Sprintf("Hist. '%s' (me)\n", usernamePlayer) + lipgloss.JoinVertical(0,
						getColumnCardHistory(myCardHistory["💧"]),
						getColumnCardHistory(myCardHistory["🧊"]),
						getColumnCardHistory(myCardHistory["🔥"]),
						"\n",
					)),
				stickers.NewFlexBoxCell(1, 3).
					SetStyle(lipgloss.NewStyle().MarginLeft(5).Align(lipgloss.Center, lipgloss.Top)).
					SetContent(fmt.Sprintf("Hist. '%s'\n", usernameOther) + lipgloss.JoinVertical(0,
						getColumnCardHistory(otherCardHistory["💧"]),
						getColumnCardHistory(otherCardHistory["🧊"]),
						getColumnCardHistory(otherCardHistory["🔥"]),
						"\n",
					)),
			},
		),
	}

	// Arena
	myPlayedCard := ""
	otherPlayedCard := ""
	if m.cardPlayedByMe.value != "" {
		myPlayedCard = cardStyle.
			BorderForeground(lipgloss.Color(m.cardPlayedByMe.color)).
			Render(m.cardPlayedByMe.value + m.cardPlayedByMe.symbol)
		otherPlayedCard = cardStyle.
			BorderForeground(lipgloss.Color(m.cardPlayedByOther.color)).
			Render(m.cardPlayedByOther.value + m.cardPlayedByOther.symbol)

		rows = append(rows,
			m.flexBox.NewRow().AddCells(
				[]*stickers.FlexBoxCell{
					stickers.NewFlexBoxCell(1, 5).
						SetStyle(lipgloss.NewStyle().AlignHorizontal(lipgloss.Right).AlignVertical(lipgloss.Center)).
						SetContent(lipgloss.JoinVertical(lipgloss.Center, fmt.Sprintf("%s (me)", usernamePlayer), myPlayedCard)),
					stickers.NewFlexBoxCell(1, 5).
						SetStyle(lipgloss.NewStyle().AlignHorizontal(lipgloss.Left).AlignVertical(lipgloss.Center)).
						SetContent(lipgloss.JoinVertical(lipgloss.Center, usernameOther, otherPlayedCard)),
				},
			),
		)
	} else {
		rows = append(rows,
			m.flexBox.NewRow().AddCells(
				[]*stickers.FlexBoxCell{stickers.NewFlexBoxCell(1, 4), stickers.NewFlexBoxCell(1, 4)},
			),
		)
	}

	// Construct deck cards
	var cards []string
	for i, choice := range m.deck {
		if i == m.cursor {
			cards = append(
				cards,
				selectedStyle.BorderForeground(lipgloss.Color(choice.color)).Render(choice.value+choice.symbol),
			)
		} else {
			cards = append(cards, cardStyle.BorderForeground(lipgloss.Color(choice.color)).Render(choice.value+choice.symbol))
		}
	}
	rows = append(rows,
		m.flexBox.NewRow().AddCells(
			[]*stickers.FlexBoxCell{
				stickers.NewFlexBoxCell(1, 5).
					SetStyle(lipgloss.NewStyle().AlignHorizontal(lipgloss.Center).AlignVertical(lipgloss.Center)).
					SetMinWidth(cardStyle.GetWidth() * 5).
					SetContent(lipgloss.JoinVertical(lipgloss.Left,
						"Deck",
						lipgloss.JoinHorizontal(lipgloss.Left, cards...)),
					),
			},
		),
	)

	m.flexBox.SetRows(rows)
}

func getColumnCardHistory(historyCards []string) string {
	return lipgloss.JoinHorizontal(0, historyCards...)
}

func (m model) playCard(cardPlayed card) {
	m.room.players[m.session].cardPlayed <- cardPlayed
}

func cardDuel(c1 card, c2 card, m model) card {
	if c1.symbol == c2.symbol {
		c1Val, _ := strconv.Atoi(c1.value)
		c2Val, _ := strconv.Atoi(c2.value)

		if c1Val == c2Val {
			return card{}
		} else if c1Val > c2Val {
			fmt.Println("c1 is winner", c1)
			return c1
		} else {
			fmt.Println("c2 wins")
			return c2
		}

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

func isWinnedFromHist(historyCards map[string][]string) bool {
	// we create a new array without color doublons and check if len == 3 meaning we won
	for _, value := range historyCards {
		clearedLine := removeDoublons(value)
		if len(clearedLine) == 0 {
			// it's impossible to win if one of the line is empty
			return false
		}
		if len(clearedLine) == 3 {
			return true
		}
	}

	// for each round we store a list of element we have already seen and compare the next line elements to it to check if it's a new unique one
	for _, waterItem := range historyCards["💧"] {
		for _, fireItem := range historyCards["🔥"] {
			for _, iceItem := range historyCards["🧊"] {
				if waterItem != fireItem && fireItem != iceItem && iceItem != waterItem {
					return true
				}
			}
		}
	}
	return false
}

func removeDoublons(array []string) []string {
	uniqueMap := make(map[string]bool)
	for _, elem := range array {
		uniqueMap[elem] = true
	}

	result := make([]string, 0, len(uniqueMap))
	for key := range uniqueMap {
		result = append(result, key)
	}
	return result
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
	if len(m.room.players) == 1 {
		return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, "⏳ Waiting for another player")
	}

	if !m.helpMenuOpen {
		m.constructFlexboxUi()

		if m.winner == "" {
			m.constructFlexboxUi()
		} else {
			return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, m.winner+" won")
		}

		return m.flexBox.Render()
	} else {
		out, _ := glamour.Render(rules, "dark")
		return out
	}
}

func TeaHandler(s ssh.Session, r Room) (tea.Model, []tea.ProgramOption) {
	return initialModel(s, r), []tea.ProgramOption{tea.WithAltScreen()}
}
