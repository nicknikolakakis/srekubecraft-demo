{{- define "app.name" -}}
{{ .Release.Name }}
{{- end }}

{{- define "app.namespace" -}}
{{ .Release.Name }}-ns
{{- end }}

{{- define "app.labels" -}}
app.kubernetes.io/name: {{ include "app.name" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "app.selectorLabels" -}}
app.kubernetes.io/name: {{ include "app.name" . }}
{{- end }}
