# Fix team-name display bug, relocate group selector/create/join, add favorite group

## Itération 1 — 2026-08-29

- **Fait directement** (bug bien identifié + demande de redesign avec suggestions concrètes de l'utilisateur)
- **Verdict** : ✅ validé, vérifié en direct (dev stack réel, tests Go + Jest, multi-groupes, dark mode)

### 1. Bug : le nom des équipes affiche la couleur au lieu du nom défini à la création

Root cause trouvée par lecture de code : `MatchesPanel.vue` (2 endroits : la carte horizontale de la liste de matchs, et l'en-tête du tableau de détails) et `MatchDetails.vue` (l'en-tête d'onglet équipe) interpolaient littéralement `{{ team.Colour }}` comme libellé, au lieu de `{{ team.Name }}` — alors que le backend renvoie bien les deux champs séparément depuis la feature "équipes nommées et colorées" d'une session précédente. La pastille de couleur (`getTeamColor(team.Colour)`) était correcte ; seul le texte à côté était faux. Corrigé aux 3 endroits. Vérifié en direct : création d'un groupe avec équipes "Les Rouges"/"Les Bleus" (couleurs hex personnalisées), un match dans ce groupe affiche bien "Les Rouges"/"Les Bleus" partout (liste, détails, onglets), plus la pastille de couleur correcte à côté.

### 2. Sélecteur de groupe déplacé, création/join déplacés, ancien sélecteur navbar supprimé

Diagnostic confirmé : le sélecteur de groupe (`GroupSwitcher.vue`, popover dans la navbar) ne scope en réalité que Matches/Standings — rien d'autre dans l'app ne lit `activeGroupId`. Le garder global dans la navbar n'avait donc plus de sens une fois cette portée clarifiée.

- **Sélecteur** : déplacé dans `MatchesAndStandings.vue`, dans la même barre que le sélecteur de saison (`.controls-bar`) — un `<select>` simple, au même niveau visuel que "Season". Bascule via `setActiveGroupId` + `window.location.reload()`, exactement le même contrat qu'avant. Un joueur sans groupe voit un texte d'invite ("No group yet — join or create one from your Profile") plutôt qu'un menu vide.
- **Création et jointure** : déplacées dans `Profile.vue`, section "Your groups" — deux petites icônes en haut à droite de l'en-tête (une flèche "entrer" pour join, un "+" pour create), ouvrant respectivement le nouveau `JoinGroupModal.vue` (juste un champ code d'invitation) et `CreateGroupModal.vue` (inchangé dans son contenu). Visibles même à 0 groupe, puisque c'est justement là qu'on en a besoin.
- **`GroupSwitcher.vue` supprimé** entièrement. `App.vue` perd tout l'état lié aux groupes qui ne servait plus qu'à l'alimenter (`groups`, `activeGroupId`, `showGroupSwitcher`, `refreshGroups()`).
- Petit nettoyage induit : `CreateGroupModal.vue` recevait `isDarkMode` en prop depuis `App.vue` via `GroupSwitcher.vue` (chaîne de props maintenant cassée puisque `GroupSwitcher` disparaît). Remplacé par un helper partagé `src/services/theme.js` (`isDarkModeActive()`, interroge `document.querySelector('.dark-mode')`) utilisé par les 3 modales teleportées (`CreateGroupModal`, `JoinGroupModal`, `GroupSettingsModal`), qui n'ont plus besoin de prop-threading pour connaître le thème.

### 3. Groupe favori

Nouvelle fonctionnalité backend + frontend, comme demandé : "le premier groupe que tu intègres est ton favori, tu ne peux pas avoir de non favori", sélection via une étoile dans Profile → Your groups, et c'est ce groupe qui devient le groupe actif par défaut à la connexion.

- **Backend** : `GroupMembership.IsFavorite bool` (nouvelle colonne). `AddPlayerToGroupWithRole` compte les memberships existants du joueur avant insertion — zéro ⇒ ce nouveau membership devient favori automatiquement (couvre création de groupe, jointure par code, et le flow admin "ghost player"). Nouvel endpoint `PATCH /groups/:id/favorite` (self-service, juste `requireGroupMemberByPathID` — pas admin) → `GroupMembershipService.SetFavoriteGroup`, qui débascule l'ancien favori et bascule le nouveau dans une transaction. `LeaveGroup` et `RemoveMember` réassignent automatiquement le favori (au membership le plus ancien restant) si celui qui vient d'être supprimé le portait — sinon un joueur pourrait se retrouver sans aucun favori après un départ/retrait. Backfill ponctuel dans `connect.go` (même style que le renommage `owner`→`admin`) pour les groupes déjà en base, qui récupèrent `is_favorite=false` par défaut d'`AutoMigrate` : promeut le membership le plus ancien de chaque joueur qui n'a encore aucun favori, idempotent.
- **Frontend** : `GET /groups/me` renvoie maintenant `is_favorite` par groupe. `resolveActiveGroup()` (`activeGroup.js`) : le fallback (quand rien n'est stocké localement, ou que l'id stocké ne correspond plus à rien — typiquement juste après connexion) devient "le groupe favori" plutôt que "le premier groupe de la liste" ; `groups[0]` reste un ultime filet de sécurité si aucun favori n'était marqué. Dans `Profile.vue`, chaque carte de groupe a une étoile : pleine et désactivée si c'est déjà le favori, contour cliquable sinon (`setFavoriteGroup` → recharge `groupMeta` pour mettre à jour les deux étoiles concernées).

### Vérifications faites

- **Go** : `go build`/`go vet`/`gofmt -l .` verts. 4 nouveaux tests d'intégration (`favoritegroup_integration_test.go`) : premier groupe devient favori automatiquement (et le second non) ; `SetFavoriteGroup` bascule bien l'ancien vers le nouveau ; `LeaveGroup` et `RemoveMember` réassignent le favori à un groupe restant quand celui supprimé le portait. Suite complète (`go test ./... -count=1`) verte, aucune régression sur les tests existants (rôles, invite, etc.) malgré le changement de signature interne d'`AddPlayerToGroupWithRole`/`RemoveMember`.
- **Frontend** : `npm run lint` / `npm run test:unit` (19/19, dont un nouveau test du fallback favori dans `activeGroup.spec.js`) / `npm run build` : tous verts.
- **Navigateur réel (Playwright)**, dev stack + reseed :
  - Bug équipes : groupe "Sunday Crew" créé avec équipes "Les Rouges"/"Les Bleus" (couleurs hex), match créé dedans → capture d'écran confirmant l'affichage correct partout (liste, détails, onglets).
  - Sélecteur navbar : `.group-switcher` absent après connexion ; sélecteur de groupe présent dans Matches, même niveau que Season, options correctes (`Default`, `Sunday Crew`), bascule fonctionnelle (vérifié via capture après sélection de `Sunday Crew`).
  - Profile : icônes join/+ présentes et fonctionnelles (modales s'ouvrent, clic sur une carte pour déplier le roster fonctionne toujours malgré `@click.stop` sur étoile/engrenage) ; étoile pleine sur "Default" au départ, clic sur l'étoile de "Sunday Crew" bascule correctement (capture avant/après) ; **reconnexion dans un contexte navigateur totalement neuf** (aucun localStorage) atterrit bien sur "Sunday Crew" (le nouveau favori) sur la page Matches — preuve de bout en bout de la fonctionnalité demandée.
  - Réassignation de favori testée via API directement (script curl) : nouveau joueur rejoint Default (devient favori) puis Sunday Crew ; admin retire ce joueur de Default → Sunday Crew devient automatiquement son nouveau favori.
  - Non-admin (`mattheo@mail.com`) : étoile présente et cliquable (préférence personnelle, pas admin-only) mais désactivée sur son seul groupe ; icônes join/+ toujours visibles ; aucune icône réglages (admin-only, inchangé).
  - Dark mode vérifié sur les 3 modales teleportées.
- Données de test nettoyées, base de dev reseedée (`-reset`) après vérification.

### Aucune itération de correction nécessaire.
