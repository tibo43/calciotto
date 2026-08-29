# Sélecteur de groupe en popover + création/adhésion déplacées hors de Groups.vue

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet
- **Verdict** : problème détecté et corrigé, ✅ validé après correction

### Ce qui a été fait

- Nouveau `GroupSwitcher.vue` remplaçant le `<select>` de la nav : bouton déclencheur (nom du groupe actif + chevron), panneau listant les groupes (clic = même comportement que l'ancien `changeActiveGroup`, `setActiveGroupId` + reload complet), séparateur, mini-formulaire "Join with a code" inline, bouton "+ Create a group" ouvrant une modale. Fermeture au clic extérieur et à Escape, listeners nettoyés dans `beforeUnmount`.
- Nouveau `CreateGroupModal.vue` reprenant exactement le formulaire de création existant (nom + 2 équipes avec `TeamColourPicker`), sans changer la validation ni l'appel API.
- `Groups.vue` recentré sur la seule liste "Groups you belong to" (code d'invitation + gestion des équipes) — formulaires création/adhésion et tout leur état retirés proprement (vérifié par grep : aucune référence résiduelle à `newGroupName`/`submitCreate`/etc.).
- **Bon réflexe non demandé explicitement mais nécessaire** : le sélecteur s'affiche maintenant même pour un joueur avec zéro groupe (avant : `v-if="groups.length"` le masquait complètement) — referme un vrai angle mort d'onboarding déjà repéré en session : un nouveau compte sans groupe n'avait auparavant aucun moyen visible de rejoindre/créer un groupe depuis la nav.
- Après création ou adhésion, l'utilisateur est maintenant automatiquement basculé sur le nouveau groupe (`setActiveGroupId` + reload) plutôt que de rester sur l'ancien groupe actif — un choix de design que j'avais explicitement laissé à la discrétion de la teammate dans le prompt, et qui est le bon par défaut.
- `App.vue.refreshGroups()` simplifié : suppression de l'heuristique `from.name === 'Groups'` (obsolète maintenant que création/adhésion passent aussi par un reload complet, comme le changement de groupe).

### Bug trouvé en revue (corrigé, une itération)

**Le modal "Create a group" s'affichait presque entièrement hors écran.** Cause précise, confirmée via `getBoundingClientRect()` dans un vrai navigateur : `.top-navbar` a un `backdrop-filter: blur(12px)`, qui — comme `filter` — crée un nouveau *containing block* pour tout descendant en `position: fixed`. `CreateGroupModal` est maintenant instancié à l'intérieur de `GroupSwitcher`, lui-même dans `.nav-actions`, donc à l'intérieur de la navbar — son `.modal-overlay` (`position: fixed; inset: 0`) se positionnait alors par rapport à la boîte de la navbar (69px de haut) au lieu du viewport réel, poussant le modal centré (644px de haut) à environ 287px au-dessus de l'écran visible. Ce piège CSS n'affectait pas le modal déjà existant dans `MatchDetails.vue`, qui n'est pas imbriqué dans la nav.

**Fix demandé et appliqué** : `<Teleport to="body">` autour de la racine du template de `CreateGroupModal.vue` — solution native Vue 3, aucune dépendance ajoutée, échappe complètement au containing block de la navbar quel que soit l'endroit du DOM où le composant est instancié.

### Vérifications faites en revue

- Relecture des 4 fichiers concernés (2 nouveaux composants + 2 modifiés) : logique cohérente, aucune référence orpheline.
- `npm run lint` / `npm run build` : verts, vérifiés indépendamment après le fix.
- **Test réel de bout en bout (Playwright, indépendant du script de la teammate)** :
  - Connexion fraîche (sans token) → sélecteur absent avant login, apparaît **immédiatement après connexion sans rechargement de page** (confirme que le fix de réactivité `showGroupSwitcher` — trouvé et corrigé par la teammate elle-même en cours de route, un vrai piège Vue de court-circuit dans un computed cassant le tracking de dépendance — fonctionne réellement).
  - Ancien `<select class="group-select">` bien absent.
  - Ouverture/fermeture du panneau (clic extérieur, Escape) : les deux fonctionnent.
  - **Bug du modal confirmé puis sa correction vérifiée** : bounding box du modal entièrement dans le viewport après le `Teleport`, capture d'écran inspectée visuellement — rendu complet, propre.
  - Création réelle d'un groupe via la modale → bascule automatique dessus, confirmé par le nom affiché dans le déclencheur du switcher.
  - `/groups` confirmé sans formulaires de création/adhésion.
  - Aucune erreur console sur l'ensemble du parcours.
  - Base de dev remise à zéro (`-reset`) après mes propres tests (qui ont réellement créé un groupe).

### Itération 2 — après correction
Corrigé exactement comme demandé, aucun problème résiduel. Validé.
