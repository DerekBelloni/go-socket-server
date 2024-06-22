package main

import (
	"sync"
	"time"

	"github.com/DerekBelloni/go-socket-server/internal/relay"
)

func main() {
	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://nos.lol",
	}

	// channel to communicate that notes have been set in redis
	finished := make(chan string)
	// channel to communicate to the wait group that all go routines are done
	done := make(chan bool)
	var wg sync.WaitGroup

	// instantiate a timer
	ticker := time.NewTicker(10 * time.Minute)
	// makes sure the ticker is stopped when the main function exits, important for resource clean up
	defer ticker.Stop()

	for _, relayUrl := range relayUrls {
		wg.Add(1)
		go func(relayUrl string) {
			defer wg.Done()
			for {
				select {
					case <-ticker.C: 
						relay.ConnectToRelay(relayUrl, finished)
						return
					case <-done:
						return
				}
			}
		// Allows calling the anon function with the current iteration in relayUrls
		// Each GO routine has its own copy
		}(relayUrl)
	}

	// Listen for OS signals to gracefully shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		// Block until a signal is received
		<-sig
		// signal to close the 'd
		close(done)
		wg.Wait()
		wg.Done()
	}
}
