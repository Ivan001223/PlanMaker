package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App            AppConfig        `mapstructure:"app"`
	DB             DBConfig         `mapstructure:"db"`
	Redis          RedisConfig      `mapstructure:"redis"`
	MongoDB        MongoDBConfig    `mapstructure:"mongodb"`
	RabbitMQ       RabbitMQConfig   `mapstructure:"rabbitmq"`
	MinIO          MinIOConfig      `mapstructure:"minio"`
	WeChat         WeChatConfig     `mapstructure:"wechat"`
	LLM            LLMConfig        `mapstructure:"llm"`
	JWT            JWTConfig        `mapstructure:"jwt"`
	PDFService     PDFServiceConfig `mapstructure:"pdf_service"`
	InternalAPIKey string           `mapstructure:"internal_api_key"`
}

type AppConfig struct {
	Name               string `mapstructure:"name"`
	Env                string `mapstructure:"env"`
	Port               int    `mapstructure:"port"`
	CORSAllowedOrigins string `mapstructure:"cors_allowed_origins"`
}

type DBConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

func (c *DBConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai",
		c.Host, c.Port, c.User, c.Password, c.Name)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func (c *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

type RabbitMQConfig struct {
	URL string `mapstructure:"url"`
}

type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

type WeChatConfig struct {
	AppID     string `mapstructure:"app_id"`
	AppSecret string `mapstructure:"app_secret"`
}

type LLMConfig struct {
	APIKey        string `mapstructure:"api_key"`
	BaseURL       string `mapstructure:"base_url"`
	DefaultModel  string `mapstructure:"default_model"`
	PlanningModel string `mapstructure:"planning_model"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpireHours int    `mapstructure:"expire_hours"`
}

type PDFServiceConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

// Validate 验证配置完整性
func (c *Config) Validate() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET 未设置，请通过环境变量 JWT_SECRET 设置")
	}
	// 生产环境必须配置微信
	if c.App.Env == "production" && (c.WeChat.AppID == "" || c.WeChat.AppSecret == "") {
		return fmt.Errorf("生产环境必须配置微信 AppID 和 AppSecret")
	}
	// 检查数据库密码
	if c.DB.Password == "" {
		return fmt.Errorf("数据库密码未设置，请通过环境变量 DB_PASSWORD 设置")
	}
	return nil
}

// Load 加载配置，支持环境变量覆盖
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 环境变量覆盖: DB_HOST -> db.host
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 尝试加载 .env 文件 (如果存在)
	viper.SetConfigName(".env")
	viper.AddConfigPath(".")
	_ = viper.MergeInConfig() // 忽略错误，.env 可能不存在

	// 重新设置回 config.yaml 并读取
	viper.SetConfigName("config")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 环境变量优先级高于配置文件
	overrideFromEnv(&cfg)

	return &cfg, nil
}

func overrideFromEnv(cfg *Config) {
	if v := viper.GetString("DB_HOST"); v != "" {
		cfg.DB.Host = v
	}
	if v := viper.GetInt("DB_PORT"); v != 0 {
		cfg.DB.Port = v
	}
	if v := viper.GetString("DB_USER"); v != "" {
		cfg.DB.User = v
	}
	if v := viper.GetString("DB_PASSWORD"); v != "" {
		cfg.DB.Password = v
	}
	if v := viper.GetString("DB_NAME"); v != "" {
		cfg.DB.Name = v
	}
	if v := viper.GetString("REDIS_HOST"); v != "" {
		cfg.Redis.Host = v
	}
	if v := viper.GetInt("REDIS_PORT"); v != 0 {
		cfg.Redis.Port = v
	}
	if v := viper.GetString("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := viper.GetString("MONGO_URI"); v != "" {
		cfg.MongoDB.URI = v
	}
	if v := viper.GetString("RABBITMQ_URL"); v != "" {
		cfg.RabbitMQ.URL = v
	}
	if v := viper.GetString("MINIO_ENDPOINT"); v != "" {
		cfg.MinIO.Endpoint = v
	}
	if v := viper.GetString("MINIO_ACCESS_KEY"); v != "" {
		cfg.MinIO.AccessKey = v
	}
	if v := viper.GetString("MINIO_SECRET_KEY"); v != "" {
		cfg.MinIO.SecretKey = v
	}
	if v := viper.GetString("DASHSCOPE_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := viper.GetString("LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := viper.GetString("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := viper.GetString("WECHAT_APP_ID"); v != "" {
		cfg.WeChat.AppID = v
	}
	if v := viper.GetString("WECHAT_APP_SECRET"); v != "" {
		cfg.WeChat.AppSecret = v
	}
	if v := viper.GetString("INTERNAL_API_KEY"); v != "" {
		cfg.InternalAPIKey = v
	}
	if v := viper.GetString("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.App.CORSAllowedOrigins = v
	}
}
