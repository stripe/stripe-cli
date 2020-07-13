go run $(go env GOROOT)/src/crypto/tls/generate_cert.go --host localhost --rsa-bits 2048 --ca --start-date "Jan 1 00:00:00 2020" --duration=1000000h
