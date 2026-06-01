# internal/docs

HTTP client for fetching and caching documentation from docs.stripe.com.
By default, nothing is cached — caching must be explicitly enabled via `WithCache`.

## Usage

```go
import (
    "net/url"
    "github.com/stripe/stripe-cli/internal/docs"
)

// Create a client with default settings.
client := docs.NewClient("0.3.0")

// Create a client with a filesystem cache.
cache, err := docs.NewFSCache("/tmp/stripe-docs-cache", docs.WithTTL(30*time.Minute))
if err != nil {
    log.Fatal(err)
}
client = docs.NewClient("0.3.0").WithOptions(docs.WithCache(cache))

// Fetch a page.
page, err := client.FetchPage(ctx, &url.URL{Path: "/payments/accept-a-payment"})

// Fetch a page with query parameters.
page, err = client.FetchPage(ctx, &url.URL{
    Path:     "/api/charges",
    RawQuery: url.Values{"api_version": {"2024-06-30"}, "lang": {"go"}}.Encode(),
})

// Access page metadata.
fmt.Println(page.URL)       // *url.URL — fully resolved with scheme and host
fmt.Println(page.FetchedAt) // time the content was retrieved
```
