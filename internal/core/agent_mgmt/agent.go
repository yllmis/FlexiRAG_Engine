package agent_mgmt

type Agent struct {
	ID uint `json:"id"`
	// 仅用于展示，如 "教务系统答疑机器人"
	Name string `json:"name"`

	// SystemPrompt 是系统提示词，通常用于设定机器人的行为和角色。
	SystemPrompt string `json:"system_prompt"`
}
