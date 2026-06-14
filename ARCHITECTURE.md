# k8dclusterlife — Architektur & Design

## Ziel

Ein **unerfahrener Nutzer** (kein Kubernetes-Experte) sieht auf einen Blick, was in einem oder mehreren Kubernetes-Clustern kaputt ist. Die App erklärt jeden Status verständlich auf Deutsch, erkennt Probleme in allen Cluster-Bereichen — inkl. Storage (CSI) — und heilt den Cluster via Claude-KI-Agent automatisch.

## Leitprinzipien

- **Unerfahrener Nutzer**: Jeder K8s-Status hat eine verständliche Erklärung auf Deutsch + Ursache
- **Nur Probleme anzeigen**: Wenn alles OK ist, sieht der Nutzer ein grünes Banner — keine Informationsflut
- **Claude als Agent**: Tool Use Agentic Loop — KI sammelt selbst Kontext und führt Heilungs-Schritte aus
- **Konfigurierbare Autonomie**: MANUAL / COUNTDOWN / FULL_AUTO — wählbar in den Einstellungen
- **Test als First-Class-Bürger**: Vollständiger E2E-Beweis inkl. KI-Heilungsverifikation
- **Stateless + horizontal skalierbar**: Redis für State, PostgreSQL für Persistenz
- **Conventional Commits**: `feat:` → minor, `fix:` → patch, `feat!:` → major

## Tech Stack

| Layer | Technologie | Begründung |
|-------|-------------|------------|
| Backend | **Go** | `client-go` nativ; Goroutinen für parallele Cluster; geringe Memory-Footprint |
| HTTP Router | `chi` | Leichtgewichtig, middleware-kompatibel |
| WebSocket | `gorilla/websocket` | Live-Updates + Claude-Streaming zum Browser |
| OIDC Auth | `coreos/go-oidc` + `golang.org/x/oauth2` | Multi-Provider: Entra ID, GitHub, Google |
| Claude API | `anthropic-sdk-go` | Tool Use + Streaming; Agentic Loop |
| Cache | **Redis** (`go-redis`) | Problem-State Cache; Pub/Sub für horizontale Skalierung |
| DB | **PostgreSQL** + `pgx` + `goose` | Sessions, Cluster-Config, Healing-History, Test-Ergebnisse |
| Metrics | Prometheus (`prometheus/client_golang`) | `/metrics` für Monitoring der App selbst |
| Frontend | **React + TypeScript + Vite** | Modern, schnelle Builds |
| UI Components | **shadcn/ui + Tailwind CSS** | Accessible, Dark Mode, themeable |
| Client State | Zustand + TanStack Query | Lokaler State + Server-State mit Caching |
| E2E Tests | **Playwright** | Browser-UI Tests inkl. KI-Healing Verifikation |
| API Tests | Go `testing` + `testify` | Unit + Integration pro Handler/Service |
| Chaos | Eigener Test-Controller + Litmus Chaos | Einfache Szenarien selbst, komplexe via Litmus |
| Versioning | **semantic-release** | Vollautomatisch ab v1.0.0 |
| Deployment | **Helm Chart** | PostgreSQL + Redis als Bitnami Subcharts |

## Systemarchitektur

```
┌────────────────────────────────────────────────────────────────────────┐
│  Browser                                                                │
│  ├── Main App (React/Vite)    Dashboard + KI-Healing                   │
│  └── Test Dashboard (React/Vite)  Test-Läufe + Playwright-Reports     │
└──────────────┬──────────────────────────────┬──────────────────────────┘
               │ REST + WebSocket              │ REST + WebSocket
┌──────────────▼────────────────┐  ┌──────────▼──────────────────────────┐
│  Main Backend (Go, stateless)  │  │  Test Orchestrator (Go)              │
│  ├── /auth/*      OIDC         │  │  Test-Controller (K8s Job)           │
│  ├── /api/clusters             │  │  Playwright Runner (Node.js)         │
│  ├── /api/problems             │  │  Litmus Chaos Integration            │
│  ├── /api/healing              │  └──────────────────────────────────────┘
│  ├── /metrics     Prometheus   │
│  ├── /ws          WebSocket    │
│  ├── ClusterController         │
│  │   ├── Core K8s Watcher      │
│  │   ├── CSI Auto-Detector     │    ┌──────────────────────────────┐
│  │   │   ├── Longhorn Watcher  │    │ Überwachte K8s Cluster        │
│  │   │   ├── Ceph Watcher      │◄───│ (alle Namespaces)             │
│  │   │   ├── Cloud CSI Watcher │    │ inkl. CSI Namespaces          │
│  │   │   └── Piraeus Watcher   │    └──────────────────────────────┘
│  ├── StatusMapper              │
│  ├── HealingAgent ↔ Claude API │
│  ├── EventBus                  │
│  └── Notifier (Teams)          │
└──────┬──────────────┬──────────┘
       │              │
  ┌────▼────┐   ┌─────▼──────┐
  │ Redis    │   │ PostgreSQL  │
  │ Problem- │   │ Persistenz  │
  │ State    │   └─────────────┘
  └──────────┘         │
                ┌──────▼─────────────┐
                │ K8s Secrets         │
                │ (kubeconfigs)       │
                └─────────────────────┘
```

## Was überwacht wird (pro Cluster)

| Kategorie | Ressourcen |
|-----------|-----------|
| Control Plane | API Server, etcd, Scheduler, Controller Manager |
| Nodes | NotReady, DiskPressure, MemoryPressure, PIDPressure |
| Pods | CrashLoopBackOff, OOMKilled, Error, ImagePullBackOff, Evicted, Pending >5min, alle Status-Übergänge |
| Deployments | unavailableReplicas > 0 |
| StatefulSets | not ready replicas |
| DaemonSets | desired vs ready mismatch |
| Jobs / CronJobs | failed |
| HPA | unable to scale |
| PVC / PV | Pending, Lost, Failed |
| Zertifikate | TLS-Certs ablaufend < 30 Tage |
| Events | Warning-Events |
| CoreDNS | Pod-Gesundheit |
| Ingress | fehlende Backend-Services |
| Helm Releases | failed / pending-upgrade |
| CSI (Longhorn, Ceph, Cloud, Piraeus) | Treiber-spezifische Probleme — auto-detect |

## CSI-Monitoring (Storage Interface)

Die App erkennt automatisch welche CSI-Treiber installiert sind.

### Longhorn (Rancher)
- Erkennung: Namespace `longhorn-system` existiert
- Überwacht: `longhorn-manager` Pod, Volume CRDs (degraded/faulted), Backups, Replicas
- Beispiel-Meldung: "Ein Speichervolume hat nicht alle Kopien — Datenverlust-Risiko"

### Rook / Ceph
- Erkennung: Namespace `rook-ceph` existiert
- Überwacht: `CephCluster` Status (HEALTH_WARN/ERR), OSD Pods, MON Quorum, MGR Pod
- Beispiel-Meldung: "Der Ceph-Speicherdienst hat einen kritischen Fehler — sofortige Aktion nötig"

### Cloud CSI (AWS EBS / Azure Disk / GCE PD)
- Erkennung: CSI-Controller-Pods in `kube-system` mit bekannten Labels
- Überwacht: Controller-Pod, Node-DaemonSet, VolumeAttachments (stuck > 5min)
- Beispiel-Meldung: "Ein Cloud-Speichervolume kann nicht an den Pod angehängt werden"

### Piraeus / LINSTOR (LINBIT)
- Erkennung: Namespace `piraeus-system` oder CRD `LinstorController`
- Überwacht: `linstor-controller` Pod, Satellite Pods, `LinstorSatelliteSet` Status
- Beispiel-Meldung: "Ein Speicher-Knoten (Satellite) ist nicht erreichbar — Volumes auf diesem Node betroffen"

### Generisches CSI (Fallback)
- VolumeAttachments stuck, PVCs Pending > 5min, PVs Failed/Released

## KI-Healing Agent

### Agentic Loop

```
Problem erkannt → "KI-Heilung starten" Button
  → Zustand-Snapshot vor Heilung speichern (PostgreSQL)
  → Claude Tool Use Loop (claude-sonnet-4-6 + Streaming):
      Lese-Tools: sofort ausführen
      Schreib-Tools: Autonomie-Check:
        MANUAL    → Pause → WebSocket-Event → Admin klickt Bestätigen
        COUNTDOWN → 60s Timer in UI, Admin kann abbrechen
        FULL_AUTO → Sofort ausführen + Audit-Log
      Nach jeder Aktion: Problem-Status re-prüfen
      Falls Verschlechterung erkannt:
        Claude schlägt Rollback vor → User bestätigt → Snapshot wiederherstellen
  → "Geheilt ✓" oder "Nicht heilbar: <Begründung>"
  → Gesamter Dialog + Aktionen in PostgreSQL (Healing-History)
```

### Claude Tools

| Tool | Typ |
|------|-----|
| `get_pod_logs`, `get_events`, `describe_resource`, `get_node_status`, `get_namespace_quota` | Lesen (sicher, sofort) |
| `delete_pod`, `scale_deployment`, `patch_resource`, `apply_manifest`, `cordon_node`, `uncordon_node`, `restore_snapshot` | **Schreibend** → Autonomie-Check |

## Status-Mapping Knowledge Base

Jedes Problem enthält:
- `status` — roher K8s-Status
- `description` — Was bedeutet das? (für Einsteiger, auf Deutsch)
- `cause` — Warum passiert das typischerweise?
- Details (ausklappbar) — Events, letzte Log-Zeilen, Resource-Specs

| Status | Beschreibung | Typische Ursache |
|--------|-------------|-----------------|
| `CrashLoopBackOff` | Container startet immer wieder und schlägt fehl | Fehler im Startprozess |
| `OOMKilled` | Container wegen Speicherüberlast beendet | Memory Limit zu niedrig |
| `ImagePullBackOff` | Container-Image nicht ladbar | Falscher Name oder fehlende Zugangsdaten |
| `Pending` >5min | Pod wartet auf Einplanung | Zu wenig Ressourcen auf Nodes |
| `NotReady` (Node) | Node nimmt keine Workloads an | Kubelet-Problem oder Ressourcen-Druck |
| `Evicted` | Pod wurde vom Node entfernt | Disk/Memory-Druck |
| `Error` | Container beendet mit Fehler-Exit | Anwendungsfehler |
| `Volume degraded` | Speichervolume hat nicht alle Kopien | Longhorn Replica-Ausfall |
| `CephCluster HEALTH_ERR` | Kritischer Ceph-Speicherfehler | OSD-Ausfall oder Netzwerkproblem |
| `VolumeAttachment stuck` | Cloud-Volume kann nicht angehängt werden | CSI-Treiber-Problem |
| `Satellite down` | Speicher-Knoten nicht erreichbar | LINSTOR Satellite-Ausfall |
| `Cert expiring` | TLS-Zertifikat läuft in X Tagen ab | Erneuerung ausstehend |

## UI-Seiten

| Seite | Inhalt |
|-------|--------|
| **Login** | Provider-Auswahl: Microsoft / GitHub / Google |
| **Dashboard** | Problemliste aller Cluster; grünes Banner wenn OK; "KI-Heilung" Button pro Problem |
| **Problem Detail** | Slide-over: Tab Details + Tab KI-Healing Live-Stream (Claude-Dialog + Tool-Calls) |
| **Healing History** | Claude-Dialoge, ausgeführte Aktionen, Rollback-Historie |
| **Cluster-Verwaltung** | Cluster hinzufügen (kubeconfig Upload), entfernen, Verbindungsstatus |
| **Einstellungen** | Autonomie-Modus, Countdown-Dauer, Claude API Token, Teams Webhook, Refresh-Intervall |
| **Test-Dashboard** | Test-Läufe starten, Live-Log-Stream, eingebetteter Playwright-Report |

## Test-Strategie

### CI-Phasen

**Phase 1** (kein externer Cluster nötig): kind-Cluster in GitHub Actions
- Go Unit Tests + Integration Tests
- Playwright E2E mit kind-Cluster
- KI-Healing Tests mit gemockter Claude API

**Phase 2** (nach erstem grünem CI): User stellt kubeconfig bereit
- Echte CSI-Tests (Longhorn, Rook, Piraeus)
- Echte Claude API E2E Tests
- kubeconfig als GitHub Secret `KUBECONFIG_TEST`

### Test-Coverage
- 100% auf kritische Pfade: `healing/agent.go`, `cluster/mapper.go`, alle API-Handler
- Mock Claude API für Unit-Tests (deterministisch, keine API-Kosten)
- Echte Claude API nur in E2E (Phase 2)

### Test-Szenarien (Test-Controller)

| Szenario | Controller-Aktion | E2E-Prüfung via Playwright |
|----------|------------------|---------------------------|
| Pod-Kill / CrashLoop | Pod löschen + crashender Container | Dashboard zeigt Problem → KI heilt |
| Node-Cordon | Node cordonieren | NotReady → KI uncordont |
| Scale=0 | Deployment auf 0 | unavailableReplicas → KI skaliert hoch |
| Bad TLS Cert | Abgelaufenes Secret einschleusen | Cert-Warnung im Dashboard |
| Alle Pod-States | Jeden Status systematisch triggern | Korrekte Mapping-Anzeige |
| Litmus Chaos | Zufälliger Pod-Kill | Dashboard erkennt + optional KI-Healing |

## API Endpoints

### Main Backend

| Endpoint | Method | Beschreibung |
|----------|--------|--------------|
| `/auth/login` | GET | OIDC-Redirect zum Provider |
| `/auth/callback` | GET | Token-Exchange, Session setzen |
| `/auth/logout` | POST | Session löschen |
| `/api/clusters` | GET | Liste konfigurierter Cluster |
| `/api/clusters` | POST | Cluster hinzufügen (kubeconfig Upload) |
| `/api/clusters/:id` | DELETE | Cluster entfernen |
| `/api/problems` | GET | Alle aktuellen Probleme (Redis Cache) |
| `/api/problems?cluster=x` | GET | Probleme eines spezifischen Clusters |
| `/api/healing` | POST | Healing-Session starten |
| `/api/healing` | GET | Healing-History |
| `/api/healing/:id` | GET | Detail: Claude-Dialog + Aktionen |
| `/api/healing/:id/approve` | POST | Schreibende Aktion bestätigen (MANUAL) |
| `/api/healing/:id/reject` | POST | Schreibende Aktion ablehnen |
| `/api/healing/:id/rollback` | POST | Rollback bestätigen (Hybrid-Modus) |
| `/api/settings` | GET/PUT | App-Einstellungen |
| `/ws` | WebSocket | Problem-Events + Healing-Streams |
| `/metrics` | GET | Prometheus Metrics |

### Test Orchestrator

| Endpoint | Method | Beschreibung |
|----------|--------|--------------|
| `/tests/scenarios` | GET | Liste verfügbarer Szenarien |
| `/tests/run` | POST | Szenario starten (K8s Job) |
| `/tests/results` | GET | Alle Test-Ergebnisse |
| `/tests/:id` | GET | Detail + Playwright-Report-URL |
| `/tests/ws` | WebSocket | Live-Log laufender Test |

## Projektstruktur

```
k8dclusterlife/
├── cmd/
│   ├── server/main.go                 # Main Backend Entry Point
│   └── testorch/main.go               # Test Orchestrator Entry Point
├── internal/
│   ├── auth/
│   │   ├── oidc.go                    # Multi-Provider OIDC
│   │   ├── middleware.go              # JWT Session Middleware
│   │   └── oidc_test.go
│   ├── cluster/
│   │   ├── controller.go              # goroutine pro Cluster
│   │   ├── watcher.go                 # K8s API Calls
│   │   ├── mapper.go                  # Status Knowledge Base
│   │   ├── types.go
│   │   ├── controller_test.go
│   │   └── mapper_test.go
│   ├── csi/
│   │   ├── detector.go                # Auto-Detect installierter CSI-Treiber
│   │   ├── longhorn.go
│   │   ├── ceph.go
│   │   ├── cloudcsi.go
│   │   ├── piraeus.go
│   │   └── *_test.go
│   ├── api/
│   │   ├── clusters.go
│   │   ├── problems.go
│   │   ├── healing.go
│   │   ├── settings.go
│   │   ├── websocket.go
│   │   └── *_test.go
│   ├── healing/
│   │   ├── agent.go                   # HealingAgent: Claude Agentic Loop
│   │   ├── tools.go                   # Tool-Implementierungen
│   │   ├── autonomy.go                # MANUAL/COUNTDOWN/FULL_AUTO
│   │   ├── rollback.go                # Snapshot + Rollback
│   │   ├── audit.go                   # Audit-Log
│   │   └── *_test.go
│   ├── notify/
│   │   ├── teams.go                   # Teams Webhook Adaptive Cards
│   │   ├── eventbus.go
│   │   └── *_test.go
│   ├── cache/
│   │   ├── redis.go
│   │   └── *_test.go
│   ├── metrics/
│   │   └── prometheus.go
│   └── store/
│       ├── postgres.go
│       ├── secrets.go                 # K8s Secret CRUD für kubeconfigs
│       └── *_test.go
├── testorch/
│   ├── controller.go                  # K8s Job Steuerung
│   ├── scenarios.go                   # Alle Test-Szenarien
│   ├── playwright.go                  # Playwright Container
│   ├── litmus.go                      # Litmus Chaos Integration
│   └── *_test.go
├── web/
│   ├── app/                           # Haupt-App (React)
│   │   └── src/
│   │       ├── components/
│   │       │   ├── ProblemCard.tsx    # + "KI-Heilung" Button
│   │       │   ├── StatusBadge.tsx    # + Tooltip mit Description
│   │       │   ├── CauseTag.tsx
│   │       │   ├── ProblemDetail.tsx  # Slide-over
│   │       │   ├── HealingStream.tsx  # Live Claude-Dialog
│   │       │   ├── ActionConfirm.tsx  # Modal MANUAL mode
│   │       │   └── CountdownBar.tsx   # COUNTDOWN mode
│   │       ├── pages/
│   │       │   ├── Dashboard.tsx
│   │       │   ├── HealingHistory.tsx
│   │       │   ├── ClusterAdmin.tsx
│   │       │   └── Settings.tsx
│   │       └── hooks/
│   │           ├── useWebSocket.ts
│   │           ├── useProblems.ts
│   │           └── useHealing.ts
│   └── testdash/                      # Test-Dashboard (React)
│       └── src/
│           ├── pages/
│           │   ├── TestRuns.tsx
│           │   └── TestDetail.tsx
│           └── hooks/
│               └── useTestStream.ts
├── e2e/
│   ├── tests/
│   │   ├── dashboard.spec.ts
│   │   ├── healing.spec.ts
│   │   ├── notifications.spec.ts
│   │   └── auth.spec.ts
│   └── playwright.config.ts
├── helm/k8dclusterlife/
│   ├── Chart.yaml
│   ├── values.yaml
│   └── templates/
│       ├── deployment.yaml
│       ├── service.yaml
│       ├── ingress.yaml
│       ├── serviceaccount.yaml
│       ├── rbac.yaml
│       ├── configmap.yaml
│       └── test-namespace.yaml
├── .github/
│   └── workflows/
│       ├── ci.yml                     # Tests bei PR (kind-Cluster)
│       └── release.yml                # semantic-release bei Push auf main
├── .releaserc.json
├── Dockerfile                         # Multi-Stage: Go + React
├── Dockerfile.testorch
├── Dockerfile.playwright
└── docker-compose.yml
```

## Helm RBAC

| ServiceAccount | Permissions |
|----------------|-------------|
| `k8dclusterlife` | Secrets CRUD (eigener Namespace — kubeconfigs) |
| `k8dclusterlife-monitor` | Alle Ressourcen get/list/watch in Ziel-Clustern (inkl. CSI-Namespaces) |
| `k8dclusterlife-healer` | Pods delete, Deployments scale, Nodes cordon/uncordon, Resources patch/apply (alle Namespaces) |

## Prometheus Metrics

| Metric | Typ | Beschreibung |
|--------|-----|--------------|
| `active_problems_total` | Gauge | Aktuelle Probleme (Labels: cluster, type) |
| `healing_attempts_total` | Counter | Gestartete Healing-Versuche |
| `healing_success_total` | Counter | Erfolgreiche Heilungen |
| `healing_duration_seconds` | Histogram | Dauer der Healing-Vorgänge |
| `cluster_scrape_errors_total` | Counter | Fehler beim Cluster-Polling |

## Ressourcen für Tests

1. **Jetzt**: Nichts nötig — kind-Cluster läuft in GitHub Actions (kostenlos)
2. **Nach erstem grünem CI**: kubeconfig als GitHub Secret `KUBECONFIG_TEST`
3. **Optional**: `ANTHROPIC_API_KEY` als GitHub Secret für echte KI-E2E-Tests
