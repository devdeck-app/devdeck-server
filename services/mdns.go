package services

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/grandcat/zeroconf"
)

func RegisterMDNS() {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Fatalf("Failed to get loopback interface: %v", err)
	}
	server, err := zeroconf.Register(
		"DevDeck",
		"_devdeck._tcp",
		"local.",
		8080,
		[]string{"txtv=0", "lo=1", "la=2"},
		ifaces,
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
