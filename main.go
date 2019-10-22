package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/dns"
)

type RecordEntry struct {
	Addresses []string
	Service   string
}

type DNSHandler struct {
	sync.RWMutex

	svcMap map[string]string
}

func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name

		h.RLock()
		address, ok := h.svcMap[domain]
		h.RUnlock()
		if !ok {
			break
		}

		msg.Answer = append(msg.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			A: net.ParseIP(address),
		})
	}
	w.WriteMsg(&msg)
}

func main() {
	configPath := flag.String("config", "", "Config file path")
	flag.Parse()
	if *configPath == "" {
		log.Fatalln("-config must be set")
	}

	config, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalln("Error loading config:", err)
	}

	srv := &dns.Server{Addr: config.Bind, Net: "udp"}
	h := &DNSHandler{
		svcMap: make(map[string]string),
	}
	srv.Handler = h

	go func() {
		log.Println("Starting DNS server on", config.Bind)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to set udp listener %s\n", err.Error())
		}
	}()

	svcs := config.Services
	notify := make(chan *RecordEntry)
	for _, svc := range svcs {
		go monitor(svc, notify)
	}

	log.Printf("Beginning monitoring of Consul services (%s)",
		strings.Join(config.Services, ","))

	go func() {
		for {
			a := <-notify

			t := a.Addresses
			sort.Strings(t)
			h.Lock()
			log.Printf("Updating service map record %s (%s)", a.Service, t[0])
			h.svcMap[a.Service+"."+config.Zone+"."] = t[0]
			h.Unlock()
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
