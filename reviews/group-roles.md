# Rôles basiques dans un groupe (owner/member)

## Itération 1 — 2026-08-28

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- `GroupMembership.Role` (`internal/models/database.go`), `gorm:"...;not null;default:member"` + constantes `RoleOwner`/`RoleMember`.
- `AddPlayerToGroup` conservé tel quel (signature inchangée, tous les appelants existants intacts) et devenu un wrapper fin autour du nouveau `AddPlayerToGroupWithRole`. `GetRole` ajouté, même forme de requête que `IsMember`.
- `GroupHandler.CreateGroup` tague le créateur `RoleOwner` ; tous les autres chemins (join par code, ajout manuel) restent `RoleMember` par défaut — vérifié qu'aucun autre call site n'a été touché.
- `RequireGroupOwner`/`RequireGroupOwnerByPathParam` (`groupowner.go`), calqués fidèlement sur `RequireGroupMembership`, réutilisant les mêmes helpers de résolution de `group_id`. Non câblés sur une route, comme demandé.
- `CLAUDE.md` mis à jour, y compris une note sur la limitation "pas d'owner pour un groupe créé hors du flux HTTP authentifié" (ex: "Default" du seed), symétrique à la note déjà existante sur `invite_code = NULL`.
- Tests d'intégration : couvrent exactement la matrice demandée (créateur→owner, join/ajout manuel→member, owner/member/outsider/sans-token sur les deux variantes du middleware).

### Vérifications faites en revue

- Relecture des diffs : conforme, rien en dehors du scope (pas de wiring dans `main.go`, `cmd/seed` intact, frontend intact).
- `go build`/`go vet`/`go test ./... -count=1` : tous verts.
- Test réel (docker + seed) : les 12 memberships seedées sont bien `role='member'` (backfill du `default:member` par Postgres/GORM sur une colonne `NOT NULL` ajoutée à une table existante — fonctionne comme attendu, pas de valeur vide/NULL comme je le craignais initialement). Création d'un groupe via l'API réelle → créateur `owner` confirmé en base.

### Aucun problème détecté, aucune itération de correction nécessaire.
