package rtwmutex

/* Simple semaphore implementation with channels */

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
