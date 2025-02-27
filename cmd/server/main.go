package main

import (
	"github.com/DerekBelloni/go-socket-server/relay"
	"github.com/DerekBelloni/go-socket-server/search"
	"github.com/DerekBelloni/go-socket-server/tracking"
	"github.com/DerekBelloni/go-socket-server/user"
)

func main() {
	trackerManager := tracking.NewTrackerManager()
	relayManager := relay.NewRelayManager(nil, nil, trackerManager)
	relayConnection := relay.NewRelayConnection(relayManager)
	relayManager.Connector = relayConnection
	searchTracker := search.NewSearchTrackerImpl()
	relayManager.SearchTracker = searchTracker

	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://relay.primal.net",
		"wss://relay.nostr.band",
	}

	userService := user.NewService(relayConnection, relayUrls, searchTracker)

	go userService.StartMetadataQueue()
	go userService.StartFollowsMetadataQueue()
	go userService.StartCreateNoteQueue()
	go userService.StartSearchQueue()
	go userService.StartFollowsNotesQueue()
	go userService.StartEmbeddedEntityQueue()

	var forever chan struct{}
	<-forever
}
