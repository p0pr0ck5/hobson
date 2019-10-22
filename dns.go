package main

import (
	"net"
	"sync"

	"github.com/miekg/dns"
)

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
