package timeparser

import (
	"regexp"
	"strconv"
)

var timeTypeToSeconds = map[string]int{
	"s":   1,
	"min": 60,
	"h":   3600,
	"d":   86400,
	"w":   604800,
	"m":   2628000,  // approx, 28-31 days in month
	"y":   31540000, // approx, leap year uncounted
}
var digitCheck = regexp.MustCompile(`^[0-9]+$`)

// addTimeDataIfFullyParsed checks whether the current token has been fully parsed. Returns true when it has been parsed & added to the time list
func addTimeDataIfFullyParsed(char string, timeType string, time string, timeTypesList *[]string, timeList *[]int) bool {
	if digitCheck.MatchString(char) && len(timeType) != 0 {
		timeInt, err := strconv.Atoi(time)
		if err != nil {
			panic("Failed to convert time before a token to int")
		}

		*timeList = append(*timeList, timeInt)
		*timeTypesList = append(*timeTypesList, timeType)

		return true
	}

	return false
}

// Parses a time in format 1y1m1w1d1h1min1s in any combination to a time delta
func ParseTimeStringToTimeDeltaSeconds(timeStr string) int {
	timeDelta := 0

	timeTypes := []string{} // s, min, h ...
	timesBeforeType := []int{}
	currTimeType := ""
	currTimeBeforeType := ""
	for i := 0; i < len(timeStr); i++ {
		currChar := string(timeStr[i])
		if addTimeDataIfFullyParsed(currChar, currTimeType, currTimeBeforeType, &timeTypes, &timesBeforeType) {
			currTimeType = ""
			currTimeBeforeType = ""
		}

		if digitCheck.MatchString(currChar) {
			currTimeBeforeType += currChar
		} else {
			currTimeType += currChar
		}
	}
	addTimeDataIfFullyParsed("1", currTimeType, currTimeBeforeType, &timeTypes, &timesBeforeType)

	// loop over times and add them together (correctly converting everything to seconds)
	for i := range timesBeforeType {
		timeDelta += timesBeforeType[i] * timeTypeToSeconds[timeTypes[i]]
	}

	return timeDelta
}
