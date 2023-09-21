{{/*
Expand the name of the chart.
*/}}
{{- define "s3gw-probe.name" -}}
{{- .Chart.Name }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "s3gw-probe.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "s3gw-probe.labels" -}}
helm.sh/chart: {{ include "s3gw-probe.chart" . }}
{{ include "s3gw-probe.commonSelectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "s3gw-probe.commonSelectorLabels" -}}
app.kubernetes.io/name: {{ include "s3gw-probe.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "s3gw-probe.selectorLabels" -}}
{{ include "s3gw-probe.commonSelectorLabels" . }}
app.kubernetes.io/component: application
{{- end }}

{{- define "s3gw-probe-ui.selectorLabels" -}}
{{ include "s3gw-probe.commonSelectorLabels" . }}
app.kubernetes.io/component: ui
{{- end }}

{{/*
Version helpers for the image tag
*/}}
{{- define "s3gw-probe.image" -}}
{{- $defaulttag := printf "%s" "latest" }}
{{- $tag := default $defaulttag .Values.probe.imageTag }}
{{- $name := default "giubacc/s3gw-probe" .Values.probe.imageName }}
{{- $registry := default "ghcr.io" .Values.probe.imageRegistry }}
{{- printf "%s/%s:%s" $registry $name $tag }}
{{- end }}

{{- define "s3gw-probe-ui.image" -}}
{{- $tag := default (printf "v%s" .Chart.Version) .Values.ui.imageTag }}
{{- $name := default "giubacc/s3gw-probe-ui" .Values.ui.imageName }}
{{- $registry := default "ghcr.io" .Values.ui.imageRegistry }}
{{- printf "%s/%s:%s" $registry $name $tag }}
{{- end }}

{{/*
Traefik Middleware CORS name
*/}}
{{- define "s3gw-probe.CORSMiddlewareName" -}}
{{- $dmcn := printf "%s-%s-cors-header" .Release.Name .Release.Namespace }}
{{- $name := $dmcn }}
{{- $name }}
{{- end }}

{{/*
probe service name
*/}}
{{- define "s3gw-probe.serviceName" -}}
{{- $dsn := printf "%s-%s" .Release.Name .Release.Namespace }}
{{- $name := default $dsn .Values.probe.serviceName }}
{{- $name }}
{{- end }}
