package mailer

import (
	"strings"

	"github.com/firewatch/internal/model"
)

// RenderTemplate substitutes {{field_id}} tokens in the template with the
// corresponding submitted values. Unknown tokens are replaced with an empty string.
func RenderTemplate(tmpl string, submission map[string]string) string {
	result := tmpl
	for id, value := range submission {
		result = strings.ReplaceAll(result, "{{"+id+"}}", value)
	}
	return result
}

// RenderPreview substitutes tokens with placeholder values for display purposes.
// It uses the English locale for field labels and placeholders.
func RenderPreview(tmpl string, fields []model.Field) string {
	result := tmpl
	for _, f := range fields {
		locale := f.Locale(model.LangEN)
		sample := locale.Placeholder
		if sample == "" {
			sample = "[" + locale.Label + "]"
		}
		result = strings.ReplaceAll(result, "{{"+f.ID+"}}", sample)
	}
	return result
}
