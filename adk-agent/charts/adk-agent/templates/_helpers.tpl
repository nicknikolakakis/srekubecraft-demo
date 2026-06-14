{{/* Expand the name of the chart. */}}
{{- define "adk-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Fully qualified app name. Truncated at 63 chars (DNS-1123 limit). */}}
{{- define "adk-agent.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/* Chart label. */}}
{{- define "adk-agent.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Common labels. */}}
{{- define "adk-agent.labels" -}}
helm.sh/chart: {{ include "adk-agent.chart" . }}
{{ include "adk-agent.selectorLabels" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/* Selector labels (stable). */}}
{{- define "adk-agent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "adk-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* ServiceAccount name. */}}
{{- define "adk-agent.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "adk-agent.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/* Image reference: repository:tag. */}}
{{- define "adk-agent.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}

{{/* Validation: inline and externalSecret are mutually exclusive. */}}
{{- define "adk-agent.validateApiKey" -}}
{{- if and .Values.apiKey.inline.enabled .Values.apiKey.externalSecret.enabled -}}
{{- fail "apiKey.inline.enabled and apiKey.externalSecret.enabled are mutually exclusive" -}}
{{- end -}}
{{- if .Values.apiKey.inline.enabled -}}
{{- if not .Values.apiKey.inline.value -}}
{{- fail "apiKey.inline.enabled is true but apiKey.inline.value is empty" -}}
{{- end -}}
{{- end -}}
{{- if .Values.apiKey.externalSecret.enabled -}}
{{- if not .Values.apiKey.externalSecret.secretStoreRef.name -}}
{{- fail "apiKey.externalSecret.enabled is true but apiKey.externalSecret.secretStoreRef.name is empty" -}}
{{- end -}}
{{- if not .Values.apiKey.externalSecret.remoteRef.key -}}
{{- fail "apiKey.externalSecret.enabled is true but apiKey.externalSecret.remoteRef.key is empty" -}}
{{- end -}}
{{- end -}}
{{- if not .Values.apiKey.existingSecret.name -}}
{{- fail "apiKey.existingSecret.name must always be set; the Deployment reads from this Secret" -}}
{{- end -}}
{{- end -}}
