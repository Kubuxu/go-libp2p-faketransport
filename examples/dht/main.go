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
	"github.com/pkg/profile"
)

var rng = rand.New(rand.NewSource(1))

func createHosts(ctx context.Context, fkNet *fk.FkNet, N int) ([]host.Host, error) {
	hosts := make([]host.Host, 0, N)
	for i := 0; i < N; i++ {
		host, err := fkNet.NewHost(ctx)
		if err != nil {
			return nil, err
		}

		hosts = append(hosts, host)
	}
	return hosts, nil
}

func createDHTs(ctx context.Context, hosts []host.Host) ([]*kad.IpfsDHT, error) {
	dhts := make([]*kad.IpfsDHT, len(hosts))
	for i, host := range hosts {
		dht, err := kad.New(ctx, host)
		if err != nil {
			return nil, err
		}
		dhts[i] = dht
	}
	return dhts, nil
}

func starConns(ctx context.Context, hosts []host.Host) (host.Host, error) {
	bootstrapHost := hosts[0]

	for i, v := range hosts[1:] {
		fmt.Printf("\rConnecting %d/%d", i+2, len(hosts))
		err := v.Connect(ctx, peer.AddrInfo{bootstrapHost.ID(), bootstrapHost.Addrs()})
		if err != nil {
			return nil, err
		}
	}
	fmt.Println()
	log.Printf("Bootstrap host conns: %d", len(bootstrapHost.Network().Conns()))
	log.Printf("Bootstrap host peerstore: %d", len(bootstrapHost.Network().Peerstore().Peers()))
	return bootstrapHost, nil
}

func dhtBootstrap(ctx context.Context, dhts []*kad.IpfsDHT) error {
	btsCfg := kad.DefaultBootstrapConfig
	btsCfg.Queries = 1

	for i, dht := range dhts {
		before := len(dht.Host().Network().Conns())
		err := dht.BootstrapOnce(ctx, btsCfg)
		if err != nil {
			return err
		}
		after := len(dht.Host().Network().Conns())
		fmt.Printf("\rBootstraping dhts %d/%d. [%03d=>%03d]\t\t\t", i+1, len(dhts), before, after)
	}
	fmt.Println()
	return nil
}

func main() {
	defer profile.Start(profile.MemProfile).Stop()
	N := 1000
	ctx, done := context.WithCancel(context.Background())
	defer done()

	fkNetwork := fk.NewFakeNetwork(rng)
	hosts, err := createHosts(ctx, fkNetwork, N)
	if err != nil {
		log.Fatal(err)
	}

	dhts, err := createDHTs(ctx, hosts)
	if err != nil {
		log.Fatal(err)
	}

	_, err = starConns(ctx, hosts)
	if err != nil {
		log.Fatal(err)
	}
	err = dhtBootstrap(ctx, dhts)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Done")

	runtime.GC()
	PrintMemUsage()
}
