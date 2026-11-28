package services

import (
	"fmt"
	"html"
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
// header's order), q=0 means "not acceptable".
func LocaleFromAcceptLanguage(header string) string {
	best, bestQ := "", 0.0
	for _, part := range strings.Split(header, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		q := 1.0
		if params != "" {
			if value, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
				parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err != nil {
					continue
				}
				q = parsed
			}
		}
		language, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(tag)), "-")
		language, _, _ = strings.Cut(language, "_")
		if q > bestQ && slices.Contains(models.SupportedLocales, language) {
			best, bestQ = language, q
		}
	}
	return best
}
