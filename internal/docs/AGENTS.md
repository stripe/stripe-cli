# internal/docs

HTTP client for fetching and caching documentation from docs.stripe.com.
By default, nothing is cached — caching must be explicitly enabled via `WithCache`.

## Usage

```go
import "github.com/stripe/stripe-cli-docs-plugin/internal/docs"

// Create a client with default settings.
client := docs.NewClient("0.3.0")

// Create a client with a filesystem cache.
cache, err := docs.NewFSCache("/tmp/stripe-docs-cache", docs.WithTTL(30*time.Minute))
if err != nil {
    log.Fatal(err)
}
client = docs.NewClient("0.3.0").WithOptions(docs.WithCache(cache))

// Fetch a page.
page, err := client.FetchPage(ctx, "/payments/accept-a-payment", nil)

// Fetch a page with params.
page, err = client.FetchPage(ctx, "/api/charges", map[string]string{
    "api_version": "2024-06-30",
    "lang":        "go",
})

// Access page metadata.
fmt.Println(page.URL)       // resolved URL
fmt.Println(page.FromCache) // true if served from cache
fmt.Println(page.CachedAt)  // time the cache entry was written
```
