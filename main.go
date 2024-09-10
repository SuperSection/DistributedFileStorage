package main

import (
	"bytes"
	"log"
	"time"

	"github.com/SuperSection/FileStorage/p2p"
)

func makeServer(listenAddr string, nodes ...string) *FileServer {

	tcpOptions := p2p.TCPTransportOptions{
		ListenAddress: listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       &p2p.DefaultDecoder{},
	}

	tcpTransport := p2p.NewTCPTransport(tcpOptions)

	fileServerOptions := FileServerOptions{
		StorageRoot:       listenAddr + "_network",
		PathTransformFunc: CASPathTransformFunc,
		Transport:         tcpTransport,
		BootstrapNodes:    nodes,
	}

	s := NewFileServer(fileServerOptions)

	tcpTransport.OnPeer = s.OnPeer

	return s
}

func main() {

	s1 := makeServer(":3000", "")
	s2 := makeServer(":4000", ":3000")

	go func() {
		log.Fatal(s1.Start())
	}()

	time.Sleep(4 * time.Second)

	go s2.Start()
	time.Sleep(4 * time.Second)

	data := bytes.NewReader([]byte("my big data file here!"))

	s2.StoreData("myprivatedata", data)

	select {}
}
