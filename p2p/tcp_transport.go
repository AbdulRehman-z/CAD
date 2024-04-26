package p2p

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"sync"
)

// TCPPeer represents the remote node over the tcp established connection.
type TCPPeer struct {
	// conn represents the underlying connection of the tcp peer
	net.Conn

	// if we dial and our connection got accepted then, outbound === true
	// if we accept an incoming connection then, outbound === false
	outbound bool
	wg       *sync.WaitGroup
}

// NewTCPPeer creates a new TCPPeer instance.
func NewTCPPeer(conn *net.TCPConn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(b []byte) error {
	n, err := p.Conn.Write(b)
	if err != nil {
		return fmt.Errorf("err writing to peer: %s", err)
	}
	fmt.Println("Write the followig data to connection: ", n)
	return nil
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
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
		rpcch:            make(chan RPC, 1024),
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

func (t *TCPTransport) Addr() string {
	return t.ListenAddr.String()
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.rpcch
}

func (t *TCPTransport) Close() error {
	return t.TcpListener.Close()
}

// Dial dials to the provided addr.
func (t *TCPTransport) Dial(port int) error {
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{
		IP:   net.ParseIP("localhost"),
		Port: port,
	},
	)

	if err != nil {
		return fmt.Errorf("failed to dial: %v", err)
	}

	fmt.Println("Dial executed", conn.RemoteAddr().String())

	go t.handleConn(conn, true)
	return nil
}

// startAcceptLoop starts the accept loop for accepting incoming connections.
func (t *TCPTransport) startAcceptLoop() {
	rateLimit := 0

	for {
		conn, err := t.TcpListener.AcceptTCP()
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
		log.Printf("Peer added: %v\n", peer)
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	for {
		rpc := RPC{}
		if err = t.Decoder.Decode(conn, &rpc); err != nil {
			fmt.Printf("Context[t.handleConn()]: Failed to decode rpc: %v\n", err)
		}

		rpc.From = conn.RemoteAddr().String()
		if rpc.Stream {
			peer.wg.Add(1)
			fmt.Printf("Received stream from %v\n", rpc.From)
			peer.wg.Wait()
			fmt.Printf("Finished receiving stream from %v\n", rpc.From)
			continue
		}
		t.rpcch <- rpc
		slog.Info("write the following buffer to payload", "conn", rpc.From, "payload", string(rpc.Payload))
	}
}
