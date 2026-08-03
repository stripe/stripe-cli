package doctor

// flow.go — the live behavioral check behind `doctor --live`: start a local
// webhook handler (the doc's Ruby sample, in Go), run `stripe listen
// --forward-to` it, `stripe trigger checkout.session.async_payment_succeeded`,
// and verify the event arrives with a valid signature. All test mode.

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
)

// drillRun implements the doc's delayed-notification handler and proves it
// works by round-tripping a triggered event through `stripe listen`. The
// local handler binds an OS-assigned port, so nothing collides with the
// classic 4242.
func drillRun() (*DrillReport, error) {
	r := &DrillReport{Command: "drill", Events: []DrillEvent{}}
	received := make(chan DrillEvent, 8)
	var secret string

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
			received <- DrillEvent{Type: evt.Type, Signature: "invalid"}
			return
		}
		switch evt.Type {
		case "checkout.session.completed",
			"checkout.session.async_payment_succeeded",
			"checkout.session.async_payment_failed":
			received <- DrillEvent{Type: evt.Type, Signature: "verified"}
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
			return r, fmt.Errorf("stripe CLI not found on PATH (set STRIPE_BIN)")
		}
	}
	listen := exec.Command(stripeBin, "listen", "--forward-to", addr+"/webhook",
		"--events", "checkout.session.completed,checkout.session.async_payment_succeeded,checkout.session.async_payment_failed")
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
		return r, fmt.Errorf("timed out waiting for stripe listen")
	}

	trigger := exec.Command(stripeBin, "trigger", "checkout.session.async_payment_succeeded")
	if out, err := trigger.CombinedOutput(); err != nil {
		return r, fmt.Errorf("stripe trigger: %v: %s", err, out)
	}
	r.Triggered = "checkout.session.async_payment_succeeded"

	want := map[string]bool{"checkout.session.completed": false, "checkout.session.async_payment_succeeded": false}
	deadline := time.After(60 * time.Second)
	for {
		select {
		case evt := <-received:
			r.Events = append(r.Events, evt)
			if evt.Signature != "verified" {
				return r, fmt.Errorf("event %s arrived with invalid signature", evt.Type)
			}
			want[evt.Type] = true
			all := true
			for _, got := range want {
				if !got {
					all = false
				}
			}
			if all {
				r.Verified = true
				return r, nil
			}
		case <-deadline:
			r.Note = "timed out before both events arrived"
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
