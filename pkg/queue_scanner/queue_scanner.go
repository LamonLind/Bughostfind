package queue_scanner

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Ctx struct {
	ScanSuccessList []interface{}
	ScanFailedList  []interface{}
	ScanComplete    int

	dataList []interface{}

	mx sync.Mutex
	context.Context
}

func (c *Ctx) Log(a ...interface{}) {
	fmt.Printf("\r\033[2K%s\n", a...)
}

func (c *Ctx) Logf(f string, a ...interface{}) {
	fmt.Printf("\r\033[2K%s\n", fmt.Sprintf(f, a...))
}

func (c *Ctx) LogReplace(a ...string) {
	scanSuccess := len(c.ScanSuccessList)
	scanFailed := len(c.ScanFailedList)
	scanCompletePercentage := float64(c.ScanComplete) / float64(len(c.dataList)) * 100
	s := fmt.Sprintf(
		"  %.2f%% - C: %d / %d - S: %d - F: %d - %s", scanCompletePercentage, c.ScanComplete, len(c.dataList), scanSuccess, scanFailed, strings.Join(a, " "),
	)
	fmt.Print("\r\033[2K", s, "\r")
}

func (c *Ctx) LogReplacef(f string, a ...interface{}) {
	c.LogReplace(fmt.Sprintf(f, a...))
}

func (c *Ctx) ScanSuccess(a interface{}, fn func()) {
	if fn != nil {
		fn()
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	c.ScanSuccessList = append(c.ScanSuccessList, a)
}

func (c *Ctx) ScanFailed(a interface{}, fn func()) {
	if fn != nil {
		fn()
	}

	c.mx.Lock()
	defer c.mx.Unlock()

	c.ScanFailedList = append(c.ScanFailedList, a)
}

type QueueScannerScanFunc func(c *Ctx, a interface{})
type QueueScannerDoneFunc func(c *Ctx)

type QueueScanner struct {
	threads  int
	scanFunc QueueScannerScanFunc
	doneFunc QueueScannerDoneFunc
	queue    chan interface{}
	wg       sync.WaitGroup

	ctx *Ctx
}

func NewQueueScanner(threads int, scanFunc QueueScannerScanFunc, doneFunc QueueScannerDoneFunc) *QueueScanner {
	t := &QueueScanner{
		threads:  threads,
		scanFunc: scanFunc,
		doneFunc: doneFunc,
		queue:    make(chan interface{}),
		ctx:      &Ctx{},
	}

	for i := 0; i < t.threads; i++ {
		go t.run()
	}

	return t
}

func (s *QueueScanner) run() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		a, ok := <-s.queue
		if !ok {
			break
		}
		s.ctx.LogReplacef("%v", a)
		s.scanFunc(s.ctx, a)

		s.ctx.mx.Lock()
		s.ctx.ScanComplete++
		s.ctx.mx.Unlock()

		s.ctx.LogReplacef("%v", a)
	}
}

func (s *QueueScanner) Add(dataList ...interface{}) {
	s.ctx.dataList = append(s.ctx.dataList, dataList...)
}

func (s *QueueScanner) Start() {
	for _, data := range s.ctx.dataList {
		s.queue <- data
	}
	close(s.queue)

	s.wg.Wait()

	if s.doneFunc != nil {
		s.doneFunc(s.ctx)
	}
}