package models

import (
	"testing"
	"time"
)

func TestDate_MarshalJSON(t *testing.T) {
	d := Date(time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC))

	data, err := d.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON returned error: %v", err)
	}
	if string(data) != `"2026-08-09"` {
		t.Errorf("MarshalJSON = %s, want \"2026-08-09\"", data)
	}
}

func TestDate_UnmarshalJSON_Valid(t *testing.T) {
	var d Date
	if err := d.UnmarshalJSON([]byte(`"2026-08-09"`)); err != nil {
		t.Fatalf("UnmarshalJSON returned error: %v", err)
	}

	want := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)
	if !time.Time(d).Equal(want) {
		t.Errorf("UnmarshalJSON set %v, want %v", time.Time(d), want)
	}
}

func TestDate_UnmarshalJSON_InvalidFormat(t *testing.T) {
	var d Date
	if err := d.UnmarshalJSON([]byte(`"09/08/2026"`)); err == nil {
		t.Fatal("expected an error for a malformed date, got nil")
	}
}

func TestDate_UnmarshalJSON_EmptyAndNull(t *testing.T) {
	for _, input := range []string{`""`, `null`} {
		var d Date
		if err := d.UnmarshalJSON([]byte(input)); err != nil {
			t.Errorf("UnmarshalJSON(%s) returned error: %v", input, err)
		}
	}
}

func TestDate_ScanAndValue(t *testing.T) {
	want := time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC)

	var d Date
	if err := d.Scan(want); err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if !time.Time(d).Equal(want) {
		t.Errorf("Scan set %v, want %v", time.Time(d), want)
	}

	value, err := d.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	got, ok := value.(time.Time)
	if !ok || !got.Equal(want) {
		t.Errorf("Value() = %v, want %v", value, want)
	}
}

func TestDate_Scan_WrongType(t *testing.T) {
	var d Date
	if err := d.Scan("not-a-time"); err == nil {
		t.Fatal("expected an error scanning a non-time.Time value, got nil")
	}
}

func TestDate_String(t *testing.T) {
	d := Date(time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC))
	if got := d.String(); got != "2026-08-09" {
		t.Errorf("String() = %q, want %q", got, "2026-08-09")
	}
}

func TestDate_Before(t *testing.T) {
	earlier := Date(time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC))
	later := Date(time.Date(2026, time.August, 10, 0, 0, 0, 0, time.UTC))

	if !earlier.Before(later) {
		t.Error("expected earlier.Before(later) to be true")
	}
	if later.Before(earlier) {
		t.Error("expected later.Before(earlier) to be false")
	}
}
