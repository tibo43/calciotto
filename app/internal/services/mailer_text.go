package services

import (
	"fmt"
	"html"
	"math"
	"slices"
	"strconv"
	"strings"

	"app/internal/models"
)

// defaultEmailLocale is the language of an email when nothing says otherwise.
const defaultEmailLocale = "en"

// passwordResetText is the wording of the password-reset email, per language
// (models.SupportedLocales). The wording follows the app's own for the same
// screens (src/locales/*.json, auth.forgot / auth.reset): tutoiement, « e-mail »
// in French, « Passwort » in German, « reimpostare » in Italian, « correo » and
// « restablecer » in Spanish. The link is the only variable part.
var passwordResetText = map[string]struct{ subject, intro, outro string }{
	"en": {
		subject: "Reset your Agoran password",
		intro:   "Click the link below to reset your Agoran password:",
		outro:   "This link expires in one hour. If you didn't request this, you can ignore this email.",
	},
	"fr": {
		subject: "Réinitialise ton mot de passe Agoran",
		intro:   "Clique sur le lien ci-dessous pour réinitialiser ton mot de passe Agoran\u00a0:",
		outro:   "Ce lien expire dans une heure. Si tu n’as pas fait cette demande, tu peux ignorer cet e-mail.",
	},
	"it": {
		subject: "Reimposta la tua password Agoran",
		intro:   "Clicca sul link qui sotto per reimpostare la tua password Agoran:",
		outro:   "Questo link scade tra un’ora. Se non hai fatto tu questa richiesta, puoi ignorare questa email.",
	},
	"de": {
		subject: "Setze dein Agoran-Passwort zurück",
		intro:   "Klicke auf den Link unten, um dein Agoran-Passwort zurückzusetzen:",
		outro:   "Dieser Link läuft in einer Stunde ab. Wenn du das nicht angefordert hast, kannst du diese E-Mail ignorieren.",
	},
	"es": {
		subject: "Restablece tu contraseña de Agoran",
		intro:   "Haz clic en el enlace de abajo para restablecer tu contraseña de Agoran:",
		outro:   "Este enlace caduca en una hora. Si no lo has solicitado, puedes ignorar este correo.",
	},
}

// PasswordResetEmail is the subject and HTML body of the reset email in the
// given language (unknown languages get English). The link is escaped here.
func PasswordResetEmail(locale, link string) (subject, htmlContent string) {
	text, ok := passwordResetText[locale]
	if !ok {
		text = passwordResetText[defaultEmailLocale]
	}
	escapedLink := html.EscapeString(link)
	return text.subject, fmt.Sprintf(
		`<p>%s</p><p><a href="%s">%s</a></p><p>%s</p>`,
		text.intro, escapedLink, escapedLink, text.outro,
	)
}

// EmailLocale picks the language of an email to someone who may not be signed
// in: the language they saved in the app (players.locale) when they chose one,
// else the first language of the request's Accept-Language header that the app
// speaks (the browser's own, which is what the signed-out screens follow too),
// else English.
func EmailLocale(saved *string, acceptLanguage string) string {
	if saved != nil && slices.Contains(models.SupportedLocales, *saved) {
		return *saved
	}
	if locale := LocaleFromAcceptLanguage(acceptLanguage); locale != "" {
		return locale
	}
	return defaultEmailLocale
}

// LocaleFromAcceptLanguage returns the supported language the header prefers
// ("fr-CA,fr;q=0.9,en;q=0.8" -> "fr"), or "" when none is spoken. Regional
// tags reduce to their language; q-values order the choices (ties keep the
// header's order), q=0 means "not acceptable". Parameter names are matched
// case-insensitively (RFC 9110) wherever they sit among the tag's parameters;
// an entry whose q is not a number between 0 and 1 is skipped.
func LocaleFromAcceptLanguage(header string) string {
	best, bestQ := "", 0.0
	for _, part := range strings.Split(header, ",") {
		tag, q, ok := parseAcceptLanguageEntry(part)
		if !ok {
			continue
		}
		language, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
		language, _, _ = strings.Cut(language, "_")
		if q > bestQ && slices.Contains(models.SupportedLocales, language) {
			best, bestQ = language, q
		}
	}
	return best
}

// parseAcceptLanguageEntry splits one entry of an Accept-Language header into
// its tag and q-value (1 when absent). ok is false when the q-value is not a
// finite number in [0, 1].
func parseAcceptLanguageEntry(entry string) (tag string, q float64, ok bool) {
	fields := strings.Split(entry, ";")
	q = 1.0
	for _, param := range fields[1:] {
		name, value, _ := strings.Cut(param, "=")
		if !strings.EqualFold(strings.TrimSpace(name), "q") {
			continue
		}
		parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || math.IsNaN(parsed) || parsed < 0 || parsed > 1 {
			return "", 0, false
		}
		q = parsed
	}
	return fields[0], q, true
}
