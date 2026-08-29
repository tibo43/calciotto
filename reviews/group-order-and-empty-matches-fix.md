# Fix : ordre non déterministe de GET /groups/me + null au lieu de [] sur GET /matches/details

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème détecté

### Contexte

Ces deux bugs avaient été trouvés en testant réellement la feature "équipes nommées et colorées" (voir `reviews/named-coloured-teams.md`) : un admin de plusieurs groupes pouvait tomber par hasard sur un groupe fraîchement créé sans matchs au chargement de la page d'accueil et voir une erreur au lieu d'un état vide.

### Ce qui a été fait

- **Bug 1** : `GroupMembershipService.GetGroupsWithRoleByPlayerID` n'avait aucun `ORDER BY`. Ajout de `.Order("group_memberships.created_at ASC")`, exactement le même tri que `GetFirstGroupForPlayer` utilise déjà (et pour la même raison : `resolveActiveGroup()` côté frontend se rabat sur `groups[0]`, qui doit rester stable).
- **Bug 2** : `MatchHandler.GetMatchesDetails` ne normalisait pas un slice nil avant de le sérialiser, contrairement à `GroupHandler.GetMyGroups` qui fait déjà exactement ce garde-fou pour le même problème. Ajout du même pattern (`if matches == nil { matches = []models.MatchWithDetails{} }`), avec un commentaire calqué sur celui de `GetMyGroups`.
- `GetMatchDetailsByID` correctement laissé intact comme demandé (retourne un pointeur unique, pas un slice — déjà 404 proprement via `ErrMatchNotFound`).
- **Bonne discipline sur le scope** : la teammate a explicitement remarqué que le même problème de sérialisation nil→null existe probablement ailleurs (`GetPlayers`, `GetTeamsByGroup`, standings...) mais n'y a pas touché, conformément à la consigne de ne corriger que ces deux bugs précis.
- Tests de non-régression, tous deux écrits pour prouver qu'ils auraient effectivement échoué sur le code non corrigé, pas juste "de forme" :
  - Le test d'ordre force délibérément un ordre de `CreatedAt` différent de l'ordre d'insertion (via backdating), pour qu'une requête sans tri ne puisse pas passer "par coïncidence" — et l'exécute deux fois pour écarter un résultat accidentellement trié.
  - Le test du body vide vérifie le **corps brut de la réponse HTTP** (`"[]"` vs `"null"`), pas juste le résultat d'un `json.Unmarshal` (qui aurait masqué le bug : `null` unmarshalé dans un slice Go donne aussi une longueur 0).

### Vérifications faites en revue

- Relecture des deux diffs : minimaux, corrects, cohérents avec les patterns déjà établis dans le repo (pas de nouvelle abstraction inventée).
- `go build ./...` / `go vet ./...` / `gofmt -l .` : verts. `go test ./... -count=1` contre la vraie base de dev : tous les packages passent, y compris les deux nouveaux tests. Tout vérifié indépendamment.
- **Test réel de bout en bout** (curl) :
  - Groupe fraîchement créé sans aucun match → `GET /matches/details` renvoie littéralement `[]` (corps brut inspecté, pas juste un code 200).
  - Création de 3 groupes successifs, `GET /groups/me` appelé deux fois de suite → même ordre exact à chaque fois, correspondant à l'ordre de création/adhésion.
- État de la base remis à zéro (`-reset`) après les tests.

### Aucune itération de correction nécessaire.
