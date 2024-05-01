package p2p

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
	tcp := NewTCPTransport(TCPTransportOpts{
		ListenAddr: &net.TCPAddr{
			IP:   net.ParseIP("localhost"),
			Port: 3000,
		},
		Handshake: NOPEHandshake,
		Decoder:   &DefaultDecoder{},
	})
	tcp.OnPeer = func(Peer) error { return nil }

	require.Equal(t, "tcp", tcp.ListenAddr.Network(), "Network should be tcp")
}
