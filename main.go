package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/dns"
)

type recordEntry struct {
	Addresses []string
	Service   string
}

func main() {
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()
	if *configPath == "" {
		log.Fatalln("-config must be set")
	}

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalln("Error loading config:", err)
	}

	srv := &dns.Server{Addr: config.Bind, Net: "udp"}
	h := newDNSHandler(config.Zone)
	srv.Handler = h

	go func() {
		log.Println("Answer queries for zone", config.Zone)
		log.Println("Starting DNS server on", config.Bind)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	svcs := config.Services
	notify := make(chan *recordEntry)
	for _, svc := range svcs {
		go monitor(svc, notify)
	}

	log.Printf("Beginning monitoring of Consul services (%s)",
		strings.Join(config.Services, ","))

	go func() {
		for {
			a := <-notify
			t := a.Addresses

			if len(t) == 0 {
				log.Printf("No records for service %q", a.Service)
				continue
			}

			sort.Strings(t)
			h.UpdateRecord(a.Service, t)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	waitCh := make(chan struct{})
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := srv.ShutdownContext(ctx); err != nil {
				log.Println("Error shutting down DNS server:", err)
			}
		}()

		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-ctx.Done():
		log.Fatalln("Timeout while shutting down server")
	case <-waitCh:
	}
}
