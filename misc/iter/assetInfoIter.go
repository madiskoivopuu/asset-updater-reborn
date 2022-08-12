package iter

import "sync"

type AssetInfo struct {
	AssetID   int
	AssetName string
}

type AssetInfoIter struct {
	List *[]AssetInfo
	iter int
	m    sync.Mutex
}

func (customIter *AssetInfoIter) GetNext() AssetInfo {
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
