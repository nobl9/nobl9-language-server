`{{ .Name }}:{{ .Type }}`
{{- if .Doc }}

{{ .Doc }}
{{- end }}
{{- if gt (len .Rules) 0 }}

**Validation rules:**
{{ range .Rules }}
- {{ markdownEscape .Description }}{{- if .Details }}; {{ markdownEscape .Details }}{{- end }}
  {{- if .Conditions }}
  Conditions:
    {{- range .Conditions }}
  - {{ markdownEscape . }}
    {{- end }}
  {{- end }}
{{- end }}
{{- end }}
{{- if gt (len .Examples) 0 }}

**Examples:**

```yaml
{{- range $index, $value := .Examples }}
  {{- if gt $index 0 }}
---
  {{- end }}
{{ . }}
{{- end }}
```
{{- end }}