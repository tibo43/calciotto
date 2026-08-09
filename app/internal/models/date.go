package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

// Date represents a calendar day with no time-of-day component. It is
// serialized over JSON as "YYYY-MM-DD" (matching the existing API contract)
// and stored as a SQL date, while giving the backend real time.Time semantics.
type Date time.Time

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(dateLayout) + `"`), nil
}

func (d *Date) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		return nil
	}
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return fmt.Errorf("invalid date %q, expected format YYYY-MM-DD", s)
	}
	*d = Date(t)
	return nil
}

func (d Date) Value() (driver.Value, error) {
	return time.Time(d), nil
}

func (d *Date) Scan(src interface{}) error {
	t, ok := src.(time.Time)
	if !ok {
		return fmt.Errorf("Scan: unable to scan type %T into Date", src)
	}
	*d = Date(t)
	return nil
}

func (d Date) String() string {
	return time.Time(d).Format(dateLayout)
}

// Before reports whether d occurs before other.
func (d Date) Before(other Date) bool {
	return time.Time(d).Before(time.Time(other))
}
