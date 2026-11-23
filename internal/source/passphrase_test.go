package source

import (
	"errors"
	"strings"
	"testing"

	"github.com/posture-sec/guage/internal/model"
)

// TestPassphraseErrorsEchoNeitherValue pins the promise in CheckPassphrase's
// comment: a mismatch names no passphrase. Both values can reach a CI log if
// one is formatted into the error, so the text is checked as well as the
// sentinel.
func TestPassphraseErrorsEchoNeitherValue(t *testing.T) {
	t.Parallel()

	const (
		testnet = "Test SDF Network ; September 2015"
		mainnet = "Public Global Stellar Network ; September 2015"
	)

	cases := []struct {
		name    string
		info    model.NetworkInfo
		want    string
		wantErr error
	}{
		{
			name:    "empty expectation",
			info:    model.NetworkInfo{Passphrase: testnet, Ledger: 1},
			want:    "",
			wantErr: ErrOptions,
		},
		{
			// A whitespace-only expectation is compared byte for byte, so it
			// is a mismatch rather than an absent value.
			name:    "whitespace-only expectation",
			info:    model.NetworkInfo{Passphrase: testnet, Ledger: 1},
			want:    " \t ",
			wantErr: ErrPassphraseMismatch,
		},
		{
			name: "matching pair",
			info: model.NetworkInfo{Passphrase: testnet, Ledger: 1},
			want: testnet,
		},
		{
			name:    "mainnet expectation against testnet",
			info:    model.NetworkInfo{Passphrase: testnet, Ledger: 1},
			want:    mainnet,
			wantErr: ErrPassphraseMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := CheckPassphrase(tc.info, tc.want)
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("CheckPassphrase rejected a matching pair: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.want != "" && strings.Contains(err.Error(), tc.want) {
				t.Fatalf("the error echoes the wanted passphrase: %v", err)
			}
			if tc.info.Passphrase != "" && strings.Contains(err.Error(), tc.info.Passphrase) {
				t.Fatalf("the error echoes the served passphrase: %v", err)
			}
		})
	}
}
