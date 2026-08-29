# Liste des membres du groupe + gestion admin (rôle, retrait), joueur retiré non sélectionnable, stats conservées

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- Backend : `models.PlayerWithRole` (nouveau DTO, même pattern "embed + Role" que `GroupWithRole`) et `GroupMembershipService.GetPlayersWithRoleByGroupID`, tri déterministe par `created_at` (bon réflexe proactif, cohérent avec le fix d'ordre appliqué plus tôt sur `GetGroupsWithRoleByPlayerID`). `GET /groups/:id/players` en bénéficie directement, reste ouvert à tout membre (seule l'action est réservée aux admins, pas la lecture).
- `IsMember bool` ajouté à `PointsStandingRow`/`ScorerRow`, calculé en **post-traitement** dans `StandingsService.GetPointsStandings`/`GetScorers` (nouvelle méthode privée `currentMemberIDs`) — les fonctions pures `ComputePointsStandings`/`ComputeScorers` restent intactes, exactement la séparation architecturale documentée dans `CLAUDE.md`. `GetPlayerProfile` (stats cross-groupe) volontairement non touché, raisonnement explicite et correct (toujours `true` de toute façon dans ce contexte).
- Frontend :
  - `Groups.vue` : nouvelle section "Members" par groupe (même pattern fetch-à-la-demande que "Manage teams"), badge "ADMIN", actions "Make admin/member" + "Remove" visibles uniquement pour un admin et jamais sur sa propre ligne (déduit du JWT décodé côté client, usage purement cosmétique — le backend refuse déjà l'auto-ciblage de toute façon, donc aucun risque même si le décodage échouait).
  - `MatchDetails.vue` : la recherche "ajouter un joueur existant" passe de `getPlayers()` (global) à `getGroupMembers(activeGroupId)` (scopé au groupe) — un joueur retiré n'est plus proposable pour un nouveau roster.
  - `Standings.vue` : tag discret "(left the group)" quand `IsMember === false`, sur les deux tableaux (points et buteurs).
- Tests backend : rôle correct pour un mélange admin/membre dans un même groupe ; retrait d'un joueur qui a marqué des buts → toujours présent dans classement/buteurs avec ses buts intacts, mais `IsMember: false`.

### Vérifications faites en revue

- Relecture de tous les diffs (2 nouveaux DTOs, 2 méthodes service, 1 handler, 4 fichiers frontend) : cohérent, aucune logique métier dans les handlers, séparation pure/impur respectée côté standings.
- `go build ./...` / `go vet ./...` / `gofmt -l .` : verts. `go test ./... -count=1` contre la vraie base de dev : tous les packages passent. `npm run lint` / `npm run build` : verts. Tout vérifié indépendamment.
- **Test réel de bout en bout (curl)** : `GET /groups/:id/players` renvoie bien le rôle de chacun ; promotion/rétrogradation/retrait d'un joueur via les endpoints déjà existants ; après retrait, le joueur n'apparaît plus dans la liste des membres mais reste dans `GET /standings/scorers` avec `IsMember: false` et ses buts intacts.
- **Test réel de bout en bout (navigateur, Playwright)** :
  - Ouverture de la liste des membres : badge ADMIN uniquement sur thibaut, boutons d'action présents sur les 11 autres lignes, absents sur la sienne (capture d'écran inspectée, rendu propre).
  - Retrait de "marcos" via le vrai bouton "Remove" de l'UI (avec confirmation) → disparaît immédiatement de la liste des membres.
  - Recherche d'ajout de joueur dans un match : "marcos" n'apparaît plus dans les résultats après son retrait.
  - Page Standings (onglet Buteurs) : "Marcos" toujours affiché en tête de classement avec ses 8 buts, tag "(left the group)" affiché juste à côté de son nom (capture d'écran inspectée) — exactement le comportement demandé.
  - Aucune erreur console sur l'ensemble du parcours.
  - Base de dev remise à zéro (`-reset`) après les tests (deux fois, une après mes tests curl, une après le test navigateur).

### Aucune itération de correction nécessaire.
