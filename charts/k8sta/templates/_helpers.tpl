{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "k8sta.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "k8sta.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "k8sta.bookkeeper.server.fullname" -}}
{{ include "k8sta.fullname" . | printf "%s-bookkeeper-server" }}
{{- end -}}

{{- define "k8sta.server.fullname" -}}
{{ include "k8sta.fullname" . | printf "%s-server" }}
{{- end -}}

{{- define "k8sta.controller.fullname" -}}
{{ include "k8sta.fullname" . | printf "%s-controller" }}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "k8sta.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "k8sta.labels" -}}
helm.sh/chart: {{ include "k8sta.chart" . }}
{{ include "k8sta.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels
*/}}
{{- define "k8sta.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k8sta.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "k8sta.bookkeeper.server.labels" -}}
app.kubernetes.io/component: bookkeeper-server
{{- end -}}

{{- define "k8sta.server.labels" -}}
app.kubernetes.io/component: server
{{- end -}}

{{- define "k8sta.controller.labels" -}}
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
