package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/DerekBelloni/go-socket-server/internal/relay"
)

func main() {
	relayUrls := []string{
		"wss://relay.damus.io",
		"wss://nos.lol",
		"wss://purplerelay.com",
		"wss://relay.primal.net",
	}

	finished := make(chan string)
	done := make(chan bool)
	var wg sync.WaitGroup

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for _, relayUrl := range relayUrls {
		wg.Add(1)
		go func(relayUrl string) {
			defer wg.Done()
			relay.ConnectToRelay(relayUrl, finished)
			for {
				select {
				case <-ticker.C:
					fmt.Println("timer worked!")
					relay.ConnectToRelay(relayUrl, finished)
					return
				case <-done:
					return
				}
			}
		}(relayUrl)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sig
		close(done)
		wg.Wait()
		// wg.Done()
		os.Exit(0)
	}()

	for url := range finished {
		fmt.Printf("Finished processing for relay: %s\n", url)
		wg.Done()
	}

	wg.Wait()
}
