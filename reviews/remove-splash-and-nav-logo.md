# Suppression de l'écran de démarrage et du logo de la nav

## Itération 1 — 2026-08-29

- **Modèle** : Haiku (changement mécanique, purement suppression, sans ambiguïté)
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- Bloc `.initial-view` ("Click to enter") entièrement retiré de `App.vue`, ainsi que `showImage`/`toggleView()` et toute la logique/CSS associée (~130 lignes de CSS retirées).
- `.app-container` rendu inconditionnel — plus de garde `v-if`.
- Aucune modification du routeur nécessaire : le guard `beforeEach` existant redirige déjà toute route non publique vers `/login` en l'absence de token, donc une fois l'écran de démarrage retiré, un visiteur non connecté atterrit directement sur `/login` comme demandé.
- Image du logo (`logo.png`) retirée de la barre de nav, ne reste que le texte "Calciotto" (comportement de clic vers l'accueil conservé).
- Fichiers image (`campo.jpg`, `logo.png`) laissés sur le disque, juste déréférencés — cohérent avec la consigne de ne pas toucher aux assets.

### Vérifications faites en revue

- Relecture du diff : suppression propre, rien d'orphelin laissé (`grep` sur `showImage`, `toggleView`, `campo.jpg`, `logo.png`, `nav-logo` : aucune occurrence restante, vérifié indépendamment).
- `npm run lint` / `npm run build` : verts, vérifiés indépendamment.
- **Test réel en navigateur (Playwright, indépendant du script de la teammate)** : visite fraîche de `/` (contexte non authentifié) → redirection directe vers `/login`, aucun écran de démarrage, aucune image de logo dans la nav, texte "Calciotto" seul. Connexion réussie, application fonctionnelle ensuite, aucune erreur console. Captures d'écran prises et inspectées visuellement — rendu propre.

### Aucune itération de correction nécessaire.
