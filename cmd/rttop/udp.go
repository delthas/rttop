package main

import (
	"fmt"
	"git.sr.ht/~sircmpwn/go-bare"
	"github.com/delthas/rttop"
	"net"
	"time"
)

func udpRun(host string) error {
	c, err := net.Dial("udp", host)
	if err != nil {
		return fmt.Errorf("dial: %v", err)
	}

	go func() {
		t := time.NewTicker(period)
		for range t.C {
			l.Lock()
			buf, err := bare.Marshal(&rttop.PacketWrapper{
				Packet: &rttop.PacketPing{
					Tick: tick,
				}})
			if err != nil {
				logErr.Fatalf("marshal: %v", err)
			}
			pongs[tick%len(pongs)] = pong{
				stamp: time.Now().Sub(origin),
			}
			tick++
			l.Unlock()
			if _, err := c.Write(buf); err != nil {
				logErr.Printf("write: %v\n", err)
			}
		}
	}()

	go func() {
		buf := make([]byte, rttop.MaxSize)
		for {
			n, err := c.Read(buf)
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
			switch p := p.Packet.(type) {
			case *rttop.PacketPing:
				ev := EventRTT{
					Tick: p.Tick,
				}
				l.Lock()
				if p.Tick >= tick-len(pongs) {
					pong := &pongs[p.Tick%len(pongs)]
					ev.RTT = time.Now().Sub(origin) - pong.stamp
				}
				l.Unlock()
				events <- &ev
			}
		}
	}()

	return nil
}
