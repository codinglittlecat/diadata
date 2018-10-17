import axios from 'axios';
import router from '@/router';
import { AtomSpinner } from 'epic-spinners';
import sortBy from 'lodash/sortBy';
import { EventBus } from '@/main';
import shared from  '@/shared/shared';
import moment from 'moment';
import getSymbolFromCurrency from 'currency-symbol-map';
import Highcharts from 'highcharts';
import stockInit from 'highcharts/modules/stock';

stockInit(Highcharts)

export default {
  components: {
    AtomSpinner
  },
  name: 'CoinDetails',
  props: {},
  data() {
    return {
      exchange_fields: [
        { key: 'Name', label: 'Exchange', sortable: true },
        { key: 'PriceFormatted', label: 'Price', sortable: true },
        { key: 'Volume24', label: 'Volume (24h)', sortable: true },
        { key: 'TimeFormatted', label: 'Last Updated', sortable: true },
        { key: 'show_trades', label: 'Trades', sortable: false },
      ],
      exchanges: [],
      last_trade_fields: [
        { key: 'Pair', label: 'Pair', sortable: true },
        { key: 'EstimatedPrice', label: 'Price', sortable: true },
        { key: 'Order', label: 'Order', sortable: true },
        { key: 'Volume', label: 'Volume', sortable: true },
        { key: 'TimeFormatted', label: 'Time', sortable: true },
      ],
      loading: true,
      errored: false,
      coinDetails:{},
      coinSymbol: '',
      coindata:null,
      selectedCurrency:'',
      chartAllOptions: {},
      chartSimexOptions: {},
      rateArray: []
    };
  },
  created() {
    this.coinSymbol = this.$route.params.coinSymbol;
    EventBus.$emit('hideSearchInput', true);
    EventBus.$on('currencyChange', this.formatPairData);
  },
  beforeDestroy: function () {
    EventBus.$off('currencyChange', this.formatPairData);
  },
  mounted() {
    if(this.$route.params.coinRank) {
      localStorage.rank = this.$route.params.coinRank;
    }
   
    // fetch the coin details
    this.fetchCoinDetails();


  
  },
  methods: {
  	formatPairData() {
      this.loading = true;
      if(localStorage.selectedCurrency) {
       this.selectedCurrency = localStorage.selectedCurrency;
      }
      else{
        this.selectedCurrency = "USD";
      }
      
      let {Coin, Change, Exchanges } = this.coindata;
      this.rateArray = Change.USD;
      // format the coin details
      const coinPrice = shared.calculateCurrencyFromRate(Coin.Price,Change.USD,this.selectedCurrency,"today");
      const coinPriceFormatted = shared.formatCurrency(coinPrice,this.selectedCurrency);
      const coinPriceYesterday = shared.calculateCurrencyFromRate(Coin.PriceYesterday,Change.USD,this.selectedCurrency,"yesterday");
      const change24 = (coinPrice  - coinPriceYesterday) / coinPriceYesterday * 100;
      const change24Formatted = shared.formatChange24(change24);
      const volume24Formatted = shared.formatMarketCapAndVolume24(shared.calculateCurrencyFromRate(Coin.VolumeYesterdayUSD,Change.USD,this.selectedCurrency,"yesterday"),this.selectedCurrency);
      const circulatingSupplyFormattedWithoutSymbol = shared.formatCirculatingSupply(Coin.CirculatingSupply, undefined);
      this.coinDetails = { 
          coinName: Coin.Name,
          coinSymbol: Coin.Symbol,
          coinPriceFormatted,
          change24,
          change24Formatted,
          rank: localStorage.rank,
          volume24Formatted,
          circulatingSupplyFormattedWithoutSymbol,
      };

      // format the exchanges
      Exchanges.forEach((exchange)=>{
        exchange.PriceFormatted = shared.formatCurrency(shared.calculateCurrencyFromRate(exchange.Price,Change.USD,this.selectedCurrency,"today"),this.selectedCurrency),
        exchange.Volume24 = shared.formatMarketCapAndVolume24(shared.calculateCurrencyFromRate(exchange.VolumeYesterdayUSD,Change.USD,this.selectedCurrency,"yesterday"),this.selectedCurrency),
        exchange.TimeFormatted = shared.formatDateTime(exchange.Time,"dddd, MMMM Do YYYY, h:mm:ss a");

        // format the last trades too
        exchange.LastTrades.forEach((lastTrade) => {
            lastTrade.EstimatedPrice = shared.formatCurrency(shared.calculateCurrencyFromRate(lastTrade.EstimatedUSDPrice,Change.USD,this.selectedCurrency,"today"),this.selectedCurrency);
            lastTrade.TimeFormatted = shared.formatDateTime(lastTrade.Time,"h:mm:ss a");
        });
      });

      Exchanges = sortBy(Exchanges, 'VolumeYesterdayUSD').reverse();
      this.exchanges = Exchanges;
      // finally fetch the chart details
      this.fetchCoinChartDetails();
     
  	},

    async fetchCoinDetails() {
      try {
        const response = await axios.get(`https://api.diadata.org/v1/symbol/${this.coinSymbol.toUpperCase()}`);
        this.coindata = response.data;
        EventBus.$emit('currencyChange');
      }
      catch (error) {
        console.log(error);
        this.errored = true;
      }
    },

    async fetchCoinChartDetails() {

      try {
        let response = await axios.get(`https://api.diadata.org/v1/chartPointsAllExchanges/MA120/${this.coinSymbol.toUpperCase()}`);
        let response1 = await axios.get(`https://api.diadata.org/v1/chartPointsAllExchanges/VOL120/${this.coinSymbol.toUpperCase()}`);
        let response2 = await axios.get(`https://api.diadata.org/v1/chartPoints/MA120/Simex//${this.coinSymbol.toUpperCase()}`);
        let response3 = await axios.get(`https://api.diadata.org/v1/chartPoints/VOL120/Simex//${this.coinSymbol.toUpperCase()}`);
        const price = 'Price (' + this.selectedCurrency + ')';
        const currencySymbol = getSymbolFromCurrency(this.selectedCurrency);
    
        
        const MA120AllArray = this.formatChartValues(response.data[0].Series[0].values);
        const VOL120AllArray = this.formatChartValues(response1.data[0].Series[0].values);
        const MA120SimexArray = this.formatChartValues(response2.data[0].Series[0].values);
        const VOL120SimexArray = this.formatChartValues(response3.data[0].Series[0].values);
        
        // all exchanges
        this.chartAllOptions = {
          rangeSelector: {
            selected: 1
          },

          title: {
              text: 'All Exchanges'
          },
          xAxis: {
              type: 'datetime',
              title: {
                  text: 'Time'
              }
          },
          yAxis: {
              title: {
                  text: price
              },
              min: 0
          },
          tooltip: {
              headerFormat: '<b>{series.name}</b><br>',
              pointFormat: `{point.x:%e. %b}: ${currencySymbol }{point.y:.2f} `
          },

          series: {
                compare: 'percent',
                showInNavigator: true
          },

          colors: ['#6CF', '#39F', '#06C', '#036', '#000'],
          series: [{
              name: "MA120",
              data: MA120AllArray
          }]};
           this.loading = false;
      
         // simex
        this.chartSimexOptions = {
          chart: {
              type: 'spline'
          },
          title: {
              text: 'Simex'
          },
          subtitle: {
              text: ''
          },
          xAxis: {
              type: 'datetime',
              dateTimeLabelFormats: { // don't display the dummy year
                  month: '%e. %b',
                  year: '%b'
              },
              title: {
                  text: 'Time'
              }
          },
          yAxis: {
              title: {
                  text: price
              },
              min: 0
          },
          tooltip: {
              headerFormat: '<b>{series.name}</b><br>',
              pointFormat: `{point.x:%e. %b}: ${currencySymbol }{point.y:.2f} `
          },

          plotOptions: {
              spline: {
                  marker: {
                      enabled: true
                  }
              }
          },

          colors: ['#6CF', '#39F', '#06C', '#036', '#000'],
          series: [{
              name: "MA120",
              data: MA120SimexArray
          },{
              name: "2 Min. MA",
              data: []
          }]};
           this.loading = false;
      }
      catch(error) {
        console.log(error);
        this.loading = false;
      }

    },
    formatChartValues(chartValues) {

    let formattedValues = [];

    chartValues.forEach((chartValue) => {
       const UTCDate = new Date(moment(chartValue[0]).utc().format()).valueOf();
       const price = parseFloat(shared.calculateCurrencyFromRate(chartValue[4],this.rateArray,this.selectedCurrency,"today").toFixed(2));
       
       formattedValues.push([UTCDate,price]);
    });

    return formattedValues;
  }
  },

  
};
