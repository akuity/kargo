{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "kargo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kargo.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "kargo.server.fullname" -}}
{{ include "kargo.fullname" . | printf "%s-server" }}
{{- end -}}

{{- define "kargo.controller.fullname" -}}
{{ include "kargo.fullname" . | printf "%s-controller" }}
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

{{- define "kargo.server.labels" -}}
app.kubernetes.io/component: server
{{- end -}}

{{- define "kargo.controller.labels" -}}
app.kubernetes.io/component: controller
{{- end -}}

{{- define "call-nested" }}
{{- $dot := index . 0 }}
{{- $subchart := index . 1 }}
{{- $template := index . 2 }}
{{- include $template (dict "Chart" (dict "Name" $subchart) "Values" (index $dot.Values $subchart) "Release" $dot.Release "Capabilities" $dot.Capabilities) }}
{{- end }}

{{/*
Return the appropriate apiVersion for a networking object.
*/}}
{{- define "networking.apiVersion" -}}
{{- if semverCompare ">=1.19-0" .Capabilities.KubeVersion.GitVersion -}}
{{- print "networking.k8s.io/v1" -}}
{{- else -}}
{{- print "networking.k8s.io/v1beta1" -}}
{{- end -}}
{{- end -}}

{{- define "networking.apiVersion.isStable" -}}
  {{- eq (include "networking.apiVersion" .) "networking.k8s.io/v1" -}}
{{- end -}}

{{- define "networking.apiVersion.supportIngressClassName" -}}
  {{- semverCompare ">=1.18-0" .Capabilities.KubeVersion.GitVersion -}}
{{- end -}}
