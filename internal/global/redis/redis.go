package redis

import (
	"activity-punch-system/config"
	"activity-punch-system/tools"
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisClient 定义全局 Redis 客户端实例
var RedisClient *redis.Client

// customLogger 实现 log.Logger 接口，用于日志输出
type customLogger struct {
	enabled bool
}

func (l customLogger) Printf(ctx context.Context, format string, v ...interface{}) {
	if l.enabled {
		fmt.Printf(format, v...)
		fmt.Println() // 添加换行
	}
}

func Init() {
	// 构建 Redis 连接地址
	addr := fmt.Sprintf("%s:%s",
		config.Get().Redis.Host,
		config.Get().Redis.Port,
	)

	// 配置 Redis 客户端选项
	redisOptions := &redis.Options{
		Addr:     addr,
		Password: config.Get().Redis.Password, // 密码可能为空
		DB:       config.Get().Redis.DB,       // 选择数据库
	}

	// 根据运行模式设置日志
	var logger customLogger
	switch config.Get().Mode {
	case config.ModeDebug:
		logger = customLogger{enabled: true} // 启用日志
	case config.ModeRelease:
		logger = customLogger{enabled: false} // 禁用日志
	}
	redis.SetLogger(logger)

	// 初始化 Redis 客户端
	redisClient := redis.NewClient(redisOptions)
	ctx := context.Background()

	// 测试连接
	if err := redisClient.Ping(ctx).Err(); err != nil {
		tools.PanicOnErr(err)
	}
	RedisClient = redisClient
}
