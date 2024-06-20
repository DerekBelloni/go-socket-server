package main

import (
	"fmt"

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

	fmt.Scanln(&input)
}
