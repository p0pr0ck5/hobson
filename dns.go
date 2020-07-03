package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"

	"github.com/miekg/dns"
)

type dnsHandler struct {
	sync.RWMutex

	zone string

	svcMap map[string]string

	shutdownCh chan struct{}
}

func newDNSServer(bind string) *dns.Server {
	return &dns.Server{Addr: bind, Net: "udp"}
}

func newDNSHandler(zone string) *dnsHandler {
	return &dnsHandler{
		zone:       zone,
		svcMap:     make(map[string]string),
		shutdownCh: make(chan struct{}),
	}
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
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
			A: net.ParseIP(address),
		})
	}
	w.WriteMsg(&msg)
}

func (h *dnsHandler) Watch(notify <-chan *recordEntry) {
	go func() {
		for {
			select {
			case <-h.shutdownCh:
				return
			default:
				a := <-notify
				t := a.Addresses

				if len(t) == 0 {
					log.Printf("No records for service %q", a.Service)
					continue
				}

				h.UpdateRecord(a.Service, t)
			}
		}
	}()
}

func (h *dnsHandler) Shutdown(ctx context.Context) error {
	close(h.shutdownCh)
	return nil
}

func (h *dnsHandler) UpdateRecord(service string, records []string) {
	h.Lock()
	defer h.Unlock()

	rec := fmt.Sprintf("%s.%s.", service, h.zone)
	cur := h.svcMap[rec]

	for _, record := range records {
		if cur == record {
			return
		}
	}

	sort.Strings(records)
	newRecord := records[0]
	log.Printf("Updating service map record %s (%s)", service, newRecord)
	h.svcMap[rec] = newRecord
}
