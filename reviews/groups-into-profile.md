# Rethink group management: fold Groups.vue into Profile.vue

## Itération 1 — 2026-08-29

- **Fait directement** (une seule direction structurelle, choisie par l'utilisateur après un brainstorm à plusieurs propositions)
- **Verdict** : ✅ validé, vérifié en direct dans le navigateur (dev stack réel, 2 groupes + joueurs fantômes de test)

### Contexte

L'utilisateur trouvait que l'onglet "Groups" (récemment recentré sur le seul groupe actif) ne servait plus à grand-chose, et proposait de déplacer la gestion de groupe dans le Profil : stats globales en haut, puis une liste horizontale de groupes avec les stats propres à chacun, et au clic la liste des membres (scrollable) avec possibilité pour un admin de retirer un joueur. Problème soulevé par l'utilisateur lui-même : ça ne laisserait qu'un seul onglet "Matches" dans la nav, ce qui serait moche.

Un brainstorm à 3 propositions a été présenté (A: tout dans Profile + Profile promu en onglet à part entière ; B: garder un onglet "Group settings" pointu ; C: tout fusionner en une seule page). L'utilisateur a choisi **A**, avec le code d'invitation et la gestion des équipes dans **un mini-dialog séparé** plutôt que dans le même volet que les membres.

### Ce qui a été fait

1. **Bug corrigé au passage** : la modal "Create a group" ignorait le dark mode. Cause : elle est `<Teleport to="body">`'d (pour échapper au `backdrop-filter` de la navbar), ce qui la sort du sous-arbre de `#app` et lui fait perdre l'héritage de la classe `.dark-mode`. Fix : `isDarkMode` transmis en prop (`App.vue` → `GroupSwitcher.vue` → `CreateGroupModal.vue`), réappliqué directement sur la racine téléportée.
2. **`GroupSettingsModal.vue`** (nouveau) : reprend exactement ce que "Show invite code" et "Manage teams" faisaient dans l'ancien `Groups.vue` (code d'invitation + édition nom/couleur des équipes), sous forme de dialog séparé plutôt que dans le volet des membres. Ouvert depuis une icône réglages (engrenage) sur la carte de groupe, admin-only. Même souci de dark mode que `CreateGroupModal` mais résolu différemment : comme ce composant est ouvert depuis une vue routée (pas de chemin de prop direct vers `App.vue`), il lit directement `document.querySelector('.dark-mode') !== null` — **pas** `getElementById('app')`, qui s'est révélé peu fiable (voir bug ci-dessous).
3. **`Profile.vue` réécrit** : garde la carte de stats globales inchangée, puis ajoute une carte "Your groups" avec un carousel horizontal (même pattern que la liste de matchs) — un badge Admin par groupe où c'est le cas, stats (Pts/Played/Goals) propres au groupe, icône réglages admin-only. Cliquer une carte déplie en dessous la liste des membres (scrollable, fetch-on-demand mise en cache par groupe) avec, repris tel quel de l'ancien `Groups.vue` : badge "No account yet" + bouton Invite pour un joueur fantôme, promote/demote, remove — tous strictement gated sur le rôle admin **du groupe concerné** (pas juste "admin de n'importe quel groupe").
4. **`App.vue`** : suppression du lien nav "Groups" ; le lien "Profile" (auparavant une icône seule dans `nav-actions`) devient un onglet à part entière dans `.nav-menu`, avec libellé — la nav garde ainsi deux vrais onglets ("Matches"/"Profile") au lieu d'un seul, réglant l'inquiétude de l'utilisateur.
5. **`router/index.js`** : `/groups` devient une redirection vers `/profile` (même traitement que `/standings` → `/` plus tôt dans la session), pour qu'un ancien lien ne tombe pas sur une 404.
6. **`Groups.vue` supprimé** entièrement (`git rm`).
7. **`CLAUDE.md`** mis à jour : section Frontend réécrite pour refléter la nouvelle architecture (routes, nav, `GroupSwitcher`/`CreateGroupModal` — qui n'étaient déjà plus documentés depuis une feature précédente de cette session — `Profile.vue`, `GroupSettingsModal.vue`), plus quelques corrections de références obsolètes (`TeamColourPicker` test description, mention du bouton "Create Match" désormais une carte "+").

### Bug trouvé et corrigé pendant la vérification

En testant le dark mode sur `GroupSettingsModal`, la modale restait claire alors que le reste de la page était bien sombre. Diagnostic : `document.getElementById('app')` renvoyait un élément **différent** de la vraie racine Vue — `public/index.html` définit déjà `<div id="app"></div>` comme conteneur de montage, et Vue 3 ne le remplace pas : il imbrique son propre rendu (qui porte, lui, la classe `.dark-mode`) à l'intérieur. Deux éléments partagent donc le même `id`, et `getElementById` renvoie le premier (le conteneur statique, toujours sans classe). Corrigé en interrogeant par classe (`document.querySelector('.dark-mode')`) plutôt que par id — fiable puisqu'un seul élément du document porte jamais cette classe. Ce doublon d'id est une particularité préexistante et inoffensive pour tout le reste de l'app (les sélecteurs CSS `#app`/`.dark-mode` fonctionnent quand même, puisqu'ils s'appliquent par correspondance, pas par unicité) — non corrigé à la racine, hors périmètre de cette feature.

### Vérifications faites (Playwright, dev stack réel)

- Créé un deuxième groupe ("Sunday Crew", thibaut admin) + un joueur fantôme ("Ghosty") via l'API pour tester le carousel à 2 groupes et le flux d'invitation.
- Nav : "Matches"+"Profile" visibles connecté, aucun lien visible déconnecté (page `/login`) ; `/groups` redirige bien vers `/profile`.
- Vue admin : carousel affichant les 2 groupes avec badge Admin et stats propres ; clic sur une carte surligne (bordure verte) et déplie la liste des membres du **bon** groupe ; clic sur l'icône réglages d'une carte ouvre bien le dialog du **bon** groupe (testé en ouvrant "Sunday Crew" pendant que "Default" était déplié, sans confusion entre les deux) ; invitation d'un joueur fantôme testée de bout en bout (badge et bouton disparaissent après rechargement, une fois l'email renseigné).
- Vue non-admin (`mattheo@mail.com`) : aucune icône réglages, aucun bouton d'action sur les membres, badge "No account yet" toujours visible en lecture seule.
- Dark mode : `CreateGroupModal` et `GroupSettingsModal` suivent tous deux correctement le thème après le fix.
- Mobile (390px) : carousel scrollable horizontalement, nav repliée normalement.
- `npm run lint` / `npm run test:unit` (18/18) / `npm run build` : tous verts.
- Données de test nettoyées, base de dev reseedée (`-reset`) après vérification.

### Aucune itération de correction nécessaire (au-delà du bug dark-mode trouvé et corrigé pendant cette même itération).
