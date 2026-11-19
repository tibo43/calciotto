package models

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// The four preferred positions a player can pick on their account, in their
// canonical order: goalkeeper, defender, midfielder, forward. Stored and
// returned in this order, whatever order the player ticked them in.
const (
	PositionGK  = "GK"
	PositionDEF = "DEF"
	PositionMID = "MID"
	PositionFWD = "FWD"
)

// CanonicalPositions lists every allowed value, in canonical order.
var CanonicalPositions = []string{PositionGK, PositionDEF, PositionMID, PositionFWD}

// PreferredPositions is a player's preferred positions, stored as a Postgres
// text[] column. It implements sql.Scanner/driver.Valuer itself rather than
// pulling in a driver-specific array type: the values are a closed set of
// plain upper-case tokens (never a comma, quote, brace or space — see
// PlayerService.UpdatePreferredPositions), so the array literal `{GK,DEF}` is
// all there is to read and write.
//
// A nil or empty value is written as `{}`, never NULL: the column is NOT NULL
// DEFAULT '{}', so "no position" has exactly one representation.
type PreferredPositions []string

// Value writes the array literal, e.g. `{DEF,MID}`.
func (p PreferredPositions) Value() (driver.Value, error) {
	return "{" + strings.Join(p, ",") + "}", nil
}

// Scan reads the array literal Postgres sends for a text[] column.
func (p *PreferredPositions) Scan(src any) error {
	var raw string
	switch v := src.(type) {
	case nil:
		*p = PreferredPositions{}
		return nil
	case []byte:
		raw = string(v)
	case string:
		raw = v
	default:
		return fmt.Errorf("PreferredPositions: cannot scan %T", src)
	}
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") || !strings.HasSuffix(raw, "}") {
		return fmt.Errorf("PreferredPositions: not an array literal: %q", raw)
	}
	inner := raw[1 : len(raw)-1]
	out := PreferredPositions{}
	if inner != "" {
		for _, item := range strings.Split(inner, ",") {
			out = append(out, strings.Trim(item, `"`))
		}
	}
	*p = out
	return nil
}

// PlayerPositionsEntry is one player's line in GET /matches/:id/positions —
// PascalCase JSON, like the other match-scoped DTOs (MatchRegistrationEntry).
type PlayerPositionsEntry struct {
	PlayerID           uuid.UUID          `json:"PlayerID"`
	PreferredPositions PreferredPositions `json:"PreferredPositions"`
}
