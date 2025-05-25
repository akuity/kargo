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
Create default controlplane user regular expression with well-known service accounts
*/}}
{{- define "kargo.controlplane.defaultUserRegex" -}}
{{- $list := list }}
{{- if .Values.api.enabled }}
{{- $list = append $list "kargo-api" }}
{{- end }}
{{- if .Values.controller.enabled }}
{{- $list = append $list "kargo-controller" }}
{{- end }}
{{- if .Values.garbageCollector.enabled }}
{{- $list = append $list "kargo-garbage-collector" }}
{{- end }}
{{- if .Values.managementController.enabled }}
{{- $list = append $list "kargo-management-controller" }}
{{- end }}
{{- if $list }}
{{- printf "^system:serviceaccount:%s:(%s)$" .Release.Namespace (join "|" $list) }}
{{- end }}
{{- end }}

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
{{- with .Values.global.labels }}
{{ toYaml . }}
{{- end }}
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

{{- define "kargo.controller.labels" -}}
app.kubernetes.io/component: controller
{{- end -}}

{{- define "kargo.dexServer.labels" -}}
app.kubernetes.io/component: dex-server
{{- end -}}

{{- define "kargo.externalWebhooksServer.labels" -}}
app.kubernetes.io/component: external-webhooks-server
{{- end -}}

{{- define "kargo.garbageCollector.labels" -}}
app.kubernetes.io/component: garbage-collector
{{- end -}}

{{- define "kargo.kubernetesWebhooksServer.labels" -}}
app.kubernetes.io/component: kubernetes-webhooks-server
{{- end -}}

{{- define "kargo.managementController.labels" -}}
app.kubernetes.io/component: management-controller
{{- end -}}

{{/*
Generate the base URL for the API service.
*/}}
{{- define "kargo.api.baseURL" -}}
{{- if or (and .Values.api.ingress.enabled .Values.api.ingress.tls.enabled) (and (not .Values.api.ingress.enabled) .Values.api.tls.enabled) .Values.api.tls.terminatedUpstream -}}
{{- printf "https://%s" .Values.api.host -}}
{{- else -}}
{{- printf "http://%s" .Values.api.host -}}
{{- end -}}
{{- end -}}

{{/*
Generate the base URL for the external webhook server.
*/}}
{{- define "kargo.externalWebhookServer.baseURL" -}}
{{- if or (and .Values.externalWebhooksServer.ingress.enabled .Values.externalWebhooksServer.ingress.tls.enabled) (and (not .Values.externalWebhooksServer.ingress.enabled) .Values.externalWebhooksServer.tls.enabled) .Values.externalWebhooksServer.tls.terminatedUpstream -}}
{{- printf "https://%s" .Values.externalWebhooksServer.host -}}
{{- else -}}
{{- printf "http://%s" .Values.externalWebhooksServer.host -}}
{{- end -}}
{{- end -}}

{{/*
Determine the most appropriate CPU resource field for GOMAXPROCS.
Prioritizes limits over requests, with a fallback to limits if neither is set.
*/}}
{{- define "kargo.selectCpuResourceField" -}}
  {{- $resources := .resources -}}
  {{- $hasLimits := and $resources (hasKey $resources "limits") (ne (toString $resources.limits.cpu) "") -}}
  {{- $hasRequests := and $resources (hasKey $resources "requests") (ne (toString $resources.requests.cpu) "") -}}
  {{- if $hasLimits -}}
    limits.cpu
  {{- else if $hasRequests -}}
    requests.cpu
  {{- else -}}
    limits.cpu
  {{- end -}}
{{- end -}}

{{- define "call-nested" }}
{{- $dot := index . 0 }}
{{- $subchart := index . 1 }}
{{- $template := index . 2 }}
{{- include $template (dict "Chart" (dict "Name" $subchart) "Values" (index $dot.Values $subchart) "Release" $dot.Release "Capabilities" $dot.Capabilities) }}
{{- end }}
