# Fix remaining role-based UI gaps (non-admin controls, login-page shell)

## Itération 1 — 2026-08-29

- **Fait directement** (pas de teammate — bugs bien scopés, un seul fichier chacun)
- **Verdict** : ✅ validé, vérifié en direct dans le navigateur

### Rapporté par l'utilisateur

1. Un membre non-admin voit le bouton "Edit Match" (avec icône crayon) sur la page Matches, et voit "Manage teams" sur la page Groups avec des champs nom/couleur éditables et un bouton Save — qui échoue silencieusement (403 backend) puisqu'il n'est pas admin.
2. La page `/login` ne se ressent pas comme une vue autonome.

### Cause et correctif

1. **`MatchesPanel.vue`** : le lien vers `/matches/:id/edit` n'a jamais été conditionné sur `isAdmin` (seuls les boutons "Create Match" l'étaient). La page de destination (`MatchDetails.vue`) reste volontairement consultable par tout membre — seuls ses propres contrôles d'édition sont gated — donc le lien ne doit pas être caché, juste correctement étiqueté : "Edit Match" + icône crayon pour un admin, "View Match" + icône œil pour un non-admin.
2. **`Groups.vue`** : le bloc "Manage teams" était un gap documenté explicitement dans `CLAUDE.md` ("There is still no role-based UI hiding on 'Manage teams'"). Corrigé en réutilisant `isGroupAdmin(groupId)` (déjà utilisé pour la gestion des membres) : le libellé du bouton devient "Team"/"Hide team" pour un non-admin, "Manage teams"/"Hide teams" pour un admin ; le contenu déplié devient une liste en lecture seule (pastille couleur + nom, aucun input) pour un non-admin au lieu du formulaire éditable.
3. **`App.vue`** : la navbar affichait toujours les liens "Matches"/"Groups" et les icônes Profile/Logout, même sur les routes publiques (`/login`, `/signup`, `/forgot-password`, `/reset-password`) où ils n'ont aucun sens (pas de session à agir dessus). Ajout d'un computed `isAuthenticatedRoute` (négation de la même liste `PUBLIC_ROUTE_NAMES` déjà utilisée par `refreshGroups()`) qui gate ces éléments ainsi que le bouton de menu mobile. Le thème toggle et le logo restent toujours visibles (préférence cosmétique, sans rapport avec l'auth).

### Vérifications faites (Playwright, dev stack réel, DB reseedée)

- Visite de `/` sans token → redirection vers `/login`, navbar réduite à juste "Calciotto" + thème (plus de Matches/Groups/Profile/Logout) — capture d'écran inspectée.
- Connexion en `mattheo@mail.com` (non-admin) : lien de match → texte exact "View Match" avec icône œil ; page Groups → bouton exact "Team", dépliage → 2 lignes en lecture seule (pastille + nom), 0 input éditable, 0 bouton Save.
- Connexion en `thibaut@mail.com` (admin) : lien de match → "Edit Match" avec icône crayon (inchangé) ; page Groups → bouton "Manage teams", dépliage → formulaire éditable complet (2 inputs, 2 color pickers, 2 boutons Save) — inchangé.
- Aucune erreur console sur l'ensemble des parcours testés.
- `npm run lint` / `npm run test:unit` (18/18) / `npm run build` : tous verts.
- Base de dev remise à zéro (`-reset`) après les tests.

### Aucune itération de correction nécessaire.
