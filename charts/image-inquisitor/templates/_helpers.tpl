{{/*
Expand the name of the chart.
*/}}
{{- define "image-inquisitor.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "image-inquisitor.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "image-inquisitor.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "image-inquisitor.labels" -}}
helm.sh/chart: {{ include "image-inquisitor.chart" . }}
{{ include "image-inquisitor.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "image-inquisitor.selectorLabels" -}}
app.kubernetes.io/name: {{ include "image-inquisitor.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Selector labels of one component: (dict "ctx" $ "component" "daemon")
*/}}
{{- define "image-inquisitor.componentSelectorLabels" -}}
{{ include "image-inquisitor.selectorLabels" .ctx }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Labels of one component: (dict "ctx" $ "component" "daemon")
*/}}
{{- define "image-inquisitor.componentLabels" -}}
{{ include "image-inquisitor.labels" .ctx }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
The daemon config file: values.config plus the settings the chart owns
*/}}
{{- define "image-inquisitor.config" -}}
{{- $config := deepCopy .Values.config }}
{{- $_ := set $config "listen" (printf ":%v" .Values.daemon.service.port) }}
{{- $_ = set $config "dataDir" "/var/lib/imginq" }}
{{- if .Values.kubeconfig.existingSecret }}
{{- $kubernetes := get $config "kubernetes" | default dict }}
{{- $_ = set $kubernetes "kubeconfig" "/etc/imginq/kubeconfig/config" }}
{{- $_ = set $config "kubernetes" $kubernetes }}
{{- end }}
{{- with .Values.renovate }}
{{- if .enabled }}
{{- $tokenSecret := .github.existingSecret | default (printf "%s-renovate" (include "image-inquisitor.fullname" $)) }}
{{- $tokenKey := ternary .github.existingSecretKey "token" (ne .github.existingSecret "") }}
{{- $token := dict "valueFrom" (dict "secretKeyRef" (dict "name" $tokenSecret "key" $tokenKey)) }}
{{- $env := list (merge (dict "name" "RENOVATE_TOKEN") $token) (merge (dict "name" "RENOVATE_GITHUB_COM_TOKEN") (deepCopy $token)) }}
{{- $renovate := dict
  "cluster" .cluster
  "namespace" $.Release.Namespace
  "image" (printf "%s:%s" .image.repository .image.tag)
  "interval" .interval
  "timeout" .timeout
  "config" .config
  "env" (concat $env .env)
  "resources" .resources
  "nodeSelector" .nodeSelector
}}
{{- if .dnsConfig }}
{{- $_ = set $renovate "dnsConfig" .dnsConfig }}
{{- end }}
{{- if .soak }}
{{- $_ = set $renovate "soak" .soak }}
{{- end }}
{{- if .cache.enabled }}
{{- $_ = set $renovate "cacheClaim" (printf "%s-renovate-cache" (include "image-inquisitor.fullname" $)) }}
{{- end }}
{{- $_ = set $config "renovate" $renovate }}
{{- end }}
{{- end }}
{{- toYaml $config }}
{{- end }}

{{/*
The Secret and key holding the GitHub token, if any
*/}}
{{- define "image-inquisitor.githubSecret" -}}
{{- if .Values.github.existingSecret }}{{ .Values.github.existingSecret }}{{ else if .Values.github.token }}{{ include "image-inquisitor.fullname" . }}-github{{ end }}
{{- end }}
