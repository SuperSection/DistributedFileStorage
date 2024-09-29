package main

import (
	"bytes"
	"fmt"
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

	time.Sleep(2 * time.Second)

	go s2.Start()
	time.Sleep(2 * time.Second)

	for i := 0; i < 10; i++ {
		data := bytes.NewReader([]byte("my big data file here!"))
		s2.Store(fmt.Sprintf("myprivatedata_d%", i), data)
		time.Sleep(time.Millisecond * 500)
	}
	// r, err := s2.Get("myprivatedata")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// b, err := io.ReadAll(r)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	//
	// fmt.Println(string(b))

	select {}
}
