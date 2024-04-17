package main

import (
	"fmt"

	"github.com/1500-bytes/CAD/p2p"
)

type FileServerOpts struct {
	ListenAddr        string
	PathTransformFunc PathTransformFunc
	RootStorage       string
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOpts

	store *Store
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.RootStorage,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		store:          NewStore(storeOpts),
		FileServerOpts: opts,
	}
}

func (s *FileServer) Start() error {
	if err := s.Transport.ListenAndAccept(); err != nil {
		return fmt.Errorf("err starting server: %s", err)
	}
	return nil
}
