package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestHealthHandler checks if the health endpoint returns the correct response
func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthHandler)

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "UP", rr.Body.String())
}

// TestMetricsHandlerWithPrices checks if the /metrics endpoint works after fetching prices
func TestMetricsHandlerWithPrices(t *testing.T) {
	// Mock some prices and set the last refresh time
	mu.Lock()
	cryptoCurrencies = make(map[string]CryptoCurrencyData)
	cryptoCurrencies["bitcoin"] = CryptoCurrencyData{USD: 67820}
	cryptoCurrencies["ethereum"] = CryptoCurrencyData{USD: 2624.91}
	cryptoCurrencies["iexec-rlc"] = CryptoCurrencyData{USD: 1.5}
	lastRefresh = time.Now()
	mu.Unlock()

	req, err := http.NewRequest("GET", "/metrics", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(MetricsHandler)

	// Call the handler
	handler.ServeHTTP(rr, req)

	// Verify the prices are present
	output := rr.Body.String()
	assert.Contains(t, output, `crypto_currency_price_usd{token="bitcoin"} 67820.000000`)
	assert.Contains(t, output, `crypto_currency_price_usd{token="ethereum"} 2624.910000`)
	assert.Contains(t, output, `crypto_currency_price_usd{token="iexec-rlc"} 1.500000`)
	assert.Contains(t, output, `crypto_currency_last_refresh_seconds`)
}

// TestFetchPricesMocked checks if FetchPrices correctly retrieves and parses prices from a mock server
func TestFetchPricesMocked(t *testing.T) {
	// Set up a mock server
	mockResponse := `{
		"bitcoin": {"usd": 67820},
		"ethereum": {"usd": 2624.91},
		"iexec-rlc": {"usd": 1.5}
	}`
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	// Initialize the client with the mock server URL
	client := &CoinGeckoClient{
		BaseURL:    mockServer.URL,
		HTTPClient: mockServer.Client(),
	}

	ids := []string{"bitcoin", "ethereum", "iexec-rlc"}
	prices, err := client.FetchPrices(ids)

	assert.NoError(t, err)
	assert.Equal(t, 67820.0, prices["bitcoin"].USD)
	assert.Equal(t, 2624.91, prices["ethereum"].USD)
	assert.Equal(t, 1.5, prices["iexec-rlc"].USD)
}

func TestRefreshPrices(t *testing.T) {
	// Mock server and response
	mockResponse := `{
		"bitcoin": {"usd": 67000},
		"ethereum": {"usd": 2600},
		"iexec-rlc": {"usd": 1.55}
	}`
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
	defer mockServer.Close()

	client := &CoinGeckoClient{
		BaseURL:    mockServer.URL,
		HTTPClient: mockServer.Client(),
	}
	cryptoCurrencies = make(map[string]CryptoCurrencyData)
	ids := []string{"bitcoin", "ethereum", "iexec-rlc"}

	// Create a done channel to signal when to stop
	done := make(chan bool)

	// Start the refresh in a separate goroutine
	go RefreshPrices(client, ids, 30*time.Second, done)

	// Allow some time for the prices to be updated
	time.Sleep(100 * time.Millisecond)

	// Signal to stop the goroutine
	close(done)

	// Now check the updated global prices
	mu.RLock()
	defer mu.RUnlock()

	// Ensure the global currencyPrices map is updated
	assert.Equal(t, 67000.0, cryptoCurrencies["bitcoin"].USD)
	assert.Equal(t, 2600.0, cryptoCurrencies["ethereum"].USD)
	assert.Equal(t, 1.55, cryptoCurrencies["iexec-rlc"].USD)

	// Ensure the last refresh timestamp is recent
	assert.WithinDuration(t, time.Now(), lastRefresh, time.Second)
}

// TestInvalidCoinGeckoResponse checks if the client handles an invalid response correctly
func TestInvalidCoinGeckoResponse(t *testing.T) {
	// Mock server returning invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`invalid json`))
	}))
	defer mockServer.Close()

	client := &CoinGeckoClient{
		BaseURL:    mockServer.URL,
		HTTPClient: mockServer.Client(),
	}

	ids := []string{"bitcoin", "ethereum", "iexec-rlc"}
	prices, err := client.FetchPrices(ids)

	assert.Error(t, err)
	assert.Nil(t, prices)
}
