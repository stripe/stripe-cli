// Package webhookcheck implements the live behavioral check behind
// `doctor --verify-webhooks`: start a local webhook handler, run `stripe listen
// --forward-to` it, `stripe trigger` one event, and verify it arrives
// with a valid signature. All test mode.
//
// This package has no idea which events matter for any given migration
// topic — Run takes the full set to listen for and which one to trigger,
// supplied by the topic's own remediation.Implementation (see
// remediation.WebhookChecker). Doctor's generic engine (cmd.go) looks that
// up per topic rather than assuming any fixed event set.
package webhookcheck

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

// Report is the webhook round-trip evidence (doctor --verify-webhooks).
type Report struct {
	Command     string  `json:"command"`
	ListenReady bool    `json:"listen_ready"`
	Triggered   string  `json:"triggered_event,omitempty"`
	Events      []Event `json:"events_received"`
	Verified    bool    `json:"verified"`
	Note        string  `json:"note,omitempty"`
}

type Event struct {
	Type      string `json:"type"`
	Signature string `json:"signature"` // verified | invalid
}

// Run starts a local webhook handler and proves it works by
// round-tripping triggerEvent through `stripe listen`, listening for the
// full events set (triggerEvent must be one of them). The local handler
// binds an OS-assigned port, so nothing collides with the classic 4242.
// Verified becomes true once triggerEvent itself arrives with a valid
// signature; any other declared events that happen to arrive (e.g. ones
// the same trigger cascades into) are recorded in Events too, but aren't
// required — this package has no way to know which, if any, a given
// topic's trigger fires alongside it.
func Run(events []string, triggerEvent string) (*Report, error) {
	r := &Report{Command: "webhook_check", Events: []Event{}}
	received := make(chan Event, 8)
	var secret string

	wantEvent := make(map[string]bool, len(events))
	for _, e := range events {
		wantEvent[e] = true
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return r, err
	}
	addr := ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, req *http.Request) {
		payload, _ := io.ReadAll(req.Body)
		sig := req.Header.Get("Stripe-Signature")
		var evt struct {
			Type string `json:"type"`
		}
		_ = json.Unmarshal(payload, &evt)
		if !verifySignature(payload, sig, secret) {
			w.WriteHeader(400)
			received <- Event{Type: evt.Type, Signature: "invalid"}
			return
		}
		if wantEvent[evt.Type] {
			received <- Event{Type: evt.Type, Signature: "verified"}
		}
		w.WriteHeader(200)
	})
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	defer srv.Close()

	stripeBin := os.Getenv("STRIPE_BIN")
	if stripeBin == "" {
		if p, err := exec.LookPath("stripe"); err == nil {
			stripeBin = p
		} else {
			return r, errorcategory.Errorf(errorcategory.UserInput, "stripe CLI not found on PATH (set STRIPE_BIN)")
		}
	}
	listen := exec.Command(stripeBin, "listen", "--forward-to", addr+"/webhook",
		"--events", strings.Join(events, ","))
	stderrPipe, _ := listen.StderrPipe()
	if err := listen.Start(); err != nil {
		return r, fmt.Errorf("stripe listen: %w", err)
	}
	defer func() { _ = listen.Process.Kill(); _, _ = listen.Process.Wait() }()

	secretCh := make(chan string, 1)
	go func() {
		buf := make([]byte, 4096)
		acc := ""
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				acc += string(buf[:n])
				if i := strings.Index(acc, "whsec_"); i >= 0 {
					end := i
					for end < len(acc) && !strings.ContainsRune(" \n\r\t", rune(acc[end])) {
						end++
					}
					secretCh <- acc[i:end]
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	select {
	case secret = <-secretCh:
		r.ListenReady = true
	case <-time.After(20 * time.Second):
		return r, errorcategory.Errorf(errorcategory.Network, "timed out waiting for stripe listen")
	}

	trigger := exec.Command(stripeBin, "trigger", triggerEvent)
	if out, err := trigger.CombinedOutput(); err != nil {
		return r, errorcategory.Errorf(errorcategory.Internal, "stripe trigger: %v: %s", err, out)
	}
	r.Triggered = triggerEvent

	deadline := time.After(60 * time.Second)
	for {
		select {
		case evt := <-received:
			r.Events = append(r.Events, evt)
			if evt.Signature != "verified" {
				return r, errorcategory.Errorf(errorcategory.Internal, "event %s arrived with invalid signature", evt.Type)
			}
			if evt.Type == triggerEvent {
				r.Verified = true
				return r, nil
			}
		case <-deadline:
			r.Note = "timed out before the triggered event arrived"
			return r, nil
		}
	}
}

// verifySignature implements Stripe's v1 scheme: HMAC-SHA256 of "t.payload".
func verifySignature(payload []byte, header, secret string) bool {
	if secret == "" || header == "" {
		return false
	}
	var t string
	var v1s []string
	for _, part := range strings.Split(header, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			t = kv[1]
		case "v1":
			v1s = append(v1s, kv[1])
		}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(t + "."))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	for _, v1 := range v1s {
		if hmac.Equal([]byte(expected), []byte(v1)) {
			return true
		}
	}
	return false
}
