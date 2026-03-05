package agent_mgmt

type Agent struct {
	ID           uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"type:varchar(100);not null" json:"name"`
	SystemPrompt string `gorm:"type:text;not null" json:"system_prompt"`
}
