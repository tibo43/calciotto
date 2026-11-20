package services

import (
	"errors"
	"reflect"
	"testing"

	"app/internal/models"
)

func TestNormalizePreferredPositions(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want models.PreferredPositions
	}{
		{"nil is no position", nil, models.PreferredPositions{}},
		{"empty is no position", []string{}, models.PreferredPositions{}},
		{"canonical order", []string{"FWD", "GK", "MID"}, models.PreferredPositions{"GK", "MID", "FWD"}},
		{"duplicates removed", []string{"DEF", "DEF", "MID", "DEF"}, models.PreferredPositions{"DEF", "MID"}},
		{"case and spaces forgiven", []string{" gk", "Def "}, models.PreferredPositions{"GK", "DEF"}},
		{"all four", []string{"MID", "FWD", "DEF", "GK"}, models.PreferredPositions{"GK", "DEF", "MID", "FWD"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := NormalizePreferredPositions(c.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("got %#v, want %#v", got, c.want)
			}
		})
	}
}

func TestNormalizePreferredPositionsRejectsUnknown(t *testing.T) {
	for _, in := range [][]string{{"FORW"}, {"GK", "ST"}, {""}, {"G K"}, {"{GK}"}, {"GK,DEF"}} {
		got, err := NormalizePreferredPositions(in)
		if !errors.Is(err, ErrInvalidPosition) {
			t.Errorf("NormalizePreferredPositions(%q) = %v, %v; want ErrInvalidPosition", in, got, err)
		}
	}
}
