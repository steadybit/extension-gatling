{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "gatling.secret.name" -}}
{{- default "steadybit-extension-gatling" .Values.gatling.existingSecret -}}
{{- end -}}
