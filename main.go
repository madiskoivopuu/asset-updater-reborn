package main

import (
	"bufio"
	"math/rand"
	"os"
	"projects/Vortex-Asset-Updater-reborn/settings"
	pricescheduler "projects/Vortex-Asset-Updater-reborn/threads/price-scheduler"
	"projects/Vortex-Asset-Updater-reborn/threads/updater"
	"sync"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	accounts := []string{}
	proxies := []string{}
	catalogProxies := []string{}

	// Load proxies and accounts
	proxiesFile, _ := os.Open("./proxies.txt")
	scanner := bufio.NewScanner(proxiesFile)
	for scanner.Scan() {
		proxies = append(proxies, scanner.Text())
	}

	catalogProxiesFile, _ := os.Open("./catalog_proxies.txt")
	scanner = bufio.NewScanner(catalogProxiesFile)
	for scanner.Scan() {
		catalogProxies = append(catalogProxies, scanner.Text())
	}

	accountsFile, _ := os.Open("./accounts.txt")
	scanner = bufio.NewScanner(accountsFile)
	for scanner.Scan() {
		accounts = append(accounts, scanner.Text())
	}

	settings.LoadAssetSettingsToVariables("./Configuration/settings.json")

	go pricescheduler.StartSchedulingPrice()
	updater.StartUpdating(accounts, proxies, catalogProxies)

	wg := sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
