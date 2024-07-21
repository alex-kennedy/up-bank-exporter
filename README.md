# Up Bank Exporter

Prometheus exporter for [Up](https://up.com.au).

This tool uses the Up API. See https://developer.up.com.au/ for docs.

If you wish to contribute, feel free to send a pull request.

## Metrics

Core Up bank metrics:

- `up_bank_accounts_count` The number of Up accounts.
- `up_bank_account_balance` Account balance of Up accounts, in the smallest
  denomination for the currency. For example, for an Australian dollar value of
  $10.56, this field will be 1056.
- `up_bank_webhooks_count` Number of active webhooks.

Metrics for HTTP requests to the Up API:

- `up_bank_http_outgoing_inflights` Inflight HTTP requests to the Up API.
- `up_bank_http_request_total` Total requests made to the Up API.
- `up_bank_http_request_latency` Latency histogram of requests to the Up API (in ms).
- `up_bank_http_response_size` Histogram of HTTP response size in bytes.
