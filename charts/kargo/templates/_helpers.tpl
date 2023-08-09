{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "kargo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create image reference as used by resources.
*/}}

{{- define "kargo.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kargo.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "kargo.labels" -}}
helm.sh/chart: {{ include "kargo.chart" . }}
{{ include "kargo.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "kargo.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kargo.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "kargo.api.labels" -}}
app.kubernetes.io/component: api
{{- end -}}

{{- define "kargo.dexServer.labels" -}}
app.kubernetes.io/component: dex-server
{{- end -}}

{{- define "kargo.controller.labels" -}}
app.kubernetes.io/component: controller
{{- end -}}

{{- define "kargo.garbageCollector.labels" -}}
app.kubernetes.io/component: garbage-collector
{{- end -}}

{{- define "kargo.webhooksServer.labels" -}}
app.kubernetes.io/component: webhooks-server
{{- end -}}

{{- define "call-nested" }}
{{- $dot := index . 0 }}
{{- $subchart := index . 1 }}
{{- $template := index . 2 }}
{{- include $template (dict "Chart" (dict "Name" $subchart) "Values" (index $dot.Values $subchart) "Release" $dot.Release "Capabilities" $dot.Capabilities) }}
{{- end }}
