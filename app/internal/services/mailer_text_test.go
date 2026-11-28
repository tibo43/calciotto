package services_test

import (
	"strings"
	"testing"

	"app/internal/models"
	"app/internal/services"
)

func TestPasswordResetEmail_EnglishIsTheOriginalText(t *testing.T) {
	subject, body := services.PasswordResetEmail("en", "https://club.example/reset-password?token=a&b=c")
	if subject != "Reset your Agoran password" {
		t.Errorf("subject = %q", subject)
	}
	want := `<p>Click the link below to reset your Agoran password:</p><p><a href="https://club.example/reset-password?token=a&amp;b=c">https://club.example/reset-password?token=a&amp;b=c</a></p><p>This link expires in one hour. If you didn't request this, you can ignore this email.</p>`
	if body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestPasswordResetEmail_EverySupportedLanguageHasItsOwnText(t *testing.T) {
	const link = "https://club.example/reset-password?token=abc"
	english, englishBody := services.PasswordResetEmail("en", link)
	seen := map[string]string{}
	for _, locale := range models.SupportedLocales {
		subject, body := services.PasswordResetEmail(locale, link)
		if subject == "" || strings.Count(body, link) != 2 {
			t.Errorf("%s: subject %q, body %q must carry the link twice", locale, subject, body)
		}
		if strings.Contains(subject, "{") || strings.Contains(body, "%!") {
			t.Errorf("%s: unfilled text: %q / %q", locale, subject, body)
		}
		if !strings.Contains(subject, "Agoran") {
			t.Errorf("%s: subject %q lost the product name", locale, subject)
		}
		if locale != "en" && (subject == english || body == englishBody) {
			t.Errorf("%s: still English", locale)
		}
		if other, dup := seen[subject]; dup {
			t.Errorf("%s and %s share the subject %q", locale, other, subject)
		}
		seen[subject] = locale
	}
}

func TestPasswordResetEmail_UnknownLanguageFallsBackToEnglish(t *testing.T) {
	want, wantBody := services.PasswordResetEmail("en", "https://x.example/r")
	for _, locale := range []string{"", "pt", "xx", "FR-ca"} {
		if subject, body := services.PasswordResetEmail(locale, "https://x.example/r"); subject != want || body != wantBody {
			t.Errorf("locale %q: got %q, want the English email", locale, subject)
		}
	}
}

func TestLocaleFromAcceptLanguage(t *testing.T) {
	for header, want := range map[string]string{
		"":                                 "",
		"de":                               "de",
		"de-AT,de;q=0.9,en;q=0.8":          "de",
		"pt-BR,pt;q=0.9,it;q=0.5,en;q=0.3": "it",
		"en;q=0.5,fr;q=0.9":                "fr",
		"fr;q=0.7,es;q=0.7":                "fr",
		"FR_ca":                            "fr",
		"ja,zh;q=0.8":                      "",
		"*":                                "",
		"fr;q=0,en;q=0.1":                  "en",
		"de;q=oops,es":                     "es",
		" es-419 ; q=0.8 , it ; q=0.9 ":    "it",
		// Parameter names are case-insensitive, and q need not come first.
		"fr;Q=0.1,de;q=0.5":   "de",
		"fr;x=1;q=0.1,de":     "de",
		"fr ; X=1 ; Q=0 , es": "es",
		"fr;level=1,de;q=0.9": "fr",
		// A q outside [0, 1], or not a finite number, makes the entry invalid.
		"fr;q=2,de;q=0.5":    "de",
		"fr;q=1e3,de;q=0.5":  "de",
		"fr;q=Inf,de;q=0.5":  "de",
		"fr;q=+Inf,de;q=0.5": "de",
		"fr;q=NaN,de;q=0.5":  "de",
		"fr;q=-0.5,de;q=0.1": "de",
		"fr;q=1,de;q=1.0001": "fr",
	} {
		if got := services.LocaleFromAcceptLanguage(header); got != want {
			t.Errorf("Accept-Language %q: got %q, want %q", header, got, want)
		}
	}
}

func TestEmailLocale(t *testing.T) {
	de, bogus := "de", "xx"
	for _, tt := range []struct {
		name   string
		saved  *string
		header string
		want   string
	}{
		{"saved language wins", &de, "fr", "de"},
		{"phone language when nothing is saved", nil, "it-IT,en;q=0.5", "it"},
		{"unsupported saved value is ignored", &bogus, "es", "es"},
		{"English when nothing matches", nil, "ja", "en"},
		{"English without a header", nil, "", "en"},
	} {
		if got := services.EmailLocale(tt.saved, tt.header); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
