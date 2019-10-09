package stripe

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// inspectHeaders is the whitelist of headers that will be printed.
var inspectHeaders = []string{
	"Authorization",
	"Content-Type",
	"Date",
	"Idempotency-Key",
	"Idempotency-Replayed",
	"Request-Id",
	"Stripe-Account",
	"Stripe-Version",
}

type verboseTransport struct {
	Transport *http.Transport
	Verbose   bool
	Out       io.Writer
}

func (t *verboseTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	if t.Verbose {
		t.dumpRequest(req)
	}

	resp, err = t.Transport.RoundTrip(req)

	if err == nil && t.Verbose {
		t.dumpResponse(resp)
	}

	return
}

func (t *verboseTransport) dumpRequest(req *http.Request) {
	info := fmt.Sprintf("> %s %s://%s%s", req.Method, req.URL.Scheme, req.URL.Host, req.URL.RequestURI())
	t.verbosePrintln(info)
	t.dumpHeaders(req.Header, ">")
}

func (t *verboseTransport) dumpResponse(resp *http.Response) {
	info := fmt.Sprintf("< HTTP %d", resp.StatusCode)
	t.verbosePrintln(info)
	t.dumpHeaders(resp.Header, "<")
}

func (t *verboseTransport) dumpHeaders(header http.Header, indent string) {
	for _, listed := range inspectHeaders {
		for name, vv := range header {
			if !strings.EqualFold(name, listed) {
				continue
			}

			for _, v := range vv {
				if v != "" {
					r := regexp.MustCompile("(?i)^(basic|bearer) (.+)")
					if r.MatchString(v) {
						v = r.ReplaceAllString(v, "$1 [REDACTED]")
					}

					info := fmt.Sprintf("%s %s: %s", indent, name, v)
					t.verbosePrintln(info)
				}
			}
		}
	}
}

func (t *verboseTransport) verbosePrintln(msg string) {
	color := ansi.Color(t.Out)
	fmt.Fprintln(t.Out, color.Cyan(msg))
}
