package nftdatascrapers

// Please implement the scraping of coingecko quotations here.

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk/client"
	"google.golang.org/grpc"

	"github.com/diadata-org/diadata/pkg/dia"
	models "github.com/diadata-org/diadata/pkg/model"
)

const (
	flowAPI1       = "access-001.mainnet1.nodes.onflow.org:9000"
	flowAPI2       = "access-001.mainnet2.nodes.onflow.org:9000"
	flowAPI3       = "access-001.mainnet3.nodes.onflow.org:9000"
	flowAPI4       = "access-001.mainnet4.nodes.onflow.org:9000"
	flowAPI5       = "access-001.mainnet5.nodes.onflow.org:9000"
	flowAPI6       = "access-001.mainnet6.nodes.onflow.org:9000"
	flowAPI7       = "access-001.mainnet7.nodes.onflow.org:9000"
	flowAPI8       = "access-001.mainnet8.nodes.onflow.org:9000"
	flowAPICurrent = "access.mainnet.nodes.onflow.org:9000"
	requestLimit   = uint64(249)
)

var (
	rootHeight1       = uint64(7601063)
	rootHeight2       = uint64(8742959)
	rootHeight3       = uint64(9737133)
	rootHeight4       = uint64(9992020)
	rootHeight5       = uint64(12020337)
	rootHeight6       = uint64(12609237)
	rootHeight7       = uint64(13404174)
	rootHeight8       = uint64(13950742)
	rootHeightCurrent = uint64(14892104)
	rootHeights       = []uint64{rootHeight1, rootHeight2, rootHeight3, rootHeight4, rootHeight5, rootHeight6, rootHeight7, rootHeight8, rootHeightCurrent}
)

// GetFlowClient returns a feasible client corresponding to the block's startheight.
func getFlowClient(startheight uint64) (*client.Client, error) {
	if startheight >= rootHeightCurrent {
		fmt.Printf("make flow client at current level with: %s\n", flowAPICurrent)
		return client.New(flowAPICurrent, grpc.WithInsecure())
	} else if startheight >= rootHeight8 {
		return client.New(flowAPI8, grpc.WithInsecure())
	} else if startheight >= rootHeight7 {
		return client.New(flowAPI7, grpc.WithInsecure())
	} else if startheight >= rootHeight6 {
		return client.New(flowAPI6, grpc.WithInsecure())
	} else if startheight >= rootHeight5 {
		return client.New(flowAPI5, grpc.WithInsecure())
	} else if startheight >= rootHeight4 {
		return client.New(flowAPI4, grpc.WithInsecure())
	} else if startheight >= rootHeight3 {
		return client.New(flowAPI3, grpc.WithInsecure())
	} else if startheight >= rootHeight2 {
		return client.New(flowAPI2, grpc.WithInsecure())
	} else if startheight >= rootHeight1 {
		return client.New(flowAPI1, grpc.WithInsecure())
	}
	return nil, errors.New("startheight too small. No client available.")
}

type NBATopshotScraper struct {
	nftscraper NFTScraper
	flowClient *client.Client
	ticker     *time.Ticker
}

type Play struct {
	SeriesID   uint32
	SetID      uint32
	PlayID     uint32
	SetName    string
	Attributes map[string]interface{}
}

type Moment struct {
	ID           uint64
	SetID        uint32
	PlayID       uint32
	SerialNumber uint32
}

func NewNBATopshotScraper(rdb *models.RelDB) *NBATopshotScraper {

	flowClient, err := client.New(flowAPICurrent, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	err = flowClient.Ping(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	nftScraper := NFTScraper{
		shutdown:     make(chan nothing),
		shutdownDone: make(chan nothing),
		error:        nil,
		relDB:        rdb,
		chanData:     make(chan dia.NFT),
	}
	s := &NBATopshotScraper{
		nftscraper: nftScraper,
		flowClient: flowClient,
		ticker:     time.NewTicker(refreshDelay),
	}

	go s.mainLoop()
	return s
}

// mainLoop runs in a goroutine until channel s is closed.
func (scraper *NBATopshotScraper) mainLoop() {
	err := scraper.UpdateNFT()
	if err != nil {
		log.Error("error updating NFT: ", err)
	}
	for {
		select {
		case <-scraper.ticker.C:
			err := scraper.UpdateNFT()
			if err != nil {
				log.Error("error updating NFT: ", err)
			}
		case <-scraper.nftscraper.shutdown: // user requested shutdown
			log.Info("NBA Topshot scraper shutting down")
			err := scraper.Close()
			scraper.cleanup(err)
			return
		}
	}
}

func (scraper *NBATopshotScraper) UpdateNFT() error {
	fmt.Println("fetch data...")
	nfts, err := scraper.FetchData()
	if err != nil {
		return err
	}
	for _, nft := range nfts {
		scraper.GetDataChannel() <- nft
	}
	return nil
}

// FetchData returns a slice of all NFTs fetched.
func (scraper *NBATopshotScraper) FetchData() (nfts []dia.NFT, err error) {

	var lastBlock uint64
	lastBlock, err = scraper.nftscraper.relDB.GetLastBlockheightTopshot(time.Now())
	if err != nil {
		log.Error("fetch last topshot block: ", err)
	}
	if lastBlock == uint64(0) {
		// No last block in db. Start from genesis block.
		lastBlock = rootHeight1
	}

	var nbaTopshotNFTs []dia.NFT
	allMoments, timestamps, blocknumbers, err := scraper.GetAllMoments(lastBlock)
	if err != nil {
		return []dia.NFT{}, err
	}

	attributeMap, err := scraper.GetAttributeMap()
	if err != nil {
		return []dia.NFT{}, err
	}

	for i, moment := range allMoments {
		m := MomentMintedEvent(moment)
		metadata := attributeMap[identifier{
			SetID:  uint32(m.SetID()),
			PlayID: uint32(m.PlayID()),
		}]
		metadata["blocknumber"] = blocknumbers[i]
		nbaTopshotNFTs = append(nbaTopshotNFTs, dia.NFT{
			NFTClass: dia.NFTClass{
				Address:      common.HexToAddress("0x0b2a3299cc857e29"),
				Symbol:       "TS",
				Name:         "TopShot",
				Blockchain:   "Flow",
				ContractType: "non-fungible",
				Category:     "Collectibles",
			},
			TokenID:        strconv.Itoa(int(m.ID())),
			CreationTime:   timestamps[i],
			CreatorAddress: common.Address{},
			URI:            "not available",
			Attributes:     metadata,
		})
	}
	fmt.Println("results: ", nbaTopshotNFTs)

	return nbaTopshotNFTs, nil
}

// ---------------------------------------------------------
// Get Data
// ---------------------------------------------------------

// GetAllMoments returns all moments from genesis to the latest block by iterating through
// blocks and looking for MomentMinted events.
func (scraper *NBATopshotScraper) GetAllMoments(startheight uint64) (mintedMoments []cadence.Event, timestamps []time.Time, blocknumbers []uint64, err error) {
	log.Info("Getting moments...")
	latestBlock, err := scraper.flowClient.GetLatestBlock(context.Background(), false)
	if err != nil {
		log.Error(err)
	}

	// Get first interval.
	var currentIndex int
	if startheight > rootHeights[len(rootHeights)-1] {
		currentIndex = len(rootHeights)
	} else {
		for i, root := range rootHeights {
			if startheight < root {
				currentIndex = i
				break
			}
		}
	}

	log.Infof("make flow client at startheight %v: ", startheight)
	log.Infof("currentIndex: %v\n", currentIndex)

	flowClient, err := getFlowClient(startheight)
	if err != nil {
		return
	}

	for startheight < latestBlock.Height {

		if currentIndex == len(rootHeights) || startheight+requestLimit < rootHeights[currentIndex] {
			// all blocks within the range of given client.
			m, t, b, err := GetMintedMoments(startheight, startheight+requestLimit, flowClient)
			if err != nil {
				log.Error("getting minted moments: ", err)
			}
			mintedMoments = append(mintedMoments, m...)
			timestamps = append(timestamps, t...)
			blocknumbers = append(blocknumbers, b...)
			startheight += requestLimit
			fmt.Println("current startheight: ", startheight)
		} else {
			// Reached new block range and thus need new client.
			fmt.Println("reached new block range")
			m, t, b, err := GetMintedMoments(startheight, rootHeights[currentIndex]-1, flowClient)
			if err != nil {
				log.Error(err)
			}
			mintedMoments = append(mintedMoments, m...)
			timestamps = append(timestamps, t...)
			blocknumbers = append(blocknumbers, b...)

			startheight = rootHeights[currentIndex]
			currentIndex += 1
			flowClient, err = getFlowClient(startheight)
			if err != nil {
				log.Error(err)
			}
		}
	}
	log.Info("... done getting moments.")
	return
}

// GetMintedMoments returns all moments minted between blocks @startheight and @endheight.
// The difference @endheight-@starthight is limited to 250.
// The range @startheight, @endheight must not be spread over more than the given @flowClient.
// https://docs.onflow.org/node-operation/past-sporks/
func GetMintedMoments(startheight, endheight uint64, flowClient *client.Client) (mintedMoments []cadence.Event, timestamps []time.Time, blockNumbers []uint64, err error) {

	blockEvents, err := flowClient.GetEventsForHeightRange(context.Background(), client.EventRangeQuery{
		Type:        "A.0b2a3299cc857e29.TopShot.MomentMinted",
		StartHeight: startheight,
		EndHeight:   endheight,
	})
	if err != nil {
		return
	}
	for _, blockEvent := range blockEvents {
		timestamp := blockEvent.BlockTimestamp
		for _, momentMintedEvent := range blockEvent.Events {
			fmt.Printf("got moment %v at time %v: \n", momentMintedEvent.Value, timestamp)
			timestamps = append(timestamps, timestamp)
			blockNumbers = append(blockNumbers, blockEvent.Height)
			mintedMoments = append(mintedMoments, momentMintedEvent.Value)
		}
	}
	return
}

type MomentMintedEvent cadence.Event

func (mme MomentMintedEvent) ID() uint64 {
	return uint64(mme.Fields[0].(cadence.UInt64))
}

func (mme MomentMintedEvent) PlayID() uint32 {
	return uint32(mme.Fields[1].(cadence.UInt32))
}

func (mme MomentMintedEvent) SetID() uint32 {
	return uint32(mme.Fields[2].(cadence.UInt32))
}

func (mme MomentMintedEvent) SerialNumber() uint32 {
	return uint32(mme.Fields[3].(cadence.UInt32))
}

// GetMetadata returns the metadata associated to the play with @playid in the set with @setid.
func (scraper *NBATopshotScraper) GetMetadata(setid uint32, playid uint32) (map[string]interface{}, error) {
	getPlaysScript := `
	import TopShot from 0x0b2a3299cc857e29

	pub struct MomentData  {
		pub var seriesId: UInt32
		pub var setId: UInt32
		pub var playId: UInt32
		
  
		pub var play: {String: String}	 
		pub var setName: String
		pub var numMoments: UInt32
	  
		init(playid: UInt32, setid: UInt32) {
		  self.seriesId = TopShot.getSetSeries(setID: setid)!
		  self.playId = playid
		  self.setId = setid
		   
		  self.play = TopShot.getPlayMetaData(playID: self.playId)!
		  self.setName = TopShot.getSetName(setID: self.setId)!
		  self.numMoments = TopShot.getNumMomentsInEdition(setID: self.setId, playID: self.playId)!
		  
		}  
	  }
	
	pub fun main(setid: UInt32, playid: UInt32): MomentData {
		var mom: MomentData = MomentData(playid: playid, setid: setid)		
		return mom
	}
	
`
	res, err := scraper.flowClient.ExecuteScriptAtLatestBlock(context.Background(), []byte(getPlaysScript), []cadence.Value{
		cadence.UInt32(setid),
		cadence.UInt32(playid),
	})
	if err != nil {
		return make(map[string]interface{}), fmt.Errorf("error fetching sale moment from flow: %w", err)
	}

	return cadenceMomentToMap(res.(cadence.Struct)), nil
}

// cadenceMomentToMap is a helper for GetMetadata and converts a moment to a map.
func cadenceMomentToMap(cadenceMoment cadence.Value) map[string]interface{} {
	castPlay := cadenceMoment.ToGoValue().([]interface{})

	numMoments := castPlay[5].(uint32)
	auxAttributes := castPlay[3].(map[interface{}]interface{})
	attributes := make(map[string]interface{})
	for key := range auxAttributes {
		attributes[key.(string)] = auxAttributes[key]
	}
	attributes["numMomentsInEdition"] = numMoments
	return attributes
}

// GetPlaysBySet returns all plays contained in a set.
func (scraper *NBATopshotScraper) GetPlaysBySet(setid uint32) ([]cadence.Value, error) {
	getPlaysScript := `
	import TopShot from 0x0b2a3299cc857e29

	pub struct MomentData  {
		pub var seriesId: UInt32
		pub var setId: UInt32
		pub var playId: UInt32
  
		pub var play: {String: String}	 
		pub var setName: String
	  
		init(playid: UInt32, setid: UInt32) {
		  self.seriesId = TopShot.getSetSeries(setID: setid)!
		  self.playId = playid
		  self.setId = setid
		   
		  self.play = TopShot.getPlayMetaData(playID: self.playId)!
		  self.setName = TopShot.getSetName(setID: self.setId)!
		  
		}  
	  }
	
	pub fun main(setid: UInt32): [MomentData] {
		var moments: [MomentData] = []
		var allplayids: [UInt32] = TopShot.getPlaysInSet(setID: setid)!

		for playid in allplayids {
			var mom: MomentData = MomentData(playid: playid, setid: setid)
			moments.append(mom)
		}
		
		return moments
	}
	
`
	res, err := scraper.flowClient.ExecuteScriptAtLatestBlock(context.Background(), []byte(getPlaysScript), []cadence.Value{
		cadence.UInt32(setid),
	})
	if err != nil {
		return []cadence.Value{}, fmt.Errorf("error fetching sale moment from flow: %w", err)
	}
	type Plays cadence.Array

	setID := Plays(res.(cadence.Array))
	fmt.Println("number of plays: ", len(setID.Values))
	return setID.Values, nil
}

// GetNumSets returns the number of available sets.
func (scraper *NBATopshotScraper) GetNumSets() (uint32, error) {
	getSetIDScript := `
	import TopShot from 0x0b2a3299cc857e29
	pub fun main(): UInt32 {
		return TopShot.nextSetID
	}
	
`
	res, err := scraper.flowClient.ExecuteScriptAtLatestBlock(context.Background(), []byte(getSetIDScript), []cadence.Value{})
	if err != nil {
		return 0, fmt.Errorf("error fetching set id from flow: %w", err)
	}
	type SetID cadence.UInt32
	setID := SetID(res.(cadence.UInt32))

	return uint32(setID), nil
}

type identifier struct {
	SetID  uint32
	PlayID uint32
}

// GetAttributesMap returns a map that uniquely maps an identifier consisting of setID and playID
// onto the corresponding attributes.
func (scraper *NBATopshotScraper) GetAttributeMap() (map[identifier]map[string]interface{}, error) {
	log.Info("Get attribute map...")
	attrMap := make(map[identifier]map[string]interface{})
	numSets, err := scraper.GetNumSets()
	if err != nil {
		return attrMap, err
	}
	for i := 1; i < int(numSets); i++ {

		values, err := scraper.GetPlaysBySet(uint32(i))
		if err != nil {
			fmt.Println("getting setID: ", err)
		}
		for _, val := range values {
			play := cadenceplayToPlay(val)
			idfier := identifier{
				SetID:  play.SetID,
				PlayID: play.PlayID,
			}
			attributes, err := scraper.GetMetadata(idfier.SetID, idfier.PlayID)
			if err != nil {
				log.Errorf("fetching attributes for setID %v and playID %v: %v", idfier.SetID, idfier.PlayID, err)
			}
			attributes["seriesID"] = play.SeriesID
			attributes["setID"] = play.SetID
			attributes["playID"] = play.PlayID
			attributes["setName"] = play.SetName
			attrMap[idfier] = attributes
			fmt.Println("attributes: ", attributes)
		}

	}
	log.Info("... done getting attribute map.")
	return attrMap, nil
}

// cadenceplayToPlay casts a play given as a cadence.Value to the struct @Play.
func cadenceplayToPlay(cadencePlay cadence.Value) (play Play) {
	castPlay := cadencePlay.ToGoValue().([]interface{})
	play.SeriesID = castPlay[0].(uint32)
	play.SetID = castPlay[1].(uint32)
	play.PlayID = castPlay[2].(uint32)
	play.SetName = castPlay[4].(string)

	auxAttributes := castPlay[3].(map[interface{}]interface{})
	attributes := make(map[string]interface{})
	for key := range auxAttributes {
		attributes[key.(string)] = auxAttributes[key]
	}
	play.Attributes = attributes

	return play
}

// GetDataChannel returns the scrapers data channel.
func (scraper *NBATopshotScraper) GetDataChannel() chan dia.NFT {
	return scraper.nftscraper.chanData
}

// closes all connected Scrapers. Must only be called from mainLoop
func (scraper *NBATopshotScraper) cleanup(err error) {
	scraper.nftscraper.errorLock.Lock()
	defer scraper.nftscraper.errorLock.Unlock()
	scraper.ticker.Stop()
	if err != nil {
		scraper.nftscraper.error = err
	}
	scraper.nftscraper.closed = true
	close(scraper.nftscraper.shutdownDone) // signal that shutdown is complete
}

// Close closes any existing API connections
func (scraper *NBATopshotScraper) Close() error {
	if scraper.nftscraper.closed {
		return errors.New("scraper already closed")
	}
	close(scraper.nftscraper.shutdown)
	<-scraper.nftscraper.shutdownDone
	scraper.nftscraper.errorLock.RLock()
	defer scraper.nftscraper.errorLock.RUnlock()
	return scraper.nftscraper.error
}
