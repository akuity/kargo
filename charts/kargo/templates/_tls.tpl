{{/* ============================================================================
   TLS helpers (per-resource). Each helper returns the correct Secret name
   using this precedence:
     - If selfSignedCert=false AND secretName is set => use that secretName (BYO)
     - Else => use the component's default self-signed secret name
   No validation / no fail-fast.
   ============================================================================ */}}

{{/* API server TLS (values: api.tls.*, default: kargo-api-cert) */}}
{{- define "kargo.api.serverTLSSecretName" -}}
{{- $tls := .Values.api.tls -}}
{{- if and (not ($tls.selfSignedCert)) ($tls.secretName) -}}
  {{- $tls.secretName -}}
{{- else -}}
  {{- "kargo-api-cert" -}}
{{- end -}}
{{- end -}}

{{/* API Ingress TLS (values: api.ingress.tls.*, default: kargo-api-ingress-cert) */}}
{{- define "kargo.api.ingressTLSSecretName" -}}
{{- $tls := .Values.api.ingress.tls -}}
{{- if and (not ($tls.selfSignedCert)) ($tls.secretName) -}}
  {{- $tls.secretName -}}
{{- else -}}
  {{- "kargo-api-ingress-cert" -}}
{{- end -}}
{{- end -}}

{{/* Dex server TLS (values: api.oidc.dex.tls.*, default: kargo-dex-server-cert) */}}
{{- define "kargo.dex.serverTLSSecretName" -}}
{{- $tls := .Values.api.oidc.dex.tls -}}
{{- if and (not ($tls.selfSignedCert)) ($tls.secretName) -}}
  {{- $tls.secretName -}}
{{- else -}}
  {{- "kargo-dex-server-cert" -}}
{{- end -}}
{{- end -}}

{{/* External Webhooks Server TLS (values: externalWebhooksServer.tls.*, default: kargo-external-webhooks-server-cert) */}}
{{- define "kargo.externalWebhooks.serverTLSSecretName" -}}
{{- $tls := .Values.externalWebhooksServer.tls -}}
{{- if and (not ($tls.selfSignedCert)) ($tls.secretName) -}}
  {{- $tls.secretName -}}
{{- else -}}
  {{- "kargo-external-webhooks-server-cert" -}}
{{- end -}}
{{- end -}}

{{/* External Webhooks Server Ingress TLS (values: externalWebhooksServer.ingress.tls.*, default: kargo-external-webhooks-server-ingress-cert) */}}
{{- define "kargo.externalWebhooks.ingressTLSSecretName" -}}
{{- $tls := .Values.externalWebhooksServer.ingress.tls -}}
{{- if and (not ($tls.selfSignedCert)) ($tls.secretName) -}}
  {{- $tls.secretName -}}
{{- else -}}
  {{- "kargo-external-webhooks-server-ingress-cert" -}}
{{- end -}}
{{- end -}}

{{/* Built-in Webhooks Server TLS (values: webhooksServer.tls.*, default: kargo-webhooks-server-cert) */}}
{{- define "kargo.webhooksServer.tlsSecretName" -}}
{{- $tls := .Values.webhooksServer.tls -}}
{{- if and (not ($tls.selfSignedCert)) ($tls.secretName) -}}
  {{- $tls.secretName -}}
{{- else -}}
  {{- "kargo-webhooks-server-cert" -}}
{{- end -}}
{{- end -}}
