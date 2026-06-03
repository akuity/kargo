{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "kargo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Optional suffix appended to every controller-owned resource name. Default
empty. Set .Values.controller.fullnameSuffix to apply a uniform tag
(e.g. an instance discriminator) to the controller's Deployment,
ServiceAccount, ConfigMap, Role/RoleBinding, ClusterRole/ClusterRoleBinding,
and the derivative names below.
*/}}
{{- define "kargo.controller.fullnameSuffix" -}}
{{- default "" .Values.controller.fullnameSuffix -}}
{{- end -}}

{{/*
Per-resource fullname helpers for the controller subsystem. Each composes
the resource's natural name with the chart-name prefix and the optional
suffix. Resources reference these directly (rather than deriving sibling
names by string concatenation) so the suffix lands at the end of every
name uniformly.
*/}}
{{- define "kargo.controller.fullname" -}}
{{- printf "%s-controller%s" (include "kargo.name" .) (include "kargo.controller.fullnameSuffix" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kargo.controller.rolloutsFullname" -}}
{{- printf "%s-controller-rollouts%s" (include "kargo.name" .) (include "kargo.controller.fullnameSuffix" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kargo.controller.argocdFullname" -}}
{{- printf "%s-controller-argocd%s" (include "kargo.name" .) (include "kargo.controller.fullnameSuffix" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "kargo.controller.readSecretsFullname" -}}
{{- printf "%s-controller-read-secrets%s" (include "kargo.name" .) (include "kargo.controller.fullnameSuffix" .) | trunc 63 | trimSuffix "-" -}}
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
Resolve the namespace where many of Kargo's own resources (accessed at runtime)
reside. By default this is the chart's release namespace. In rare situations,
Kargo workloads may run in one namespace (the chart's release namespace) while
some resources accessed at runtime reside in another.
*/}}
{{- define "kargo.dataNamespace" -}}
{{- .Values.global.kargoNamespace | default .Release.Namespace -}}
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
{{- printf "^system:serviceaccount:%s:(%s)$" (include "kargo.dataNamespace" .) (join "|" $serviceAccounts) -}}
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

{{/*
kargo.cabundle.enabled returns the string "true" when the passed cabundle dict
has either `configMapName` or `secretName` set; empty string otherwise. Useful
inside `or` expressions in callers' outer guards.

Usage:
  {{- if or $kc (include "kargo.cabundle.enabled" .Values.api.cabundle) }}
  volumeMounts:
  ...
*/}}
{{- define "kargo.cabundle.enabled" -}}
{{- if or .configMapName .secretName -}}
true
{{- end -}}
{{- end -}}

{{/*
kargo.cabundle.volumeMount renders the volumeMount entry for the
cabundle-derived trust bundle at /etc/ssl/certs, if either configMapName or
secretName is set on the passed cabundle dict. Empty otherwise.

Pair with kargo.cabundle.initContainer and kargo.cabundle.volumes.

Usage:
  volumeMounts:
  ...
  {{- include "kargo.cabundle.volumeMount" .Values.api.cabundle | nindent 8 }}
*/}}
{{- define "kargo.cabundle.volumeMount" -}}
{{- if or .configMapName .secretName -}}
- mountPath: /etc/ssl/certs
  name: certs
{{- end -}}
{{- end -}}

{{/*
kargo.cabundle.initContainer renders the `parse-cabundle` init container that
unpacks the customer-supplied CA cert source (mounted at /tmp/source) into a
writable trust bundle mounted at /tmp/target. Pair with kargo.cabundle.volumes
which exposes the `cabundle` and `certs` volumes the init container expects.

The init container uses the kargo image (which has `update-ca-certificates`).

Args (dict):
  cabundle — the per-component cabundle config (configMapName / secretName)
  context  — the chart context whose .Values.image and .Chart back the kargo
             image lookup. Templates inside this chart pass `.` (their chart
             root). Templates in a parent chart that depends on this one as a
             subchart pass `.Subcharts.kargo` to point at this chart's context.

Empty when the passed cabundle dict has neither configMapName nor secretName.

Usage:
  initContainers:
  ...
  {{- include "kargo.cabundle.initContainer" (dict "cabundle" .Values.api.cabundle "context" .) | nindent 6 }}
*/}}
{{- define "kargo.cabundle.initContainer" -}}
{{- if or .cabundle.configMapName .cabundle.secretName -}}
- name: parse-cabundle
  image: {{ include "kargo.image" .context }}
  imagePullPolicy: {{ .context.Values.image.pullPolicy }}
  securityContext:
    runAsUser: 0
  command:
  - "/bin/sh"
  - "-c"
  args:
  - |
    for file in /tmp/source/*; do
      base_filename=$(basename "$file" .crt)
      awk 'BEGIN {c=0;} /BEGIN CERT/{c++} { print > "/usr/local/share/ca-certificates/" base_filename "." c ".crt"}' base_filename="$base_filename" < $file
    done
    /usr/sbin/update-ca-certificates
    find /etc/ssl/certs -type l -exec cp --remove-destination {} /etc/ssl/certs/ \;
    cp -r /etc/ssl/certs/* /tmp/target/
  volumeMounts:
  - name: cabundle
    mountPath: /tmp/source
  - name: certs
    mountPath: /tmp/target
{{- end -}}
{{- end -}}

{{/*
kargo.cabundle.volumes renders the `cabundle` source volume (sourced from a
Secret if `secretName` is set, otherwise from a ConfigMap by `configMapName`)
and the writable `certs` emptyDir volume that the init container populates.

Empty when the passed cabundle dict has neither configMapName nor secretName.

Usage:
  volumes:
  ...
  {{- include "kargo.cabundle.volumes" .Values.api.cabundle | nindent 6 }}
*/}}
{{- define "kargo.cabundle.volumes" -}}
{{- if .secretName -}}
- name: cabundle
  secret:
    secretName: {{ .secretName }}
{{ end -}}
{{- if and .configMapName (not .secretName) -}}
- name: cabundle
  configMap:
    name: {{ .configMapName }}
{{ end -}}
{{- if or .configMapName .secretName -}}
- name: certs
  emptyDir: {}
{{- end -}}
{{- end -}}
