package main

import (
	"flag"
	"git.sr.ht/~sircmpwn/go-bare"
	"github.com/delthas/rttop"
	"log"
	"net"
	"os"
)

const defaultListen = ":13770"

var logErr = log.New(os.Stderr, "err: ", log.LstdFlags)

func main() {
	listen := flag.String("listen", defaultListen, "server listen host:port")
	flag.Parse()

	c, err := net.ListenPacket("udp", *listen)
	if err != nil {
		logErr.Fatalf("listen: %v", err)
	}
	log.Printf("listening on %q\n", *listen)
	buf := make([]byte, rttop.MaxSize)
	for {
		n, addr, err := c.ReadFrom(buf)
		if err == net.ErrClosed {
			break
		}
		if err != nil {
			logErr.Printf("read: %v\n", err)
			continue
		}
		var p rttop.PacketWrapper
		if err := bare.Unmarshal(buf[:n], &p); err != nil {
			logErr.Printf("unmarshal: %v\n", err)
			continue
		}
		var r *rttop.PacketWrapper
		switch p := p.Packet.(type) {
		case *rttop.PacketPing:
			r = &rttop.PacketWrapper{
				Packet: &rttop.PacketPing{
					Tick: p.Tick,
				},
			}
		default:
			continue
		}
		if r == nil {
			continue
		}
		br, err := bare.Marshal(r)
		if err != nil {
			logErr.Printf("unmarshal: %v\n", err)
			continue
		}
		if _, err := c.WriteTo(br, addr); err != nil {
			logErr.Printf("write: %v\n", err)
		}
	}
}
