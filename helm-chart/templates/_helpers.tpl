{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "trivy-runner.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Expand the name of the chart.
*/}}
{{- define "trivy-runner.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "trivy-runner.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- printf "%s-%s" $name .Release.Namespace | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "trivy-runner.namespace" -}}
{{- if (and .Values.namespace.create .Values.namespace.name) }}
{{- .Values.namespace.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- .Release.Namespace }}
{{- end }}
{{- end }}

{{/*
Namespace
*/}}
{{- define "trivy-runner.serviceAccountName" -}}
{{- if (and .Values.serviceAccount.create .Values.serviceAccount.name) }}
{{- .Values.serviceAccount.name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" (include "trivy-runner.name" .) "sa" | trunc 63 }}
{{- end }}
{{- end }}


{{/*
Labels - Trivy
*/}}
{{- define "trivy-runner.labels" -}}
helm.sh/chart: {{ include "trivy-runner.chart" . }}
{{ include "trivy-runner.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}


{{/*
Selector labels - trivy
*/}}
{{- define "trivy-runner.selectorLabels" -}}
app.kubernetes.io/name: {{ include "trivy-runner.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Common environment variables among trivy containers
*/}}
{{- define "trivy-runner.redisEnv" -}}
- name: REDIS_HOST
  value: localhost
- name: REDIS_PORT
  value: "6379"
{{- end -}}

{{/*
Errbit variables. For now we reuse main Rails app Errbit keys
*/}}
{{- define "trivy-runner.pushworkerEnv" -}}
- name: WEBHOOK_URL
  value: "{{ .Values.pushworker.webhookUrl }}"
{{- end -}}

{{- define "trivy-runner.scanworkerEnv" -}}
{{- if .Values.pushworker.enabled }}
- name: PUSH_TO_CATALOG
  value: "true"
{{- end }}
{{- if .Values.scanworker.scanParallelism }}
- name: SCAN_PARALLELISM
  value: "{{ .Values.scanworker.scanParallelism }}"
{{- end -}}
{{- if .Values.scanworker.scanTimeout }}
- name: SCAN_TIMEOUT
  value: "{{ .Values.scanworker.scanTimeout }}"
{{- end -}}
{{- end -}}

{{- define "trivy-runner.trivyEnv" -}}
- name: TRIVY_ENV
  value: "{{ .Values.environment }}"

- name: SENTRY_DSN
  valueFrom:
    secretKeyRef:
      name: secrets
      key: sentry-dsn

- name: REGISTRY_USERNAME
  valueFrom:
    secretKeyRef:
      name: secrets
      key: registry-user
- name: REGISTRY_PASSWORD
  valueFrom:
    secretKeyRef:
      name: secrets
      key: registry-password
- name: IMAGES_APP_DIR
  value: /pool/images

- name: REPORTS_APP_DIR
  value: /pool/reports
{{- if .Values.pushworker.enabled }}
- name: PUSH_TO_CATALOG
  value: "1"
{{- end }}
{{- end -}}

{{/*
Expand fully qualified image name (repo/image:tag@sha)
*/}}
{{- define "trivy-runner.fullImageName" -}}
{{ .registry }}/{{ .repository }}
{{- if .tag }}:{{ .tag }}{{- end }}
{{- if .sha }}@sha256{{ .tag }}{{- end }}
{{- end }}
