package cluster

// entry defines the human-readable mapping for one K8s status.
type entry struct {
	description string
	cause       string
	severity    Severity
}

// statusMap maps "<Kind>/<status>" → entry for the dashboard.
var statusMap = map[string]entry{
	// ── Pods ──────────────────────────────────────────────────────────
	"Pod/CrashLoopBackOff": {
		"Container startet immer wieder und schlägt fehl.",
		"Fehler im Startprozess des Containers — prüfe die Logs.",
		SeverityCritical,
	},
	"Pod/OOMKilled": {
		"Container wurde wegen zu viel Speicherverbrauch beendet.",
		"Memory-Limit zu niedrig oder Speicherleck in der Anwendung.",
		SeverityCritical,
	},
	"Pod/ImagePullBackOff": {
		"Das Container-Image konnte nicht geladen werden.",
		"Falscher Image-Name, fehlende Zugangsdaten oder kein Netz.",
		SeverityCritical,
	},
	"Pod/ErrImagePull": {
		"Erstmaliger Fehler beim Laden des Container-Images.",
		"Falscher Image-Name, fehlende Zugangsdaten oder kein Netz.",
		SeverityCritical,
	},
	"Pod/Error": {
		"Container hat mit einem Fehler-Exit-Code beendet.",
		"Anwendungsfehler — prüfe die Logs des Containers.",
		SeverityCritical,
	},
	"Pod/Evicted": {
		"Pod wurde vom Node entfernt.",
		"Ressourcen-Druck auf dem Node (Disk oder Memory zu knapp).",
		SeverityWarning,
	},
	"Pod/Pending": {
		"Pod wartet darauf eingeplant zu werden.",
		"Zu wenig Ressourcen auf Nodes oder Node-Selector passt nicht.",
		SeverityWarning,
	},
	"Pod/Terminating": {
		"Pod wird gerade beendet und reagiert nicht mehr.",
		"Finalizer blockiert oder Node nicht erreichbar.",
		SeverityWarning,
	},
	"Pod/Unknown": {
		"Zustand des Pods ist unbekannt.",
		"Node nicht erreichbar — Kubelet antwortet nicht.",
		SeverityCritical,
	},
	"Pod/CreateContainerError": {
		"Container konnte nicht erstellt werden.",
		"Fehlende ConfigMap/Secret oder ungültige Volume-Konfiguration.",
		SeverityCritical,
	},
	"Pod/RunContainerError": {
		"Container konnte nicht gestartet werden.",
		"Fehlende Binaries oder falsche Einstiegspunkt-Konfiguration.",
		SeverityCritical,
	},
	"Pod/ContainerCreating": {
		"Container wird seit längerer Zeit erstellt.",
		"Image-Download zu langsam oder Volume-Attachment hängt.",
		SeverityWarning,
	},
	"Pod/Init:CrashLoopBackOff": {
		"Init-Container schlägt immer wieder fehl.",
		"Abhängige Services nicht bereit oder Fehler im Init-Container.",
		SeverityCritical,
	},
	"Pod/Init:Error": {
		"Init-Container hat mit einem Fehler beendet.",
		"Fehler in der Initialisierungslogik — prüfe Init-Container Logs.",
		SeverityCritical,
	},

	// ── Nodes ─────────────────────────────────────────────────────────
	"Node/NotReady": {
		"Dieser Node nimmt keine Workloads an.",
		"Kubelet nicht erreichbar oder Ressourcen-Druck auf dem Node.",
		SeverityCritical,
	},
	"Node/DiskPressure": {
		"Dem Node geht der Speicherplatz aus.",
		"Logs, Images oder temporäre Dateien belegen zu viel Platz.",
		SeverityWarning,
	},
	"Node/MemoryPressure": {
		"Dem Node geht der Arbeitsspeicher aus.",
		"Zu viele Pods mit hohem Memory-Verbrauch auf diesem Node.",
		SeverityWarning,
	},
	"Node/PIDPressure": {
		"Zu viele Prozesse auf dem Node.",
		"Prozess-Limit des Betriebssystems fast erreicht.",
		SeverityWarning,
	},
	"Node/NetworkUnavailable": {
		"Netzwerk auf diesem Node nicht verfügbar.",
		"CNI-Plugin nicht initialisiert oder Netzwerk-Konfigurationsfehler.",
		SeverityCritical,
	},

	// ── Deployments ───────────────────────────────────────────────────
	"Deployment/Unavailable": {
		"Deployment hat nicht genug laufende Pods.",
		"Pods crashen, Image fehlt oder Ressourcen sind erschöpft.",
		SeverityCritical,
	},
	"Deployment/Progressing": {
		"Deployment steckt fest und macht keinen Fortschritt.",
		"Rollout-Timeout überschritten — neue Pods starten nicht.",
		SeverityWarning,
	},

	// ── StatefulSets ──────────────────────────────────────────────────
	"StatefulSet/NotReady": {
		"StatefulSet hat nicht alle Pods bereit.",
		"Pods crashen oder PVCs können nicht gebunden werden.",
		SeverityCritical,
	},

	// ── DaemonSets ────────────────────────────────────────────────────
	"DaemonSet/Mismatch": {
		"DaemonSet läuft nicht auf allen Nodes.",
		"Node-Selektor, Taint oder Ressourcen-Mangel verhindert Start.",
		SeverityWarning,
	},

	// ── Jobs / CronJobs ───────────────────────────────────────────────
	"Job/Failed": {
		"Job hat alle Versuche erschöpft und ist fehlgeschlagen.",
		"Anwendungsfehler oder externe Abhängigkeit nicht erreichbar.",
		SeverityCritical,
	},
	"CronJob/Suspended": {
		"CronJob ist pausiert und läuft nicht.",
		"Manuell pausiert oder automatisch wegen zu vieler Fehler.",
		SeverityInfo,
	},

	// ── HPA ───────────────────────────────────────────────────────────
	"HPA/ScaleFailed": {
		"Horizontal Pod Autoscaler kann nicht skalieren.",
		"Fehlende Metriken (kein metrics-server) oder Ressourcen-Limits.",
		SeverityWarning,
	},

	// ── PVC / PV ──────────────────────────────────────────────────────
	"PVC/Pending": {
		"Speicher-Anforderung (PVC) kann nicht erfüllt werden.",
		"Kein passender PersistentVolume vorhanden oder CSI-Treiber fehlt.",
		SeverityCritical,
	},
	"PVC/Lost": {
		"Speicher-Anforderung hat sein Volume verloren.",
		"Das zugehörige PersistentVolume wurde gelöscht.",
		SeverityCritical,
	},
	"PV/Failed": {
		"PersistentVolume hat einen Fehler.",
		"Storage-Backend nicht erreichbar oder Volume beschädigt.",
		SeverityCritical,
	},

	// ── Zertifikate ───────────────────────────────────────────────────
	"Certificate/Expiring": {
		"TLS-Zertifikat läuft bald ab.",
		"Zertifikats-Erneuerung (z.B. cert-manager) ist ausstehend.",
		SeverityWarning,
	},
	"Certificate/Expired": {
		"TLS-Zertifikat ist bereits abgelaufen.",
		"Sofortige Erneuerung notwendig — HTTPS funktioniert nicht mehr.",
		SeverityCritical,
	},

	// ── Helm ──────────────────────────────────────────────────────────
	"HelmRelease/Failed": {
		"Helm-Release ist fehlgeschlagen.",
		"Fehler beim letzten helm upgrade oder install.",
		SeverityCritical,
	},
	"HelmRelease/PendingUpgrade": {
		"Helm-Release steckt in einem laufenden Upgrade fest.",
		"Vorheriger Upgrade-Versuch wurde unterbrochen.",
		SeverityWarning,
	},

	// ── CSI: Longhorn ─────────────────────────────────────────────────
	"LonghornVolume/Degraded": {
		"Ein Longhorn-Speichervolume hat nicht alle Replikate.",
		"Eine Replik ist ausgefallen — Datenverlust-Risiko steigt.",
		SeverityWarning,
	},
	"LonghornVolume/Faulted": {
		"Longhorn-Speichervolume ist nicht mehr verwendbar.",
		"Alle Replikate sind ausgefallen — sofortiger Handlungsbedarf.",
		SeverityCritical,
	},
	"LonghornManager/NotRunning": {
		"Der Longhorn Storage-Manager läuft nicht.",
		"Alle Longhorn-Volumes sind betroffen — Storage nicht verfügbar.",
		SeverityCritical,
	},

	// ── CSI: Rook/Ceph ────────────────────────────────────────────────
	"CephCluster/HEALTH_WARN": {
		"Der Ceph-Speicherdienst meldet eine Warnung.",
		"Ein OSD oder Monitor hat ein Problem — Redundanz reduziert.",
		SeverityWarning,
	},
	"CephCluster/HEALTH_ERR": {
		"Der Ceph-Speicherdienst hat einen kritischen Fehler.",
		"OSD-Ausfall oder Quorum-Verlust — sofortige Aktion nötig.",
		SeverityCritical,
	},
	"CephOSD/Down": {
		"Eine Ceph-Speichereinheit (OSD) ist ausgefallen.",
		"Disk-Fehler oder Node-Ausfall — Redundanz reduziert.",
		SeverityWarning,
	},

	// ── CSI: Cloud / Generic ──────────────────────────────────────────
	"VolumeAttachment/Stuck": {
		"Ein Cloud-Speichervolume kann nicht an den Pod angehängt werden.",
		"CSI-Treiber hängt oder Cloud-API nicht erreichbar.",
		SeverityCritical,
	},
	"CSIDriver/NotRunning": {
		"Der Speicher-Treiber (CSI) läuft nicht.",
		"Neue Volumes können nicht erstellt oder gemountet werden.",
		SeverityCritical,
	},

	// ── CSI: Piraeus/LINSTOR ──────────────────────────────────────────
	"LinstorController/NotRunning": {
		"Der LINSTOR Storage-Controller läuft nicht.",
		"Kein Piraeus-Speicher verfügbar bis Controller wieder läuft.",
		SeverityCritical,
	},
	"LinstorSatellite/NotRunning": {
		"Ein LINSTOR Speicher-Knoten (Satellite) ist nicht erreichbar.",
		"Volumes auf diesem Node können nicht erstellt oder gemountet werden.",
		SeverityWarning,
	},
}

// Map returns description, cause and severity for a given Kind+status.
// Falls back to a generic message when the status is unknown.
func Map(kind, status string) (description, cause string, severity Severity) {
	key := kind + "/" + status
	if e, ok := statusMap[key]; ok {
		return e.description, e.cause, e.severity
	}
	return "Unbekannter Zustand: " + status + ".",
		"Prüfe die Kubernetes-Events für weitere Details.",
		SeverityWarning
}
