# Suppression de match réservée aux admins + masquage UI par rôle

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème bloquant détecté

### Ce qui a été fait

- Nouvelle capacité qui n'existait pas du tout : `DELETE /matches/:id`, gardé par le `requireGroupAdmin` déjà en place pour la création/édition. `MatchService.DeleteMatch` scope sa recherche sur `(id, group_id)` ensemble — même pattern IDOR-safe que `GetMatchDetailsByID` — et **gère correctement l'absence de `ON DELETE CASCADE`** sur `MatchPlayer.MatchID` (vérifié directement contre la vraie base : contrainte FK simple, sans cascade) en supprimant les lignes `match_players` avant le match, dans une transaction.
- Le handler réutilise `resolveGroupID` (déjà utilisé par les autres routes GET de ce fichier) plutôt que d'inventer une troisième façon de résoudre le groupe — j'ai vérifié que son comportement (query param, sinon premier groupe du joueur) coïncide exactement avec celui de la middleware `requireGroupAdmin` pour une requête DELETE sans body, donc pas de risque de divergence entre le groupe autorisé et le groupe effectivement ciblé.
- Frontend : premier endroit du code à faire du masquage UI par rôle. `MatchesAll.vue` et `MatchDetails.vue` passent de `resolveActiveGroupId()` à `resolveActiveGroup()` (qui expose déjà `role` par groupe depuis une feature précédente) pour dériver `isAdmin`, avec dégradation propre en cas d'échec (comportement identique à l'ancien wrapper). Masque : bouton "Create Match", "Add Player", "Save Changes", bouton de suppression de joueur (X), les deux boutons +/- de buts. Le nom du joueur, l'avatar et le score restent visibles en lecture seule pour tout le monde.
- Nouveau bouton "Delete Match" (`.btn-danger`, classe déjà existante réutilisée), confirmation via `window.confirm` en reprenant exactement le pattern déjà présent dans ce même fichier (`beforeRouteLeave`'s "unsaved changes" guard) plutôt que d'inventer une modale, séparé visuellement de "Save Changes" par une marge supplémentaire.
- Tests : couverture service (suppression réelle des `match_players` + du match, pas juste absence d'erreur ; scope cross-groupe et ID inconnu → `ErrMatchNotFound`) et handler (401/403/404 cross-groupe/200 + `GET .../details` confirmant un 404 après coup, preuve que la suppression a réellement eu lieu).
- Documentation `CLAUDE.md` mise à jour, y compris la correction d'une affirmation devenue fausse ("aucun masquage UI par rôle nulle part") plutôt que de la laisser traîner.

### Vérifications faites en revue

- Relecture ligne à ligne des diffs (service, handler, `main.go`, les deux composants frontend) : cohérent, rien oublié.
- `go build ./...` / `go vet ./...` / `gofmt -l .` : verts. `go test ./... -count=1` contre la vraie base de dev : tous les packages passent. `npm run lint` / `npm run build` : verts. Tout vérifié indépendamment.
- **Test réel de bout en bout** (curl) :
  - Création d'un match, ajout d'un joueur au roster (peuple réellement `match_players`), puis : 401 sans token, 403 pour un membre simple, 200 pour l'admin.
  - **Vérifié en base** que les lignes `match_players` ET la ligne `matches` ont bien disparu (pas juste un 200 côté HTTP).
  - `GET .../details` sur le match supprimé → 404. Re-suppression du même match → 404 (idempotent côté résultat, pas d'erreur serveur).
  - **Test IDOR direct** : admin d'un groupe fraîchement créé tente de supprimer un match du groupe "Default" en passant l'ID de son propre groupe comme `group_id` → 404, match toujours présent en base ensuite.
  - **Navigateur réel (Playwright)** : connecté en tant que membre simple → aucun des boutons admin (Create Match, Add Player, Save Changes, Delete Match, boutons de but, suppression de joueur) n'est visible dans le DOM. Connecté en tant qu'admin → tous visibles. Aucune erreur console liée à cette feature (une erreur "Invalid matches data" est apparue lors du test, mais c'est le bug déjà identifié et signalé séparément dans la review "named-coloured-teams" — non lié à cette feature, provoqué par mes propres groupes de test accumulés pendant les sessions précédentes).
  - État de la base remis à zéro (`-reset`) après les tests.

### Note indépendante de cette feature

Le repo contenait, en dehors de tout travail de cette session (`app/pkg/database/connect.go` modifié pour supporter `DATABASE_URL`/`DATABASE_URL_UNPOOLED`, plus des fichiers/dossiers non trackés `.agents/`, `.claude/`, `skills-lock.json`), du travail non committé et sans rapport avec cette feature — la teammate l'a correctement signalé plutôt que d'y toucher ou de l'inclure dans son diff. Je l'ai laissé tel quel, non stagé, en dehors du commit de cette feature.

### Aucune itération de correction nécessaire.
