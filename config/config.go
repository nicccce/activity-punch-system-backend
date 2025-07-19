package config

type Mode string

const (
	ModeDebug   Mode = "debug"
	ModeRelease Mode = "release"
)

type Config struct {
	Host   string `envconfig:"HOST"`
	Port   string `envconfig:"PORT"`
	Prefix string `envconfig:"PREFIX"`
	Mode   Mode   `envconfig:"MODE"`
	Mysql  Mysql
	Redis  Redis
	JWT    JWT
	Log    Log
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
	AccessSecret string `envconfig:"ACCESS_SECRET"`
	AccessExpire int64  `envconfig:"ACCESS_EXPIRE"`
}

type Log struct {
	FilePath   string `envconfig:"LOG_FILE_PATH"`   // 日志文件路径
	Level      string `envconfig:"LOG_LEVEL"`       // 日志级别：debug, info, warn, error
	MaxSize    int    `envconfig:"LOG_MAX_SIZE"`    // 日志文件最大大小（MB）
	MaxBackups int    `envconfig:"LOG_MAX_BACKUPS"` // 保留的旧日志文件数
	MaxAge     int    `envconfig:"LOG_MAX_AGE"`     // 日志文件保留天数
	Compress   bool   `envconfig:"LOG_COMPRESS"`    // 是否压缩旧日志文件
}
