package rtwmutex

import (
	"sync"
)

type lockState uint8

const (
	Empty = iota
	Pending
	Entered
)

type RTWMutex struct {
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

func (rw *RTWMutex) RTWLock() {
	rw.rtw.Lock()
	rw.RLock()
}

func (rw *RTWMutex) RTWUnlock() {
	rw.RUnlock()
	rw.rtw.Unlock()
}

func (rw *RTWMutex) Upgrade() {
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

func (rw *RTWMutex) RTWUpgradeUnlock() {
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

func (rw *RTWMutex) RLock() {
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

func (rw *RTWMutex) RUnlock() {
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

func (rw *RTWMutex) Unlock() {
	rw.l.Lock()
	p := rw.readersPending
	rw.readers = p
	rw.readersPending = 0
	rw.writer = Empty
	rw.w.Unlock()
	rw.l.Unlock()

	rw.readerCond.Release(int(p))
}

func (rw *RTWMutex) Lock() {
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

func NewRWMutex() *RTWMutex {
	return &RTWMutex{
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

// RLocker returns a Locker interface that implements
// the Lock and Unlock methods by calling rw.RLock and rw.RUnlock.
func (rw *RTWMutex) RLocker() sync.Locker {
	return (*rlocker)(rw)
}

type rlocker RTWMutex

func (r *rlocker) Lock()   { (*RTWMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*RTWMutex)(r).RUnlock() }
