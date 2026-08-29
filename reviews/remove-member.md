# Retirer un membre (action réservée à l'owner)

## Itération 1 — 2026-08-28

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- `GroupMembershipService.RemoveMember(groupID, actingPlayerID, targetPlayerID)` : refuse l'auto-retrait (`ErrCannotRemoveSelf`), propage `gorm.ErrRecordNotFound` si la cible n'est pas membre, sinon suppression simple (pas de transaction nécessaire, une seule écriture — choix cohérent, pas de logique de dernier membre puisque l'owner reste en place).
- `GroupHandler.RemoveMember`, route `DELETE /groups/:id/members/:playerId`, protégée par `RequireGroupOwnerByPathParam`.
- Tests service et handler couvrant exactement la matrice demandée.

### Vérifications faites en revue

- Relecture des diffs : conforme au scope, `LeaveGroup` et sa route intacts.
- `go build`/`go vet`/`go test ./... -count=1` : tous verts.
- **Point d'attention identifié en revue, vérifié spécifiquement** : `DELETE /groups/:id/members/me` (existant) et `DELETE /groups/:id/members/:playerId` (nouveau) partagent le même préfixe de route pour la même méthode HTTP. Aucun test de la teammate ne les enregistre ensemble sur le même routeur Gin (chaque test construit son propre routeur isolé avec une seule des deux routes) — un conflit de routing (wildcard vs segment statique) ne se manifeste qu'au démarrage réel de `main.go`, ni `go build` ni `go vet` ne l'auraient détecté. Vérifié manuellement :
  - Démarrage du vrai backend (docker) : aucun panic, les deux routes apparaissent dans le log `[GIN-debug]`.
  - `DELETE /groups/:id/members/me` route bien vers `LeaveGroup` (réponse `{"left":true}`, promotion d'un successeur observée en base) — pas vers `RemoveMember` qui aurait échoué sur `uuid.Parse("me")`.
  - `DELETE /groups/:id/members/<uuid>` route bien vers `RemoveMember` : auto-retrait refusé (400), retrait d'un membre réussi (200 `{"removed":true}`), retrait par un non-owner refusé (403).
- Gin gère nativement la coexistence segment statique / paramètre nommé au même niveau (priorité au segment statique en cas de correspondance exacte) — comportement confirmé empiriquement, pas juste supposé.

### Aucun problème détecté, aucune itération de correction nécessaire. Point méthodologique pour la suite : toujours vérifier le démarrage réel de `main.go` quand une nouvelle route partage un préfixe avec une route existante, ce que les tests unitaires/intégration en routeur isolé ne peuvent pas détecter.
