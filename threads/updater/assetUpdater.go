package updater

import (
	"fmt"
	"math/rand"
	"net/url"
	"projects/Vortex-Asset-Updater-reborn/managers"
	"projects/Vortex-Asset-Updater-reborn/misc/iter"
	"projects/Vortex-Asset-Updater-reborn/settings"
	"projects/Vortex-Asset-Updater-reborn/threads/catalog"
	pricescheduler "projects/Vortex-Asset-Updater-reborn/threads/price-scheduler"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/SparklyCatTF2/Roblox-go/rblx"
	regen "github.com/zach-klippenstein/goregen"
)

var prefix = "[UpdateThread]"
var rateLimitedAccountCookies = []string{}
var rlCookiesMutex = sync.Mutex{} // TODO: make this variable local to each thread

type UpdateStats struct {
	DevelopAPIRateLimited bool
	PriceAPIRateLimited   bool
	InvalidCookieErrors   int64
}

func IsAccountBlocked(accSession *managers.AccountSession) bool {
	rlCookiesMutex.Lock()
	defer rlCookiesMutex.Unlock()
	for _, cookie := range rateLimitedAccountCookies {
		if cookie == accSession.RblxSession.Cookie {
			return true
		}
	}
	return false
}

func BlockAccountSessionForXSeconds(accSession *managers.AccountSession, seconds int) {
	rlCookiesMutex.Lock()
	rateLimitedAccountCookies = append(rateLimitedAccountCookies, accSession.RblxSession.Cookie)
	rlCookiesMutex.Unlock()

	time.Sleep(time.Duration(seconds) * time.Second)

	rlCookiesMutex.Lock()
	for i, cookie := range rateLimitedAccountCookies {
		if cookie == accSession.RblxSession.Cookie {
			rateLimitedAccountCookies = append(rateLimitedAccountCookies[:i], rateLimitedAccountCookies[i+1:]...)
			break
		}
	}
	rlCookiesMutex.Unlock()
}

// Fetches assets that haven't been moderated from a group
func GetValidAssetsForGroup(groupId int, assetType string, catalogProxies []string) []iter.AssetInfo {
	cursor := "--"
	var proxy *url.URL
	groupIDstr := strconv.Itoa(groupId)
	assetInfo := make([]iter.AssetInfo, 0, settings.MaxAssetsToParse)
	proxyIter := iter.CustomStringIterator{List: &catalogProxies}
	RblxSession := rblx.RBLXSession{Client: managers.InitHttpClientForAccount()}

	fmt.Printf("%s(%s) Fetching assets for group %d.\n", prefix, assetType, groupId)

	for cursor != "" && len(assetInfo) < settings.MaxAssetsToParse {
		if len(catalogProxies) != 0 {
			proxy = PickProxy(&proxyIter)
		} else {
			proxy = nil
		}
		if cursor == "--" { // reset the cursor for this do...while loop nonsense
			cursor = ""
		}

		catalogSearch, searchErr := RblxSession.SearchCatalog("Clothing", assetType, groupIDstr, "Group", cursor, "100", "Desc", "3", proxy)
		if searchErr != nil { // try 2 more times
			for i := 0; i < 2; i++ {
				catalogSearch, searchErr = RblxSession.SearchCatalog("Clothing", assetType, groupIDstr, "Group", cursor, "100", "Desc", "3", proxy)
				if searchErr == nil {
					break
				}
			}

			if searchErr != nil {
				fmt.Printf("%s(%s) Failed to fetch catalog page after 3 tries, continuing with %d assets. Error: %s\n", prefix, assetType, len(assetInfo), searchErr.Error())
				break
			}
		}
		cursor = catalogSearch.NextPageCursor

		if len(catalogSearch.Data) == 0 {
			break
		}

		thumbnailAssetIDs := make([]int, 0, len(catalogSearch.Data))
		for _, itemData := range catalogSearch.Data {
			thumbnailAssetIDs = append(thumbnailAssetIDs, itemData.ID)
		}

		thumbnailsData, thumbnailErr := RblxSession.GetThumbnailsBatch(thumbnailAssetIDs, proxy)
		if thumbnailErr != nil { // try 2 more times
			for i := 0; i < 2; i++ {
				thumbnailsData, thumbnailErr = RblxSession.GetThumbnailsBatch(thumbnailAssetIDs, proxy)
				if thumbnailErr == nil {
					break
				}
			}
		}
		if thumbnailErr != nil {
			fmt.Printf("%s(%s) Failed to fetch thumbnails for a catalog page, skipping all assets on this page. Error: %s\n", prefix, assetType, thumbnailErr.Error())
			continue
		}

		// get catalog details & fetch names for assets
		catalogDetails, detailsError := RblxSession.GetCatalogDetails(thumbnailAssetIDs, proxy)
		if detailsError != nil { // try 2 more times
			for i := 0; i < 2; i++ {
				catalogDetails, detailsError = RblxSession.GetCatalogDetails(thumbnailAssetIDs, proxy)
				if detailsError == nil {
					break
				}
			}

			if detailsError != nil {
				fmt.Printf("%s(%s) Failed to fetch catalog details after 3 tries, skipping current page. Error: %s\n", prefix, assetType, detailsError.Error())
				continue
			}
		}

		if len(catalogDetails.Data) == 0 {
			fmt.Printf("%s(%s) Fetched catalog details but got 0 items in the response, skipping current page. Error: %s\n", prefix, assetType, detailsError.Error())
			continue
		}
		assetNamesMap := map[int]string{}
		for _, item := range catalogDetails.Data {
			assetNamesMap[item.ID] = item.Name
		}

		tempAssetIDs := make([]iter.AssetInfo, 0, len(catalogDetails.Data)) // temporarily store ids in case one of them is pending, we wont be adding them to the main asset ids slice
		for _, itemThumbnailData := range thumbnailsData.Data {
			if (len(assetInfo) + len(tempAssetIDs)) >= settings.MaxAssetsToParse {
				break
			}

			if itemThumbnailData.TargetID < settings.MinParsedAssetID {
				continue
			}
			if settings.CatalogIgnoreThumbnailStatus {
				tempAssetIDs = append(tempAssetIDs, iter.AssetInfo{AssetID: itemThumbnailData.TargetID, AssetName: assetNamesMap[itemThumbnailData.TargetID]})
				continue
			}

			if itemThumbnailData.State == "Blocked" || itemThumbnailData.State == "Pending" {
				continue
			}
			if itemThumbnailData.State == "Completed" {
				tempAssetIDs = append(tempAssetIDs, iter.AssetInfo{AssetID: itemThumbnailData.TargetID, AssetName: assetNamesMap[itemThumbnailData.TargetID]})
				continue
			}
			fmt.Printf("%s(%s) Unknown thumbnail response for %d. Error code: %d | Error message: %s | State: %s\n", prefix, assetType, itemThumbnailData.TargetID, itemThumbnailData.ErrorCode, itemThumbnailData.ErrorMessage, itemThumbnailData.State)
		}

		assetInfo = append(assetInfo, tempAssetIDs...)
	}

	fmt.Printf("%s(%s) Fetched a total of %d assets for group %d.\n", prefix, assetType, len(assetInfo), groupId)
	return assetInfo
}

func PickProxy(proxyIter *iter.CustomStringIterator) *url.URL {
	proxy := proxyIter.GetNext()
	parsedProxy, parseErr := url.Parse(proxy)
	if parseErr != nil {
		return nil
	}

	return parsedProxy
}

func ShouldUpdaterExit(accManager *managers.AccountManager) bool {
	if accManager.GetCookiesExpiredPastMinute() >= settings.MaxInvalidatedCookiesIn1Min ||
		accManager.GetCookiesExpiredTotal() >= settings.MaxInvalidatedCookies {
		return true
	}

	return false
}

func ProcessUpdateRequest(apiType string, assetType string, wg *sync.WaitGroup, proxyIter *iter.CustomStringIterator, updateStats *UpdateStats, assetInfo *iter.AssetInfo, accountSession *managers.AccountSession) {
	var reqSuccess bool
	var err *rblx.Error
	var apiSuffix string

	// get a random proxy for account if needed
	proxyURL := PickProxy(proxyIter)
	if proxyURL == nil || !settings.RotateProxies {
		proxyURL = accountSession.DefaultProxy
	}

	switch apiType {
	case "DescriptionAPI":
		apiSuffix = "DA"

		randomStr, _ := regen.Generate("[a-zA-Z0-9]{2}")
		newDescription := fmt.Sprintf("%s %s", settings.AssetDescription, randomStr)

		reqSuccess, err = accountSession.BoostAssetByUpdating(assetInfo.AssetID, assetInfo.AssetName, newDescription, proxyURL)
	case "PriceAPI":
		apiSuffix = "PA"
		// TODO: price scheduling system
		semiRandomPrice := pricescheduler.GetPriceForAssetID(assetInfo.AssetID) + rand.Intn(12)
		reqSuccess, err = accountSession.BoostAssetByChangingPrice(assetInfo.AssetID, semiRandomPrice, proxyURL)
	}

	if !reqSuccess {
		switch err.Type {
		case rblx.TooManyRequests:

			switch apiType {
			case "DescriptionAPI":
				updateStats.DevelopAPIRateLimited = true
			case "PriceAPI":
				updateStats.PriceAPIRateLimited = true
			}
			fmt.Printf("%s(%s) Failed to update %d due to rate limit [%s].\n", prefix, assetType, assetInfo.AssetID, apiSuffix)
		case rblx.AuthorizationDenied:
			fmt.Printf("%s(%s) Failed to update %d due to expired cookie [%s].\n", prefix, assetType, assetInfo.AssetID, apiSuffix)
			atomic.AddInt64(&updateStats.InvalidCookieErrors, 1)
		default:
			fmt.Printf("%s(%s) Failed to update %d. ErrType: %d | Error: %s [%s].\n", prefix, assetType, assetInfo.AssetID, err.Type, err.Error(), apiSuffix)
		}
	} else {
		fmt.Printf("%s(%s) Successfully updated %d [%s].\n", prefix, assetType, assetInfo.AssetID, apiSuffix)
	}

	wg.Done()
}

func UpdateThread(assetType string, accManager *managers.AccountManager, proxies []string, catalogProxies []string) {
	fmt.Printf("%s(%s) Starting asset updater.\n", prefix, assetType)

	validAssets := GetValidAssetsForGroup(settings.GroupID, assetType, catalogProxies)
	catalogObserver := &catalog.CatalogUpdateObserver{BottingTimeNotifier: make(chan int)}
	go catalogObserver.StartObservingCatalog(assetType, catalogProxies)

	// add asset ids to price scheduler
	for _, assetInfo := range validAssets {
		pricescheduler.AddAssetIDToList(assetInfo.AssetID)
	}

	assetIterator := iter.AssetInfoIter{List: &validAssets}
	proxyIter := iter.CustomStringIterator{List: &proxies}

	sleepTimePerAssetMilliSeconds := int(1000 / settings.UpdatesPerSecond)
	fmt.Printf("%s(%s) Sleep time between update requests: %dms.\n", prefix, assetType, sleepTimePerAssetMilliSeconds)

	continueUpdating := false
	go func() {
		timer := time.NewTimer(0)
		for {
			updateForXSeconds := <-catalogObserver.BottingTimeNotifier

			if updateForXSeconds == 0 && continueUpdating {
				fmt.Printf("%s(%s) Updater received signal from catalog observer about catalog update, going to sleep mode.\n", prefix, assetType)

				timer.Stop()
				continueUpdating = false
				continue
			}

			if updateForXSeconds != 0 {
				fmt.Printf("%s(%s) Updater received signal to start botting.\n", prefix, assetType)
				continueUpdating = true

				updatesPerSec := len(accManager.AccountSessions) * accManager.TotalRateLimitsForAccount() / int(1.1*float64(updateForXSeconds))
				sleepTimePerAssetMilliSeconds = int(1000 / updatesPerSec)

				timer = time.AfterFunc(time.Duration(updateForXSeconds)*time.Second, func() {
					fmt.Printf("%s(%s) Updater stop timer was called, going to sleep mode.\n", prefix, assetType)
					continueUpdating = false
				})
			}
		}
	}()

	for {
		// restart observer if needed
		if !catalogObserver.IsObserverRunning() {
			catalogObserver = &catalog.CatalogUpdateObserver{BottingTimeNotifier: catalogObserver.BottingTimeNotifier}
			catalogObserver.BottingTimeNotifier <- 0
			go catalogObserver.StartObservingCatalog(assetType, catalogProxies)
		}

		if !continueUpdating {
			time.Sleep(1 * time.Millisecond)
			continue
		}

		accountSession := accManager.GetNextAccount()
		wg := sync.WaitGroup{}
		updateStats := UpdateStats{}

		if IsAccountBlocked(accountSession) {
			continue
		}

		assetInfo := assetIterator.GetNext()
		for _, apiType := range settings.ApisToUseForUpdating { // loop through different apis one at a time and create goroutine for each update request
			if ShouldUpdaterExit(accManager) {
				fmt.Printf("%s(%s) Updater cannot continue, stopping.\n", prefix, assetType)
				return
			}
			if !continueUpdating {
				goto OUTMOST_LOOP
			}

			wg.Add(1)
			go ProcessUpdateRequest(
				apiType,
				assetType,
				&wg,
				&proxyIter,
				&updateStats,
				&assetInfo,
				accountSession,
			)
			time.Sleep(time.Duration(sleepTimePerAssetMilliSeconds) * time.Millisecond)
		}

		// wait until everything is done and check if any of the apis are rate limited for this account
		go func(accountSession *managers.AccountSession, wg *sync.WaitGroup, updateStats *UpdateStats) {
			wg.Wait()

			accountSession.RblxSession.Client.CloseIdleConnections()
			if updateStats.DevelopAPIRateLimited || updateStats.PriceAPIRateLimited {
				BlockAccountSessionForXSeconds(accountSession, 65)
			}
			// fuck this piece of code for now
			// TODO: somehow rework the invalid cookie system
			/*if updateData.InvalidCookieErrors >= int64(accManager.TotalRateLimitsForAccount()/2) {
				accManager.ReportInvalidCookie(accountSession)
			}*/
		}(accountSession, &wg, &updateStats)

	OUTMOST_LOOP:
		time.Sleep(time.Duration(sleepTimePerAssetMilliSeconds) * time.Millisecond)
	}
}

func StartUpdating(accounts, proxies, catalogProxies []string) {
	assetTypeCount := len(settings.AssetTypesToBot)
	accountsPerThread, catalogProxiesPerThread := int(len(accounts)/assetTypeCount), int(len(catalogProxies)/assetTypeCount)
	currentAccountIdx, currentCatalogProxyIdx := 0, 0

	for _, assetType := range settings.AssetTypesToBot {
		accManager := &managers.AccountManager{}
		accManager.LoadAccounts(accounts[currentAccountIdx:currentAccountIdx+accountsPerThread], proxies[currentAccountIdx:currentAccountIdx+accountsPerThread])

		go UpdateThread(assetType, accManager, proxies[currentAccountIdx:currentAccountIdx+accountsPerThread], catalogProxies[currentCatalogProxyIdx:currentCatalogProxyIdx+catalogProxiesPerThread])

		currentAccountIdx += accountsPerThread
		currentCatalogProxyIdx += catalogProxiesPerThread // why are my fucking var names so long
	}
}
