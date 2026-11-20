# Reset de mot de passe (avec stub d'envoi en dev)

## Itération 1 — 2026-08-28

- **Modèle** : Opus
- **Verdict** : ✅ validé, aucun problème bloquant détecté (un point mineur noté, non bloquant)

### Ce qui a été fait

- `models.PasswordResetToken` : hash SHA-256 du token (jamais le token brut en base), avec une justification claire du choix SHA-256 vs bcrypt (entropie du token vs mot de passe choisi par un humain). Ajouté à `AutoMigrate`.
- `AuthService.ForgotPassword`/`ResetPassword` : anti-énumération correcte (même réponse, email connu ou non, avec un travail équivalent — génération d'un token jeté pour l'email inconnu), `ErrInvalidResetToken` unique pour tous les cas d'échec (inconnu/expiré/déjà utilisé), reset réussi = transaction qui change le mot de passe ET invalide tous les autres tokens actifs du joueur.
- Stub d'envoi : `log.Printf` uniquement, comme convenu, avec TODO explicite. `FRONTEND_BASE_URL` ajouté à `devops/env/develop.env`.
- Frontend : `ForgotPassword.vue`/`ResetPassword.vue` réutilisant le style existant, lien "Forgot password?" sur `Login.vue`, hand-off `reset=success` → notice de confirmation. Bon réflexe : a pensé à étendre `isRouterRoute` ET la liste des routes publiques dans `refreshGroups` (App.vue) — deux endroits distincts qui devaient tous les deux connaître les nouvelles routes publiques, les deux ont été mis à jour.
- `CLAUDE.md` : nouvelle section Auth complète, y compris la limitation "un reset ne révoque pas les JWT déjà émis" documentée comme prévu.
- Tests : couvrent exactement la matrice demandée, plus un test vérifiant explicitement que seul le hash est stocké (pas le token en clair) — bon réflexe de sécurité. Technique de capture des logs pour récupérer le token brut dans les tests bien pensée (c'est le seul endroit où il existe, cohérent avec le design).

### Vérifications faites en revue

- Relecture ligne à ligne du service (la partie la plus sensible) : logique correcte.
- `go build`/`go vet`/`go test ./... -count=1` : tous verts. `npm run lint`/`npm run build` : verts.
- **Test réel de bout en bout** (docker + seed + curl) : `forgot-password` renvoie le même message/status pour un email connu et inconnu ; le lien loggé contient un token exploitable ; reset avec ce token → 200, l'ancien mot de passe échoue ensuite au login (401), le nouveau fonctionne (200) ; rejeu du même token → 400 ; token bidon → 400, même message générique dans les deux cas.

### Point mineur noté, non bloquant

Il existe une fenêtre de course théorique : `ResetPassword` lit le token (hors transaction) puis fait le hash bcrypt (potentiellement lent) puis ouvre une transaction qui met à jour le mot de passe et marque les tokens comme utilisés — deux requêtes concurrentes avec le **même** token valide pourraient toutes les deux passer la lecture initiale avant que l'une ne marque le token comme utilisé. Impact pratique quasi nul : l'attaquant devrait déjà posséder le token valide (donc avoir déjà la capacité de reset le mot de passe de toute façon), et il faudrait un timing quasi parfait. Je ne considère pas ça comme un problème de sécurité réel pour une app de ce contexte (groupe d'amis), donc je ne renvoie pas la teammate dessus — à garder en tête si l'app grandit vers un contexte avec des enjeux de sécurité plus élevés (un `UPDATE ... WHERE used_at IS NULL RETURNING ...` atomique réglerait ça).

### Aucune itération de correction nécessaire.
