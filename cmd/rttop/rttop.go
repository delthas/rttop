package main

import (
	"flag"
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/montanaflynn/stats"
	"log"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const defaultServer = "delthas.fr:13770"

var history = 10 * time.Second
var period = 16 * time.Millisecond
var drawPeriod = 50 * time.Millisecond
var timeout = 1500 * time.Millisecond

type pong struct {
	ok    bool
	stamp time.Duration
	rtt   time.Duration
}

var protocol string
var server = defaultServer

var logErr = log.New(os.Stderr, "err: ", log.LstdFlags|log.Lmsgprefix)

var screen tcell.Screen
var exit atomic.Value // bool
var w, h int

var origin = time.Now() // used to generate monotonic offsets
var tick = 0
var l sync.Mutex
var pongs = make([]pong, history/period)

var events = make(chan any, 128)

type EventRTT struct {
	Tick int
	RTT  time.Duration
}

var charsLeft = []rune{'⡀', '⠄', '⠂', '⠁'}
var charsRight = []rune{'⢀', '⠠', '⠐', '⠈'}
var charsBoth = []rune{
	'⣀', '⡠', '⡐', '⡈',
	'⢄', '⠤', '⠔', '⠌',
	'⢂', '⠢', '⠒', '⠊',
	'⢁', '⠡', '⠑', '⠉',
}
var charsCount = len(charsLeft)

var data = make([]float64, len(pongs))

func must(v float64, err error) float64 {
	if err != nil {
		panic(err)
	}
	return v
}

func fDur(v float64, n int) string {
	return fFloat(v/float64(time.Millisecond), n)
}

func fFloat(v float64, n int) string {
	d := 1 + int(math.Log10(v))
	if d > n {
		return strings.Repeat("9", n)
	}
	if d == n {
		return fmt.Sprintf("%"+strconv.Itoa(n)+".0f", v)
	}
	if d == n-1 {
		return fmt.Sprintf("%"+strconv.Itoa(n-1)+".0f.", v)
	}
	if d < 1 {
		return fmt.Sprintf("%-0."+strconv.Itoa(n-2)+"f", v)
	}
	return fmt.Sprintf("%-"+strconv.Itoa(n)+"."+strconv.Itoa(n-d-1)+"f", v)
}

func print(x int, y int, s string) {
	for _, r := range s {
		screen.SetCell(x, y, tcell.Style{}, r)
		x++
	}
}

func initScreen() {
	var err error
	screen, err = tcell.NewScreen()
	if err != nil {
		logErr.Fatalf("open screen: %v", err)
	}
	if err := screen.Init(); err != nil {
		logErr.Fatalf("init screen: %v", err)
	}
	screen.HideCursor()
	screen.Clear()
	w, h = screen.Size()

	go func() {
		for !exit.Load().(bool) {
			ev := screen.PollEvent()
			if ev == nil {
				exit.Store(true)
				break
			}
			events <- ev
		}
	}()
}

func draw() {
	l.Lock()
	defer l.Unlock()
	st := tcell.Style{}
	n := 0
	loss := 0
	deadline := time.Now().Sub(origin) - timeout
	for _, p := range pongs {
		if !p.ok {
			if p.stamp > 0 && p.stamp < deadline {
				loss++
			}
			continue
		}
		data[n] = float64(p.rtt)
		n++
	}
	if n == 0 {
		return
	}
	d := data[:n]

	if screen == nil {
		initScreen()
	}
	screen.Clear()

	lr := float64(loss) / float64(loss+n)
	sMin := must(stats.Min(d))
	sMax := must(stats.Max(d))
	sAvg := must(stats.Mean(d))
	sDev := must(stats.StandardDeviation(d))
	sQua, _ := stats.Quartile(d)
	header := fmt.Sprintf("rttop @ %v://%v │ %vms±%vms │ loss=%v%% | p0=%vms — p25=%vms — p50=%vms — p75=%vms — p100=%vms", protocol, server, fDur(sAvg, 4), fDur(sDev, 3), fFloat(lr*100, 4), fDur(sMin, 4), fDur(sQua.Q1, 4), fDur(sQua.Q2, 4), fDur(sQua.Q3, 4), fDur(sMax, 4))

	min := time.Duration(sMin * 0.95)
	max := time.Duration(sMax / 0.95)
	wOff := 5
	hOff := 2

	print(0, 0, header)
	for x := 0; x < w; x++ {
		screen.SetCell(x, 1, st, '─')
	}
	screen.SetCell(wOff-1, 1, st, '┬')

	for y := 0; y < h-hOff; y++ {
		v := min + (max-min)*time.Duration(y)/time.Duration(h-hOff)
		s := fDur(float64(v), 4)
		print(0, h-y-1, s)
		screen.SetCell(4, h-y-1, st, '│')
	}

outer:
	for x := 0; x < w-wOff; x++ {
		loss := false
		var left time.Duration = 0
		for i := x * len(pongs) / (w - wOff); i < len(pongs) && (left == 0 || i < (2*x+1)*len(pongs)/(2*(w-wOff))); i++ {
			if i >= tick {
				continue outer
			}
			p := pongs[i]
			if !p.ok {
				if p.stamp < deadline {
					loss = true
				}
				left = 0
				break
			}
			if p.rtt > left {
				left = p.rtt
			}
		}
		var right time.Duration = 0
		for i := (2*x + 1) * len(pongs) / (2 * (w - wOff)); i < len(pongs) && (right == 0 || i < (x+1)*len(pongs)/(w-wOff)); i++ {
			if i >= tick {
				continue outer
			}
			p := pongs[i]
			if !p.ok {
				if p.stamp < deadline {
					loss = true
				}
				right = 0
				break
			}
			if p.rtt > right {
				right = p.rtt
			}
		}
		if left == 0 || right == 0 {
			ch := '┊'
			if loss {
				ch = '┋'
			}
			for y := hOff; y < h; y++ {
				screen.SetCell(x+wOff, y, st, ch)
			}
			continue
		}
		rl := float64(left-min) / float64(max-min) * float64(h-hOff)
		rr := float64(right-min) / float64(max-min) * float64(h-hOff)
		cl := int(rl*float64(charsCount)) % charsCount
		cr := int(rr*float64(charsCount)) % charsCount
		if int(rl) != int(rr) {
			screen.SetCell(x+5, h-int(rl)-1, st, charsLeft[cl])
			screen.SetCell(x+5, h-int(rr)-1, st, charsRight[cr])
		} else {
			screen.SetCell(x+5, h-int(rl)-1, st, charsBoth[cl*charsCount+cr])
		}
	}
}

func main() {
	flag.Parse()
	if flag.NArg() > 0 {
		server = flag.Arg(0)
	}
	if u, err := url.Parse(server); err == nil && u.Scheme != "" && u.Host != "" {
		protocol = u.Scheme
		server = u.Host
	} else if _, _, err := net.SplitHostPort(server); err == nil {
		protocol = "udp"
	} else {
		protocol = "ping"
	}
	switch protocol {
	case "udp":
		if err := udpRun(server); err != nil {
			log.Fatalf("run: %v", err)
		}
	case "ping", "icmp":
		if err := pingRun(server); err != nil {
			log.Fatalf("run: %v", err)
		}
	default:
		log.Fatalf("unknown server: %q", flag.Arg(0))
	}

	exit.Store(false)
	drawTimer := time.NewTimer(drawPeriod)
	for !exit.Load().(bool) {
		select {
		case ev := <-events:
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyCtrlC:
					exit.Store(true)
				}
			case *tcell.EventResize:
				w, h = screen.Size()
				draw()
				screen.Sync()
			case *EventRTT:
				l.Lock()
				if ev.Tick >= tick-len(pongs) {
					p := &pongs[ev.Tick%len(pongs)]
					p.ok = true
					p.rtt = ev.RTT
				}
				l.Unlock()
			}
		case <-drawTimer.C:
			draw()
			if screen != nil {
				screen.Show()
			}
			drawTimer.Reset(drawPeriod)
		}
	}

	if screen != nil {
		screen.Fini()
	}
}
