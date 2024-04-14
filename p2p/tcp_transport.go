package p2p

import (
	"fmt"
	"net"
	"sync"
)

// TCPPeer represents the remote node over the tcp established connection.
type TCPPeer struct {
	// conn represents the underlying connection of the tcp peer
	conn *net.TCPConn

	// if we dial and our connection got accepted then, outbound === true
	// if we accept an incoming connection then, outbound === false
	outbound bool
}

func NewTCPPeer(conn *net.TCPConn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

type TCPTransport struct {
	listenAddr  *net.TCPAddr
	tcpListener *net.TCPListener

	mu    sync.RWMutex
	peers map[*net.TCPAddr]Peer
}

func NewTCPTransport(listenAddr *net.TCPAddr) *TCPTransport {
	return &TCPTransport{
		listenAddr: listenAddr,
	}
}

func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.tcpListener, err = net.ListenTCP("tcp", t.listenAddr)
	if err != nil {
		fmt.Printf("Failed to listen on %s: %v\n", t.listenAddr, err)
	}

	fmt.Printf("Listening on %v\n", t.listenAddr.AddrPort())

	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.tcpListener.AcceptTCP()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go t.handleConn(conn)
	}
}

func (t *TCPTransport) handleConn(conn *net.TCPConn) {

	peer := NewTCPPeer(conn, true)
	fmt.Printf("Accepted new incoming connection from %v\n", peer)
}
