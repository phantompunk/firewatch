package model

type AppSettings struct {
	DestinationEmail      string `json:"destinationEmail"`
	EmailSubjectTemplate  string `json:"emailSubjectTemplate"`
	SMTPHost              string `json:"smtpHost"`
	SMTPPort              int    `json:"smtpPort"`
	SMTPUser              string `json:"smtpUser"`
	SMTPPass              string `json:"smtpPass"`
	SMTPFromAddress       string `json:"smtpFromAddress"`
	SMTPFromName          string `json:"smtpFromName"`
	ReportRetentionPolicy string `json:"reportRetentionPolicy"`
	MaintenanceMode       bool   `json:"maintenanceMode"`
}
