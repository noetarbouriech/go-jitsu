# 🐧 go-jitsu

A remake of the Card-Jitsu minigame from the game Club Penguin in Go using ssh.

# How to start

With docker (using port 3000):

```bash
docker build . -t go-jitsu-server

docker run -p 3000:3000 go-jitsu-server
```

Connect to the server with ssh:

```bash
ssh ssh://localhost:3000

# or if you want to change your username:
ssh ssh://<your username>@localhost:3000
```

# Libraries used

> Mainly the libraries developed by [Charm.sh](https://charm.sh/).

- UI lib -> [Bubbletea](https://github.com/charmbracelet/bubbletea)
- UI styling -> [Lipgloss](https://github.com/charmbracelet/lipgloss)
- UI flexbox for Bubbletea -> [Stickers](https://github.com/76creates/stickers)
- SSH server -> [Wish](https://github.com/charmbracelet/wish)
