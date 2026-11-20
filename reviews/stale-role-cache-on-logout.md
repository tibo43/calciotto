# Fix : cache de rôle/groupe obsolète après changement de compte sans refresh

## Itération 1 — 2026-08-29

- **Modèle** : Haiku (fix isolé, deux points de code, sans ambiguïté de conception)
- **Verdict** : ✅ validé, aucun problème détecté

### Contexte

Bug signalé par l'utilisateur : en se connectant en admin puis en se déconnectant pour se reconnecter avec un compte non-admin (sans rafraîchir la page), les boutons réservés aux admins restaient visibles (et inversement dans l'autre sens) jusqu'à un F5 forcé.

### Diagnostic (posé avant de lancer la teammate)

`activeGroup.js` met en cache la promesse de `GET /groups/me` au niveau du **module** (`myGroupsPromise`), pour que plusieurs composants d'une même page ne fassent qu'une seule requête — sûr uniquement parce que le groupe actif ne change normalement que via un rechargement complet de page (le sélecteur de groupe fait un `window.location.reload()`). Se déconnecter/reconnecter ne recharge pas la page, donc ce cache mémoire survivait au changement de compte et renvoyait les groupes/rôles de l'ancien utilisateur au nouveau.

### Ce qui a été fait

- Nouvelle fonction exportée `clearMyGroupsCache()` dans `activeGroup.js` (remet `myGroupsPromise` à `null`), avec un commentaire expliquant pourquoi (transition logout/login non couverte par l'invariant "ne change que via un reload complet").
- Appelée dans `logout()` (`App.vue`), juste à côté de `clearActiveGroupId()` déjà existant.
- Correction minimale et chirurgicale : 2 fichiers, ~7 lignes au total, aucune réécriture de la stratégie de cache elle-même.

### Vérifications faites en revue

- Relecture du diff : exactement le fix demandé, rien de plus.
- `npm run lint` / `npm run build` : verts, vérifiés indépendamment.
- **Reproduction réelle en navigateur (Playwright, indépendante du script de la teammate)**, dans un seul et même onglet sans aucun rechargement de page entre les étapes :
  - Connexion admin (`thibaut@mail.com`) → "Create Match" visible.
  - Déconnexion → connexion `marcos@mail.com` (membre simple), **sans reload** → "Create Match", "Save Changes" et "Delete Match" bien **absents**.
  - Déconnexion → reconnexion `thibaut@mail.com`, **sans reload** → "Create Match" réapparaît bien.
  - Aucune erreur console sur l'ensemble du scénario.
- La teammate avait elle-même d'abord reproduit le bug sur le code non corrigé (boutons admin visibles à tort pour le compte non-admin) avant d'appliquer le fix et de confirmer sa résolution — bonne discipline, pas de correction appliquée à l'aveugle.

### Aucune itération de correction nécessaire.
