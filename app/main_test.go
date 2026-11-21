package main

import (
	"slices"
	"testing"
)

func TestAllowedOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://calcio-otto.vercel.app , https://club.agoran.app ,, ")
	if got, want := allowedOrigins(), []string{"https://calcio-otto.vercel.app", "https://club.agoran.app"}; !slices.Equal(got, want) {
		t.Fatalf("allowedOrigins() = %v, want %v", got, want)
	}

	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	if got, want := allowedOrigins(), []string{"http://localhost:4000", "http://127.0.0.1:4000"}; !slices.Equal(got, want) {
		t.Fatalf("allowedOrigins() without the variable = %v, want %v", got, want)
	}
}
