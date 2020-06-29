# Setup
If you want to serve over HTTPS, you'll need to generate a self-signed certificate for the TLS server.

`cd pkg/vcr`

`./setup_for_https.sh`

Otherwise, no other setup is needed.

# Usage

You start and configure the server via the `stripe vcr` command. See `stripe vcr --help` for a description of the flags, which configure address, record vs replay mode, HTTP vs HTTPS, etc.

`go run cmd/stripe/main.go vcr --help`

# Testing
Some of the tests require a `.env` file to be present at `/pkg/vcr/.env` containing
`STRIPE_SECRET_KEY="sk_test_..."`. To run the tests, create this file and define your own secret testmode key in it.
