package services

import (
	"math/big"
	"strings"

	"github.com/google/uuid"
)

// shortLinkAlphabet and shortLinkLength mirror frontend/src/services/
// shortLink.js exactly: a match's short code is its UUID's 128 bits written
// in base62 and left-padded with '0' to the width of the largest 128-bit
// value. The frontend decodes these codes (the `?match=` deep link), so the
// two encoders must agree byte for byte; shortlink_test.go pins vectors the
// JS implementation produces.
const shortLinkAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const shortLinkLength = 22

// MatchShortCode is the code the frontend's `/?match=<code>` deep link
// expects, so a push notification can open its match directly.
func MatchShortCode(id uuid.UUID) string {
	n := new(big.Int).SetBytes(id[:])
	base := big.NewInt(int64(len(shortLinkAlphabet)))
	digit := new(big.Int)
	var out []byte
	for n.Sign() > 0 {
		n.DivMod(n, base, digit)
		out = append(out, shortLinkAlphabet[digit.Int64()])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	code := string(out)
	if len(code) < shortLinkLength {
		code = strings.Repeat("0", shortLinkLength-len(code)) + code
	}
	return code
}

// MatchDeepLink is the in-app path that opens a match on the Matches tab.
func MatchDeepLink(id uuid.UUID) string {
	return "/?match=" + MatchShortCode(id)
}
