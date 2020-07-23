package main

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/hashicorp/consul/api"
)

const backoffMax = 30000
const backoffBase = 500

type monitor struct {
	services []string

	client *api.Client

	shutdownCh chan struct{}

	backoff func(*uint64)
	reset   func(*uint64)
}

func newMonitor(services []string) (*monitor, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	m := &monitor{
		services:   services,
		client:     client,
		shutdownCh: make(chan struct{}),
	}

	backoff, reset := backoffFuncs()
	m.backoff = backoff
	m.reset = reset

	return m, nil
}

func backoffFuncs() (func(*uint64), func(*uint64)) {
	backoff := func(n *uint64) {
		*n++
		sleep := math.Min(math.Pow(2, float64(*n))*backoffBase, backoffMax)
		time.Sleep(time.Millisecond * time.Duration(sleep))
	}
	reset := func(n *uint64) {
		*n = 0
	}
	return backoff, reset
}

func (m *monitor) monitorService(service string, notify chan<- *recordEntry) {
	var wait, n uint64

	for {
		var a []string

		svcs, meta, err := m.client.Health().Service(service, "", true, &api.QueryOptions{
			WaitIndex: wait,
		})
		if err != nil {
			log.Println(err)
			consulMonitorError.WithLabelValues(service).Inc()
			m.backoff(&n)
			continue
		}
		m.reset(&n)

		if meta != nil {
			wait = meta.LastIndex
		}

		for _, svc := range svcs {
			a = append(a, svc.Node.Address)
		}

		notify <- &recordEntry{
			addresses: a,
			service:   service,
		}
	}
}

func (m *monitor) Run(notify chan<- *recordEntry) {
	for _, svc := range m.services {
		go m.monitorService(svc, notify)
	}
}

func (m *monitor) Shutdown(ctx context.Context) error {
	close(m.shutdownCh)
	return nil
}
