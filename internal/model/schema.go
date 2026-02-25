package model

import (
	"time"
)

const (
	LangEN = "en"
	LangES = "es"
	LangFR = "fr"
	LangDE = "de"
	LangPT = "pt"
)

type LangInfo struct {
	Code string `json:"Code"`
	Name string `json:"Name"`
}

var SupportedLanguages = []LangInfo{
	{LangEN, "English"},
	{LangES, "Español"},
	{LangFR, "Français"},
	{LangDE, "Deutsch"},
	{LangPT, "Português"},
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
	ID       string                `json:"id"`
	Type     string                `json:"type"` // text, textarea, accordion
	Order    int                   `json:"order"`
	Required bool                  `json:"required"`
	Options  []string              `json:"options,omitempty"`
	I18n     map[string]FieldLocale `json:"i18n"`
}

type FieldLocale struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Placeholder string `json:"placeholder"`
	Order       int    `json:"order"` // per-language display order; 0 = use Field.Order
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
				LangFR: {
					Title:             "Rapport d'Incident Communautaire",
					Subtitle:          "Toutes les soumissions sont anonymes. Aucune information d'identification n'est collectée.",
					SubmitButtonLabel: "Soumettre le Rapport",
				},
				LangDE: {
					Title:             "Gemeinschaftlicher Vorfallsbericht",
					Subtitle:          "Alle Einreichungen sind anonym. Es werden keine identifizierenden Informationen gesammelt.",
					SubmitButtonLabel: "Bericht Einreichen",
				},
				LangPT: {
					Title:             "Relatório de Incidente Comunitário",
					Subtitle:          "Todas as submissões são anônimas. Nenhuma informação de identificação é coletada.",
					SubmitButtonLabel: "Enviar Relatório",
				},
			},
		},
		Fields: []Field{
			{
				ID: "size", Type: "text", Order: 1, Required: true,
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Size", Description: "Describe the number of people or scale of the incident.", Placeholder: "Approximately 10 individuals...", Order: 1},
					LangES: {Label: "Cantidad", Description: "Describa el número de personas o la magnitud del incidente.", Placeholder: "Aproximadamente 10 personas...", Order: 1},
					LangFR: {Label: "Nombre", Description: "Décrivez le nombre de personnes ou l'ampleur de l'incident.", Placeholder: "Environ 10 personnes...", Order: 1},
					LangDE: {Label: "Anzahl", Description: "Beschreiben Sie die Anzahl der Personen oder das Ausmaß des Vorfalls.", Placeholder: "Ungefähr 10 Personen...", Order: 1},
					LangPT: {Label: "Quantidade", Description: "Descreva o número de pessoas ou a dimensão do incidente.", Placeholder: "Aproximadamente 10 pessoas...", Order: 1},
				},
			},
			{
				ID: "activity", Type: "text", Order: 2, Required: true,
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Activity", Description: "What was happening? Describe the activity or behavior observed.", Placeholder: "A group was seen...", Order: 2},
					LangES: {Label: "Actividad", Description: "¿Qué estaba sucediendo? Describa la actividad o comportamiento observado.", Placeholder: "Se observó a un grupo...", Order: 2},
					LangFR: {Label: "Activité", Description: "Que se passait-il ? Décrivez l'activité ou le comportement observé.", Placeholder: "Un groupe a été vu...", Order: 2},
					LangDE: {Label: "Aktivität", Description: "Was geschah? Beschreiben Sie die beobachtete Aktivität oder das Verhalten.", Placeholder: "Eine Gruppe wurde gesehen...", Order: 2},
					LangPT: {Label: "Atividade", Description: "O que estava acontecendo? Descreva a atividade ou comportamento observado.", Placeholder: "Um grupo foi visto...", Order: 2},
				},
			},
			{
				ID: "location", Type: "text", Order: 3, Required: true,
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Location", Description: "Where did this occur?", Placeholder: "Near the east gate...", Order: 3},
					LangES: {Label: "Ubicación", Description: "¿Dónde ocurrió esto?", Placeholder: "Cerca de la puerta este...", Order: 3},
					LangFR: {Label: "Lieu", Description: "Où cela s'est-il produit ?", Placeholder: "Près de la porte est...", Order: 3},
					LangDE: {Label: "Ort", Description: "Wo ist dies passiert?", Placeholder: "In der Nähe des Osttors...", Order: 3},
					LangPT: {Label: "Localização", Description: "Onde isso ocorreu?", Placeholder: "Perto do portão leste...", Order: 3},
				},
			},
			{
				ID: "uniform", Type: "text", Order: 4, Required: false,
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Uniform", Description: "Describe any uniforms, markings, or affiliations observed.", Placeholder: "No visible markings...", Order: 4},
					LangES: {Label: "Uniforme", Description: "Describa uniformes, insignias o afiliaciones observadas.", Placeholder: "Sin marcas visibles...", Order: 4},
					LangFR: {Label: "Uniforme", Description: "Décrivez les uniformes, insignes ou affiliations observés.", Placeholder: "Aucun marquage visible...", Order: 4},
					LangDE: {Label: "Uniformen", Description: "Beschreiben Sie beobachtete Uniformen, Kennzeichnungen oder Zugehörigkeiten.", Placeholder: "Keine sichtbaren Kennzeichnungen...", Order: 4},
					LangPT: {Label: "Uniformes", Description: "Descreva uniformes, marcações ou afiliações observadas.", Placeholder: "Sem marcações visíveis...", Order: 4},
				},
			},
			{
				ID: "time", Type: "text", Order: 5, Required: true,
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Time", Description: "When did this occur?", Placeholder: "Around 14:30 today...", Order: 5},
					LangES: {Label: "Hora", Description: "¿Cuándo ocurrió esto?", Placeholder: "Alrededor de las 14:30 hoy...", Order: 5},
					LangFR: {Label: "Heure", Description: "Quand cela s'est-il produit ?", Placeholder: "Vers 14h30 aujourd'hui...", Order: 5},
					LangDE: {Label: "Zeit", Description: "Wann ist dies passiert?", Placeholder: "Heute gegen 14:30 Uhr...", Order: 5},
					LangPT: {Label: "Hora", Description: "Quando isso ocorreu?", Placeholder: "Por volta das 14h30 hoje...", Order: 5},
				},
			},
			{
				ID: "equipment", Type: "text", Order: 6, Required: false,
				I18n: map[string]FieldLocale{
					LangEN: {Label: "Equipment", Description: "Describe any equipment, vehicles, or tools observed.", Placeholder: "Two unmarked vehicles...", Order: 6},
					LangES: {Label: "Equipo", Description: "Describa cualquier equipo, vehículos o herramientas observadas.", Placeholder: "Dos vehículos sin identificación...", Order: 6},
					LangFR: {Label: "Équipement", Description: "Décrivez tout équipement, véhicule ou outil observé.", Placeholder: "Deux véhicules non marqués...", Order: 6},
					LangDE: {Label: "Ausrüstung", Description: "Beschreiben Sie beobachtete Ausrüstung, Fahrzeuge oder Werkzeuge.", Placeholder: "Zwei unmarkierte Fahrzeuge...", Order: 6},
					LangPT: {Label: "Equipamento", Description: "Descreva qualquer equipamento, veículos ou ferramentas observadas.", Placeholder: "Dois veículos sem identificação...", Order: 6},
				},
			},
		},
		EmailTemplates: map[string]string{
			LangEN: "New Community Report\n\nSize:\n{{size}}\n\nActivity:\n{{activity}}\n\nLocation:\n{{location}}\n\nUniform:\n{{uniform}}\n\nTime:\n{{time}}\n\nEquipment:\n{{equipment}}\n\n---\nThis report was submitted anonymously.",
		},
	}
}
