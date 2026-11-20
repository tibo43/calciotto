# Colour picker pour les équipes (remplace la liste de couleurs fixe)

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet (jugement nécessaire sur la conversion mot-clé legacy → hex, au-delà du simple mécanique)
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- `Groups.vue` : les deux `<select>` (formulaire de création + "Manage teams") remplacés par `<input type="color">`. `TEAM_COLOUR_OPTIONS` et sa data property supprimées (plus aucune référence après grep).
- **Point le plus délicat correctement traité** : les équipes créées avant ce changement stockent encore un mot-clé (`black`, `white`, éventuellement les 8 autres) et non un hex — un `<input type="color">` alimenté directement avec `"black"` afficherait un rendu cassé/par défaut. La teammate a ajouté `toHexColour()` (même palette que `getTeamColor`, retourne la valeur telle quelle si elle commence par `#`, sinon lookup, sinon gris de repli `#6b7280`) et l'applique à la volée quand "Manage teams" charge les équipes existantes. Testé et confirmé fonctionnel (voir plus bas) — sans ça, éditer une équipe seedée aurait été cassé silencieusement.
- Valeurs par défaut du formulaire de création changées de `'black'`/`'white'` vers les hex exacts déjà utilisés par `getTeamColor` (`#1f2937`/`#f8fafc`) pour préserver le rendu par défaut à l'identique.
- `getTeamColor` dans `MatchesAll.vue` ET `MatchDetails.vue` (dupliqué dans les deux fichiers, mis à jour dans les deux) : passthrough direct si la valeur commence par `#`, sinon comportement inchangé (lookup + repli gris) — assure que les anciennes équipes (mot-clé) et les nouvelles (hex) s'affichent correctement sans migration de données.
- Aucune sur-ingénierie : pas de nouveau module partagé pour la palette, cohérent avec la duplication déjà existante entre les fichiers concernés.

### Vérifications faites en revue

- Relecture du diff (3 fichiers) : cohérent, correspond exactement au périmètre demandé.
- `npm run lint` / `npm run build` : verts, vérifiés indépendamment.
- **Test réel en navigateur (Playwright, indépendant du script de la teammate)** : plus aucun `<select class="team-colour-select">` sur la page ; 2 `<input type="color">` dans le formulaire de création ; ouverture de "Manage teams" sur le groupe "Default" (équipes stockées `black`/`white` en base) → les deux color pickers affichent bien `#1f2937` et `#f8fafc`, pas de noir cassé par défaut. Aucune erreur console.
- Base de dev vérifiée propre après les tests de la teammate (`teams` toujours `Black/black`, `White/white`).

### Aucune itération de correction nécessaire.
