package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"

	"github.com/1500-bytes/CAD/p2p"
	"github.com/1500-bytes/CAD/store"
)

type FileServerOpts struct {
	PathTransformFunc store.PathTransformFunc
	RootStorage       string
	Transport         p2p.Transport
	// BootstrapNodes represents the list of nodes to connect to when starting the server
	BootstrapNodes []int
}

type FileServer struct {
	FileServerOpts

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	store    *store.Store
	quitChan chan struct{}
}

type Payload struct {
	Key  string
	data []byte
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := store.StoreOpts{
		Root:              opts.RootStorage,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		quitChan:       make(chan struct{}),
		store:          store.NewStore(storeOpts),
		FileServerOpts: opts,
		peers:          make(map[string]p2p.Peer),
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return fmt.Errorf("err starting server: %s", err)
	}

	s.Bootstrap()
	s.loop()

	return nil
}

func (s *FileServer) Bootstrap() {
	if len(s.BootstrapNodes) == 0 {
		return
	}

	for _, port := range s.BootstrapNodes {
		go func(port int) {
			if err := s.Transport.Dial(port); err != nil {
				log.Printf("Failed to dial to %d: %v", port, err)
			}
		}(port)
	}
}

func (s *FileServer) Stop() {
	close(s.quitChan)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddr().String()] = p
	log.Printf("connected with remote: %s", p.RemoteAddr().String())

	return nil
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	fmt.Println("Storing data: ", key)
	buf := new(bytes.Buffer)
	// io.TeeReader returns a Reader that writes to w what it reads from r.
	tee := io.TeeReader(r, buf)
	if err := s.store.Write(key, tee); err != nil {
		return fmt.Errorf("err writing to store: %s", err)
	}

	p := &Payload{
		Key:  key,
		data: buf.Bytes(),
	}

	return s.broadcast(p)
}

func (s *FileServer) broadcast(p *Payload) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(p)
}

func (s *FileServer) loop() {
	defer func() {
		log.Printf("Shutting down server")
		s.Transport.Close()
	}()

loop:
	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Println("consumed")
			var p Payload
			if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&p); err != nil {
				log.Printf("Failed to decode payload: %v", err)
			}
		case <-s.quitChan:
			break loop
		}
	}

}
