package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	defaultRefreshInterval = 30 * time.Second
	defaultListenAddress   = ":8080"
)

// CryptoCurrencyData struct holds the price of a currency in USD
type CryptoCurrencyData struct {
	USD float64 `json:"usd"`
}

// CoinGeckoClient struct to manage the HTTP client
type CoinGeckoClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewCoinGeckoClient initializes the CoinGecko client
func NewCoinGeckoClient() *CoinGeckoClient {
	return &CoinGeckoClient{
		BaseURL: "https://api.coingecko.com/api/v3/simple/price",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Global variable to store currency prices (map of token name to CurrencyData) and last refresh timestamp
var (
	cryptoCurrencies map[string]CryptoCurrencyData
	lastRefresh      time.Time
	mu               sync.RWMutex
	done             chan bool
)

// FetchPrices takes a list of cryptocurrency IDs and fetches their prices in USD
func (c *CoinGeckoClient) FetchPrices(ids []string) (map[string]CryptoCurrencyData, error) {
	// Join the list of ids into a comma-separated string
	idList := strings.Join(ids, ",")

	// Prepare the full URL with query parameters
	url := fmt.Sprintf("%s?ids=%s&vs_currencies=USD", c.BaseURL, idList)

	// Make the HTTP request
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse the JSON response into a map
	var result map[string]CryptoCurrencyData
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// MetricsHandler serves the /metrics endpoint with the OpenMetrics formatted data
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the map for reading
	mu.RLock()
	defer mu.RUnlock()

	// Set the Content-Type header for OpenMetrics
	w.Header().Set("Content-Type", "text/plain")

	// Loop through the currency prices and write the metrics
	for cryptoCurrency, data := range cryptoCurrencies {
		fmt.Fprintf(w, "crypto_currency_price_usd{token=\"%s\"} %f\n", cryptoCurrency, data.USD)
	}

	// Write the last refresh timestamp as a metric
	fmt.Fprintf(w, "crypto_currency_last_refresh_seconds %d\n", lastRefresh.Unix())
}

// HealthHandler serves the root endpoint (/) and returns "UP" with HTTP 200 status
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "UP")
}

func RefreshPrices(client *CoinGeckoClient, ids []string, interval time.Duration, done <-chan bool) {
	for {
		// Fetch the latest prices
		prices, err := client.FetchPrices(ids)
		if err != nil {
			log.Printf("Error fetching prices: %v", err)
			time.Sleep(interval)
			continue
		}

		// Lock the map for writing and update the prices
		mu.Lock()
		for id, price := range prices {
			cryptoCurrencies[id] = price
		}
		lastRefresh = time.Now() // Update the last refresh time
		mu.Unlock()

		// Check if the done channel is closed, indicating it's time to stop
		select {
		case <-done:
			return
		case <-time.After(interval):
			// continue after sleeping for the interval duration
		}
	}
}

func main() {

	var currencies, listenAddress string

	flag.StringVar(&currencies, "currencies", "bitcoin,ethereum,iexec-rlc", "List of currency separated by comma to fetch from coingecko.")
	flag.StringVar(&listenAddress, "listen-address", defaultListenAddress, "Address to listen on.")
	flag.Parse()

	if currencies == "" || listenAddress == "" {
		log.Println("missing required flags")
		flag.Usage()
		os.Exit(1)
	}

	// Initialize the CoinGecko client
	client := NewCoinGeckoClient()
	ids := strings.Split(currencies, ",")
	// Initialize the cryptoCurrencies map and pre-fill with the currency names
	cryptoCurrencies = make(map[string]CryptoCurrencyData)
	for _, id := range ids {
		cryptoCurrencies[id] = CryptoCurrencyData{USD: 0.0} // Initial value of 0.0 USD
	}

	// Set the initial last refresh time to 0 (Unix epoch)
	lastRefresh = time.Unix(0, 0)

	// Initialize the done channel for graceful shutdown of the refresh loop
	done = make(chan bool)

	// Start the background goroutine to refresh prices every 30 seconds
	go RefreshPrices(client, ids, defaultRefreshInterval, done)

	// Expose the /metrics endpoint
	http.HandleFunc("/metrics", MetricsHandler)

	// Expose the health check endpoint at /
	http.HandleFunc("/", HealthHandler)

	// Catch OS signals and close the `done` channel to stop the refresh goroutine.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signalChan
		log.Printf("Received signal %s, stopping refresh...", sig)
		close(done) // This will stop the refresh loop
		os.Exit(0)
	}()

	// Start the HTTP server
	log.Println("Prometheus exporter running on ", listenAddress)
	if err := http.ListenAndServe(listenAddress, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
