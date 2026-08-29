# Fusion Matches + Standings en une page avec sélecteur de saison partagé

## Itération 1 — 2026-08-29

- **Modèle** : Opus (plus grosse restructuration frontend de la session, risque de régression élevé si mal géré)
- **Verdict** : ✅ validé, aucun problème détecté — aucune itération de correction nécessaire

### Ce qui a été fait

- **Backend** : `MatchService.GetMatchesDetails` gagne un paramètre `season`, filtré en Go via `FilterMatchesBySeason` (déjà existant, réutilisé tel quel — pas de nouvelle logique de filtrage). Les **4 points d'appel** existants ont été mis à jour correctement : `MatchHandler.GetMatchesDetails` passe la vraie saison de la query string, et les **3** appels internes de `standings.go` (`GetPointsStandings`, `GetScorers`, et `GetSeasons` — ce dernier n'était pas nommé explicitement dans mon prompt, bon réflexe de l'avoir trouvé par grep comme demandé) passent `""` explicitement pour préserver exactement leur comportement actuel (ils filtrent déjà eux-mêmes après coup). Tests étendus sur le test de saisons déjà existant plutôt que dupliqués.
- **Frontend** : décomposition propre plutôt qu'un fichier monstre unique — `MatchesAndStandings.vue` (nouvelle page racine, route `/`) possède l'état partagé (groupe actif, rôle admin, saison, saisons disponibles, onglet actif) et compose trois enfants : `MatchesPanel.vue` (renommage de `MatchesAll.vue`, devenu un composant enfant recevant `activeGroupId`/`isAdmin`/`season` en props, avec un watcher sur `season` qui recharge les matchs), `PointsStandingsTable.vue` et `ScorersTable.vue` (extraits quasi verbatim de l'ancien `Standings.vue`, y compris le tag "(left the group)" de la feature précédente).
- Bon soin du détail visuel : le bouton "Create Match" a été correctement déplacé de l'ancien header en dégradé (où il avait un style translucide spécifique) vers une toolbar simple sur fond neutre, avec un style de bouton adapté en conséquence plutôt que collé tel quel dans un contexte visuel différent.
- `/standings` devient une redirection vers `/` plutôt que supprimée sans rien — un ancien lien/favori ne tombe pas sur une 404.
- Nav simplifiée : plus qu'un seul lien "Matches" (en plus de "Groups"), cohérent avec la fusion.
- Toutes les références à l'ancien nom de route (`MatchesAll`/`Standings`) mises à jour partout (`App.vue`, y compris `isRouterRoute` et le watcher de route) — vérifié par grep qu'il n'en reste aucune trace, à part le libellé d'affichage "Standings" du sous-onglet Points (normal, ce n'est pas une référence technique).

### Vérifications faites en revue

- Relecture complète de tous les diffs (backend : 2 fichiers + tests ; frontend : 1 nouvelle page, 1 renommage, 2 nouveaux composants de tableau, router, nav, 4 fichiers avec juste des commentaires à corriger) — cohérent de bout en bout, aucune régression identifiée à la lecture.
- `go build ./...` / `go vet ./...` / `gofmt -l .` : verts. `go test ./... -count=1` contre la vraie base de dev : tous les packages passent, y compris les 4 nouvelles assertions sur le filtre de saison des matchs. `npm run lint` / `npm run build` : verts. Tout vérifié indépendamment, pas seulement sur la base du rapport de la teammate.
- **Test réel de bout en bout (navigateur, Playwright, indépendant du script de la teammate)** :
  - `/` : nav réduite à "Matches"/"Groups", sélecteur de saison présent, 3 sous-onglets (Matches/Points/Scorers), bouton "Create Match" visible pour l'admin, 6 cartes de match.
  - Bascule vers Points → 12 lignes ; vers Scorers → 12 lignes.
  - Visite directe de `/standings` → redirige bien vers `/`.
  - Connexion en membre simple → "Create Match" absent, mais les matchs restent visibles (lecture toujours ouverte à tout membre).
  - Requêtes réseau capturées et inspectées : `season=2025-2026` correctement propagé sur `/matches/details`, `/standings/points` et `/standings/scorers` — les trois onglets partagent bien la même saison sélectionnée.
  - Capture d'écran du sous-onglet Matches inspectée visuellement : rendu propre et cohérent, aucune régression visuelle perceptible par rapport à l'ancienne page.
  - Aucune erreur console sur l'ensemble du parcours.
  - Base de dev remise à zéro (`-reset`) après mes propres tests (un match de test créé par la teammate pendant sa vérification avait déjà été nettoyé par ce même reseed).

### Aucune itération de correction nécessaire.
