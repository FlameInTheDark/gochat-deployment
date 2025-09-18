{{- define "gochat.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "gochat.fullname" -}}
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

{{- define "gochat.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" -}}
{{- end -}}

{{- define "gochat.labels" -}}
helm.sh/chart: {{ include "gochat.chart" . }}
{{ include "gochat.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "gochat.selectorLabels" -}}
app.kubernetes.io/name: {{ include "gochat.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "gochat.componentLabels" -}}
{{ include "gochat.labels" .Context }}
app.kubernetes.io/component: {{ .Component }}
{{- if .ExtraLabels }}
{{- range $key, $value := .ExtraLabels }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "gochat.componentSelectorLabels" -}}
app.kubernetes.io/name: {{ include "gochat.name" .Context }}
app.kubernetes.io/instance: {{ .Context.Release.Name }}
app.kubernetes.io/component: {{ .Component }}
{{- if .ExtraLabels }}
{{- range $key, $value := .ExtraLabels }}
{{ $key }}: {{ $value | quote }}
{{- end }}
{{- end }}
{{- end -}}

{{- define "gochat.traefikServiceAccountName" -}}
{{- if .Values.serviceAccount.name -}}
{{- .Values.serviceAccount.name -}}
{{- else -}}
{{- printf "%s-traefik" (include "gochat.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
