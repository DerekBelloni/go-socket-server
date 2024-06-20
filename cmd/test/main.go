package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/DerekBelloni/go-socket-server/internal/relay"
)

func main() {
	var input string
	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://nos.lol",
	}

	for _, relayUrl := range relayUrls {
		go relay.ConnectToRelay(relayUrl)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Scanln(&input)
}
