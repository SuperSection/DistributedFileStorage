package main

import (
	"log"

	"github.com/SuperSection/FileStorage/p2p"
)

func main() {
	tr := p2p.NewTCPTransport(":5500")

	if err := tr.ListenAndAccept(); err != nil {
		log.Fatal(err)
	}

	select {}
}
