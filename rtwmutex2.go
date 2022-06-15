package lala

import (
	"sync"
)

type lockState uint8

const (
	Empty = iota
	Pending
	Entered
)

type RWMutex2 struct {
	rtw            sync.Mutex
	w              sync.Mutex
	l              sync.Mutex
	readers        int32
	readersPending int32
	writer         lockState
	upgrade        lockState
	writerCond     semaphore
	upgradingCond  semaphore
	readerCond     semaphore
}

func (rw *RWMutex2) RTWLock() {
	rw.rtw.Lock()
	rw.RLock()
}

func (rw *RWMutex2) RTWUnlock() {
	rw.RUnlock()
	rw.rtw.Unlock()
}

func (rw *RWMutex2) Upgrade() {
	rw.l.Lock()
	if rw.readers == 1 {
		rw.upgrade = Entered
		rw.l.Unlock()
		return
	}

	rw.upgrade = Pending
	rw.l.Unlock()

	// wait readers to finish
	rw.upgradingCond.Acquire(1)
}

func (rw *RWMutex2) RTWUpgradeUnlock() {
	rw.l.Lock()
	rw.readers--
	rw.upgrade = Empty
	if rw.writer == Pending {
		rw.writer = Entered
		rw.writerCond.Release(1)
	} else {
		p := rw.readersPending 
		rw.readersPending = 0
		rw.readers = p
		rw.readerCond.Release(int(p))
	}
	rw.rtw.Unlock()
	rw.l.Unlock()
}

func (rw *RWMutex2) RLock() {
	wait := false
	rw.l.Lock()
	if rw.writer == Pending || rw.upgrade == Pending || rw.writer == Entered || rw.upgrade == Entered {
		wait = true
		rw.readersPending++
	} else {
		rw.readers++
	}
	rw.l.Unlock()
	if wait {
		rw.readerCond.Acquire(1)
	}
}

func (rw *RWMutex2) RUnlock() {
	rw.l.Lock()
	rw.readers--
	if rw.readers == 0 && rw.writer == Pending {
		// wake writer
		rw.writer = Entered
		rw.writerCond.Release(1)
	} else if rw.readers == 1 && rw.upgrade == Pending {
	 	// wake upgrading
		rw.upgrade = Entered
		rw.upgradingCond.Release(1)
	}
	rw.l.Unlock()
}

func (rw *RWMutex2) Unlock() {
	rw.l.Lock()
	p := rw.readersPending
	rw.readers = p
	rw.readersPending = 0
	rw.writer = Empty
	rw.w.Unlock()
	rw.l.Unlock()

	rw.readerCond.Release(int(p))
}

func (rw *RWMutex2) Lock() {
	rw.w.Lock()
	rw.l.Lock()
	if rw.readers == 0 {
		rw.writer = Entered
		rw.l.Unlock()
		return
	}

	rw.writer = Pending
	rw.l.Unlock()

	// wait readers to finish
	rw.writerCond.Acquire(1)
}

func NewRWMutex2() *RWMutex2 {
	return &RWMutex2{
		rtw:            sync.Mutex{},
		w:              sync.Mutex{},
		l:              sync.Mutex{},
		readers:        0,
		readersPending: 0,
		writer:         Empty,
		upgrade:        Empty,
		writerCond:     make(semaphore),
		upgradingCond:  make(semaphore),
		readerCond:     make(semaphore),
	}
}

type empty struct{}
type semaphore chan empty

// acquire n resources
func (s semaphore) Acquire(n int) {
	e := empty{}
	for i := 0; i < n; i++ {
		s <- e
	}
}

// release n resources
func (s semaphore) Release(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}
