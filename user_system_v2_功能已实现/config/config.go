package config

import (
	"fmt"
	"os"
)

type Config struct {
	MySQLHost     string
	MySQLPort     string
	MySQLUser     string
	MySQLPassword string
	MySQLDatabase string
	
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	
	HTTPServerPort string
	TCPServerPort  string
	
	SessionExpiration int // 秒
}

func LoadConfig() *Config {
	// 添加调试信息
	mysqlUser := getEnv("MYSQL_USER", "user_system")  // 改为user_system
	mysqlPassword := getEnv("MYSQL_PASSWORD", "password123")  // 改为password123
	
	fmt.Printf("DEBUG: MYSQL_USER from env: %s\n", mysqlUser)
	fmt.Printf("DEBUG: MYSQL_PASSWORD from env: %s\n", mysqlPassword)
	
	return &Config{
		MySQLHost:     getEnv("MYSQL_HOST", "localhost"),
		MySQLPort:     getEnv("MYSQL_PORT", "3306"),
		MySQLUser:     mysqlUser,
		MySQLPassword: mysqlPassword,
		MySQLDatabase: getEnv("MYSQL_DATABASE", "user_system"),
		
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		
		HTTPServerPort: getEnv("HTTP_PORT", "8080"),
		TCPServerPort:  getEnv("TCP_PORT", "9090"),
		
		SessionExpiration: 3600, // 1小时
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
} 