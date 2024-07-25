package p2p

/*
	HandshakeFun
*/
type HandshakeFunc func(Peer) error

func NOPHandshakeFunc(Peer) error { return nil }
