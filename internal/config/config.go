package config

import (
	"os"

	"github.com/spf13/viper"
)

// DatabaseConfig controls which GORM driver is used and how to connect.
type DatabaseConfig struct {
	Type string `mapstructure:"type" json:"type"` // sqlite | postgres | mysql
	DSN  string `mapstructure:"dsn" json:"dsn"`   // file path for sqlite, connection string otherwise
}

// PomodoroConfig holds the timing preferences for the Pomodoro cycle.
type PomodoroConfig struct {
	WorkDuration       int  `mapstructure:"work_duration" json:"work_duration"`
	ShortBreak         int  `mapstructure:"short_break" json:"short_break"`
	LongBreak          int  `mapstructure:"long_break" json:"long_break"`
	SessionsBeforeLong int  `mapstructure:"sessions_before_long" json:"sessions_before_long"`
	DailyGoal          int  `mapstructure:"daily_goal" json:"daily_goal"` // work sessions per day
	Sound              bool `mapstructure:"sound" json:"sound"`           // chime when a session ends
}

// AppConfig holds general application preferences.
type AppConfig struct {
	Theme              string `mapstructure:"theme" json:"theme"`   // surface mode: cosy | dark | light
	Accent             string `mapstructure:"accent" json:"accent"` // accent/mascot: duck | tomato | orange
	MinimizeToTray     bool   `mapstructure:"minimize_to_tray" json:"minimize_to_tray"`
	AgentNotifications bool   `mapstructure:"agent_notifications" json:"agent_notifications"` // toast when agents create/complete tasks
}

// MCPConfig controls the embedded MCP server that lets coding agents
// (Claude Code etc.) manage tasks without driving the UI.
type MCPConfig struct {
	Enabled bool `mapstructure:"enabled" json:"enabled"`
	Port    int  `mapstructure:"port" json:"port"` // listens on 127.0.0.1 only
}

// WindowConfig remembers where the user last put the widget. Saved is the
// explicit "a position was stored" flag — (0,0) and negative coordinates are
// all valid on multi-monitor setups, so no coordinate can act as a sentinel.
type WindowConfig struct {
	X     int  `mapstructure:"x" json:"x"`
	Y     int  `mapstructure:"y" json:"y"`
	Saved bool `mapstructure:"saved" json:"saved"`
}

// Config is the root configuration object loaded from config.yaml.
type Config struct {
	Database DatabaseConfig `mapstructure:"database" json:"database"`
	Pomodoro PomodoroConfig `mapstructure:"pomodoro" json:"pomodoro"`
	App      AppConfig      `mapstructure:"app" json:"app"`
	Window   WindowConfig   `mapstructure:"window" json:"window"`
	MCP      MCPConfig      `mapstructure:"mcp" json:"mcp"`
}

// setDefaults registers sensible defaults so the app works with zero config.
func setDefaults(v *viper.Viper) {
	v.SetDefault("database.type", "sqlite")
	v.SetDefault("database.dsn", "tasks.db")

	v.SetDefault("pomodoro.work_duration", 25)
	v.SetDefault("pomodoro.short_break", 5)
	v.SetDefault("pomodoro.long_break", 15)
	v.SetDefault("pomodoro.sessions_before_long", 4)
	v.SetDefault("pomodoro.daily_goal", 8)
	v.SetDefault("pomodoro.sound", true)

	v.SetDefault("app.theme", "cosy")
	v.SetDefault("app.accent", "duck")
	v.SetDefault("app.minimize_to_tray", true)
	v.SetDefault("app.agent_notifications", true)

	v.SetDefault("window.x", 0)
	v.SetDefault("window.y", 0)
	v.SetDefault("window.saved", false)

	v.SetDefault("mcp.enabled", true)
	v.SetDefault("mcp.port", 7823)
}

// Load reads configuration from the given YAML path. If the file does not
// exist it is created with defaults. Missing individual keys fall back to
// their defaults, so the app always starts with a valid configuration.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			// First run: persist the defaults so the user has a file to edit.
			if werr := v.WriteConfigAs(path); werr != nil {
				return nil, werr
			}
		} else {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	cfg.applyFallbacks()
	return &cfg, nil
}

// Save writes the configuration back to the given YAML path, preserving the
// snake_case keys used by the rest of the app.
func Save(path string, cfg *Config) error {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	v.Set("database.type", cfg.Database.Type)
	v.Set("database.dsn", cfg.Database.DSN)

	v.Set("pomodoro.work_duration", cfg.Pomodoro.WorkDuration)
	v.Set("pomodoro.short_break", cfg.Pomodoro.ShortBreak)
	v.Set("pomodoro.long_break", cfg.Pomodoro.LongBreak)
	v.Set("pomodoro.sessions_before_long", cfg.Pomodoro.SessionsBeforeLong)
	v.Set("pomodoro.daily_goal", cfg.Pomodoro.DailyGoal)
	v.Set("pomodoro.sound", cfg.Pomodoro.Sound)

	v.Set("app.theme", cfg.App.Theme)
	v.Set("app.accent", cfg.App.Accent)
	v.Set("app.minimize_to_tray", cfg.App.MinimizeToTray)
	v.Set("app.agent_notifications", cfg.App.AgentNotifications)

	v.Set("window.x", cfg.Window.X)
	v.Set("window.y", cfg.Window.Y)
	v.Set("window.saved", cfg.Window.Saved)

	v.Set("mcp.enabled", cfg.MCP.Enabled)
	v.Set("mcp.port", cfg.MCP.Port)

	return v.WriteConfigAs(path)
}

// applyFallbacks guards against a partially-filled config file leaving zero
// values that would break the timer (e.g. a 0-minute work session).
func (c *Config) applyFallbacks() {
	if c.Database.Type == "" {
		c.Database.Type = "sqlite"
	}
	if c.Database.DSN == "" {
		c.Database.DSN = "tasks.db"
	}
	if c.Pomodoro.WorkDuration <= 0 {
		c.Pomodoro.WorkDuration = 25
	}
	if c.Pomodoro.ShortBreak <= 0 {
		c.Pomodoro.ShortBreak = 5
	}
	if c.Pomodoro.LongBreak <= 0 {
		c.Pomodoro.LongBreak = 15
	}
	if c.Pomodoro.SessionsBeforeLong <= 0 {
		c.Pomodoro.SessionsBeforeLong = 4
	}
	if c.Pomodoro.DailyGoal <= 0 {
		c.Pomodoro.DailyGoal = 8
	}
	if c.App.Theme == "" {
		c.App.Theme = "cosy"
	}
	if c.App.Accent == "" {
		c.App.Accent = "duck"
	}
	if c.MCP.Port <= 0 || c.MCP.Port > 65535 {
		c.MCP.Port = 7823
	}
}
