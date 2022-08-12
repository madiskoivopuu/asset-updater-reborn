package iter

import (
	"sync"
)

type CustomIntIterator struct {
	List *[]int
	iter int
	m    sync.Mutex
}

func (customIter *CustomIntIterator) GetNext() int {
	customIter.m.Lock()
	defer customIter.m.Unlock()
	if len(*customIter.List) == 0 {
		panic("CustomIntIterator: GetNext called but list size is 0")
	}
	if customIter.iter >= len(*customIter.List) {
		customIter.iter = 0
	}

	n := (*customIter.List)[customIter.iter]
	customIter.iter++
	return n
}
