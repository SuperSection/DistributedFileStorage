package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/SuperSection/FileStorage/p2p"
)

type FileServerOptions struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOptions

	peerLock sync.Mutex
	peers    map[string]p2p.Peer

	storage *Storage
	quitch  chan struct{}
}

func NewFileServer(opts FileServerOptions) *FileServer {
	storageOpts := StorageOptions{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOptions: opts,
		storage:           NewStorage(storageOpts),
		quitch:            make(chan struct{}),
		peers:             make(map[string]p2p.Peer),
	}
}

func (server *FileServer) Stop() {
	close(server.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	s.peerLock.Lock()
	defer s.peerLock.Unlock()

	s.peers[p.RemoteAddress().String()] = p

	log.Printf("connected with remote %s", p.RemoteAddress())
	return nil
}

func (server *FileServer) loop() {
	defer func() {
		log.Println("file server stopped due to user quit action")
		server.Transport.Close()
	}()

	for {
		select {
		case msg := <-server.Transport.Consume():
			fmt.Println(msg)
		case <-server.quitch:
			return
		}
	}
}

func (server *FileServer) bootstrapNetwork() error {
	for _, addr := range server.BootstrapNodes {
		if len(addr) == 0 {
			continue
		}

		go func(addr string) {
			fmt.Println("attempting to connect with remote: ", addr)

			if err := server.Transport.Dial(addr); err != nil {
				log.Println("dial error: ", err)
			}
		}(addr)
	}

	return nil
}

func (server *FileServer) Start() error {
	if err := server.Transport.ListenAndAccept(); err != nil {
		return err
	}

	if len(server.BootstrapNodes) != 0 {

		server.bootstrapNetwork()
	}

	server.loop()

	return nil
}
