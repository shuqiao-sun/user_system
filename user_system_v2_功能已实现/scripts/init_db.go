package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"
	
	"user_system_v1/config"
	"user_system_v1/database"
)

func main() {
	fmt.Println("数据库初始化脚本")
	fmt.Println("================")
	
	// 加载配置
	cfg := config.LoadConfig()
	
	// 连接数据库
	mysqlDB, err := database.NewMySQLDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer mysqlDB.Close()
	
	// 连接Redis
	redisDB, err := database.NewRedisDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisDB.Close()
	
	// 创建表
	fmt.Println("创建数据库表...")
	if err := mysqlDB.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
	fmt.Println("✓ 数据库表创建成功")
	
	// 检查现有数据
	count, err := mysqlDB.GetUserCount()
	if err != nil {
		log.Fatalf("Failed to get user count: %v", err)
	}
	
	if count > 0 {
		fmt.Printf("数据库已包含 %d 条用户记录\n", count)
		fmt.Print("是否要重新插入测试数据？(y/N): ")
		
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("跳过数据插入")
			return
		}
		
		// 清空现有数据
		fmt.Println("清空现有数据...")
		if err := clearExistingData(mysqlDB); err != nil {
			log.Fatalf("Failed to clear existing data: %v", err)
		}
	}
	
	// 插入测试数据
	fmt.Println("开始插入测试数据...")
	start := time.Now()
	
	if err := mysqlDB.InsertTestUsers(10000000); err != nil {
		log.Fatalf("Failed to insert test users: %v", err)
	}
	
	duration := time.Since(start)
	fmt.Printf("✓ 成功插入 10,000,000 条测试数据，耗时: %v\n", duration)
	
	// 验证数据
	fmt.Println("验证数据...")
	finalCount, err := mysqlDB.GetUserCount()
	if err != nil {
		log.Fatalf("Failed to get final user count: %v", err)
	}
	
	fmt.Printf("✓ 数据库中共有 %d 条用户记录\n", finalCount)
	
	// 测试几个用户
	fmt.Println("测试用户数据:")
	for i := 1; i <= 5; i++ {
		user, err := mysqlDB.GetUserByUsername(fmt.Sprintf("user_%d", i))
		if err != nil {
			log.Printf("Failed to get user_%d: %v", i, err)
			continue
		}
		fmt.Printf("  用户%d: %s (%s)\n", i, user.Username, user.Nickname)
	}
	
	// 测试随机用户
	randomUser, err := mysqlDB.GetRandomUser()
	if err != nil {
		log.Printf("Failed to get random user: %v", err)
	} else {
		fmt.Printf("  随机用户: %s (%s)\n", randomUser.Username, randomUser.Nickname)
	}
	
	fmt.Println("\n数据库初始化完成！")
}

func clearExistingData(mysqlDB *database.MySQLDB) error {
	// 这里应该实现清空数据的逻辑
	// 为了安全起见，我们只是打印警告
	fmt.Println("警告: 清空数据功能需要手动实现")
	fmt.Println("请手动执行: DELETE FROM users;")
	return nil
}

// 生成密码哈希（用于测试）
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
} 