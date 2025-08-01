package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"user_system_v1/client"
	"user_system_v1/config"
	"user_system_v1/database"
	"user_system_v1/server"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 初始化数据库连接
	mysqlDB, err := database.NewMySQLDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}
	defer mysqlDB.Close()

	// 初始化Redis连接
	redisDB, err := database.NewRedisDB(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisDB.Close()

	// 创建数据库表
	if err := mysqlDB.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// 检查是否需要插入测试数据
	count, err := mysqlDB.GetUserCount()
	if err != nil {
		log.Fatalf("Failed to get user count: %v", err)
	}

	// 检查是否有快速模式环境变量
	quickMode := os.Getenv("QUICK_MODE")
	userCount := 10000000

	if quickMode == "true" {
		userCount = 1000 // 快速模式只插入1000条数据
		log.Println("Quick mode enabled, inserting only 1000 users for testing")
	}

	if count == 0 {
		log.Println("Inserting test users...")
		if err := mysqlDB.InsertTestUsers(userCount); err != nil {
			log.Fatalf("Failed to insert test users: %v", err)
		}
		log.Printf("Inserted %d test users", userCount)
	} else {
		log.Printf("Database already contains %d users", count)

		// 检查是否需要更新密码哈希
		log.Println("Checking password hashes...")
		if err := mysqlDB.UpdatePasswordHashes(); err != nil {
			log.Printf("Warning: Failed to update password hashes: %v", err)
		} else {
			log.Println("Password hashes updated successfully")
		}
	}

	// 启动TCP服务器（直接传递数据库连接）
	tcpServer := server.NewTCPServer(mysqlDB, redisDB)

	// 使用WaitGroup等待服务器启动
	var wg sync.WaitGroup

	// 启动TCP服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := tcpServer.Start(cfg.TCPServerPort); err != nil {
			log.Printf("TCP Server error: %v", err)
		}
	}()

	// 等待TCP服务器启动
	log.Println("Waiting for TCP server to start...")
	time.Sleep(3 * time.Second)

	// 创建RPC客户端
	log.Println("Creating RPC client...")
	// 创建RPC客户端连接到TCP服务器，RPC客户端是HTTP与TCP服务器通信的桥梁
	rpcClient, err := client.NewRPCClient("localhost:" + cfg.TCPServerPort)
	if err != nil {
		log.Fatalf("Failed to create RPC client: %v", err)
	}
	defer rpcClient.Close()

	// 启动HTTP服务器
	httpServer := server.NewHTTPServer(rpcClient)

	// 启动HTTP服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := httpServer.Start(cfg.HTTPServerPort); err != nil {
			log.Printf("HTTP Server error: %v", err)
		}
	}()

	log.Printf("System started successfully!")
	log.Printf("HTTP Server: http://localhost:%s", cfg.HTTPServerPort)
	log.Printf("TCP Server: localhost:%s", cfg.TCPServerPort)

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")

	// 优雅关闭
	_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭TCP服务器
	if err := tcpServer.Stop(); err != nil {
		log.Printf("Error stopping TCP server: %v", err)
	}

	log.Println("Servers stopped successfully")
}
