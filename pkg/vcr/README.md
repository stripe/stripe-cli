# Setup
If you want to serve over HTTPS, you'll need to generate a self-signed certificate for the TLS server.

`cd pkg/vcr`

`./setup_for_https.sh`

Otherwise, no other setup is needed.

# Usage

You start and configure the server via the `stripe vcr` command. See `stripe vcr --help` for a description of the flags, which configure address, record vs replay mode, HTTP vs HTTPS, etc.

`go run cmd/stripe/main.go vcr --help`

When running in both record/replay mode, the server will print out interactions with it:

```
### When in recordmode (playing responses from the remote API)
--> POST to /v1/customers
<-- 200 OK from HTTPS://API.STRIPE.COM

--> GET to /v1/customers
<-- 200 OK from HTTPS://API.STRIPE.COM

--> GET to /v1/balance
<-- 200 OK from HTTPS://API.STRIPE.COM

```
```
### When in replaymode (playing responses from CASSETTE)
--> POST to /v1/customers
<-- 200 OK from CASSETTE

--> GET to /v1/customers
<-- 200 OK from CASSETTE

--> GET to /v1/balance
<-- 200 OK from CASSETTE
```


## Example Interaction
### In Window 1:

`go run cmd/stripe/main.go vcr`
(Start a recordmode HTTP server at localhost:8080, writing to the default cassette)

### In Window 2:

Record some test interactions using the stripe CLI, but proxy through the `stripe vcr` server:

`stripe customers create --api-base="http://localhost:8080"`

`stripe customers list --api-base="http://localhost:8080"`

`stripe balance retrieve --api-base="http://localhost:8080"`

Stop recording:

`curl http://localhost:8080/vcr/stop`

### In Window 1:
Ctrl-C the record server to shut it down.

Then, start the replay server, which should read from the same default cassette.

`go run cmd/stripe/main.go vcr --replaymode`

### In Window 2:

Replay the same sequence of interactions using the stripe CLI, and notice that we are now replaying from the cassette:

`stripe customers create --api-base="http://localhost:8080"`

`stripe customers list --api-base="http://localhost:8080"`

`stripe balance retrieve --api-base="http://localhost:8080"`



# Testing
Some of the tests require a `.env` file to be present at `/pkg/vcr/.env` containing
`STRIPE_SECRET_KEY="sk_test_..."`. To run the tests, create this file and define your own secret testmode key in it.
