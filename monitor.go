package main

import (
	"log"
	"math"
	"time"

	"github.com/hashicorp/consul/api"
)

const backoffMax = 30000
const backoffBase = 500

func monitor(svc string, notify chan<- *RecordEntry) {
	var wait, n uint64

	for {
		var a []string

		client, _ := api.NewClient(api.DefaultConfig())
		svcs, meta, err := client.Health().Service(svc, "", true, &api.QueryOptions{
			WaitIndex: wait,
		})
		if err != nil {
			log.Println(err)
			n += 1
			sleep := math.Min(math.Pow(2, float64(n))*backoffBase, backoffMax)
			time.Sleep(time.Millisecond * time.Duration(sleep))
			continue
		}
		n = 0

		if meta != nil {
			wait = meta.LastIndex
		}

		for _, svc := range svcs {
			a = append(a, svc.Node.Address)
		}

		notify <- &RecordEntry{
			Addresses: a,
			Service:   svc,
		}
	}
}
