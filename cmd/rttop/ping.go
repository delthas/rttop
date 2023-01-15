package main

import (
	"fmt"
	"github.com/go-ping/ping"
	"time"
)

func pingRun(host string) error {
	pinger, err := ping.NewPinger(host)
	if err != nil {
		return fmt.Errorf("init ping: %v", err)
	}
	pinger.Interval = period
	pinger.RecordRtts = false

	seq := make(map[int]int, 65536)
	pinger.OnSend = func(p *ping.Packet) {
		l.Lock()
		seq[p.Seq] = tick
		pongs[tick%len(pongs)] = pong{
			stamp: time.Now().Sub(origin),
		}
		tick++
		l.Unlock()
	}
	pinger.OnRecv = func(p *ping.Packet) {
		events <- &EventRTT{
			Tick: seq[p.Seq],
			RTT:  p.Rtt,
		}
	}
	go func() {
		if err := pinger.Run(); err != nil {
			logErr.Printf("run ping: %v\n", err)
		}
	}()
	return nil
}
