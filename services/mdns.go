package services

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/grandcat/zeroconf"
)

func RegisterMDNS() {
	server, err := zeroconf.Register(
		"DevDeck",
		"_devdeck._tcp",
		"local.",
		4242,
		[]string{"txtv=0", "lo=1", "la=2", "usb=true"},
		nil,
	)

	if err != nil {
		log.Printf("Failed to register mDNS service: %v", err)
		return
	}
	defer server.Shutdown()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case <-sig:
		log.Println("Received shutdown signal for mDNS")
	}

	server.Shutdown()
}
