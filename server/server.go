package server

import (
	"fmt"
	"log"

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

	store    *store.Store
	quitChan chan struct{}
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
		// if port is empty, skip

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

func (s *FileServer) loop() {
	defer func() {
		log.Printf("Shutting down server")
		s.Transport.Close()
	}()

loop:
	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Printf("Consumed %+v\n", msg)
		case <-s.quitChan:
			break loop
		}
	}

}
