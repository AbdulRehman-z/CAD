package server

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

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

type MessageStoreFile struct {
	Key  string
	Size int64
}

type Message struct {
	Payload any
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

	if len(s.BootstrapNodes) > 0 {
		s.Bootstrap()
	}
	s.loop()

	return nil
}

func (s *FileServer) Bootstrap() {
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

	return nil
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	var err error

	buf := new(bytes.Buffer)
	msg := Message{
		Payload: MessageStoreFile{
			Key:  key,
			Size: 15,
		},
	}

	if err = gob.NewEncoder(buf).Encode(&msg); err != nil {
		log.Printf("err encoding: %s\n", err)
	}
	for _, peer := range s.peers {
		fmt.Println("Sending key")
		if err = peer.Send(buf.Bytes()); err != nil {
			log.Printf("err sending key: %s\n", err)
		}
	}

	time.Sleep(1 * time.Second)

	payload := []byte("Hello, Wordawd!")
	for _, peer := range s.peers {
		fmt.Println("Sending payload")
		if err = peer.Send(payload); err != nil {
			log.Printf("err sending payload: %s\n", err)
		}
	}

	return nil
}

func (s *FileServer) broadcast(p *Message) error {
	peers := []io.Writer{}
	for _, peer := range s.peers {
		peers = append(peers, peer)
	}

	mw := io.MultiWriter(peers...)
	return gob.NewEncoder(mw).Encode(p)
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("Shutting down server")
		s.Transport.Close()
	}()

loop:
	for {
		select {
		case rpc := <-s.Transport.Consume():
			var m Message
			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&m); err != nil {
				log.Printf("Context[s.loop()]: Failed to decode payload: %s", err)
			}

			fmt.Println("Consumed message: ", m)

			if err := s.handleMessage(rpc.From, &m); err != nil {
				log.Println(err)
			}

		case <-s.quitChan:
			break loop
		}
	}

}

func (s *FileServer) handleMessage(from string, m *Message) error {
	switch v := m.Payload.(type) {
	case MessageStoreFile:
		fmt.Println("Handling message store file")
		return s.handleMessageStoreFile(from, v)
	}
	return nil
}

func (s *FileServer) handleMessageStoreFile(from string, m MessageStoreFile) error {
	peer, ok := s.peers[from]
	if !ok {
		return fmt.Errorf("peer not found: %s", from)
	}

	if err := s.store.Write(m.Key, io.LimitReader(peer, m.Size)); err != nil {
		return fmt.Errorf("err writing to store: %s", err)
	}

	peer.(*p2p.TCPPeer).Wg.Done()
	return nil
}

func init() {
	gob.Register(MessageStoreFile{})
}
