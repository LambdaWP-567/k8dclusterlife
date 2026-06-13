package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap_KnownStatuses(t *testing.T) {
	cases := []struct {
		kind        string
		status      string
		wantSev     Severity
		wantDescHas string
	}{
		{"Pod", "CrashLoopBackOff", SeverityCritical, "startet immer wieder"},
		{"Pod", "OOMKilled", SeverityCritical, "Speicherverbrauch"},
		{"Pod", "ImagePullBackOff", SeverityCritical, "Image"},
		{"Pod", "Error", SeverityCritical, "Fehler-Exit"},
		{"Pod", "Evicted", SeverityWarning, "entfernt"},
		{"Pod", "Pending", SeverityWarning, "eingeplant"},
		{"Node", "NotReady", SeverityCritical, "keine Workloads"},
		{"Node", "DiskPressure", SeverityWarning, "Speicherplatz"},
		{"Node", "MemoryPressure", SeverityWarning, "Arbeitsspeicher"},
		{"Deployment", "Unavailable", SeverityCritical, "nicht genug"},
		{"PVC", "Pending", SeverityCritical, "Speicher-Anforderung"},
		{"PVC", "Lost", SeverityCritical, "verloren"},
		{"Certificate", "Expiring", SeverityWarning, "bald ab"},
		{"Certificate", "Expired", SeverityCritical, "abgelaufen"},
		{"HelmRelease", "Failed", SeverityCritical, "fehlgeschlagen"},
		{"LonghornVolume", "Degraded", SeverityWarning, "Replikate"},
		{"CephCluster", "HEALTH_ERR", SeverityCritical, "kritischen Fehler"},
		{"VolumeAttachment", "Stuck", SeverityCritical, "angehängt"},
		{"LinstorController", "NotRunning", SeverityCritical, "LINSTOR"},
	}

	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.status, func(t *testing.T) {
			desc, cause, sev := Map(tc.kind, tc.status)
			assert.Equal(t, tc.wantSev, sev)
			assert.Contains(t, desc, tc.wantDescHas)
			assert.NotEmpty(t, cause)
		})
	}
}

func TestMap_UnknownStatus(t *testing.T) {
	desc, cause, sev := Map("Pod", "SomeWeirdStatus")
	assert.Equal(t, SeverityWarning, sev)
	assert.Contains(t, desc, "SomeWeirdStatus")
	assert.NotEmpty(t, cause)
}

func TestMap_AllEntriesHaveContent(t *testing.T) {
	for key, e := range statusMap {
		assert.NotEmpty(t, e.description, "empty description for %s", key)
		assert.NotEmpty(t, e.cause, "empty cause for %s", key)
	}
}
