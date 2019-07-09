package fk

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"

	"github.com/libp2p/go-libp2p"
	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/peer"
	transport "github.com/libp2p/go-libp2p-core/transport"

	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

func NewFakeNetwork(rnd io.Reader) *FkNet {
	return &FkNet{
		transports: make(map[peer.ID]*fkTransport),
		rnd:        rnd,
	}
}

type FkNet struct {
	sync.RWMutex

	rnd io.Reader

	transports map[peer.ID]*fkTransport
	ids        []peer.ID
}

func (fk *FkNet) NewHost(ctx context.Context) (host.Host, error) {
	priv, pubkey, _ := ic.GenerateEd25519Key(fk.rnd)
	peerid, err := peer.IDFromPublicKey(pubkey)
	if err != nil {
		return nil, err
	}
	listenAddr := dummyIP6MA(peerid)
	return libp2p.New(ctx, libp2p.Transport(fk.newTransport),
		libp2p.Identity(priv),
		libp2p.ListenAddrs(listenAddr),
	)
}

func dummyIP6MA(peerid peer.ID) ma.Multiaddr {
	var ip net.IP = append([]byte{0xfd}, peerid[len(peerid)-15:]...)
	multiaddr, err := manet.FromIP(ip)
	if err != nil {
		panic(err)
	}
	return multiaddr
}

// Use it with combinations of `libp2p.Transport` option
func (fk *FkNet) NewTransport(priv ic.PrivKey) (*fkTransport, error) {
	peerid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, err
	}

	trans := &fkTransport{
		net:            fk,
		privKey:        priv,
		pubKey:         priv.GetPublic(),
		peerid:         peerid,
		multiaddr:      dummyIP6MA(peerid),
		incommingConns: make(chan transport.CapableConn, 1),
	}
	fk.Lock()
	fk.transports[peerid] = trans
	fk.ids = append(fk.ids, peerid)
	fk.Unlock()

	return trans, nil
}

func (fk *FkNet) Peers() []peer.ID {
	fk.RLock()
	defer fk.RUnlock()
	res := make([]peer.ID, len(fk.ids))
	copy(res, fk.ids)
	return res
}

func (fk *FkNet) dial(local peer.ID, remote peer.ID) (transport.CapableConn, error) {
	fk.RLock()
	localTransport := fk.transports[local]
	remoteTransport := fk.transports[remote]
	fk.RUnlock()

	streams1 := make(chan mux.MuxedStream, 1)
	streams2 := make(chan mux.MuxedStream, 1)

	localConn := &fkConn{
		net:             fk,
		transport:       localTransport,
		remotePubKey:    remoteTransport.pubKey,
		remotePeer:      remote,
		remoteMultiaddr: remoteTransport.multiaddr,
		inStreams:       streams1,
		outStreams:      streams2,
	}
	remoteConn := &fkConn{
		net:             fk,
		transport:       remoteTransport,
		remotePubKey:    localTransport.pubKey,
		remotePeer:      local,
		remoteMultiaddr: localTransport.multiaddr,
		inStreams:       streams2,
		outStreams:      streams1,
	}
	remoteTransport.incommingConns <- remoteConn
	return localConn, nil
}

type fkTransport struct {
	net    *FkNet
	closed bool

	privKey   ic.PrivKey
	pubKey    ic.PubKey
	peerid    peer.ID
	multiaddr ma.Multiaddr

	incommingConns chan transport.CapableConn
}

func (fk *fkTransport) Dial(ctx context.Context, raddr ma.Multiaddr, p peer.ID) (transport.CapableConn, error) {
	return fk.net.dial(fk.peerid, p)
}

func (*fkTransport) CanDial(addr ma.Multiaddr) bool {
	return len(addr.Protocols()) == 1 && addr.Protocols()[0].Code == ma.P_IP6
}

func (fk *fkTransport) Listen(laddr ma.Multiaddr) (transport.Listener, error) {
	return fk, nil
}

func (*fkTransport) Protocols() []int {
	return []int{ma.P_IP6}
}

func (*fkTransport) Proxy() bool {
	return false
}

// transport.Listner implementation

func (fk *fkTransport) Accept() (transport.CapableConn, error) {
	if fk.closed {
		return nil, errors.New("Listener is closed")
	}
	conn := <-fk.incommingConns
	if fk.closed {
		return nil, errors.New("Listener is closed")
	}
	return conn, nil
}

func (fk *fkTransport) Close() error {
	fk.closed = true
	fk.incommingConns <- nil // notify Accept
	return nil
}

func (fk *fkTransport) Addr() net.Addr {
	addr, _ := manet.ToNetAddr(fk.multiaddr)
	return addr
}

func (fk *fkTransport) Multiaddr() ma.Multiaddr {
	return fk.multiaddr
}
