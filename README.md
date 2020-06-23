# Usage
## Recording
1. Set the value of `recordMode` in `http_vcr.go:main()` to true for record mode. (TODO: configure via a /vcr endpoint)
2. Start the server: `go run http_vcr.go simpler.go serializer.go`
3. Send requests to `localhost:8080` via curl (TODO: currently it doesn't matter what endpoint you specify, the request will always be proxied to "http://gobyexample.com".)
4. Stop the server by hitting `/vcr/end`

## Replaying
1. Set the value of `recordMode` in `http_vcr.go:main()` to false for record mode.
2. Start the server: `go run http_vcr.go simpler.go serializer.go`
3. Send requests to `localhost:8080` via curl. They should replay the recorded requests and stop once the cassette has finished. TODO: make the end of cassette stopping more graceful)