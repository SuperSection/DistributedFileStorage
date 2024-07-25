package main

import (
	"log"

	"github.com/SuperSection/FileStorage/p2p"
)

func main() {
	tcpOtions := p2p.TCPTransportOptions{
		ListenAddress: ":5500",
		Decoder:       &p2p.GOBDecoder{},
		HandshakeFunc: p2p.NOPHandshakeFunc,
	}

	tr := p2p.NewTCPTransport(tcpOtions)

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
