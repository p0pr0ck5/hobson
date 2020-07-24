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

// RecordEntry associated a set of DNS records with a given Consul service
type RecordEntry struct {
	addresses []string
	service   string
}

// DNSHandler stores DNS record information for monitored Consul services, and implement
// dns.ServeDNS()
type DNSHandler struct {
	mu sync.RWMutex

	zone string

	svcMap map[string]net.IP

	shutdownCh chan struct{}
}

// NewDNSServer creates a new dns.Server on a given address
func NewDNSServer(bind string) *dns.Server {
	return &dns.Server{Addr: bind, Net: "udp"}
}

// NewDNSHandler creates a new DNSHandler object for a given zone
func NewDNSHandler(zone string) *DNSHandler {
	return &DNSHandler{
		zone:       zone,
		svcMap:     make(map[string]net.IP),
		shutdownCh: make(chan struct{}),
	}
}

// ServeDNS implements dns.ServeDNS, which responds to DNS queries
// on a given dns.Server
func (h *DNSHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
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

// Watch spawns a goroutine to listen for messages on a channel that indicate
// an update to a service's record set has occured
func (h *DNSHandler) Watch(notify <-chan *RecordEntry) {
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

// Shutdown ends DNSHandler activity
func (h *DNSHandler) Shutdown(ctx context.Context) error {
	close(h.shutdownCh)
	return nil
}

// UpdateRecord updates the record value that hobson will serve for a
// given service. In order to avoid unnecessary flapping during service
// health/registration churn, UpdateRecord will only update the record
// value when the candidate record in a given set of records is not the
// current record value.
func (h *DNSHandler) UpdateRecord(service string, records []string) {
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
