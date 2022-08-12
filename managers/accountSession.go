package managers

import (
	"math/rand"
	"net/url"

	"github.com/SparklyCatTF2/Roblox-go/rblx"
)

type AccountSession struct {
	DefaultProxy *url.URL
	RblxSession  *rblx.RBLXSession
}

type AccRateLimit struct {
	PriceAPI       int
	DescriptionAPI int
}

func (account *AccountSession) BoostAssetByUpdating(assetId int, name string, description string, proxyURL *url.URL) (bool, *rblx.Error) {
	genres := []string{"All", "Tutorial", "Scary", "TownAndCity", "War", "Funny", "Fantasy", "Adventure", "SciFi", "Pirate", "FPS", "RPG", "Sports", "Ninja", "WildWest"}
	randomGenre := genres[rand.Intn(len(genres))]

	// [DISABLED] sadly we have to create a brand new temporary RBLXSession due to the request content (along w the proxy) staying the same through multiple reqs
	//newRblxSess := rblx.RBLXSession{Cookie: account.RblxSession.Cookie, XCSRFToken: account.RblxSession.XCSRFToken, Client: InitHttpClientForAccount()}

	_, updateErr := account.RblxSession.UpdateAsset(assetId, name, description, randomGenre, proxyURL)
	if updateErr != nil && updateErr.Type == rblx.TokenValidation {
		_, updateErr = account.RblxSession.UpdateAsset(assetId, name, description, randomGenre, proxyURL)
	}

	if updateErr != nil {
		if updateErr.Type == rblx.AuthorizationDenied {
			return false, updateErr
		}

		if updateErr.Type == rblx.TooManyRequests {
			return false, updateErr
		}

		return false, updateErr
	}
	return true, nil
}

func (account *AccountSession) BoostAssetByChangingPrice(assetId int, price int, proxyURL *url.URL) (bool, *rblx.Error) {
	// [DISABLED] sadly we have to create a brand new temporary RBLXSession due to the request content (along w the proxy) staying the same through multiple reqs
	//newRblxSess := rblx.RBLXSession{Cookie: account.RblxSession.Cookie, XCSRFToken: account.RblxSession.XCSRFToken, Client: InitHttpClientForAccount()}

	_, updateErr := account.RblxSession.UpdateAssetPrice(assetId, price, proxyURL)
	if updateErr != nil && updateErr.Type == rblx.TokenValidation {
		_, updateErr = account.RblxSession.UpdateAssetPrice(assetId, price, proxyURL)
	}

	if updateErr != nil {
		if updateErr.Type == rblx.AuthorizationDenied {
			return false, updateErr
		}

		if updateErr.Type == rblx.TooManyRequests {
			return false, updateErr
		}

		return false, updateErr
	}
	return true, nil
}
