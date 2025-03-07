`{{ .Name }}` {{ .Kind }}
{{- if .Description }}

{{ .Description }}
{{- end }}

```yaml
{{ .YAML -}}
```