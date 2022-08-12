package settings

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

var UpdatesPerSecond = 65
var ConnectionTimeout = 12
var CatalogObserverConnectionTimeout = 4
var RotateProxies = false
var KeepConnectionAlive = true
var IdleConnectionTimeoutInSeconds = 660
var UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.63 Safari/537.36"

var ApisToUseForUpdating = []string{"DescriptionAPI", "PriceAPI"}
var DescriptionAPIRateLimit = 8
var MaxInvalidatedCookies = 100
var MaxInvalidatedCookiesIn1Min = 30

var AssetTypesToBot = []string{"Clothing", "ClassicShirts", "ClassicPants"}
var MaxAssetsToParse = 800
var MinParsedAssetID = 9000000000
var GroupID = 5751082
var AssetPrice = 499
var AssetDescription = "Like what you see? Join the group for more ✨𝑷𝑹𝑬𝑴𝑰𝑼𝑴✨ clothing: https://www.roblox.com/groups/5751082/meemus-in-pajamas-2\n cute cool kawaii kitty cherry anime japanese vamp emo gloomy blossom sad goth baddie cutie y2k cyber swag drip limited aesthetic lovely hot trendy stylish classic popular old vintage exquisite exclusive cheap sale beautiful prestigeous villain hero supreme nike adidas gucci off white original fashion king rp roleplay marshmello alan walker deadmau5 david guetta martin ksi garrix paul zara lantern festival festivity celebration mardi gras st saint patrick halloween thanksgiving father christmas xmas boxing santa summer sunny beach holiday vacation red ruby pink black blue gold diamond glowy denim grunge ripped trim checkered plaid rainbow colourful bikini baggy pastel casual crop top jeans shoes gyaru girl pants shirt logan"
var CatalogIgnoreThumbnailStatus = false

type SettingsJSON struct {
	AssetTypesToBot              []string `json:"assetTypesToBot"`
	MaxAssetsToParse             int      `json:"maxAssetsToParse"`
	GroupID                      int      `json:"groupId"`
	AssetPrice                   int      `json:"assetPrice"`
	AssetDescription             string   `json:"assetDescription"`
	MinParsedAssetID             int      `json:"minParsedAssetID"`
	CatalogIgnoreThumbnailStatus bool     `json:"catalogIgnoreThumbnailStatus"`
}

func LoadAssetSettingsToVariables(filename string) {
	settingsFromFile, _ := ioutil.ReadFile(filename)

	settingsJson := SettingsJSON{}
	unmarshalErr := json.Unmarshal(settingsFromFile, &settingsJson)
	if unmarshalErr != nil {
		fmt.Printf("Error (%s) loading settings file! Using default settings.\n", unmarshalErr.Error())
		return
	}

	AssetTypesToBot = settingsJson.AssetTypesToBot
	MaxAssetsToParse = settingsJson.MaxAssetsToParse
	GroupID = settingsJson.GroupID
	AssetPrice = settingsJson.AssetPrice
	AssetDescription = settingsJson.AssetDescription
	MinParsedAssetID = settingsJson.MinParsedAssetID
	CatalogIgnoreThumbnailStatus = settingsJson.CatalogIgnoreThumbnailStatus
}
