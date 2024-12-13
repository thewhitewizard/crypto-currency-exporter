# crypto-currency-exporter

Export Crypto currency price to Prometheus.

```bash
Usage of crypto-currency-exporter:
  -currencies string
        List of currency separated by comma to fetch from coingecko. (default "bitcoin,ethereum,iexec-rlc")
  -listen-address string
        Address to listen on. (default ":8080")
```

``` bash
crypto_currency_price_usd{token="ethereum"} 3899.110000
crypto_currency_price_usd{token="iexec-rlc"} 2.580000
crypto_currency_price_usd{token="bitcoin"} 100292.000000
crypto_currency_last_refresh_seconds 1734080567
```

## Exported Metrics

| Metric                               | Meaning                                                 | Labels |
|--------------------------------------|---------------------------------------------------------|--------|
| crypto_currency_last_refresh_seconds | Was the last refresh of price from coingecko successful |        |
| crypto_currency_price_usd            | Crypto currency price.                                  | token  |
