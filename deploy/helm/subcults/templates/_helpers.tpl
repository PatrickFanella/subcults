{{/*
Common template helpers for the subcults chart.
*/}}

{{/*
Expand the name of the chart.
*/}}
{{- define "subcults.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "subcults.fullname" -}}
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
{{- define "subcults.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "subcults.labels" -}}
helm.sh/chart: {{ include "subcults.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: subcults
{{- end }}

{{/*
Selector labels for a specific component.
Usage: {{ include "subcults.selectorLabels" (dict "root" . "component" "api") }}
*/}}
{{- define "subcults.selectorLabels" -}}
app.kubernetes.io/name: {{ include "subcults.name" .root }}
app.kubernetes.io/instance: {{ .root.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}

{{/*
Full image path for a component.
Usage: {{ include "subcults.image" (dict "root" . "image" .Values.api.image) }}
*/}}
{{- define "subcults.image" -}}
{{- printf "%s/%s:%s" .root.Values.global.imageRegistry .image.repository .image.tag }}
{{- end }}
