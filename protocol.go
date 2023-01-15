package rttop

import "git.sr.ht/~sircmpwn/go-bare"

type PacketWrapper struct {
	Packet Packet
}

type Packet interface {
	bare.Union
}

type PacketPing struct {
	Tick int
}

func (p PacketPing) IsUnion() {}

func init() {
	bare.RegisterUnion((*Packet)(nil)).
		Member(PacketPing{}, 1)
}

const MaxSize = 508
