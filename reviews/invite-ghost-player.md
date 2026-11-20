# Réclamation de compte pour un joueur fantôme (invitation par un admin)

## Itération 1 — 2026-08-29

- **Modèle** : Opus (feature la plus sensible du plan côté sécurité : attache des identifiants à un compte tiers, ré-émission d'un token à usage unique, et un vrai risque d'IDOR à fermer explicitement)
- **Verdict** : ✅ validé, aucun problème bloquant détecté

### Ce qui a été fait

- `AuthService.InviteExistingPlayer(playerID, email) error` : réutilise **intégralement** le mécanisme de reset de mot de passe déjà en place (`PasswordResetToken`, `generateResetToken`, `sendPasswordResetLink`) plutôt que d'inventer un type de token dédié — raisonnement explicite et juste : `ResetPassword` écrase déjà `PasswordHash` sans se soucier de savoir s'il y en avait un avant, donc consommer un token d'invitation ou un token de reset classique, c'est la même opération. Conséquence appréciable : **aucune page frontend nouvelle**, le lien `/reset-password?token=...` existant fonctionne tel quel.
- Bon réflexe de refactor : extraction de `issuePasswordResetToken(db, playerID)` partagée entre `ForgotPassword` et cette nouvelle méthode, sans complexifier `ForgotPassword` (le comportement de cette dernière n'a pas changé, juste raccourcie).
- **Garde-fou sécurité correctement identifié et fermé** : `RequireGroupAdminByPathParam` prouve seulement que l'appelant est admin du groupe `:id` — rien ne garantit que `:playerId` appartient à ce groupe. Sans vérification supplémentaire, un admin du groupe A aurait pu inviter n'importe quel ID de joueur, y compris un membre du groupe B qu'il n'administre pas. La teammate a ajouté un check explicite `IsMember(groupID, targetPlayerID)` → 404 avant d'appeler le service, exactement le genre de vérification qu'il fallait absolument voir apparaître dans cette feature — c'était le point sur lequel j'étais le plus vigilant en écrivant le prompt, et il a été traité correctement sans qu'il faille le renvoyer.
- Transaction correcte : attacher l'email et émettre le token se fait dans un seul `s.DB.Transaction`, avec l'envoi du lien (`sendPasswordResetLink`) **après** le commit — évite d'envoyer un lien pointant vers un token qui aurait été annulé par un rollback.
- Réutilisation intelligente des sentinelles existantes (`ErrPlayerNotFound`, `ErrPlayerAlreadyClaimed`, `ErrEmailAlreadyUsed`, `ErrEmailRequired`) — aucune nouvelle sentinelle inventée pour un sens qui existait déjà, conforme à la consigne.
- `GroupHandler` a gagné une dépendance `AuthService` (nécessaire puisque gérer des identifiants n'est pas la responsabilité de `GroupService`) — tous les points d'appel (`main.go` + 11 tests) mis à jour.
- Tests : couverture complète service + handler, y compris le chemin complet bout-en-bout (capture du log → `ResetPassword` → `Login`) et la vérification explicite qu'une tentative sur un compte déjà réclamé ne modifie ni l'email existant ni n'émet de token.

### Vérifications faites en revue

- Relecture ligne à ligne des diffs (service, handler, `main.go`) : cohérent, la logique de garde IDOR est bien positionnée dans le handler (validation liée à l'autorisation, pas de la logique métier — argument explicitement documenté par la teammate, bien vu).
- `go build ./...` / `go vet ./...` / `gofmt -l .` : verts, vérifiés indépendamment. Vérifié aussi que les 14 points d'appel de `NewGroupHandler` compilent tous après l'ajout du 3ème paramètre.
- `go test ./... -count=1` contre la vraie base Postgres de dev : tous les packages passent.
- **Test réel de bout en bout** (curl + lecture des logs du conteneur backend, comme pour la feature de reset de mot de passe) :
  - 401 sans token, 403 pour un membre simple, 404 pour une cible qui n'est pas membre de CE groupe (UUID inexistant).
  - Admin invite un vrai fantôme du groupe → 200 `{"invited":true}`.
  - Ré-invitation du même joueur (déjà réclamé) → 400 `"player already has an account"`.
  - Lien récupéré dans les vrais logs du conteneur (`docker logs devops-backend-1`), consommé via `POST /auth/reset-password` (endpoint existant, aucune modification) → 200, puis `POST /auth/login` avec le nouveau mot de passe → 200 avec JWT valide. La chaîne complète fonctionne réellement, pas seulement au niveau des codes HTTP.
  - État de la base remis à zéro (`-reset`) après les tests.

### Aucune itération de correction nécessaire.
