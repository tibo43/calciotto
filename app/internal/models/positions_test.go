package models

import (
	"reflect"
	"testing"
)

func TestPreferredPositionsValue(t *testing.T) {
	cases := []struct {
		in   PreferredPositions
		want string
	}{
		{nil, "{}"},
		{PreferredPositions{}, "{}"},
		{PreferredPositions{"GK"}, "{GK}"},
		{PreferredPositions{"DEF", "MID"}, "{DEF,MID}"},
	}
	for _, c := range cases {
		got, err := c.in.Value()
		if err != nil {
			t.Fatalf("Value(%v) error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("Value(%v) = %v, want %q", c.in, got, c.want)
		}
	}
}

func TestPreferredPositionsScan(t *testing.T) {
	cases := []struct {
		in   any
		want PreferredPositions
	}{
		{nil, PreferredPositions{}},
		{"{}", PreferredPositions{}},
		{[]byte("{GK}"), PreferredPositions{"GK"}},
		{"{DEF,MID,FWD}", PreferredPositions{"DEF", "MID", "FWD"}},
		{`{"DEF","MID"}`, PreferredPositions{"DEF", "MID"}},
	}
	for _, c := range cases {
		var got PreferredPositions
		if err := got.Scan(c.in); err != nil {
			t.Fatalf("Scan(%v) error: %v", c.in, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Scan(%v) = %#v, want %#v", c.in, got, c.want)
		}
	}
	var bad PreferredPositions
	if err := bad.Scan("GK,DEF"); err == nil {
		t.Error("Scan of a non-array literal succeeded, want an error")
	}
	if err := bad.Scan(42); err == nil {
		t.Error("Scan of an int succeeded, want an error")
	}
}
