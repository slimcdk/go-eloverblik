package cmd

import (
	"io"
	"testing"
	"time"

	"github.com/slimcdk/go-eloverblik/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChargeLinksCmdSendsCopenhagenMidnight covers the one command whose dates reach the
// API as timestamps rather than calendar dates. --from 2026-07-01 means the Danish day
// 1 July, which starts at midnight in Copenhagen; read as midnight in UTC it went out as
// 02:00 Copenhagen time, two hours into the day.
func TestChargeLinksCmdSendsCopenhagenMidnight(t *testing.T) {
	cph := copenhagenForTest(t)
	// sent is what the client puts in the request body: GetChargeLinksWithCharges sends
	// from and to as RFC 3339 timestamps in Copenhagen time.
	sent := func(tm time.Time) string { return tm.In(cph).Format(time.RFC3339) }

	for _, api := range []string{"customer", "thirdparty"} {
		t.Run(api, func(t *testing.T) {
			var from, to time.Time
			clientInstance = &MockClient{
				GetChargeLinksWithChargesFunc: func(_ []string, f, tt time.Time) (*eloverblik.ChargeLinksWithChargesResponse, error) {
					from, to = f, tt
					return &eloverblik.ChargeLinksWithChargesResponse{}, nil
				},
			}
			defer func() { clientInstance = nil }()

			oldOutput := output
			output = io.Discard
			defer func() { output = oldOutput }()

			_, err := execute(t, api, "charge-links", "571313000000000003",
				"--from", "2026-01-01", "--to", "2026-07-01", "--token", "dummy")

			require.NoError(t, err)
			assert.Equal(t, "2026-01-01T00:00:00+01:00", sent(from))
			assert.Equal(t, "2026-07-01T00:00:00+02:00", sent(to))
		})
	}
}

// TestChargeLinksCmdHelpSaysToIsExclusive: charge-links asks for [from, to), like
// timeseries, and its --to must say so in the same words.
func TestChargeLinksCmdHelpSaysToIsExclusive(t *testing.T) {
	for _, api := range []string{"customer", "thirdparty"} {
		t.Run(api, func(t *testing.T) {
			out, err := execute(t, api, "charge-links", "--help")

			require.NoError(t, err)
			assert.Contains(t, out, "end date, exclusive (YYYY-MM-DD, now, now-30d/w/m/y, defaults to today)")
		})
	}
}
