{{/*
Expand the name of the chart.
*/}}
{{- define "k8dclusterlife.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "k8dclusterlife.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "k8dclusterlife.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "k8dclusterlife.labels" -}}
helm.sh/chart: {{ include "k8dclusterlife.chart" . }}
{{ include "k8dclusterlife.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "k8dclusterlife.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8dclusterlife.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "k8dclusterlife.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "k8dclusterlife.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Database URL
*/}}
{{- define "k8dclusterlife.databaseURL" -}}
{{- if .Values.postgresql.enabled }}
{{- printf "postgres://%s:%s@%s-postgresql:5432/%s?sslmode=disable" .Values.postgresql.auth.username .Values.postgresql.auth.password .Release.Name .Values.postgresql.auth.database }}
{{- else }}
{{- printf "postgres://%s:%s@%s:%d/%s?sslmode=disable" .Values.externalPostgresql.username .Values.externalPostgresql.password .Values.externalPostgresql.host (.Values.externalPostgresql.port | int) .Values.externalPostgresql.database }}
{{- end }}
{{- end }}

{{/*
Redis address
*/}}
{{- define "k8dclusterlife.redisAddr" -}}
{{- if .Values.redis.enabled }}
{{- printf "%s-redis-master:6379" .Release.Name }}
{{- else }}
{{- printf "%s:%d" .Values.externalRedis.host (.Values.externalRedis.port | int) }}
{{- end }}
{{- end }}
