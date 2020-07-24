package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

type recordEntry struct {
	addresses []string
	service   string
}

func main() {
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()
	if *configPath == "" {
		log.Fatalln("-config must be set")
	}

	config, err := NewConfig(*configPath)
	if err != nil {
		log.Fatalln("Error loading config:", err)
	}

	srv := newDNSServer(config.Bind)
	h := newDNSHandler(config.Zone)
	srv.Handler = h

	go func() {
		log.Println("Answer queries for zone", config.Zone)
		log.Println("Starting DNS server on", config.Bind)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	notify := make(chan *recordEntry)
	m, err := NewMonitor(config.Services)
	if err != nil {
		log.Fatalln("Failed to setup monitor:", err)
	}

	log.Printf("Beginning monitoring of Consul services (%s)",
		strings.Join(config.Services, ","))

	m.Run(notify)
	h.Watch(notify)

	p := NewMetricsHandler(config.PromBind)
	p.RegisterPrometheus()
	go func() {
		log.Println("Exporting Prometheus metrics on", config.PromBind)
		if err := p.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalln("Failed to start Prometheus exposition server:", err)
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

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.Shutdown(ctx); err != nil {
				log.Println("Error shutting down DNS handler:", err)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.Shutdown(ctx); err != nil {
				log.Println("Error shutting down monitor:", err)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := p.Shutdown(ctx); err != nil {
				log.Println("Error shutting down Prometheus exposition server:", err)
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
