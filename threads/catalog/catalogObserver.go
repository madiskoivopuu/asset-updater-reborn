package catalog

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"projects/Vortex-Asset-Updater-reborn/managers"
	"projects/Vortex-Asset-Updater-reborn/misc/iter"
	"projects/Vortex-Asset-Updater-reborn/settings"

	"github.com/SparklyCatTF2/Roblox-go/rblx"
)

var MaxCatalogErrorsInARow = 50
var observerPrefix = "[CatalogObserver]"

type CatalogUpdateObserver struct {
	catalogStr          string
	timesBetweenUpdates []int
	tbuMutex            sync.Mutex
	lastUpdateTime      int
	errorsInARow        int
	BottingTimeNotifier chan int
	stopped             bool
}

func GetTimeInSeconds() int {
	return int(time.Now().Unix())
}

func MinMax(array []int) (int, int) {
	var max int = array[0]
	var min int = array[0]
	for _, value := range array {
		if max < value {
			max = value
		}
		if min > value {
			min = value
		}
	}
	return min, max
}

func (observer *CatalogUpdateObserver) GetCatalogUpdateTimeAmplitudeInSeconds() int {
	if len(observer.timesBetweenUpdates) < 5 {
		return 15
	}

	observer.tbuMutex.Lock()
	defer observer.tbuMutex.Unlock()

	min, max := MinMax(observer.timesBetweenUpdates)
	amplitude := max - min
	if amplitude < 15 {
		return 15
	} else {
		return amplitude
	}
}

func (observer *CatalogUpdateObserver) GetCatalogUpdateTimeAverageInSeconds() int {
	if len(observer.timesBetweenUpdates) < 5 {
		return 60
	}

	observer.tbuMutex.Lock()
	defer observer.tbuMutex.Unlock()

	sum := 0
	for _, updateTime := range observer.timesBetweenUpdates {
		sum += updateTime
	}

	return sum / len(observer.timesBetweenUpdates)
}

func (observer *CatalogUpdateObserver) AddCatalogUpdateTime(timeInSeconds int) {
	observer.tbuMutex.Lock()

	observer.timesBetweenUpdates = append(observer.timesBetweenUpdates, timeInSeconds)
	if len(observer.timesBetweenUpdates) > 25 {
		observer.timesBetweenUpdates = observer.timesBetweenUpdates[1:]
	}

	observer.tbuMutex.Unlock()
}

func (observer *CatalogUpdateObserver) FetchCatalog(assetType, proxy string) string {
	parsedProxy, parseErr := url.Parse(proxy)
	if parseErr != nil {
		return ""
	}
	if proxy == "" {
		parsedProxy = nil
	}

	rblxSess := &rblx.RBLXSession{Client: managers.InitHttpClientForAccount()}
	rblxSess.Client.Timeout = time.Duration(settings.CatalogObserverConnectionTimeout) * time.Second

	catalogResp, catalogErr := rblxSess.SearchCatalog("Clothing", assetType, "", "", "", "10", "", "3", parsedProxy)
	if catalogErr != nil {
		return ""
	}

	catalogRespStr, marshalErr := json.Marshal(catalogResp)
	if marshalErr != nil {
		return ""
	}

	return string(catalogRespStr)
}

func (observer *CatalogUpdateObserver) IsObserverRunning() bool {
	return !observer.stopped
}

func (observer *CatalogUpdateObserver) StartObservingCatalog(assetType string, proxies []string) {
	l := []string{""}
	proxyIter := iter.CustomStringIterator{List: &l}
	if len(proxies) != 0 {
		proxyIter = iter.CustomStringIterator{List: &proxies}
	}

	// initial catalog fetch
	for {
		if observer.errorsInARow >= MaxCatalogErrorsInARow {
			fmt.Printf("%s(%s) Too many errors trying to fetch the catalog for the first time, stopping observer. This also means the updater will not work.\n", observerPrefix, assetType)
			observer.stopped = true
			return
		}

		catalogStr := observer.FetchCatalog(assetType, proxyIter.GetNext())
		if catalogStr != "" {
			observer.catalogStr = catalogStr
			break
		}

		observer.errorsInARow += 1
	}
	observer.errorsInARow = 0

	// continuous fetch
	firstUpdate := true
	timer := time.NewTimer(0)
	for observer.errorsInARow < MaxCatalogErrorsInARow {
		proxy := proxyIter.GetNext()
		catalogStr := observer.FetchCatalog(assetType, proxy)
		if catalogStr == "" {
			observer.errorsInARow += 1
			fmt.Printf("%s(%s) Failed to fetch catalog with %s, total failures in a row: %d.\n", observerPrefix, assetType, proxy, observer.errorsInARow)
			continue
		}
		observer.errorsInARow = 0

		if catalogStr != observer.catalogStr {
			timer.Stop()
			observer.catalogStr = catalogStr
			observer.BottingTimeNotifier <- 0

			if !firstUpdate {
				timediff := GetTimeInSeconds() - observer.lastUpdateTime
				if timediff > 15 { // sanity check: only add timediff to list if its larger than 15s, no way the catalog updates in less than 15 seconds
					observer.AddCatalogUpdateTime(timediff)
				}
			} else {
				fmt.Printf("%s(%s) First catalog update, bot will start doing its thing in soon.\n", observerPrefix, assetType)
				firstUpdate = false
			}
			observer.lastUpdateTime = GetTimeInSeconds()

			avgUpdateTime, updateTimeAmplitude := observer.GetCatalogUpdateTimeAverageInSeconds(), observer.GetCatalogUpdateTimeAmplitudeInSeconds()
			fmt.Printf("%s(%s) Average update time: %ds | Average update time amplitude: %ds\n", observerPrefix, assetType, avgUpdateTime, updateTimeAmplitude)

			timer = time.AfterFunc(time.Duration(avgUpdateTime-(updateTimeAmplitude*2/3))*time.Second, func() {
				observer.BottingTimeNotifier <- updateTimeAmplitude
			})
		}
	}

	fmt.Printf("%s(%s) Too many errors trying to fetch the catalog, stopping observer. This also means the updater will not work.\n", observerPrefix, assetType)
	observer.stopped = true
}
