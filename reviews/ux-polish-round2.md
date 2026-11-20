# UX polish round 2: ghost-player invite, decluttered Matches page, simplified colour picker, Groups scoped to active group

## Itération 1 — 2026-08-29

- **Fait directement** (5 demandes ciblées, pas de teammate)
- **Verdict** : ✅ validé, vérifié en direct dans le navigateur (dev stack réel + reseed)

### Demandes de l'utilisateur et ce qui a été fait

1. **Inviter un joueur "fantôme" par email depuis la page Groups.**
   Le backend le supportait déjà (`POST /groups/:id/members/:playerId/invite`, `AuthService.InviteExistingPlayer`) mais n'était câblé nulle part côté frontend. Ajout de `invitePlayer(groupId, playerId, email)` dans `api.js`. Dans `Groups.vue`, chaque ligne de membre sans email (`!member.email` — le champ existe déjà en JSON via `PlayerWithRole` embarquant `Player`, `omitempty`) affiche un badge "No account yet" et, pour un admin, un bouton "Invite" qui déplie un mini-formulaire (email + Send invite + Cancel). Sur succès, la liste des membres est rechargée — le badge et le bouton disparaissent puisque `member.email` est maintenant renseigné. Le message de succès précise explicitement qu'aucun envoi d'email réel n'est câblé dans cet environnement (le lien est seulement loggé côté serveur), pour éviter toute confusion.

2. **UI Matches trop fouillie (saison / onglets / create match dans 3 divs séparées).**
   `MatchesAndStandings.vue` : fusion de `.season-bar` et `.sub-tabs-bar` en une seule `.controls-bar` (onglets à gauche, sélecteur de saison à droite, empilés verticalement en mobile). `MatchesPanel.vue` : suppression de la toolbar séparée "Create Match" ; le bouton devient une carte "+" en pointillés à la fin de la liste de matchs elle-même (`.add-match-card`, admin-only), au lieu d'un bouton au-dessus. L'état vide (aucun match) garde son propre CTA "Create Your First Match", inchangé — il n'y a alors rien à intégrer dans une liste vide.

3. **Simplifier le colour picker (retirer la grille de 10 préréglages).**
   `TeamColourPicker.vue` réécrit : une seule pastille (fond = couleur actuelle, ou dégradé arc-en-ciel si vide) avec l'`<input type="color">` natif du navigateur superposé en transparent — "juste le global qui permet de choisir n'importe quelle couleur". Utilisé aux deux endroits existants (`CreateGroupModal.vue` et le formulaire d'édition d'équipe de `Groups.vue`) sans changement supplémentaire, puisque c'est le même composant partagé. Les tests unitaires du composant ont été entièrement réécrits pour matcher la nouvelle structure (6 tests : rendu, valeur reflétée, fond par défaut vs personnalisé, émission au changement, désactivation).

4. **Modal "Create a group" cassée / hors thème, textboxes d'équipe trop petites.**
   Diagnostic en direct (Playwright) : la modal utilise déjà les classes globales du thème (`.modal-overlay`, `.form-input`, etc. dans `assets/global-styles.css`, bien importées) — le vrai problème était que la grille de 10 pastilles de l'ancien colour picker prenait toute la largeur disponible et wrappait sur 2 lignes, écrasant l'input du nom d'équipe. La simplification du point 3 résout ce problème sans changement supplémentaire — capture d'écran après coup confirmant une mise en page propre et cohérente avec le thème.

5. **La page Groups doit refléter le groupe actif (sélecteur en haut à droite), pas "tous mes groupes".**
   Réécriture de `Groups.vue` : suppression de la boucle sur `groups` (state auparavant garder en dictionnaires `groupId -> ...`) au profit d'un seul groupe résolu une fois via `resolveActiveGroup()` (même pattern que `MatchesAndStandings.vue`) — `activeGroupId`, `groupName`, `isAdmin` deviennent des valeurs plates. Le titre de page devient le nom du groupe actif lui-même. Changer de groupe se fait toujours via le sélecteur en haut à droite (rechargement complet de page), donc cette page n'a jamais besoin de re-résoudre en cours de session. État vide ("No group selected") si le joueur n'appartient à aucun groupe, avec renvoi vers le sélecteur. Toute la logique métier (code d'invitation, gestion d'équipes, membres, et maintenant l'invitation ghost-player) est conservée à l'identique, juste dé-dictionnarisée puisqu'il n'y a plus qu'un seul groupe à la fois.

### Vérifications faites (Playwright, dev stack réel, reseed avant/après)

- Colour picker : 18/18 tests unitaires passent (6 réécrits + 11 `activeGroup` inchangés) ; `npm run lint`/`build` verts.
- Page Matches : capture admin montrant la barre fusionnée (onglets + saison) et la carte "+" en fin de liste (scrollée jusqu'à elle) ; clic sur la carte "+" ouvre bien la modale de création ; carte absente pour un non-admin (`mattheo@mail.com`) ; mobile (390px) : la barre empile proprement onglets puis saison.
- Create Group modal : capture après simplification du colour picker — mise en page propre, aucun débordement, cohérente avec le thème.
- Groups.vue : capture admin montrant directement "Default" (le groupe actif) sans liste ; création de deux joueurs fantômes via l'API (`Ghosty`, `Ghosty2`) pour tester le flux d'invitation de bout en bout — badge "No account yet" + bouton "Invite" visibles avant, formulaire d'email fonctionnel, message de succès affiché, lien de reset loggé côté serveur (`docker logs devops-backend-1`) confirmant l'appel réel à `InviteExistingPlayer`, badge/bouton disparus après rechargement de la liste (email désormais renseigné). Vue non-admin (`mattheo@mail.com`) : bouton "Team" en lecture seule, aucun bouton Invite/Make admin/Remove nulle part (vérifié avec un sélecteur précis après qu'un premier test au texte trop large ait donné un faux positif à cause de "Show invite code" contenant "invite"), badge "No account yet" toujours visible en lecture seule sur Ghosty2.
- Joueurs fantômes de test supprimés puis DB reseedée (`-reset`) après vérification.
- `npm run lint` / `npm run test:unit` (18/18) / `npm run build` : tous verts après l'ensemble des changements.

### Aucune itération de correction nécessaire.
