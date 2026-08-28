# Quitter un groupe (self-service)

## Itération 1 — 2026-08-28

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- `GroupMembershipService.LeaveGroup(groupID, playerID)` dans une transaction : dernier membre → `ErrLastMember` ; owner avec d'autres membres → promotion du membre restant le plus ancien (`created_at ASC`, même logique que `GetFirstGroupForPlayer`) avant suppression ; simple member → suppression directe.
- `GroupHandler.LeaveGroup`, route `DELETE /groups/:id/members/me` (`authRequired` + `requireGroupMemberByPathID`), exactement le pattern des autres routes `/groups/:id/*`.
- Tests service : promotion du bon membre (ordre vérifié explicitement avec un "earlier" et un "later" member), départ d'un simple membre sans effet de bord sur l'owner, dernier membre bloqué.
- Tests handler : happy path (200 + vérifie que l'accès au groupe est ensuite 403), sans token (401), non-membre (403 via le middleware).

### Vérifications faites en revue

- Relecture des diffs : conforme, transaction correcte, rien hors scope (pas de "retirer un membre", `cmd/seed` et frontend intacts).
- `go build`/`go vet`/`go test ./... -count=1` : tous verts.
- **Test réel de bout en bout** (docker + seed + curl) : owner seul → 400 ; ajout d'un second membre puis départ de l'owner → 200, le second membre est bien promu `owner` en base ; l'ancien owner perd l'accès aux données du groupe (403 sur `/matches/details`). Comportement exactement conforme à la règle métier spécifiée.

### Aucun problème détecté, aucune itération de correction nécessaire.
