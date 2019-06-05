package fk

import (
	"errors"
	"io"

	ic "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/mux"
	"github.com/libp2p/go-libp2p-core/peer"
	transport "github.com/libp2p/go-libp2p-core/transport"

	ma "github.com/multiformats/go-multiaddr"
)

type fkConn struct {
	net       *FkNet
	transport *fkTransport

	remotePubKey    ic.PubKey
	remotePeer      peer.ID
	remoteMultiaddr ma.Multiaddr

	inStreams  chan mux.MuxedStream
	outStreams chan mux.MuxedStream

	closed bool
}

func (fk *fkConn) Close() error {
	fk.inStreams <- nil
	fk.closed = true
	return nil
}

func (fk *fkConn) IsClosed() bool {
	return fk.closed
}

func (fk *fkConn) OpenStream() (mux.MuxedStream, error) {
	lr, rw := io.Pipe()
	rr, lw := io.Pipe()
	localStream := &fkStream{lr, lw}
	remoteStream := &fkStream{rr, rw}
	fk.outStreams <- remoteStream
	return localStream, nil
}

func (fk *fkConn) AcceptStream() (mux.MuxedStream, error) {
	if fk.closed {
		return nil, errors.New("Listener is closed")
	}
	conn := <-fk.inStreams
	if fk.closed {
		return nil, errors.New("Listener is closed")
	}
	return conn, nil
}

func (fk *fkConn) LocalPeer() peer.ID {
	return fk.transport.peerid
}

func (fk *fkConn) LocalPrivateKey() ic.PrivKey {
	return fk.transport.privKey
}

func (fk *fkConn) RemotePeer() peer.ID {
	return fk.remotePeer
}

func (fk *fkConn) RemotePublicKey() ic.PubKey {
	return fk.remotePubKey
}

func (fk *fkConn) LocalMultiaddr() ma.Multiaddr {
	return fk.transport.multiaddr
}

func (fk *fkConn) RemoteMultiaddr() ma.Multiaddr {
	return fk.remoteMultiaddr
}

func (fk *fkConn) Transport() transport.Transport {
	return fk.transport
}
