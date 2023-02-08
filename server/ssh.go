package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	bm "github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/noetarbouriech/go-jitsu/ui"
)

func InitServer(host string, port int) {
	server, err := wish.NewServer(
		wish.WithAddress(fmt.Sprintf("%s:%d", host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			roomMiddleware(),
			logging.Middleware(),
			bm.Middleware(ui.TeaHandler),
		),
		wish.WithIdleTimeout(1*time.Hour),
	)
	if err != nil {
		log.Fatalf("Couldn't create the server: %s", err)
	}

	startServer(server, err, host, port)
}

func startServer(server *ssh.Server, err error, host string, port int) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s:%d", host, port)
	go func() {
		if err = server.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}

// All users connected
var users []ssh.Session

func roomMiddleware() wish.Middleware {
	return func(handler ssh.Handler) ssh.Handler {
		return func(session ssh.Session) {
			if len(users)+1 > 2 {
				if session.Close() != nil {
					return
				}
				return
			}

			wait := make(chan ssh.Signal, 1)
			go handlePlayer(session)
			<-wait
			handler(session)
		}
	}
}

func handlePlayer(session ssh.Session) {
	users = append(users, session)

	log.Printf("Room has %d users", len(users))
	for _, s := range users {
		wish.Println(s, fmt.Sprintf("User %s is connected ;\n\tid -> %s\n", s.User(), s.Context().SessionID()))
	}
}
