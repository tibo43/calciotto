# Réagencement de la nav : Profile à côté de Logout, boutons icône seule

## Itération 1 — 2026-08-29

- **Modèle** : Haiku (changement mécanique, sans ambiguïté de conception, réutilisant un pattern déjà présent dans le fichier)
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- Le lien "Profile" retiré de `.nav-menu` (ne reste que Matches/Standings/Groups, cohérent avec le fait que ces trois-là sont scopés au groupe actif alors que Profile est cross-groupe).
- Nouveau bouton Profile ajouté dans `.nav-actions`, juste avant "Log out" — reste un `router-link` (pas un bouton), ce qui préserve le highlighting actif natif de Vue Router sans réinventer la logique.
- "Log out" passe en icône seule (une icône de sortie/porte, cohérente visuellement avec les autres icônes du fichier), texte visible supprimé.
- Nouvelle classe `.icon-nav-button` factorisant le style commun aux deux boutons (padding, hover, état actif), copiée sur le pattern déjà existant du bouton `theme-toggle`.
- `aria-label` correctement posé sur les deux boutons pour ne pas régresser l'accessibilité malgré la disparition du texte visible.

### Vérifications faites en revue

- Relecture du diff : cohérent, pas de sur-ingénierie, réutilise le pattern déjà en place plutôt que d'en inventer un nouveau.
- `npm run lint` / `npm run build` : verts, vérifiés indépendamment.
- **Test réel en navigateur (Playwright)** : texte de `.nav-menu` ne contient plus que "Matches Standings Groups" ; `aria-label="Profile"` et `aria-label="Log out"` bien présents ; clic sur l'icône Profile navigue vers `/profile` ; aucune erreur console. Capture d'écran prise et inspectée visuellement — rendu propre, bouton Profile et Logout bien alignés avant le sélecteur de thème.

### Aucune itération de correction nécessaire.
