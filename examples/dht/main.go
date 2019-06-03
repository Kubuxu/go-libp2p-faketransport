package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime"

	fk "github.com/Kubuxu/go-libp2p-faketransport"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	kad "github.com/libp2p/go-libp2p-kad-dht"
)

var rng = rand.New(rand.NewSource(1))

func main() {
	N := 1000
	ctx, done := context.WithCancel(context.Background())
	defer done()
	fkNetwork := fk.NewFakeNetwork(rng)

	hosts := make([]host.Host, 0, N)
	for i := 0; i < N; i++ {
		host, err := fkNetwork.NewHost(ctx)
		if err != nil {
			panic(err)
		}

		hosts = append(hosts, host)
	}

	dhts := make([]*kad.IpfsDHT, 0, N)
	for _, host := range hosts {
		dht, err := kad.New(ctx, host)
		if err != nil {
			log.Fatalf("creating dht: %v", err)
		}
		dhts = append(dhts, dht)
	}

	bootstrapHost := hosts[0]

	for i, v := range hosts[1:] {
		fmt.Printf("\rConnecting %d/%d", i+2, N)
		err := v.Connect(ctx, peer.AddrInfo{bootstrapHost.ID(), bootstrapHost.Addrs()})
		if err != nil {
			log.Fatalf("connecting: %v", err)
		}
	}

	btsCfg := kad.DefaultBootstrapConfig
	btsCfg.Queries = 1

	fmt.Println()
	log.Printf("Bootstrap host conns: %d", len(bootstrapHost.Network().Conns()))
	log.Printf("Bootstrap host peerstore: %d\n", len(bootstrapHost.Network().Peerstore().Peers()))
	for i, dht := range dhts {
		before := len(dht.Host().Network().Conns())
		err := dht.BootstrapOnce(ctx, btsCfg)
		if err != nil {
			log.Fatalf("walking: %v", err)
		}
		after := len(dht.Host().Network().Conns())
		fmt.Printf("\rBootstraping dhts %d/%d. [%03d=>%03d]\t\t\t", i+1, N, before, after)
	}
	fmt.Println()
	log.Printf("Done")

	runtime.GC()
	PrintMemUsage()
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
