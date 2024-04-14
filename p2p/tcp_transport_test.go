package p2p

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTCPTransport(t *testing.T) {
	c := NewTCPTransport(&net.TCPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: 4000,
	})

	require.Equal(t, "tcp", c.listenAddr.Network(), "Network should be tcp")
}
