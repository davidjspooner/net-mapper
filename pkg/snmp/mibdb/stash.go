package mibdb

import "sync"

var lock sync.RWMutex

type Stash map[string]any

func (stash *Stash) Get(name string) any {
	lock.RLock()
	defer lock.RUnlock()
	if stash == nil || *stash == nil {
		return nil
	}
	return (*stash)[name]
}

func (stash *Stash) Set(name string, i any) {
	lock.Lock()
	defer lock.Unlock()
	if *stash == nil {
		*stash = make(map[string]any)
	}
	(*stash)[name] = i
}
