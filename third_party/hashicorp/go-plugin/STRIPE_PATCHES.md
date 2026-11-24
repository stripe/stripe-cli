# Stripe patches to hashicorp/go-plugin

This directory is a copy of [`github.com/hashicorp/go-plugin`](https://github.com/hashicorp/go-plugin)
`v1.8.0`, with one local patch applied, minus `.github/`, `docs/`, `examples/`, `CHANGELOG.md`,
`README.md`, and CI-only config files (`buf.yaml`, `buf.gen.yaml`, `.go-version`, `.gitignore`)
-- none of which affect building or testing the package. It's wired in via a `replace` directive
in this repo's `go.mod`:

```
replace github.com/hashicorp/go-plugin => ./third_party/hashicorp/go-plugin
```

This exists so the patch doesn't require maintaining a separate forked repo (no extra remote,
no separate release process) -- the patched source just lives here, checked in like any other
code in this repo.

## The patch

`mtls.go`'s `generateCert()` -- used to generate the client's AutoMTLS identity certificate --
is changed from `elliptic.P521()` to `elliptic.P256()`.

**Why:** AutoMTLS (RUN_DX-5908) requires the plugin's gRPC listener to require and verify a
client certificate. TypeScript-based plugins (built on `@stripe/stripe-cli-plugin-bootstrap`,
shipped as `bun build --compile` binaries) can do this correctly -- except Bun's peer
certificate retrieval (confirmed via `node:tls`, `node:http2`, and Bun's own native socket API,
independent of Bun version) returns nothing for an ECDSA P-521 certificate specifically. P-256
works correctly. `generateCert()` is the only place this repo's AutoMTLS client cert comes
from, so switching its curve here is enough to make AutoMTLS actually work against those
plugins -- no change needed on the plugin side.

Verified directly: the real `stripe-cli` host, patched this way, connects successfully to a
real `bun build --compile` binary (Bun >= 1.4.3) running
[`@stripe/stripe-cli-plugin-bootstrap`](https://github.com/stripe/stripe-cli-ts-plugins)'s
AutoMTLS support.

This has no effect on Go-based plugins (e.g. `apps`): they use their own, separate
`hashicorp/go-plugin` dependency in their own repo, unaffected by this patch, and Go-to-Go
AutoMTLS already works regardless of curve.

## Updating to a newer upstream release

1. Replace this directory's contents with the new version from the Go module cache or
   `github.com/hashicorp/go-plugin`'s release tag, keeping this file, then strip the same
   non-build directories/files listed above.
2. Re-apply the one-line change to `generateCert()` in `mtls.go` (switch `elliptic.P521()` to
   `elliptic.P256()`). Check `mtls.go`'s history upstream first -- if a future release makes the
   curve configurable or changes the default itself, this patch (and the `replace` directive in
   `go.mod`) can likely be dropped entirely.
3. Run `go build ./...`, `go vet ./...`, and `make test` from the repo root to confirm nothing
   else changed in a way that affects stripe-cli.

## License

Unchanged from upstream: Mozilla Public License 2.0 (see `LICENSE` in this directory).
