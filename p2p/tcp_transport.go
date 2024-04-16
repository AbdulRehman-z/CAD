package p2p

import (
	"fmt"
	"net"
)

// TCPPeer represents the remote node over the tcp established connection.
type TCPPeer struct {
	// conn represents the underlying connection of the tcp peer
	conn *net.TCPConn

	// if we dial and our connection got accepted then, outbound === true
	// if we accept an incoming connection then, outbound === false
	outbound bool
}

// NewTCPPeer creates a new TCPPeer instance.
func NewTCPPeer(conn *net.TCPConn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) Close() error {
	return p.conn.Close()
}

// TCPTransportOpts represents the options for the TCPTransport.
type TCPTransportOpts struct {
	ListenAddr  *net.TCPAddr
	TcpListener *net.TCPListener
	Handshake   HandshakeFunc
	Decoder     Decoder
	OnPeer      func(Peer) error
}

// TCPTransport represents the transport layer for the TCP protocol.
type TCPTransport struct {
	TCPTransportOpts
	rpcch chan RPC
}

// NewTCPTransport creates a new TCPTransport instance.
func NewTCPTransport(opts TCPTransportOpts) *TCPTransport {
	return &TCPTransport{
		TCPTransportOpts: opts,
		rpcch:            make(chan RPC),
	}
}

// ListenAndAccept listens on the provided address and accepts incoming connections.
func (t *TCPTransport) ListenAndAccept() error {
	var err error

	t.TcpListener, err = net.ListenTCP("tcp", t.ListenAddr)
	if err != nil {
		fmt.Printf("Failed to listen on %s: %v\n", t.ListenAddr, err)
	}

	fmt.Printf("Listening on %v\n", t.ListenAddr.AddrPort())

	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) startAcceptLoop() {
	for {
		conn, err := t.TcpListener.AcceptTCP()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go t.handleConn(conn)
	}
}

// handleConn handles the incoming connection.
func (t *TCPTransport) handleConn(conn *net.TCPConn) {
	var err error

	defer func() {
		fmt.Println("terminating connection:", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, true)

	if err = t.Handshake(peer); err != nil {
		fmt.Printf("Handshake failed: %v\n", err)
		return
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	msg := RPC{}
	for {
		if err = t.Decoder.Decode(conn, &msg); err != nil {
			fmt.Printf("Failed to decode RPC: %v\n", err)
			continue
		}

		msg.From = conn.RemoteAddr()
		t.rpcch <- msg
		fmt.Printf("Received RPC: %v %v\n", msg.From.String(), string(msg.Payload))
	}
}
