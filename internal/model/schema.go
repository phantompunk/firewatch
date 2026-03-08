package model

import (
	"time"
)

const (
	LangEN = "en"
	LangES = "es"
)

type LangInfo struct {
	Code string `json:"Code"`
	Name string `json:"Name"`
}

var SupportedLanguages = []LangInfo{
	{LangEN, "English"},
	{LangES, "Español"},
}

type ReportSchema struct {
	SchemaVersion  int               `json:"schemaVersion"`
	UpdatedAt      time.Time         `json:"updatedAt"`
	UpdatedBy      string            `json:"updatedBy,omitempty"`
	Languages      []string          `json:"languages"`
	Page           PageMeta          `json:"page"`
	Fields         []Field           `json:"fields"`
	EmailTemplates map[string]string `json:"emailTemplates"`
}

type PageMeta struct {
	I18n map[string]PageLocale `json:"i18n"`
}

type PageLocale struct {
	Title             string `json:"title"`
	Subtitle          string `json:"subtitle"`
	SubmitButtonLabel string `json:"submitButtonLabel"`
}

type Field struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"` // text, textarea, accordion
	Order    int                    `json:"order"`
	Required bool                   `json:"required"`
	Prefix   string                 `json:"prefix,omitempty"` // optional accented letter shown before the field label
	Options  []string               `json:"options,omitempty"`
	I18n     map[string]FieldLocale `json:"i18n"`
}

type FieldLocale struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Placeholder string `json:"placeholder"`
	Prefix      string `json:"prefix,omitempty"` // overrides Field.Prefix for this language
	Order       int    `json:"order"`             // per-language display order; 0 = use Field.Order
}

// DefaultLang returns the first language in Languages, falling back to LangEN.
func (s *ReportSchema) DefaultLang() string {
	if len(s.Languages) > 0 {
		return s.Languages[0]
	}
	return LangEN
}

// Locale returns the PageLocale for lang, falling back to English.
func (pm PageMeta) Locale(lang string) PageLocale {
	if l, ok := pm.I18n[lang]; ok {
		return l
	}
	if l, ok := pm.I18n[LangEN]; ok {
		return l
	}
	return PageLocale{}
}

// Locale returns the FieldLocale for lang, falling back to English.
func (f Field) Locale(lang string) FieldLocale {
	if l, ok := f.I18n[lang]; ok {
		return l
	}
	if l, ok := f.I18n[LangEN]; ok {
		return l
	}
	return FieldLocale{}
}

// DisplayOrder returns the per-language display order, falling back to Field.Order.
func (f Field) DisplayOrder(lang string) int {
	if l, ok := f.I18n[lang]; ok && l.Order != 0 {
		return l.Order
	}
	return f.Order
}

// DefaultSALUTESchema returns the initial SALUTE report schema (v2).
func DefaultSALUTESchema() ReportSchema {
	return ReportSchema{
		SchemaVersion: 2,
		UpdatedAt:     time.Now().UTC(),
		Languages:     []string{LangEN},
		Page: PageMeta{
			I18n: map[string]PageLocale{
				LangEN: {
					Title:             "Community Incident Report",
					Subtitle:          "All submissions are anonymous. No identifying information is collected.",
					SubmitButtonLabel: "Submit Report",
				},
				LangES: {
					Title:             "Informe de Incidentes Comunitarios",
					Subtitle:          "Todas las presentaciones son anónimas. No se recopila información de identificación.",
					SubmitButtonLabel: "Enviar Informe",
				},
			},
		},
		Fields: []Field{
			{
				ID: "size", Type: "text", Order: 1, Required: true, Prefix: "S",
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Size", Description: "Describe the number of people or scale of the incident.", Placeholder: "Approximately 10 individuals...", Order: 1},
					LangES: {Label: "Cantidad", Description: "Describa el número de personas o la magnitud del incidente.", Placeholder: "Aproximadamente 10 personas...", Prefix: "C", Order: 1},
				},
			},
			{
				ID: "activity", Type: "text", Order: 2, Required: true, Prefix: "A",
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Activity", Description: "What was happening? Describe the activity or behavior observed.", Placeholder: "A group was seen...", Order: 2},
					LangES: {Label: "Actividad", Description: "¿Qué estaba sucediendo? Describa la actividad o comportamiento observado.", Placeholder: "Se observó a un grupo...", Prefix: "A", Order: 2},
				},
			},
			{
				ID: "location", Type: "text", Order: 3, Required: true, Prefix: "L",
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Location", Description: "Where did this occur?", Placeholder: "Near the east gate...", Order: 3},
					LangES: {Label: "Ubicación", Description: "¿Dónde ocurrió esto?", Placeholder: "Cerca de la puerta este...", Prefix: "U", Order: 3},
				},
			},
			{
				ID: "uniform", Type: "text", Order: 4, Required: false, Prefix: "U",
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Uniform", Description: "Describe any uniforms, markings, or affiliations observed.", Placeholder: "No visible markings...", Order: 4},
					LangES: {Label: "Uniforme", Description: "Describa uniformes, insignias o afiliaciones observadas.", Placeholder: "Sin marcas visibles...", Prefix: "U", Order: 4},
				},
			},
			{
				ID: "time", Type: "text", Order: 5, Required: true, Prefix: "T",
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Time", Description: "When did this occur?", Placeholder: "Around 14:30 today...", Order: 5},
					LangES: {Label: "Hora", Description: "¿Cuándo ocurrió esto?", Placeholder: "Alrededor de las 14:30 hoy...", Prefix: "H", Order: 5},
				},
			},
			{
				ID: "equipment", Type: "text", Order: 6, Required: false, Prefix: "E",
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Equipment", Description: "Describe any equipment, vehicles, or tools observed.", Placeholder: "Two unmarked vehicles...", Order: 6},
					LangES: {Label: "Equipo", Description: "Describa cualquier equipo, vehículos o herramientas observadas.", Placeholder: "Dos vehículos sin identificación...", Prefix: "E", Order: 6},
				},
			},
		},
		EmailTemplates: map[string]string{
			LangEN: "New Community Report\n\nSize:\n{{size}}\n\nActivity:\n{{activity}}\n\nLocation:\n{{location}}\n\nUniform:\n{{uniform}}\n\nTime:\n{{time}}\n\nEquipment:\n{{equipment}}\n\n---\nThis report was submitted anonymously.",
		},
	}
}
