package rtwmutex

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"
)

func writer2(rwm IRTWMutex, num_iterations int, activity *int32, incr *int32, cdone chan bool) {
	for i := 0; i < num_iterations; i++ {
		rwm.Lock()
		*incr = *incr + 1 
		n := atomic.AddInt32(activity, 10000)
		if n != 10000 {
			rwm.Unlock()
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}
		for i := 0; i < 100; i++ {
		}
		atomic.AddInt32(activity, -10000)
		rwm.Unlock()
	}
	cdone <- true
}

func readtowriter(rwm IRTWMutex, num_iterations int, activity *int32, incr *int32, cdone chan bool) {
	for i := 0; i < num_iterations; i++ {
		rwm.RTWLock()
		firstRead := *incr
		n := atomic.AddInt32(activity, 1)
		if n < 1 || n >= 10000 {
			//rwm.RUnlock()
			panic(fmt.Sprintf("wlock(%d)\n", n))
		}

		// upgrade or continue with read lock
		if rand.Intn(2) == 0 {
			rwm.Upgrade()
			n := atomic.AddInt32(activity, 10000)
			if n != 10001 {
				//rwm.Unlock()
				panic(fmt.Sprintf("wlock(%d)\n", n))
			}
			for i := 0; i < 100; i++ {
			}
			if firstRead != *incr{
				panic("rwt lock priority violated")
			}
			*incr = *incr+1
			atomic.AddInt32(activity, -10001)
			rwm.RTWUpgradeUnlock()
		
		}else{
			for i := 0; i < 100; i++ {
			}
			atomic.AddInt32(activity, -1)
			rwm.RTWUnlock()
		}
	}
	cdone <- true
}


func HammerRTWMutex(gomaxprocs, numReaders, num_iterations int) {
	runtime.GOMAXPROCS(gomaxprocs)
	// Number of active readers + 10000 * number of active writers.
	var activity, incr int32
	rwm := new()
	cdone := make(chan bool)
	go writer2(rwm, num_iterations, &activity, &incr, cdone)
	var i int
	for i = 0; i < numReaders/2; i++ {
		go reader(rwm, num_iterations, &activity, cdone)
	}
	go writer(rwm, num_iterations, &activity, cdone)
	for ; i < numReaders; i++ {
		go readtowriter(rwm, num_iterations, &activity, &incr, cdone)
	}
	// Wait for the 2 writers and all readers to finish.
	for i := 0; i < 2+numReaders; i++ {
		<-cdone
	}
}

func TestRTWMutexHammer(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(-1))
	n := 1000
	if testing.Short() {
		n = 5
	}
 	HammerRTWMutex(1, 1, n)
	HammerRTWMutex(1, 3, n)
	HammerRTWMutex(1, 10, n)
	HammerRTWMutex(4, 1, n)
	HammerRTWMutex(4, 3, n)
	HammerRTWMutex(4, 10, n)
	HammerRTWMutex(10, 1, n)
	HammerRTWMutex(10, 3, n)
	HammerRTWMutex(10, 10, n)
	HammerRTWMutex(10, 5, n)
	HammerRTWMutex(100, 5, n)
	HammerRTWMutex(1000, 5, n)
	HammerRTWMutex(1000, 50, n)
	HammerRTWMutex(100, 100, n)
}

func TestRTWMutexStress(t *testing.T){
	n := 100
	for i := 0; i < n; i++ {
		TestRTWMutexHammer(t)	
	}
}