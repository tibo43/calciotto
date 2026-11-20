# Signup self-service (création de compte avec nom libre)

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème bloquant détecté (une remarque mineure hors scope notée, non bloquante)

### Ce qui a été fait

- `AuthService.SignupNewPlayer(name, email, password) (uuid.UUID, error)` : crée un nouveau `Player` et lui attache email/password en une seule étape. Valide nom/email/password (sentinelles réutilisées : `ErrEmptyPlayerName`, `ErrEmailRequired`, `ErrPasswordRequired`, `ErrEmailAlreadyUsed`). **N'impose aucune unicité sur le nom** — décision produit délibérée actée avec l'utilisateur (un joueur n'a qu'un seul nom de compte partagé entre tous ses groupes, donc ni l'unicité globale ni par groupe n'ont de sens ici).
- Ancien `AuthService.Signup(playerID, ...)` conservé tel quel (non renommé, ~20 call sites dans les tests et `cmd/seed`), juste redocumenté comme flow "claim by name" non routé aujourd'hui, réservé à une future feature de réclamation de joueur fantôme.
- `AuthHandler.Signup` : bind `{name, email, password}`, retourne `{"id": ...}`.
- **Changement de schéma nécessaire et bien identifié** : `Player.Name` portait un `uniqueIndex` appliqué au niveau DB, indépendant du check applicatif de `PlayerService.CreatePlayer`. La teammate l'a retiré du modèle et a ajouté un `DROP INDEX IF EXISTS idx_players_name` explicite dans `connect.go`, avec un commentaire expliquant pourquoi `AutoMigrate` seul ne suffit pas (il n'enlève jamais un index qui n'est plus déclaré). Bon réflexe technique, correctement documenté dans `CLAUDE.md`.
- `PlayerService.CreatePlayer` (flow admin de création de joueur fantôme, feature séparée à venir) intentionnellement non touché — garde bien son propre check de doublon.
- Frontend : `Signup.vue` réécrit en formulaire nom/email/password classique, suppression du `getPlayers()` et du dropdown de sélection. `api.js` mis à jour en conséquence.
- Tests : couverture service + handler complète, y compris un test explicite `TestSignupNewPlayer_Integration_DuplicateNameSucceeds` qui prouve noir sur blanc l'absence de contrainte d'unicité (deux comptes distincts, deux logins indépendants).

### Vérifications faites en revue

- Relecture ligne à ligne des diffs (service, handler, modèle, migration, frontend, doc) : cohérent avec les conventions du projet, aucune logique métier dans le handler.
- `go build ./...` / `go vet ./...` : verts (vérifiés indépendamment, pas juste sur le rapport de la teammate).
- **Test réel de bout en bout** (stack docker déjà up, backend `air` en hot-reload) :
  - `curl POST /auth/signup` avec deux comptes différents portant **le même nom** → 200 les deux fois, deux UUID distincts, deux lignes réellement présentes en base (vérifié via `psql` directement dans `devops-db-1`, pas juste via l'API).
  - `\d players` dans la vraie base : `idx_players_name` bien absent, `idx_players_email` toujours unique — la migration s'applique correctement sur une base déjà existante (pas seulement sur un volume neuf), exactement le cas de figure signalé par la teammate comme nécessitant cette étape explicite.
  - Email dupliqué → 400 `email already in use`. Nom vide (espaces) → 400 `player name must not be empty`.
  - Login juste après signup → 200, token JWT valide.
  - **Navigateur réel (Playwright, chromium déjà installé dans le scratchpad)** : le formulaire de signup affiche bien les nouveaux champs nom/email/password (plus de dropdown "Select your name"), soumission → redirection `/login`, login avec les identifiants fraîchement créés → redirection vers l'accueil, nav affichant "Log out" (session bien établie).
  - `npm run lint` / `npm run build` : verts, vérifiés indépendamment.

### Point mineur noté, hors scope, non bloquant

Un compte tout nouvellement créé n'appartient à aucun groupe (il n'existait auparavant aucun moyen de créer un compte sans passer par un joueur pré-seedé/déjà membre d'un groupe). En arrivant sur `/` après ce genre de compte, la page Matches déclenche des erreurs console (404 sur `getMatchesDetails`/`getSeasons` faute de `group_id` valide) mais **affiche quand même correctement l'état vide** ("No matches available — Create your first match to get started"), donc pas de page cassée pour l'utilisateur. C'est un angle mort pré-existant de `MatchesAll.vue`/`activeGroup.js` (aucun groupe actif du tout, pas juste un groupe vide), qui devient visible seulement maintenant qu'un compte peut exister sans jamais avoir rejoint de groupe. Hors scope de cette feature (qui ne devait pas toucher aux groupes) — à garder en tête pour une itération future sur l'onboarding post-signup (probablement : rediriger un nouvel utilisateur sans groupe vers `/groups` plutôt que `/`).

### Aucune itération de correction nécessaire.
