package cache

import (
	pb "mini-cache/proto"
)

// PeerPicker is the interface that must be implemented to locate
// the peer that owns a specific key.
type PeerPicker interface {
	PickPeer(key string) (peer PeerServer, ok bool)
}

// PeerServer is the interface that must be implemented by a peer.
type PeerServer interface {
	// Get(group string, key string) ([]byte, error)
	Get(in *pb.Request, out *pb.Response) error
}
