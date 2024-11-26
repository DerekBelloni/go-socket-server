package main

import (
	"github.com/DerekBelloni/go-socket-server/relay"
	"github.com/DerekBelloni/go-socket-server/user"
)

func main() {
	relayManager := relay.NewRelayManager(nil)
	relayConnection := relay.NewRelayConnection(relayManager)
	relayManager.Connector = relayConnection

	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://relay.primal.net",
		"wss://relay.nostr.band",
	}

	userService := user.NewService(relayConnection, relayUrls)

	go userService.StartMetadataQueue()
	go userService.StartFollowsMetadataQueue()
	go userService.StartCreateNoteQueue()

	var forever chan struct{}
	<-forever
}
