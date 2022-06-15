package rtwmutex

// IRTWMutex is an extended version of classic RWMutex that lets you acquire a read-to-write lock which could be
// upgraded to an exclusive write lock when needed.
type IRTWMutex interface{
	RTWLock()
	RTWUnlock()
	Upgrade()
	RTWUpgradeUnlock()

	RLock()
	RUnlock()

	Lock()
	Unlock()
}