package lala

import (
	"sync"
)

type RWMutex2 struct {
	rtw            sync.Mutex
	w              sync.Mutex
	l              sync.Mutex
	readers        int32
	readersPending int32
	pendingWriter  bool
	pendingUpgrade bool
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
	rw.pendingUpgrade = true
	
	if rw.readers == 1 {
		rw.l.Unlock()
		return
	}

	rw.l.Unlock()

	// wait readers to finish
	rw.upgradingCond.Acquire(1)
}


func (rw *RWMutex2) RTWUpgradeUnlock() {
	panic("implement me")
}


func (rw *RWMutex2) RLock() {
	wait := false
	rw.l.Lock()
	if rw.pendingWriter || rw.pendingUpgrade {
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
	if rw.readers == 0 && rw.pendingWriter{
		// wake writer
		rw.writerCond.Release(1)
	} else if rw.readers == 1 && rw.pendingUpgrade {
		// wake upgrading
		rw.upgradingCond.Release(1)
	}
	rw.l.Unlock()
}

func (rw *RWMutex2) Unlock() {
	rw.l.Lock()
	if rw.pendingUpgrade && !rw.pendingWriter {
		rw.readers += rw.readersPending
		rw.readersPending = 0
		rw.pendingUpgrade = false
		rw.rtw.Unlock()
		rw.readerCond.Release(int(rw.readers))
	} else if rw.pendingUpgrade && rw.pendingWriter {
		rw.pendingUpgrade = false
		rw.readers--
		rw.rtw.Unlock()
		rw.writerCond.Release(1)
	} else {
		p := rw.readersPending
		rw.readers += p // NOTE: readers must be 0 already
		rw.readersPending = 0
		rw.pendingWriter = false
		rw.w.Unlock()
 		rw.readerCond.Release(int(p))
	}
	rw.l.Unlock()
}

func (rw *RWMutex2) Lock() {
	rw.w.Lock()
	rw.l.Lock()
	rw.pendingWriter = true
	
	if rw.readers == 0 {
		rw.l.Unlock()
		return
	}

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
		pendingWriter:  false,
		pendingUpgrade: false,
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