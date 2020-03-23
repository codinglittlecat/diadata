package main

import (
	"github.com/diadata-org/diadata/internal/pkg/exchange-scrapers"
	_ "github.com/diadata-org/diadata/pkg/dia"
	_ "github.com/diadata-org/diadata/pkg/model"
	"sync"
	log "github.com/sirupsen/logrus"
)

// TODO: Read the instruments from DB (formerly: deribitOptionsMetaFilename)
// pairs contains all pairs currently supported by the DIA scrapers
var (
	instruments = []string{

/*		"BTC-27MAR20-6000-C",
		"BTC-24APR20-6000-C",
		"BTC-27MAR20-6000-P",
		"BTC-24APR20-6000-P",

		"BTC-27MAR20-7000-C",
		"BTC-24APR20-7000-C",
		"BTC-27MAR20-7000-P",
		"BTC-24APR20-7000-P",

		"BTC-27MAR20-8000-C",
		"BTC-24APR20-8000-C",
		"BTC-27MAR20-8000-P",
		"BTC-24APR20-8000-P",*/

		"BTC-27MAR20-9000-C",
		"BTC-24APR20-9000-C",
		"BTC-27MAR20-9000-P",
		"BTC-24APR20-9000-P",

		/*"BTC-27MAR20-10000-C",
		"BTC-24APR20-10000-C",
		"BTC-27MAR20-10000-P",
		"BTC-24APR20-10000-P",

		"BTC-27MAR20-11000-C",
		"BTC-24APR20-11000-C",
		"BTC-27MAR20-11000-P",
		"BTC-24APR20-11000-P",

		"BTC-27MAR20-12000-C",
		"BTC-24APR20-12000-C",
		"BTC-27MAR20-12000-P",
		"BTC-24APR20-12000-P",*/
	}
)

// main manages all Scrapers and handles incoming trade information
func main() {
	wg := sync.WaitGroup{}
	allScrapers := scrapers.NewAllDeribitOptionsScrapers(&wg, instruments, "rHQ8rYAo", "UmX8Ea0FelZzvT0nB44ZUWdRu6jyBYUMZDqE_gtQ2us")
	go allScrapers.GetMetas()
	//go allScrapers.RefreshMetas("BTC")
	allScrapers.ScrapeMarkets()

	log.Info(wg)
	defer wg.Wait()
}
