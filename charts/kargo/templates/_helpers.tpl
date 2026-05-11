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
Check if TLS should be used for a service.
*/}}
{{- define "kargo.useTLS" -}}
{{- $service := . -}}
{{- or (and $service.ingress.enabled $service.ingress.tls.enabled)
       (and (not $service.ingress.enabled) $service.tls.enabled)
       $service.tls.terminatedUpstream -}}
{{- end -}}

{{/*
Generate base URL for a service.
*/}}
{{- define "kargo.baseURL" -}}
{{- $service := .service -}}
{{- $host := .host -}}
{{- if eq (include "kargo.useTLS" $service) "true" -}}
{{- printf "https://%s" $host -}}
{{- else -}}
{{- printf "http://%s" $host -}}
{{- end -}}
{{- end -}}

{{/*
Generate the base URL for the API service.
*/}}
{{- define "kargo.api.baseURL" -}}
{{- include "kargo.baseURL" (dict "service" .Values.api "host" .Values.api.host) -}}
{{- end -}}

{{/*
Generate the base URL for the external webhook server.
*/}}
{{- define "kargo.externalWebhooksServer.baseURL" -}}
{{- $apiService := .Values.api -}}
{{- $webhookService := .Values.externalWebhooksServer -}}
{{- if and (not $webhookService.ingress.enabled) $apiService.enabled $apiService.ingress.enabled -}}
{{- printf "%s/webhooks" (include "kargo.api.baseURL" .) -}}
{{- else -}}
{{- include "kargo.baseURL" (dict "service" $webhookService "host" $webhookService.host) -}}
{{- end -}}
{{- end -}}

{{/*
Create default controlplane user regular expression with well-known service accounts.
*/}}
{{- define "kargo.controlplane.defaultUserRegex" -}}
{{- $components := dict
    "api" .Values.api.enabled
    "controller" .Values.controller.enabled
    "garbage-collector" .Values.garbageCollector.enabled
    "management-controller" .Values.managementController.enabled -}}
{{- $serviceAccounts := list -}}
{{- range $name, $enabled := $components -}}
{{- if $enabled -}}
{{- $serviceAccounts = append $serviceAccounts (printf "kargo-%s" $name) -}}
{{- end -}}
{{- end -}}
{{- if $serviceAccounts -}}
{{- printf "^system:serviceaccount:%s:(%s)$" .Release.Namespace (join "|" $serviceAccounts) -}}
{{- end -}}
{{- end -}}

{{/*
Determine the most appropriate CPU resource field for GOMAXPROCS.
*/}}
{{- define "kargo.selectCpuResourceField" -}}
{{- $resources := .resources -}}
{{- $cpu := dict -}}
{{- if $resources -}}
{{- if and $resources.limits $resources.limits.cpu -}}
{{- $cpu = set $cpu "field" "limits.cpu" -}}
{{- else if and $resources.requests $resources.requests.cpu -}}
{{- $cpu = set $cpu "field" "requests.cpu" -}}
{{- else -}}
{{- $cpu = set $cpu "field" "limits.cpu" -}}
{{- end -}}
{{- else -}}
{{- $cpu = set $cpu "field" "limits.cpu" -}}
{{- end -}}
{{- $cpu.field -}}
{{- end -}}

{{/*
kargo.annotations renders a complete metadata annotations block by merging
.Values.global.annotations with a per-component annotations map.

Call it by passing a dict with two keys:
  - "root": the top-level chart context (.)
  - "annotations": the component-specific annotations map (e.g. .Values.controller.annotations)

Example:
  {{- include "kargo.annotations" (dict "root" . "annotations" .Values.controller.annotations) | nindent 2 }}

When the merged result is empty the helper emits nothing, so it is always safe to
include unconditionally — no surrounding `with` is required.

Use this helper only when the annotations block consists solely of the merged
global + component values. When additional static annotations are required (e.g.
cert-manager CA injection, configmap checksums), write the annotations block
inline and call mergeOverwrite directly.

For resources that have no component-specific annotations, omit the "annotations"
key or pass an empty dict:
  {{- include "kargo.annotations" (dict "root" .) | nindent 2 }}
*/}}
{{- define "kargo.annotations" -}}
{{- with (mergeOverwrite (deepCopy .root.Values.global.annotations) (.annotations | default dict)) -}}
annotations:
  {{- range $key, $value := . }}
  {{ $key }}: {{ $value | quote }}
  {{- end }}
{{- end }}
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
