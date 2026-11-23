package source

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// TestBodyCapBoundary pins the one byte read past the cap. Reading maxBody+1
// bytes separates a body that fits from one that does not, and the refusal must
// stay non-retrying: a regression that drops the plus one, compares with >=, or
// returns a retryable attempt would otherwise turn a large body into four
// requests and an ErrDecode.
func TestBodyCapBoundary(t *testing.T) {
	t.Parallel()

	const (
		maxBody = 256
		marker  = "limit-marker-must-not-be-echoed"
		root    = `{"network_passphrase":"Test SDF Network ; September 2015","history_latest_ledger":5466897,"marker":"` + marker + `"}`
	)

	// pad returns the valid root JSON padded with spaces to exactly size bytes.
	// Trailing whitespace keeps the body decodable while fixing its length.
	pad := func(body string, size int) string {
		if len(body) > size {
			t.Fatalf("the %d byte JSON does not fit in a %d byte cap", len(body), size)
		}
		return body + strings.Repeat(" ", size-len(body))
	}

	cases := []struct {
		name    string
		size    int
		wantErr error
	}{
		{name: "exactly the cap", size: maxBody},
		{name: "one byte over", size: maxBody + 1, wantErr: ErrTooLarge},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := pad(root, tc.size)
			if len(body) != tc.size {
				t.Fatalf("body is %d bytes, want %d", len(body), tc.size)
			}

			var requests int32
			srv := startServer(t, func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&requests, 1)
				serve(w, http.StatusOK, []byte(body))
			})

			// MaxRetries is above zero so a retryable refusal would show up as
			// more than one request rather than passing unnoticed.
			h := clientFor(t, srv, Options{MaxBody: maxBody, MaxRetries: 3})
			info, err := h.Network(context.Background())

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("a %d byte body failed under a %d byte cap: %v", tc.size, maxBody, err)
				}
				if info.Ledger != 5466897 {
					t.Fatalf("ledger = %d, want 5466897", info.Ledger)
				}
			} else {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("err = %v, want %v", err, tc.wantErr)
				}
				if strings.Contains(err.Error(), marker) || strings.Contains(err.Error(), "Test SDF Network") {
					t.Fatalf("the error echoes the served body: %v", err)
				}
			}

			if got := atomic.LoadInt32(&requests); got != 1 {
				t.Fatalf("requests = %d, want exactly 1", got)
			}
		})
	}
}
