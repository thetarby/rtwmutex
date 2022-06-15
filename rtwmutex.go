package lala

import (
	"sync"
	"sync/atomic"
)

type RTWMutex struct {
	rtw         sync.Mutex
	w           sync.Mutex // held if there are pending writers
	readerCount int32      // number of pending readers
	readerWait  int32      // number of departing readers
	writerCond  semaphore
	upgradeCond semaphore
	readerCond  semaphore
}

const rwmutexMaxReaders = 1000

func (rw *RTWMutex) RTWLock() {
	rw.rtw.Lock()
	rw.RLock()
}

func (rw *RTWMutex) RTWUnlock() {
	rw.RUnlock()
	rw.rtw.Unlock()
}

func (rw *RTWMutex) Upgrade() {
	// Announce to readers there is a pending upgrade.
	r := atomic.AddInt32(&rw.readerCount, -2*rwmutexMaxReaders)

	pw, pu := rw.pendingState(r)
	if !pu || (r+2*rwmutexMaxReaders) == 0 {
		panic("invalid state")
	}

	// Wait for active readers, however if there is already a pending writer
	// readerWait is already set no need to change it.
	if pw {
		if readerWait := atomic.AddInt32(&rw.readerWait, 0); readerWait != 1{
			rw.upgradeCond.Acquire(1)
		}
	} else {
		readers := r + 2*rwmutexMaxReaders
		if readerWait := atomic.AddInt32(&rw.readerWait, readers); readerWait != 1{
			rw.upgradeCond.Acquire(1)
		}
	}
}

func (rw *RTWMutex) RTWUpgradeUnlock() {
	r := atomic.AddInt32(&rw.readerCount, (2*rwmutexMaxReaders)-1)
	if r >= rwmutexMaxReaders {
		panic("sync: Unlock of unlocked RWMutex")
	}

	pw, pu := rw.pendingState(r)
	if pu {
		panic("invalid state")
	}

	atomic.AddInt32(&rw.readerWait, -1)
	
	if pw {
		rw.writerCond.Release(1)
	} else {
		rw.readerCond.Release(int(r))
	}

	rw.rtw.Unlock()
}

func (rw *RTWMutex) RLock() {
	if r := atomic.AddInt32(&rw.readerCount, 1); r < 0 {
		// A writer is pending, wait for it.
		rw.readerCond.Acquire(1)
	} else if r == 0 || r == -rwmutexMaxReaders || r == -2*rwmutexMaxReaders {
		panic("sync: More readers than allowed")
	}
}

func (rw *RTWMutex) RUnlock() {
	if r := atomic.AddInt32(&rw.readerCount, -1); r < 0 {
		pw, pu := rw.pendingState(r)
		readerWait := atomic.AddInt32(&rw.readerWait, -1)
		if readerWait == 1 && pu {
			rw.upgradeCond.Release(1)
		} else if readerWait == 0 && pw {
			rw.writerCond.Release(1)
		}
	}
}

func (rw *RTWMutex) Lock() {
	// First, resolve competition with other writers.
	rw.w.Lock()
	// Announce to readers there is a pending writer.
	r := atomic.AddInt32(&rw.readerCount, -rwmutexMaxReaders)

	pw, pu := rw.pendingState(r)
	if !pw {
		panic("invalid state")
	}

	// Wait for active readers.
	if pu {
		rw.writerCond.Acquire(1)
	} else {
		if (r + rwmutexMaxReaders) == 0 {
			// no reader to wait
			return
		}

		if readerWait := atomic.AddInt32(&rw.readerWait, r+rwmutexMaxReaders); readerWait != 0 {
			rw.writerCond.Acquire(1)
		}
	}
}

func (rw *RTWMutex) Unlock() {
	r := atomic.AddInt32(&rw.readerCount, rwmutexMaxReaders)
	if r >= rwmutexMaxReaders {
		panic("sync: Unlock of unlocked RWMutex")
	}
	rw.w.Unlock()
	rw.readerCond.Release(int(r))
}

func (rw *RTWMutex) pendingState(readerCount int32) (pendingWriter bool, pendingUpgrade bool) {
	if readerCount >= 0 {
		return false, false
	}
	if readerCount >= -rwmutexMaxReaders {
		return true, false
	}
	if readerCount >= -2*rwmutexMaxReaders {
		return false, true
	}
	return true, true
}

func NewRWMutex() *RTWMutex {
	return &RTWMutex{
		rtw:         sync.Mutex{},
		w:           sync.Mutex{},
		readerCount: 0,
		readerWait:  0,
		writerCond:  make(semaphore),
		upgradeCond: make(semaphore),
		readerCond:  make(semaphore),
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
