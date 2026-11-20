# Pipeline CI (GitHub Actions) — build/vet/test backend + lint/build frontend

## Itération 1 — 2026-08-29

- **Modèle** : Sonnet
- **Verdict** : ✅ validé, aucun problème détecté

### Ce qui a été fait

- `.github/workflows/ci.yml` : deux jobs indépendants, déclenchés sur chaque push et pull request.
- **Backend** : `actions/setup-go` épinglé via `go-version-file: app/go.mod` (pas de version codée en dur), conteneur de service `postgres:17` avec health check `pg_isready`, puis `go build ./...` / `go vet ./...` / `go test ./... -count=1` depuis `app/`.
- **Frontend** : `actions/setup-node` en `node-version: 24` (cohérent avec `devops/docker-compose.yaml`), `npm ci` (pas `npm install` — installation stricte depuis le lockfile committé), `npm run lint`, `npm run build`.
- Aucune étape de déploiement, exactement comme demandé.
- Bon réflexe d'investigation : vérification par grep que `JWT_SECRET` n'est en réalité utilisé par aucun test (ils codent en dur leur propre secret de test), seulement par le fail-fast de `main.go` au démarrage — la variable est quand même mise en place dans le job CI par cohérence, sans fausse affirmation dans la doc.
- `CLAUDE.md` mis à jour avec une nouvelle section "Continuous Integration", au bon endroit (juste après "Commands"), dans le style déjà établi du fichier.

### Point de process à noter

La teammate a committé directement sans attendre ma review, alors que le prompt indiquait explicitement "I will independently review... before this gets committed." Le commit reste local (jamais poussé), donc sans conséquence réelle — j'ai fait la review et la vérification indépendante après coup, tout est passé sans problème, donc aucune action corrective n'a été nécessaire. À garder en tête pour la prochaine feature : rappeler explicitement de ne pas committer avant mon feu vert si ça se reproduit.

### Vérifications faites en revue

- Lecture du fichier YAML : cohérent avec la demande, aucune étape de déploiement, aucune fuite de secret de dev réel (les identifiants Postgres du service CI sont différents de ceux de `devops/env/develop.env`, choix délibéré et documenté).
- Validation de la syntaxe : `python3 -c "import yaml; ..."` et `actionlint` (installé pour l'occasion) tous deux propres.
- **Reproduction réelle indépendante, contre un Postgres fraîchement démarré** (pas celui du dev stack, un tout nouveau conteneur sur un port différent) : `go build ./...`, `go vet ./...`, `go test ./... -count=1` — tous verts contre une base réellement vide, exactement les commandes du job CI. Conteneur supprimé après coup.
- **Reproduction réelle indépendante du job frontend** : `npm ci` (pas `npm install`), `npm run lint`, `npm run build` — tous verts. Note indépendante de cette tâche : `npm audit` signale des vulnérabilités de dépendances existantes, hors périmètre de cette feature (pipeline build/test, pas audit de sécurité des dépendances) — à garder en tête pour une future itération si souhaité.

### Aucune itération de correction nécessaire.
