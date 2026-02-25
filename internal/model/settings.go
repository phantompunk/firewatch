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
	PGPKey                string `json:"pgpKey"`

	// Verification state — set automatically on save and at startup.
	SMTPVerified bool   `json:"smtpVerified"`
	SMTPError    string `json:"smtpError"`
	PGPVerified  bool   `json:"pgpVerified"`
	PGPError     string `json:"pgpError"`
}
