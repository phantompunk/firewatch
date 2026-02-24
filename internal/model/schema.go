package model

import "time"

type ReportSchema struct {
	SchemaVersion int       `json:"schemaVersion"`
	UpdatedAt     time.Time `json:"updatedAt"`
	UpdatedBy     string    `json:"updatedBy,omitempty"`
	Page          PageMeta  `json:"page"`
	Fields        []Field   `json:"fields"`
	EmailTemplate string    `json:"emailTemplate"`
}

type PageMeta struct {
	Title             string `json:"title"`
	Subtitle          string `json:"subtitle"`
	SubmitButtonLabel string `json:"submitButtonLabel"`
}

type Field struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"` // text, textarea, select
	Order       int      `json:"order"`
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Placeholder string   `json:"placeholder"`
	Required    bool     `json:"required"`
	Options     []string `json:"options,omitempty"` // for select fields
}

// DefaultSALUTESchema returns the initial SALUTE report schema.
func DefaultSALUTESchema() ReportSchema {
	return ReportSchema{
		SchemaVersion: 1,
		UpdatedAt:     time.Now().UTC(),
		Page: PageMeta{
			Title:             "Community Incident Report",
			Subtitle:          "All submissions are anonymous. No identifying information is collected.",
			SubmitButtonLabel: "Submit Report",
		},
		Fields: []Field{
			{ID: "size", Type: "text", Order: 1, Label: "Size", Description: "Describe the number of people or scale of the incident.", Placeholder: "Approximately 10 individuals...", Required: true},
			{ID: "activity", Type: "text", Order: 2, Label: "Activity", Description: "What was happening? Describe the activity or behavior observed.", Placeholder: "A group was seen...", Required: true},
			{ID: "location", Type: "text", Order: 3, Label: "Location", Description: "Where did this occur?", Placeholder: "Near the east gate...", Required: true},
			{ID: "unit", Type: "text", Order: 4, Label: "Unit", Description: "Describe any uniforms, markings, or affiliations observed.", Placeholder: "No visible markings...", Required: false},
			{ID: "time", Type: "text", Order: 5, Label: "Time", Description: "When did this occur?", Placeholder: "Around 14:30 today...", Required: true},
			{ID: "equipment", Type: "text", Order: 6, Label: "Equipment", Description: "Describe any equipment, vehicles, or tools observed.", Placeholder: "Two unmarked vehicles...", Required: false},
		},
		EmailTemplate: "New Community Report\n\nSize:\n{{size}}\n\nActivity:\n{{activity}}\n\nLocation:\n{{location}}\n\nUnit:\n{{unit}}\n\nTime:\n{{time}}\n\nEquipment:\n{{equipment}}\n\n---\nThis report was submitted anonymously.",
	}
}
