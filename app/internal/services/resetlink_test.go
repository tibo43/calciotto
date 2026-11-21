package services

import "testing"

func TestResetLinkBase(t *testing.T) {
	tests := []struct {
		name, origin, env, want string
	}{
		{"the requesting frontend wins", "https://club.agoran.app", "https://calcio-otto.vercel.app", "https://club.agoran.app"},
		{"the legacy frontend keeps its own links", "https://calcio-otto.vercel.app", "https://club.agoran.app", "https://calcio-otto.vercel.app"},
		{"FRONTEND_BASE_URL without a vouched origin", "", "https://calcio-otto.vercel.app/", "https://calcio-otto.vercel.app"},
		{"the local frontend when nothing is set", "", "", defaultFrontendBaseURL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("FRONTEND_BASE_URL", tt.env)
			if got := ResetLinkBase(tt.origin); got != tt.want {
				t.Fatalf("ResetLinkBase(%q) = %q, want %q", tt.origin, got, tt.want)
			}
		})
	}
}
