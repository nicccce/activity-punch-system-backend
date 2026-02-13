package config

type Mode string

const (
	ModeDebug   Mode = "debug"
	ModeRelease Mode = "release"
)

type Config struct {
	Host     string `envconfig:"HOST"`
	Port     string `envconfig:"PORT"`
	Domain   string `envconfig:"DOMAIN"`
	Prefix   string `envconfig:"PREFIX"`
	Storage  Storage
	Mode     Mode `envconfig:"MODE"`
	Mysql    Mysql
	Redis    Redis
	JWT      JWT
	Log      Log      `mapstructure:"Log"`
	Sdulogin Sdulogin `yaml:"Sdulogin" envconfig:"SDULOGIN"`
	S3       S3
	Sentry   Sentry `mapstructure:"Sentry"`
}

type Sentry struct {
	Dsn         string  `mapstructure:"dsn" envconfig:"SENTRY_DSN"`          // Sentry DSN
	Environment string  `mapstructure:"environment" envconfig:"SENTRY_ENV"`  // 环境标识，如 production, staging
	SampleRate  float64 `mapstructure:"sample_rate" envconfig:"SENTRY_RATE"` // 采样率，0.0-1.0
	Tracing     Tracing `mapstructure:"tracing"`                             // 性能追踪配置
}

// Tracing Sentry 性能追踪配置
type Tracing struct {
	// 数据库慢查询阈值（毫秒），仅记录超过此阈值的查询，0 表示记录所有查询
	DBSlowThresholdMs int64 `mapstructure:"db_slow_threshold_ms" envconfig:"SENTRY_DB_SLOW_MS"`
	// Redis 慢操作阈值（毫秒），仅记录超过此阈值的操作，0 表示记录所有操作
	RedisSlowThresholdMs int64 `mapstructure:"redis_slow_threshold_ms" envconfig:"SENTRY_REDIS_SLOW_MS"`
	// 是否记录所有外部 HTTP 调用（通常较慢且数量少，建议开启）
	TraceHTTPCalls bool `mapstructure:"trace_http_calls" envconfig:"SENTRY_TRACE_HTTP"`
}

type Storage struct {
	Home string
}

type S3 struct {
	Endpoint        string `mapstructure:"endpoint"`
	BaseURL         string `mapstructure:"base_url"`
	BackupHost      string `mapstructure:"backup_host"`
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	AccessKey       string `mapstructure:"access_key"`
	SecretAccessKey string `mapstructure:"secret_key"`
	Prefix          string `mapstructure:"prefix"`
	UsePathStyle    bool   `mapstructure:"path_style"`
}

type Sdulogin struct {
	CasKey string `yaml:"caskey" envconfig:"CASKEY" mapstructure:"caskey"`
	Mode   Mode   `yaml:"mode" envconfig:"MODE"`
}

type Mysql struct {
	Host     string `envconfig:"HOST"`
	Port     string `envconfig:"PORT"`
	Username string `envconfig:"USERNAME"`
	Password string `envconfig:"PASSWORD"`
	DBName   string `envconfig:"DB_NAME"`
}

type Redis struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWT struct {
	AccessSecret string `envconfig:"ACCESS_SECRET" mapstructure:"access_secret"`
	AccessExpire int64  `envconfig:"ACCESS_EXPIRE" mapstructure:"access_expire"`
}

type Log struct {
	FilePath   string `envconfig:"LOG_FILE_PATH" mapstructure:"file_path"`     // 日志文件路径
	Level      string `envconfig:"LOG_LEVEL" mapstructure:"level"`             // 日志级别：debug, info, warn, error
	MaxSize    int    `envconfig:"LOG_MAX_SIZE" mapstructure:"max_size"`       // 日志文件最大大小（MB）
	MaxBackups int    `envconfig:"LOG_MAX_BACKUPS" mapstructure:"max_backups"` // 保留的旧日志文件数
	MaxAge     int    `envconfig:"LOG_MAX_AGE" mapstructure:"max_age"`         // 日志文件保留天数
	Compress   bool   `envconfig:"LOG_COMPRESS" mapstructure:"compress"`       // 是否压缩旧日志文件
}
