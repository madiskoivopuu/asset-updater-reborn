package managers

import (
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"projects/Vortex-Asset-Updater-reborn/settings"

	"github.com/SparklyCatTF2/Roblox-go/rblx"
)

var prefix string = "[AccountManager]"

type AccountManager struct {
	AccountSessions          []*AccountSession
	currentAccIndex          int
	proxyless                bool
	cookiesExpiredPastMinute int
	cookiesExpiredTotal      int
	listMutex                sync.Mutex
}

type AddHeaderTransport struct {
	T *http.Transport
}

var defaultTransport = &AddHeaderTransport{T: &http.Transport{Proxy: func(req *http.Request) (*url.URL, error) {
	currentProxy, _ := req.Context().Value("proxy").(*url.URL)
	return currentProxy, nil
}, MaxIdleConns: 3, IdleConnTimeout: time.Duration(settings.IdleConnectionTimeoutInSeconds) * time.Second, DisableKeepAlives: !settings.KeepConnectionAlive}}

func (adt *AddHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("User-Agent", settings.UserAgent)
	req.Header.Add("Sec-Fetch-Site", "same-site")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("Sec-Fetch-Dest", "empty")
	req.Header.Add("sec-ch-ua", "\" Not A;Brand\";v=\"99\", \"Chromium\";v=\"100\", \"Google Chrome\";v=\"100\"")
	req.Header.Add("sec-ch-ua-platform", "\"Windows\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("Accept", "application/json, text/plain, /*")
	req.Header.Add("Accept-Language", "en-GB,en;q=0.9")
	req.Header.Add("Referer", "https://www.roblox.com/")
	req.Header.Add("Origin", "https://www.roblox.com")
	return adt.T.RoundTrip(req)
}

func (accManager *AccountManager) GetCookiesExpiredTotal() int {
	return accManager.cookiesExpiredTotal
}

func (accManager *AccountManager) GetCookiesExpiredPastMinute() int {
	return accManager.cookiesExpiredPastMinute
}

func (accManager *AccountManager) ReportInvalidCookie(accountSession *AccountSession) {
	accManager.cookiesExpiredPastMinute += 1
	accManager.cookiesExpiredTotal += 1

	accManager.removeAccountSession(accountSession)
	go accManager.decrementCookieExpiredPastMinute()
}

func (accManager *AccountManager) decrementCookieExpiredPastMinute() {
	time.Sleep(60 * time.Second)
	accManager.cookiesExpiredPastMinute -= 1
}

func (accManager *AccountManager) removeAccountSession(accountSession *AccountSession) {
	accManager.listMutex.Lock()
	for i, accSessionInList := range accManager.AccountSessions {
		if accSessionInList == accountSession {
			accManager.AccountSessions = append(accManager.AccountSessions[:i], accManager.AccountSessions[i+1:]...)
			break
		}
	}
	accManager.listMutex.Unlock()
}

func (accManager *AccountManager) IsProxyless() bool {
	return accManager.proxyless
}

func (accManager *AccountManager) GetNextAccount() *AccountSession {
	accManager.listMutex.Lock()
	defer accManager.listMutex.Unlock()

	if len(accManager.AccountSessions) == 0 {
		return nil
	}
	if accManager.currentAccIndex >= len(accManager.AccountSessions) {
		accManager.currentAccIndex = 0
	}

	n := accManager.AccountSessions[accManager.currentAccIndex]
	accManager.currentAccIndex++
	return n
}

func InitHttpClientForAccount() *http.Client {
	client := &http.Client{Timeout: time.Duration(settings.ConnectionTimeout) * time.Second}
	client.Transport = &AddHeaderTransport{T: &http.Transport{Proxy: func(req *http.Request) (*url.URL, error) {
		currentProxy, _ := req.Context().Value("proxy").(*url.URL)
		return currentProxy, nil
	}, MaxIdleConns: 3, IdleConnTimeout: time.Duration(settings.IdleConnectionTimeoutInSeconds) * time.Second, DisableKeepAlives: !settings.KeepConnectionAlive}}

	return client
}

func (accManager *AccountManager) TotalRateLimitsForAccount() int {
	return settings.DescriptionAPIRateLimit
}

func (accManager *AccountManager) LoadAccounts(cookies []string, proxies []string) bool {
	if len(proxies) == 0 || len(proxies) < len(cookies) {
		accManager.proxyless = true
		fmt.Printf("%s No proxies provided or not enough proxies for all accounts, using proxyless mode.\n", prefix)
	}

	accManager.AccountSessions = make([]*AccountSession, len(cookies))
	for i := 0; i < len(cookies); i++ {
		accManager.AccountSessions[i] = &AccountSession{}
		accManager.AccountSessions[i].RblxSession = &rblx.RBLXSession{Cookie: cookies[i]}

		if !accManager.proxyless {
			parsedProxy, parseErr := url.Parse(proxies[i])
			if parseErr != nil {
				fmt.Printf("%s Failed to parse proxy, aborting: %s\n", prefix, parseErr.Error())
				return false
			}

			accManager.AccountSessions[i].DefaultProxy = parsedProxy
			accManager.AccountSessions[i].RblxSession.Client = InitHttpClientForAccount()
		} else {
			accManager.AccountSessions[i].RblxSession.Client = InitHttpClientForAccount()
		}
	}

	return true
}
