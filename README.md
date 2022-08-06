# RTWMutex

RTWMutex is an extended version of classic sync.RWMutex that lets you acquire a read-to-write lock which could be upgraded to an exclusive write lock when needed.

## Usage
It implements below methods. ```RLock(), RUnlock(), Lock(), Unlock()``` methods works exactly the same way as ```sync.RWMutex```.
```go
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
```
* To acquire an upgradable read lock ```RTWLock()``` should be called. ```RTWLock``` can be released before upgrading by calling ```RTWUnlock```. 

* If an upgrade is required ```Upgrade()``` should be called. ```Upgrade``` should be called when you already have a ```RTWLock```. 

* Once ```Upgrade``` returns, it is an exlusive lock there is no other thread that has a read or write lock so that resource can be modifed.

* If an ```RTWLock``` is upgraded to release it ```RTWUpgradeUnlock()```should be called.

### RTW Lock Priority
* If an rtw lock is acquired and upgrade is called on it, it enters the lock when no pending readers is left even if a writer is pending since writer might change the state of the resource. 
* When an upgraded rtw lock is released, pending writer acquires the lock if there is any. (Because reader locks are already shared giving writers some priority seems more reasonable option.)

## License

MIT
