package skill

// Manifest Skill 声明文件 (skill.yaml)
type Manifest struct {
	Name        string      `yaml:"name"`
	Version     string      `yaml:"version"`
	Description string      `yaml:"description"`
	Permissions Permissions `yaml:"permissions"`
	Entrypoint  string      `yaml:"entrypoint"`
	Triggers    []Trigger   `yaml:"triggers,omitempty"`
}

// Permissions 权限声明
type Permissions struct {
	Network     []string `yaml:"network,omitempty"`     // 允许的域名
	Filesystem  string   `yaml:"filesystem,omitempty"`  // "none"|"read"|"write"
	Credentials []string `yaml:"credentials,omitempty"` // 需要的凭证名
}

// Trigger 触发条件
type Trigger struct {
	Type  string `yaml:"type"`  // "cron"|"keyword"|"event"
	Value string `yaml:"value"` // cron 表达式 / 关键词 / 事件名
}
