package services

import (
	"errors"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestNormalizeLocale(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   *string
		want string // "" = nil
	}{
		{"nil follows the phone", nil, ""},
		{"empty follows the phone", strPtr(""), ""},
		{"blank follows the phone", strPtr("  "), ""},
		{"a language", strPtr("fr"), "fr"},
		{"case and space are normalised", strPtr(" DE "), "de"},
		{"english is a choice too", strPtr("en"), "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeLocale(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if tc.want == "" {
				if got != nil {
					t.Errorf("got %q, want nil", *got)
				}
				return
			}
			if got == nil || *got != tc.want {
				t.Errorf("got %v, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizeLocale_RefusesWhatTheAppDoesNotSpeak(t *testing.T) {
	for _, in := range []string{"pt", "fr-CA", "fr_FR", "xx", "français", "de;DROP", "english"} {
		if got, err := NormalizeLocale(strPtr(in)); !errors.Is(err, ErrUnsupportedLocale) || got != nil {
			t.Errorf("NormalizeLocale(%q) = %v, %v; want ErrUnsupportedLocale", in, got, err)
		}
	}
}
