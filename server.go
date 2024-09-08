package main

import (
	"fmt"
	"log"

	"github.com/SuperSection/FileStorage/p2p"
)

type FileServerOptions struct {
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOptions

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
	}
}

func (server *FileServer) Stop() {
	close(server.quitch)
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

func (server *FileServer) Start() error {
	if err := server.Transport.ListenAndAccept(); err != nil {
		return err
	}

	server.loop()

	return nil
}
