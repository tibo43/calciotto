package services

import (
	"testing"

	"github.com/google/uuid"
)

// Vectors produced by frontend/src/services/shortLink.js's encodeMatchId, which
// decodes these codes: the two must never drift apart.
func TestMatchShortCode_MatchesFrontendEncoder(t *testing.T) {
	cases := map[string]string{
		"92bbe7a1-fa67-49dc-9058-85837504d0d4": "4SskeXDGXctO9kCsU06kpo",
		"00000000-0000-0000-0000-000000000000": "0000000000000000000000",
		"ffffffff-ffff-ffff-ffff-ffffffffffff": "7n42DGM5Tflk9n8mt7Fhc7",
		"0000ffff-0000-4000-8000-000000000001": "000SKgXNxZcfXxYfKcWe9J",
	}
	for id, want := range cases {
		if got := MatchShortCode(uuid.MustParse(id)); got != want {
			t.Errorf("MatchShortCode(%s) = %q, want %q", id, got, want)
		}
	}
}
