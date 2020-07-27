package main

import (
	"context"
	"errors"
	"log"
	"math"
	"time"

	"github.com/hashicorp/consul/api"
)

const backoffMax = 30000
const backoffBase = 500

// Fetcher is used to fetch service addresses for a given service
type Fetcher interface {
	// Fetch retrieves a slice of addresses for a given service
	Fetch(string) []string
}

// Monitor provides the ability to watch a number of Consul services and communicate
// the associated healthy services to a channel-based consumer
type Monitor struct {
	Fetcher func(string) (Fetcher, error)

	services []string

	shutdownCh chan struct{}
}

// NewMonitor creates a new Monitor object, given a set of Consul
// services to monitor
func NewMonitor(services []string) (*Monitor, error) {
	m := &Monitor{
		services:   services,
		shutdownCh: make(chan struct{}),
	}

	return m, nil
}

// ConsulFetcher implements Fetcher to retrieve a list of addresses for a given service
type ConsulFetcher struct {
	service string
	client  *api.Client

	wait  uint64
	delay uint64

	backoff func(*uint64)
	reset   func(*uint64)
}

// NewConsulFetcher creates a ConsulFetcher using the default Consul config.
// This relies on the Consul SDK's behavior of reading various configs from
// environment variables.
func NewConsulFetcher(service string) (Fetcher, error) {
	c := &ConsulFetcher{
		service: service,
	}

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}
	c.client = client

	backoff, reset := backoffFuncs()
	c.backoff = backoff
	c.reset = reset

	return c, nil
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

// Fetch retrieves a list of addresses for a Consul service. It uses an exponential
// backoff to retry on errors, and relies on blocking queries to immediately act
// on service registration changes.
func (c *ConsulFetcher) Fetch(service string) []string {
	for {
		var a []string

		svcs, meta, err := c.client.Health().Service(service, "", true, &api.QueryOptions{
			WaitIndex: c.wait,
		})
		if err != nil {
			log.Println(err)
			consulMonitorError.WithLabelValues(service).Inc()
			c.backoff(&c.delay)
			continue
		}
		c.reset(&c.delay)

		if meta != nil {
			c.wait = meta.LastIndex
		}

		for _, svc := range svcs {
			a = append(a, svc.Node.Address)
		}

		return a
	}
}

func (m *Monitor) monitorService(service string, notify chan<- *RecordEntry) {
	addressesCh := make(chan []string)
	fetcher, _ := m.Fetcher(service)

	for {
		go func() {
			addressesCh <- fetcher.Fetch(service)
		}()

		select {
		case <-m.shutdownCh:
			return
		case addresses := <-addressesCh:
			notify <- &RecordEntry{
				addresses: addresses,
				service:   service,
			}
		}
	}
}

// Run spawns a goroutine to watch the addresses for each associated Consul service
func (m *Monitor) Run(notify chan<- *RecordEntry) error {
	if m.Fetcher == nil {
		return errors.New("No Fetcher defined")
	}

	for _, svc := range m.services {
		go m.monitorService(svc, notify)
	}

	return nil
}

// Shutdown ends monitoring activity
func (m *Monitor) Shutdown(ctx context.Context) error {
	close(m.shutdownCh)
	return nil
}
