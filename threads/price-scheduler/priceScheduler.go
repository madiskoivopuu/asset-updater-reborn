package pricescheduler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	timeparser "projects/Vortex-Asset-Updater-reborn/misc/time-parser"
	"projects/Vortex-Asset-Updater-reborn/settings"
	"time"
)

type SchedulerPricepoint struct {
	TimeOffset      string `json:"timeOffset"`
	Price           int    `json:"price"`
	parsedTimeDelta int
}
type SchedulerSettings struct {
	Pricepoints      []SchedulerPricepoint     `json:"pricepoints"`
	RatiosAndOffsets []SchedulerRatioAndOffset `json:"ratiosAndOffsets"`
	totalRatio       int
	AnchorTime       int `json:"anchorTime"`
}
type SchedulerRatioAndOffset struct {
	Ratio  int `json:"ratio"`
	Offset int `json:"offset"`
}

var prefix = "[PriceScheduler]"

// sheduler state
var priceForAssetID = map[int]int{}
var assetIdCount = 0

var priceSettings = &SchedulerSettings{}
var currentTime = time.Now().UTC()
var anchorTime = time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 1, currentTime.Location())
var currentPricepointIdx = 0
var isThreadRunning = false

// Calculates the index within the bounds of an array for cases when the index is outside
func calculateIndexSafe(index, length int) int {
	return (index%length + length) % length // we have to do this weirdness to replicate python's modulo -1%5 == 4
}

func initializePricepointIndex() int {
	// figure out the current pricepoint based on time & anchoring time (1st pricepoint time)
	currentTime = time.Now().UTC()
	currentTimeUnix := currentTime.Unix()
	timeDelta := currentTime.Unix() - anchorTime.Unix()

	pricepointsLen := len(priceSettings.Pricepoints)
	tempAnchorTime := anchorTime.Unix()
	for {
		currPricePoint := priceSettings.Pricepoints[calculateIndexSafe(currentPricepointIdx, pricepointsLen-1)] // pricepointsLen-1 since last pricepoint is not considered valid
		nextPricePoint := priceSettings.Pricepoints[calculateIndexSafe(currentPricepointIdx+1, pricepointsLen-1)]

		pricepointTimeDelta := nextPricePoint.parsedTimeDelta - currPricePoint.parsedTimeDelta
		// we don't want to continue if we subtract last element time from first
		if pricepointTimeDelta < 0 {
			if timeDelta < 0 {
				currentPricepointIdx--
			} else {
				currentPricepointIdx++
			}
			continue
		}

		// if the time is in the past, we subtract pptimedelta from the temporary anchor and then redo
		// until current time goes between the anchor and anchor-pptimedelta
		// if the time is after anchor time, we do the exact opposite (adding)
		if timeDelta < 0 {
			if (tempAnchorTime-int64(pricepointTimeDelta)) < currentTimeUnix && currentTimeUnix < tempAnchorTime { // correct index found
				tempAnchorTime -= int64(pricepointTimeDelta)
				break
			}
			tempAnchorTime -= int64(pricepointTimeDelta)
			currentPricepointIdx--
		} else {
			if tempAnchorTime < currentTimeUnix && (tempAnchorTime+int64(pricepointTimeDelta)) > currentTimeUnix {
				break
			}
			tempAnchorTime += int64(pricepointTimeDelta)
			currentPricepointIdx++
		}
	}

	// since we repeatedly added to currentPricepointIdx, we will be adjusting it to be within the bounds of array
	currentPricepointIdx = calculateIndexSafe(currentPricepointIdx, pricepointsLen-1)

	return int(tempAnchorTime)
}

func updateAssetPrices() {
	if !isThreadRunning || assetIdCount == 0 {
		return
	}

	currRatioIdx := 0
	currItemIdx := 0
	updateUntilItemIdx := int(math.Round((float64(priceSettings.RatiosAndOffsets[currRatioIdx].Ratio) / float64(priceSettings.totalRatio)) * float64(assetIdCount)))
	for assetID := range priceForAssetID {
		if currItemIdx >= updateUntilItemIdx {
			currRatioIdx++
			updateUntilItemIdx += int(math.Round((float64(priceSettings.RatiosAndOffsets[currRatioIdx].Ratio) / float64(priceSettings.totalRatio)) * float64(assetIdCount)))
		}

		currentPricepointOffset := priceSettings.RatiosAndOffsets[currRatioIdx].Offset
		pricepointIdx := calculateIndexSafe(currentPricepointIdx+currentPricepointOffset, len(priceSettings.Pricepoints)-1) // len-1 because the last element and first element are considered same pricepoints

		priceForAssetID[assetID] = priceSettings.Pricepoints[pricepointIdx].Price
		currItemIdx++
	}
}

// Completely load the settings & format pricepoints to int
func loadSettings() bool {
	settingsFile, err := ioutil.ReadFile("./Configuration/pricescheduler.json")
	if err != nil {
		fmt.Printf("%s Failed to load configuration file. %s\n", prefix, err.Error())
		return false
	}

	unmarshalErr := json.Unmarshal(settingsFile, priceSettings)
	if unmarshalErr != nil {
		fmt.Printf("%s Failed to parse json from config. %s\n", prefix, unmarshalErr.Error())
		return false
	}

	for i := range priceSettings.Pricepoints {
		priceSettings.Pricepoints[i].parsedTimeDelta = timeparser.ParseTimeStringToTimeDeltaSeconds(priceSettings.Pricepoints[i].TimeOffset)
	}

	// if we have a custom anchor time set then overwrite that
	if priceSettings.AnchorTime > 0 {
		if priceSettings.AnchorTime <= 31 {
			anchorTime = time.Date(currentTime.Year(), currentTime.Month(), priceSettings.AnchorTime, 0, 0, 0, 1, currentTime.Location())
		} else {
			anchorTime = time.Unix(int64(priceSettings.AnchorTime), 0).UTC()
		}
	}

	// add up the ratios for updateAssetPrices() use
	for _, item := range priceSettings.RatiosAndOffsets {
		priceSettings.totalRatio += item.Ratio
	}

	return true
}

func GetPriceForAssetID(assetID int) int {
	price, keyFound := priceForAssetID[assetID]
	if !keyFound {
		return settings.AssetPrice
	}

	return price
}

func AddAssetIDToList(assetID int) {
	if _, exists := priceForAssetID[assetID]; !exists {
		assetIdCount += 1
	}
	priceForAssetID[assetID] = settings.AssetPrice

	updateAssetPrices()
}

func StartSchedulingPrice() {
	if isThreadRunning {
		return
	}
	isThreadRunning = true

	if !loadSettings() {
		isThreadRunning = false
		return
	}

	prevPriceTime := initializePricepointIndex()
	nextPriceTime := 0

	for {
		time.Sleep(1 * time.Second)
		currentTime = time.Now().UTC()

		nextPricepointIdx := calculateIndexSafe(currentPricepointIdx+1, len(priceSettings.Pricepoints))
		// edge case: the time cycle has wrapped around to the end, and since we treat the first and last element as one
		// it would bug out the code because the parsedTimeDelta subtraction would return a negative and nextPriceTime would actually be in the past, bugging out the code
		// thus we have to manually reset the cycle with this check
		if nextPricepointIdx == 0 && currentPricepointIdx == len(priceSettings.Pricepoints)-1 {
			prevPriceTime = nextPriceTime
			currentPricepointIdx = 0

			updateAssetPrices()
			continue
		}

		nextPriceTime = prevPriceTime + (priceSettings.Pricepoints[nextPricepointIdx].parsedTimeDelta - priceSettings.Pricepoints[currentPricepointIdx].parsedTimeDelta)
		// price doesn't need to change
		if int64(prevPriceTime) < currentTime.Unix() && currentTime.Unix() < int64(nextPriceTime) {
			continue
		}

		currentPricepointIdx++
		updateAssetPrices()
		prevPriceTime = nextPriceTime
	}
}
