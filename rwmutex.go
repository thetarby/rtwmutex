package sync

import (
	"sync"
	"sync/atomic"
)

type RWMutex struct {
	w           sync.Mutex // held if there are pending writers
	writerSem   uint32     // semaphore for writers to wait for completing readers
	readerSem   uint32     // semaphore for readers to wait for completing writers
	readerCount int32      // number of pending readers
	readerWait  int32      // number of departing readers
	writerCond  *sync.Cond
	readerCond  *sync.Cond
}

const rwmutexMaxReaders = 1 << 30

func (rw *RWMutex) RLock() {
	if atomic.AddInt32(&rw.readerCount, 1) < 0 {
		// A writer is pending, wait for it.
		rw.readerCond.L.Lock()
		rw.readerCond.Wait()
	}
}

func (rw *RWMutex) RUnlock() {
	if r := atomic.AddInt32(&rw.readerCount, -1); r < 0 {
		// Outlined slow-path to allow the fast-path to be inlined
		rw.rUnlockSlow(r)
	}
}

func (rw *RWMutex) rUnlockSlow(r int32) {
	// A writer is pending.
	if atomic.AddInt32(&rw.readerWait, -1) == 0 {
		// The last reader unblocks the writer.
		rw.writerCond.Signal()
	}
}

func (rw *RWMutex) Lock() {
	// First, resolve competition with other writers.
	rw.w.Lock()
	// Announce to readers there is a pending writer.
	r := atomic.AddInt32(&rw.readerCount, -rwmutexMaxReaders) + rwmutexMaxReaders
	// Wait for active readers.
	if r != 0 && atomic.AddInt32(&rw.readerWait, r) != 0 {
		rw.writerCond.L.Lock()
		rw.writerCond.Wait()
	}
}

func (rw *RWMutex) Unlock() {
	// Announce to readers there is no active writer.
	r := atomic.AddInt32(&rw.readerCount, rwmutexMaxReaders)
	if r >= rwmutexMaxReaders {
		panic("sync: Unlock of unlocked RWMutex")
	}
	// Unblock blocked readers, if any.
	for i := 0; i < int(r); i++ {
		rw.readerCond.Broadcast()
	}
	// Allow other writers to proceed.
	rw.w.Unlock()
}

func NewRWMutex() *RWMutex{
	w,r := sync.Mutex{}, sync.Mutex{}
	return  &RWMutex{
		w:           sync.Mutex{},
		writerSem:   0,
		readerSem:   0,
		readerCount: 0,
		readerWait:  0,
		writerCond:  sync.NewCond(&w),
		readerCond:  sync.NewCond(&r),
	}
}

// RLocker returns a Locker interface that implements
// the Lock and Unlock methods by calling rw.RLock and rw.RUnlock.
func (rw *RWMutex) RLocker() sync.Locker {
	return (*rlocker)(rw)
}

type rlocker RWMutex

func (r *rlocker) Lock()   { (*RWMutex)(r).RLock() }
func (r *rlocker) Unlock() { (*RWMutex)(r).RUnlock() }
