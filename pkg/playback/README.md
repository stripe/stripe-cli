# Introduction
`stripe playback` is a prototype feature for the Stripe CLI. It is still under development. This project is inspired by the [VCR](https://github.com/vcr/vcr) approach to testing popularized in Ruby.

When using the Ruby VCR gem, you record any HTTP interactions made in your test suite (request & response). These recordings are stored in serialized format, and can be replayed in future Źests to make them "fast, deterministic, and accurate".

While Ruby VCR is implemented as a Gem that you can directly use and configure in your test source code - `stripe playback` runs as a separate HTTP proxy server on your machine. That means any configuration happens either at startup via the CLI, or via interprocess HTTP calls.

This WIP document aims to give an unfamiliar developer enough information to begin using `stripe playback`.
# Usage

You start and configure the server via the `stripe playback` command. See `stripe playback --help` for a description of the flags, which configures address, record vs replay mode, etc.

`go run cmd/stripe/main.go playback --help`

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

## Controlling the playback server
Besides the command line flags at startup, there are also HTTP endpoints that allow you to control and modify the server's behavior while it is running.

`GET:` `/pb/mode/[record, replay]`: Switch to the specified mode.

`GET:` `/pb/cassette/load?filepath=[filepath]`: Read/write from/to (depending on mode) to the cassette at `filepath`.

`GET:` `/pb/casette/eject`: Eject the cassette. In `record` mode this writes the recorded interactions to the loaded cassette file. In `replay` mode this is a no-op.


## Example
### In Window 1:

`go run cmd/stripe/main.go playback`
(Start a recordmode HTTP server at localhost:13111, writing to the default cassette)

### In Window 2:

Record some test interactions using the stripe CLI, but proxy through the `stripe playback` server:

`stripe customers create --api-base="http://localhost:13111"`

`stripe customers list --api-base="http://localhost:13111"`

`stripe balance retrieve --api-base="http://localhost:13111"`

Stop recording:

`curl http://localhost:13111/pb/stop`

### In Window 1:
Ctrl-C the record server to shut it down.

Then, start the replay server, which should read from the same default cassette.

`go run cmd/stripe/main.go playback --replaymode`

### In Window 2:

Replay the same sequence of interactions using the stripe CLI, and notice that we are now replaying from the cassette:

`stripe customers create --api-base="http://localhost:13111"`

`stripe customers list --api-base="http://localhost:13111"`

`stripe balance retrieve --api-base="http://localhost:13111"`


## [WIP] Webhooks
Webhooks are a WIP feature, but currently have basic functionality working. If you don't plan to make use of webhook recording/replaying, you can ignore this section.

Skeleton demo of functionality:

```
# Terminal 1
# Start the playback server using the default settings. (in record mode, and using the default ports)

> go run cmd/stripe/main.go playback

Seting up playback server...

/pb/mode/: Setting mode to  record
/pb/cassette/load: Loading cassette  [default_cassette.yaml]

------ Server Running ------
Recording...
Using cassette: "default_cassette.yaml".

Listening via HTTP on localhost:13111
Forwarding webhooks to http://localhost:13112
-----------------------------

```

```
# Terminal 2
# Use stripe listen to forward webhooks to the playback server's webhook endpoint

> stripe listen --forward-to localhost:13111/pb/webhooks
```


```
# Terminal 3
# This effectively sets up a basic HTTP server on localhost:13112 that will echo out all requests
# and always respond with a 200 status code.

> socat -v -s tcp-listen:13112,reuseaddr,fork    "exec:printf \'HTTP/1.0 200  OK\r\n\r\n\'"
```
Finally, use the Stripe CLI to send requests and trigger webhooks to the playback server.
```
# Terminal 4

# Send a normal request
stripe balance retrieve --api-base "localhost:13111"

# Trigger webhooks afterwards
stripe trigger payment_intent.created

```
You should see the `socat` server in Terminal 3 receive the forwarded webhooks. You should also see the `playback` server logging (and recording) all interactions (outbound API requests and inbound webhooks).

After all this, you can re-run the server in replay mode:

`go run cmd/stripe/main.go playback --replaymode`

Then, rerun the same commands in the same order in Terminal 4, and you should see **recorded** responses **and** webhooks being returned to your Stripe CLI client and to your `socat` server.


# Testing
Some of the tests require a `.env` file to be present at `/pkg/playback/.env` containing
`STRIPE_SECRET_KEY="sk_test_..."`. To run the tests, create this file and define your own secret testmode key in it.
