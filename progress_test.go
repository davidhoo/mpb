package mpb_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sync"
	"testing"
	"time"

	. "github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/cwriter"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestBarCount(t *testing.T) {
	p := New(WithOutput(ioutil.Discard))

	var wg sync.WaitGroup
	wg.Add(1)
	b := p.AddBar(100)
	go func() {
		for i := 0; i < 100; i++ {
			if i == 33 {
				wg.Done()
			}
			b.Increment()
			time.Sleep(randomDuration(100 * time.Millisecond))
		}
	}()

	wg.Wait()
	count := p.BarCount()
	if count != 1 {
		t.Errorf("BarCount want: %q, got: %q\n", 1, count)
	}

	p.Abort(b)
	p.Wait()
}

func TestBarAbort(t *testing.T) {
	p := New(WithOutput(ioutil.Discard))

	var wg sync.WaitGroup
	wg.Add(1)
	bars := make([]*Bar, 3)
	for i := 0; i < 3; i++ {
		b := p.AddBar(100)
		bars[i] = b
		go func(n int) {
			for i := 0; i < 100; i++ {
				if n == 0 && i == 33 {
					p.Abort(b)
					wg.Done()
				}
				b.Increment()
				time.Sleep(randomDuration(100 * time.Millisecond))
			}
		}(i)
	}

	wg.Wait()
	count := p.BarCount()
	if count != 2 {
		t.Errorf("BarCount want: %q, got: %q\n", 2, count)
	}
	p.Abort(bars[1])
	p.Abort(bars[2])
	p.Wait()
}

func TestWithCancel(t *testing.T) {
	cancel := make(chan struct{})
	shutdown := make(chan struct{})
	p := New(
		WithOutput(ioutil.Discard),
		WithCancel(cancel),
		WithShutdownNotifier(shutdown),
	)

	numBars := 3
	bars := make([]*Bar, 0, numBars)
	for i := 0; i < numBars; i++ {
		bar := p.AddBar(int64(1000), BarID(i))
		bars = append(bars, bar)
		go func() {
			for !bar.Completed() {
				time.Sleep(randomDuration(100 * time.Millisecond))
				bar.Increment()
			}
		}()
	}

	time.AfterFunc(100*time.Millisecond, func() {
		close(cancel)
	})

	p.Wait()

	select {
	case <-shutdown:
	case <-time.After(200 * time.Millisecond):
		t.FailNow()
	}
}

var (
	cursorUp           = fmt.Sprintf("%c[%dA", cwriter.ESC, 1)
	clearLine          = fmt.Sprintf("%c[2K\r", cwriter.ESC)
	clearCursorAndLine = cursorUp + clearLine
)

func TestWithFormat(t *testing.T) {
	var buf bytes.Buffer
	customFormat := "╢▌▌░╟"
	p := New(WithOutput(&buf), WithFormat(customFormat))
	bar := p.AddBar(100, BarTrim())

	for i := 0; i < 100; i++ {
		if i == 33 {
			p.Abort(bar)
			break
		}
		time.Sleep(randomDuration(100 * time.Millisecond))
		bar.Increment()
	}

	p.Wait()

	bb := bytes.Split(buf.Bytes(), []byte("\n"))
	lastLine := bb[len(bb)-2]
	lastLine = lastLine[len(clearCursorAndLine):]

	for _, r := range customFormat {
		if !bytes.ContainsRune(lastLine, r) {
			t.Errorf("Rune %#U not found in bar\n", r)
		}
	}
}

func randomDuration(max time.Duration) time.Duration {
	return time.Duration(rand.Intn(10)+1) * max / 10
}
