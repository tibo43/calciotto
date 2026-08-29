# Création de joueur fantôme réservée aux admins + unicité de nom repensée par groupe

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème bloquant détecté (un choix de design mineur relevé, accepté tel quel)

### Ce qui a été fait

- `POST /players` passe de complètement public à `authRequired + requireGroupAdmin` (réutilise telle quelle la middleware body/query-based ajoutée pour les matchs dans la feature précédente — pas de nouvelle middleware créée, bon réflexe de réutilisation).
- **Bug pré-existant corrigé au passage, à raison** : `MatchDetails.vue` ne envoyait jamais de `group_id` en créant un joueur fantôme depuis l'éditeur de match — le joueur atterrissait silencieusement dans `GroupService.GetDefaultGroup()` (un groupe quasi aléatoire), pas dans le groupe du match en cours d'édition. C'était exactement le genre de bug qu'une "création de fantôme réservée à l'admin du groupe" doit fermer, sinon le gating n'aurait aucun sens. Corrigé côté frontend (`group_id: this.match.GroupID`, **snake_case exact**, en évitant explicitement le piège déjà documenté dans `CLAUDE.md` sur le binding JSON de Gin qui ne comble pas l'absence d'underscore) et côté backend (fallback `GetFirstGroupForPlayer` au lieu de `GetDefaultGroup`, pour rester cohérent avec ce que la middleware a déjà autorisé).
- `GroupService.GetDefaultGroup` entièrement supprimé (plus aucun appelant réel après ce changement, vérifié par grep) plutôt que laissé mort — bonne hygiène, et tous les commentaires qui le décrivaient comme "seul chemin non authentifié restant" mis à jour partout (`groupscope.go`, `groupmembership.go`, `matches.go`, `groupMemberships.go`, `CLAUDE.md`).
- Contrainte d'unicité de nom repensée : suppression complète du check global case-insensitive (`ErrPlayerAlreadyExists`) dans `PlayerService.CreatePlayer` — cohérent avec la décision déjà actée pour le signup (un nom n'est unique nulle part globalement). À la place, nouveau garde-fou **soft, scopé au groupe** : `GroupMembershipService.HasMemberNamed(groupID, name)`, appelé depuis le handler après résolution du groupe cible, avant la création. `PlayerService` reste group-agnostic comme le veut la convention du projet — la vérification vit dans la couche qui connaît déjà les groupes.
- Tests : couverture complète et bien ciblée — 401 sans token, 403 non-admin, 200 admin avec vérification que le joueur atterrit bien dans le bon groupe (pas juste le code HTTP), 400 doublon dans le même groupe, 200 même nom dans un groupe différent. Test dédié pour `HasMemberNamed` (vrai/faux selon le groupe). Suppression propre des tests qui ciblaient l'ancien comportement global (`GetDefaultGroup`, unicité globale).

### Vérifications faites en revue

- Relecture ligne à ligne de tous les diffs (handler, services, modèle, `main.go`, frontend) : cohérent, logique métier bien confinée.
- `go build ./...` / `go vet ./...` / `gofmt -l .` : verts, vérifiés indépendamment.
- `go test ./... -count=1` contre la vraie base Postgres de dev : tous les packages passent.
- **Test réel de bout en bout** (curl) : 401 sans token, 403 pour un membre simple, 200 pour l'admin avec le fantôme confirmé dans le bon groupe via `psql` direct (pas juste via l'API), 400 sur doublon dans le même groupe, 200 pour le même nom dans un groupe fraîchement créé par le même admin.
- **Navigateur réel (Playwright)** : connexion en admin, ouverture d'un match, ouverture de la modale "Add Players", recherche d'un nom inexistant, clic sur "Create", toast de succès affiché, aucune erreur console. Vérifié en base que le joueur créé via l'UI atterrit bien dans le groupe du match édité (pas dans un groupe par défaut aléatoire) — c'est exactement le bug pré-existant que cette feature devait fermer, et il l'est.
- État de la base remis à zéro (`-reset`) après les tests.

### Point mineur noté, non bloquant

La teammate signale elle-même un choix de design : `ErrDuplicatePlayerNameInGroup` est retourné en construisant directement `services.ErrDuplicatePlayerNameInGroup.Error()` dans le handler plutôt que via `errors.Is` sur une erreur remontée par le service (puisque `HasMemberNamed` retourne `(bool, error)`, pas directement la sentinelle). Ça fonctionne correctement et c'est un choix défendable pour une méthode de requête booléenne, mais c'est légèrement en dehors du pattern `errors.Is` utilisé partout ailleurs dans le repo. Pas assez significatif pour renvoyer la teammate dessus — à garder en tête si ce pattern se répète.

### Aucune itération de correction nécessaire.
