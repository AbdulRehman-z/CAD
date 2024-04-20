package p2p

import (
	"errors"
	"fmt"
	"net"
	"time"
)

// TCPPeer represents the remote node over the tcp established connection.
type TCPPeer struct {
	// conn represents the underlying connection of the tcp peer
	net.Conn

	// if we dial and our connection got accepted then, outbound === true
	// if we accept an incoming connection then, outbound === false
	outbound bool
}

// NewTCPPeer creates a new TCPPeer instance.
func NewTCPPeer(conn *net.TCPConn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
	}
}

func (p *TCPPeer) Send(b []byte) error {
	if _, err := p.Conn.Write(b); err != nil {
		return fmt.Errorf("err writing to peer: %s", err)
	}
	return nil
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

	go t.startAcceptLoop()
	return nil
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) Close() error {
	return t.TcpListener.Close()
}

// Dial dials to the provided addr.
func (t *TCPTransport) Dial(port int) error {

	fmt.Println("Dial is executed")

	time.Sleep(2 * time.Second)
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   net.ParseIP("localhost"),
		Port: port,
	},
	)

	if err != nil {
		return fmt.Errorf("failed to dial: %v", err)
	}

	go t.handleConn(conn, true)
	return nil
}

// startAcceptLoop starts the accept loop for accepting incoming connections.
func (t *TCPTransport) startAcceptLoop() {
	rateLimit := 0

	for {
		conn, err := t.TcpListener.AcceptTCP()

		fmt.Println("Accepted connection: ", conn)

		if errors.Is(err, net.ErrClosed) {
			fmt.Println("TCPListener closed")
			return
		}

		if rateLimit > 5 {
			fmt.Println("Rate limit exceeded")
			break
		}

		if err != nil {
			rateLimit++
			continue
		}

		go t.handleConn(conn, false)
	}
}

// handleConn handles the incoming connection.
func (t *TCPTransport) handleConn(conn *net.TCPConn, outbound bool) {
	var err error

	defer func() {
		fmt.Println("terminating connection:", err)
		conn.Close()
	}()

	peer := NewTCPPeer(conn, outbound)
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
		}
		fmt.Println("handling 1")

		msg.From = conn.RemoteAddr()
		t.rpcch <- msg
		fmt.Printf("Received RPC: %v %v\n", msg.From.String(), string(msg.Payload))
	}
}
