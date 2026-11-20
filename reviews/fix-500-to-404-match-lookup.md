# Fix : `GetMatchDetailsByID` 500 → 404 sur mismatch de groupe

## Itération 1 — 2026-08-28

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- `app/internal/services/matches.go` : nouveau sentinel `ErrMatchNotFound`, retourné par `GetMatchDetailsByID` à la place de `gorm.ErrRecordNotFound` (bonne amélioration annexe : ça évite de laisser fuiter un type d'erreur GORM hors de la couche service). Bon commentaire expliquant pourquoi "n'existe pas" et "existe dans un autre groupe" partagent délibérément le même sentinel/404 (pas de signal distinct qui permettrait de deviner qu'un match existe ailleurs).
- `app/internal/handlers/matches.go` : `GetMatchDetailsByID` fait maintenant un `switch`/`errors.Is` mappant `ErrMatchNotFound` → 404, tout le reste → 500. Suit exactement le pattern déjà utilisé dans `AuthHandler.Signup`.
- Tests existants mis à jour sans rien casser (`TestGetMatchDetailsByID_Integration_NotFound`, `TestGetMatchDetailsByID_Integration_WrongGroupNotFound` dans `matches_integration_test.go`, plus le check dans `TestMatchesAndStandings_Integration_ScopedPerGroup`).
- Nouveau test handler `matchdetailsnotfound_integration_test.go` : bon réflexe d'isoler la couche testée en mettant le joueur membre des deux groupes (le gate `RequireGroupMembership` ne bloque pas la requête, donc le test vérifie bien le comportement du handler/service, pas l'autorisation). Vérifie aussi le cas "même match, bon groupe → 200 toujours" et "ID totalement inconnu → 404".

### Vérifications faites en revue

- Relecture des diffs : conforme au scope demandé, rien touché en dehors (`GetMatchesDetails`, `resolveGroupID`, `RequireGroupMembership`, frontend intacts).
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` : tous verts.
- Test réel (docker compose + seed + curl) : `GET /matches/:id/details?group_id=<autre_groupe>` renvoie bien `404 {"error":"match not found"}` au lieu de 500.

### Aucun problème détecté, aucune itération de correction nécessaire.
