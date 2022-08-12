package iter

import "sync"

type CustomStringIterator struct {
	List *[]string
	iter int
	m    sync.Mutex
}

func (customIter *CustomStringIterator) GetNext() string {
	customIter.m.Lock()
	defer customIter.m.Unlock()
	if len(*customIter.List) == 0 {
		panic("CustomStringIterator: GetNext called but list size is 0")
	}
	if customIter.iter >= len(*customIter.List) {
		customIter.iter = 0
	}

	n := (*customIter.List)[customIter.iter]
	customIter.iter++
	return n
}
