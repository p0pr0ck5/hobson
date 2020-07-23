package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
)

type dnsHandler struct {
	mu sync.RWMutex

	zone string

	svcMap map[string]net.IP

	shutdownCh chan struct{}
}

func newDNSServer(bind string) *dns.Server {
	return &dns.Server{Addr: bind, Net: "udp"}
}

func newDNSHandler(zone string) *dnsHandler {
	return &dnsHandler{
		zone:       zone,
		svcMap:     make(map[string]net.IP),
		shutdownCh: make(chan struct{}),
	}
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	timer := prometheus.NewTimer(queryHandleDuration)
	defer timer.ObserveDuration()

	msg := dns.Msg{}
	msg.SetReply(r)
	switch r.Question[0].Qtype {
	case dns.TypeA:
		msg.Authoritative = true
		domain := msg.Question[0].Name

		h.mu.RLock()
		address, ok := h.svcMap[domain]
		h.mu.RUnlock()
		if !ok {
			queryUnknownName.Inc()
			msg.SetRcode(r, dns.RcodeNameError)
			break
		}

		msg.Answer = append(msg.Answer, &dns.A{
			Hdr: dns.RR_Header{
				Name:   domain,
				Rrtype: dns.TypeA,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			A: address,
		})
		recordServed.WithLabelValues(strings.Split(domain, ".")[0]).Inc() // TODO clean this up
	}
	w.WriteMsg(&msg)
}

func (h *dnsHandler) Watch(notify <-chan *recordEntry) {
	go func() {
		for {
			select {
			case <-h.shutdownCh:
				return
			case a := <-notify:
				t := a.addresses

				if len(t) == 0 {
					log.Printf("No records for service %q", a.service)
					continue
				}

				h.UpdateRecord(a.service, t)
			}
		}
	}()
}

func (h *dnsHandler) Shutdown(ctx context.Context) error {
	close(h.shutdownCh)
	return nil
}

func (h *dnsHandler) UpdateRecord(service string, records []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	rec := fmt.Sprintf("%s.%s.", service, h.zone)
	cur := h.svcMap[rec]

	for _, record := range records {
		if bytes.Compare(net.ParseIP(record), cur) == 0 {
			return
		}
	}

	sort.Strings(records)
	newRecord := records[0]
	log.Printf("Updating service map record %s (%s)", service, newRecord)
	h.svcMap[rec] = net.ParseIP(newRecord)
	recordUpdateTime.WithLabelValues(service).SetToCurrentTime()
}
