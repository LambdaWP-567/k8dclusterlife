# TODO — Implementierungs-Reihenfolge

## Commit 1: Plan + Architektur-Dokumentation ✅
- [x] ARCHITECTURE.md mit vollständigem Plan
- [x] TODO.md mit dieser Liste

## Commit 2: Projekt-Scaffolding
- [ ] Go-Modul initialisieren (`go mod init github.com/lambdawp-567/k8dclusterlife`)
- [ ] React Apps erstellen (Vite + TypeScript): `web/app` und `web/testdash`
- [ ] Dockerfile (multi-stage: Go Build + React Build → ein Image)
- [ ] docker-compose.yml (App + PostgreSQL + Redis)
- [ ] `.github/workflows/ci.yml` (kind-Cluster + Go Tests + Playwright)
- [ ] `.releaserc.json` (semantic-release Konfiguration)
- [ ] `.gitignore`, `.editorconfig`

## Commit 3: Datenbank-Schema + Redis
- [ ] PostgreSQL Schema + goose Migrations
  - `sessions` — Auth-Sessions
  - `clusters` — Cluster-Konfigurationen (Name, Secret-Ref)
  - `healing_sessions` — KI-Heilungs-Vorgänge
  - `healing_actions` — Einzelne Aktionen pro Healing-Session
  - `healing_snapshots` — Ressource-Snapshots vor Heilung (für Rollback)
  - `test_results` — Test-Lauf-Ergebnisse
- [ ] Redis Client + Problem-State Cache (`internal/cache/redis.go`)
- [ ] Tests für Store-Layer

## Commit 4: Cluster-Controller + Status-Mapper
- [ ] `ClusterController`: goroutine pro Cluster, Polling-Loop (`internal/cluster/controller.go`)
- [ ] K8s-API Watcher: Pods, Nodes, Deployments, StatefulSets, DaemonSets (`internal/cluster/watcher.go`)
- [ ] `StatusMapper`: Knowledge Base — alle K8s-Statuses → Description (DE) + Cause (`internal/cluster/mapper.go`)
- [ ] REST API: `GET /api/problems` (aus Redis Cache)
- [ ] WebSocket Hub: Problem-Events streamen (`internal/api/websocket.go`)
- [ ] Tests: `mapper_test.go` (100% Coverage auf Knowledge Base), `controller_test.go`

## Commit 5: Erweitertes Monitoring
- [ ] Jobs/CronJobs Monitoring
- [ ] HPA Monitoring
- [ ] PVC/PV Monitoring
- [ ] TLS-Zertifikat-Scanner (Secret type kubernetes.io/tls, expiry < 30 Tage)
- [ ] Kubernetes Warning-Events
- [ ] Helm Release Status (Helm Secrets)
- [ ] CoreDNS Pod-Gesundheit
- [ ] Ingress + Service Cross-Check

## Commit 6: CSI-Monitoring (Storage)
- [ ] CSI Auto-Detector (`internal/csi/detector.go`)
- [ ] Longhorn Watcher (`internal/csi/longhorn.go`) — `longhorn-system` Namespace
- [ ] Rook/Ceph Watcher (`internal/csi/ceph.go`) — `rook-ceph` Namespace, CephCluster CRD
- [ ] Cloud CSI Watcher (`internal/csi/cloudcsi.go`) — AWS EBS, Azure Disk, GCE PD
- [ ] Piraeus/LINSTOR Watcher (`internal/csi/piraeus.go`) — `piraeus-system`, LinstorController CRD
- [ ] Generischer CSI Fallback (VolumeAttachments stuck, PVCs Pending)
- [ ] Status-Mapping für alle CSI-Fehler (auf Deutsch für Einsteiger)
- [ ] Tests für alle CSI-Watcher

## Commit 7: Frontend MVP
- [ ] React Router Setup (Dashboard, Healing History, Cluster Admin, Settings)
- [ ] `ProblemCard.tsx` — mit "KI-Heilung starten" Button
- [ ] `StatusBadge.tsx` — Badge + Tooltip mit Description (für Einsteiger)
- [ ] `CauseTag.tsx` — Ursachen-Anzeige
- [ ] `ProblemDetail.tsx` — Slide-over Panel (Tab: Details, Tab: KI-Healing)
- [ ] Dashboard: Problemliste + grünes "Alles OK" Banner
- [ ] `useWebSocket.ts` Hook für Live-Updates
- [ ] Header-Badge (Anzahl aktiver Probleme, rot)
- [ ] Toast-Notifications bei neuen Problemen
- [ ] Light Mode + Dark Mode Toggle (shadcn/ui)

## Commit 8: OIDC Authentifizierung
- [ ] Multi-Provider OIDC: Entra ID, GitHub, Google (`internal/auth/oidc.go`)
- [ ] JWT Session Middleware (`internal/auth/middleware.go`)
- [ ] Login-Seite mit Provider-Auswahl
- [ ] Logout
- [ ] Geschützte Routen (alle /api/* außer /auth/*)
- [ ] Tests: `oidc_test.go`

## Commit 9: Cluster-Verwaltung
- [ ] REST API: POST /api/clusters (kubeconfig Upload)
- [ ] K8s Secret CRUD für kubeconfigs (`internal/store/secrets.go`)
- [ ] REST API: GET /api/clusters, DELETE /api/clusters/:id
- [ ] Cluster-Management UI: Upload-Formular, Cluster-Liste, Verbindungsstatus
- [ ] Tests: `clusters_test.go`, `secrets_test.go`

## Commit 10: KI-Healing Agent — Phase 1 (MANUAL)
- [ ] `HealingAgent`: Claude Tool Use Agentic Loop mit Streaming (`internal/healing/agent.go`)
- [ ] Lese-Tools: `get_pod_logs`, `get_events`, `describe_resource`, `get_node_status`, `get_namespace_quota` (`internal/healing/tools.go`)
- [ ] Schreibendes Tool: `delete_pod` (erstes Schreib-Tool)
- [ ] Autonomie-Modus MANUAL: WebSocket-Pause + Bestätigungs-Event (`internal/healing/autonomy.go`)
- [ ] Snapshot vor Heilung → PostgreSQL (`internal/healing/rollback.go`)
- [ ] REST API: POST /api/healing, GET /api/healing, GET /api/healing/:id
- [ ] REST API: POST /api/healing/:id/approve, POST /api/healing/:id/reject
- [ ] `HealingStream.tsx` — Live Claude-Dialog + Tool-Calls
- [ ] `ActionConfirm.tsx` — Modal für MANUAL mode
- [ ] Audit-Log (`internal/healing/audit.go`)
- [ ] Tests: `agent_test.go` (mit gemockter Claude API), `tools_test.go`

## Commit 11: KI-Healing Agent — Phase 2 (Alle Tools + Modi)
- [ ] Schreib-Tools: `scale_deployment`, `patch_resource`, `apply_manifest`, `cordon_node`, `uncordon_node`
- [ ] Rollback-Tool: `restore_snapshot`
- [ ] COUNTDOWN Modus: 60s Timer UI (`CountdownBar.tsx`)
- [ ] FULL_AUTO Modus: Sofort ausführen
- [ ] Rollback: Claude erkennt Verschlechterung → Hybrid-Bestätigung durch User
- [ ] REST API: POST /api/healing/:id/rollback
- [ ] Healing History Seite (`HealingHistory.tsx`)
- [ ] Tests: Alle Autonomie-Modi, Rollback-Mechanismus

## Commit 12: Benachrichtigungen + Metriken
- [ ] Teams Webhook (`internal/notify/teams.go`) — Adaptive Cards für:
  - Neues Problem erkannt
  - KI-Heilung gestartet
  - Heilung erfolgreich / fehlgeschlagen
  - Problem automatisch aufgelöst
- [ ] Prometheus `/metrics` Endpoint (`internal/metrics/prometheus.go`)
  - `active_problems_total` (Labels: cluster, type)
  - `healing_attempts_total`, `healing_success_total`
  - `healing_duration_seconds`
  - `cluster_scrape_errors_total`
- [ ] Einstellungen-Seite: Teams Webhook URL konfigurieren
- [ ] Tests: `teams_test.go`

## Commit 13: Test-Ökosystem
- [ ] Test-Controller (Go, K8s Job): alle Szenarien (`testorch/scenarios.go`)
  - Pod-Kill / CrashLoop
  - Node-Cordon
  - Deployment Scale=0
  - Bad TLS Cert
  - Alle Pod-States systematisch
- [ ] Playwright E2E Tests (`e2e/tests/`)
  - `dashboard.spec.ts` — Problem erscheint nach Test-Trigger
  - `healing.spec.ts` — Trigger → KI heilt → Playwright verifiziert "Geheilt"
  - `notifications.spec.ts` — Toast + Header-Badge
  - `auth.spec.ts` — Login/Logout
- [ ] Test-Orchestrator Backend (`cmd/testorch/main.go`)
- [ ] Test-Dashboard React UI (`web/testdash/`)
- [ ] Litmus Chaos Integration (`testorch/litmus.go`)

## Commit 14: Helm Chart
- [ ] `helm/k8dclusterlife/Chart.yaml`
- [ ] `helm/k8dclusterlife/values.yaml` (vollständig dokumentiert)
- [ ] Templates: deployment, service, ingress, serviceaccount, rbac, configmap, test-namespace
- [ ] PostgreSQL als Bitnami Subchart
- [ ] Redis als Bitnami Subchart
- [ ] RBAC: Monitor (read-only), Healer (write), App (Secrets CRUD)
- [ ] Helm Chart Tests: `helm lint`, `helm template` im CI

## Commit 15: Responsive Design + Polish
- [ ] Mobile-responsives Layout (375px lesbar)
- [ ] Dark Mode vollständig implementiert
- [ ] Fehlerseiten: Cluster nicht erreichbar, Auth-Fehler, 404
- [ ] Loading-States für alle Komponenten (Skeleton UI)
- [ ] Leere Zustände (kein Cluster konfiguriert, etc.)

## Commit 16: Semantic-Release + CI/CD
- [ ] `.github/workflows/release.yml` (semantic-release bei Push auf main)
- [ ] Docker Image Build + Push zu GHCR
  - `ghcr.io/lambdawp-567/k8dclusterlife:v{version}`
  - `ghcr.io/lambdawp-567/k8dclusterlife:latest`
  - `ghcr.io/lambdawp-567/k8dclusterlife-testorch:v{version}`
- [ ] Helm Chart Release (GitHub Pages oder OCI Registry)
- [ ] CHANGELOG.md auto-generiert durch semantic-release

---

## Ressourcen für Phase 2 Tests (nach erstem grünem CI)

Wenn GitHub Actions grün sind, werden folgende GitHub Secrets benötigt:
- `KUBECONFIG_TEST` — kubeconfig für einen echten Test-Cluster
- `ANTHROPIC_API_KEY` — Claude API Token für echte KI-E2E-Tests (optional, kann auch Mock bleiben)

CSI-Tests benötigen einen Cluster mit mindestens einem installierten CSI-Treiber
(Longhorn, Rook, AWS EBS, Azure Disk, GCE PD oder Piraeus).
