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
}

type Storage struct {
	Home string
}

type S3 struct {
	Endpoint        string `mapstructure:"endpoint"`
	BaseURL         string `mapstructure:"base_url"`
	Bucket          string `mapstructure:"bucket"`
	Region          string `mapstructure:"region"`
	AccessKey       string `mapstructure:"access_key"`
	SecretAccessKey string `mapstructure:"secret_key"`
	Prefix          string `mapstructure:"prefix"`
	UsePathStyle    bool   `mapstructure:"path_style"`
}

type Sdulogin struct {
	CasKey string `yaml:"caskey" envconfig:"CASKEY"`
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
	AccessSecret string `envconfig:"ACCESS_SECRET"`
	AccessExpire int64  `envconfig:"ACCESS_EXPIRE"`
}

type Log struct {
	FilePath   string `envconfig:"LOG_FILE_PATH" mapstructure:"file_path"`     // 日志文件路径
	Level      string `envconfig:"LOG_LEVEL" mapstructure:"level"`             // 日志级别：debug, info, warn, error
	MaxSize    int    `envconfig:"LOG_MAX_SIZE" mapstructure:"max_size"`       // 日志文件最大大小（MB）
	MaxBackups int    `envconfig:"LOG_MAX_BACKUPS" mapstructure:"max_backups"` // 保留的旧日志文件数
	MaxAge     int    `envconfig:"LOG_MAX_AGE" mapstructure:"max_age"`         // 日志文件保留天数
	Compress   bool   `envconfig:"LOG_COMPRESS" mapstructure:"compress"`       // 是否压缩旧日志文件
}
