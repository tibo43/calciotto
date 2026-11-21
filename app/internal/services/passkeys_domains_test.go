package services

import "testing"

// While the legacy and the new frontend run side by side, each allowed
// origin gets its own relying party, named after its own host: a passkey is
// bound to the domain it was created on.
func TestNewPasskeyService_OneRelyingPartyPerFrontend(t *testing.T) {
	service, err := NewPasskeyService(nil, []string{
		"https://calcio-otto.vercel.app",
		"https://club.agoran.app",
		"https://club.agoran.app/", // not a bare origin: skipped
		"http://localhost:4000",
	})
	if err != nil {
		t.Fatalf("NewPasskeyService: %v", err)
	}
	want := map[string]string{
		"https://calcio-otto.vercel.app": "calcio-otto.vercel.app",
		"https://club.agoran.app":        "club.agoran.app",
		"http://localhost:4000":          "localhost",
	}
	if len(service.sites) != len(want) {
		t.Fatalf("got %d relying parties, want %d", len(service.sites), len(want))
	}
	for origin, rpID := range want {
		rp, ok := service.sites[origin]
		if !ok {
			t.Fatalf("no relying party for %s", origin)
		}
		if rp.Config.RPID != rpID || len(rp.Config.RPOrigins) != 1 || rp.Config.RPOrigins[0] != origin {
			t.Fatalf("%s: RPID %q origins %v", origin, rp.Config.RPID, rp.Config.RPOrigins)
		}
	}
}
