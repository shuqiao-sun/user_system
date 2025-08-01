package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	
	"github.com/go-redis/redis/v8"
	"user_system_v1/config"
)

type RedisDB struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisDB(cfg *config.Config) (*RedisDB, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
		PoolSize: 100,
	})
	
	ctx := context.Background()
	
	// 测试连接
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	
	return &RedisDB{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisDB) Close() error {
	return r.client.Close()
}

// 生成Session Token
func (r *RedisDB) GenerateSessionToken(userID int64) (string, error) {
	// 使用UUID生成token
	token := fmt.Sprintf("session_%d_%d", userID, time.Now().UnixNano())
	return token, nil
}

// 存储Session
func (r *RedisDB) StoreSession(token string, userID int64, expiration time.Duration) error {
	sessionData := map[string]interface{}{
		"user_id": userID,
		"created": time.Now().Unix(),
	}
	
	data, err := json.Marshal(sessionData)
	if err != nil {
		return err
	}
	
	key := fmt.Sprintf("session:%s", token)
	return r.client.Set(r.ctx, key, data, expiration).Err()
}

// 获取Session
func (r *RedisDB) GetSession(token string) (int64, error) {
	key := fmt.Sprintf("session:%s", token)
	
	data, err := r.client.Get(r.ctx, key).Bytes()
	if err != nil {
		return 0, err
	}
	
	var sessionData map[string]interface{}
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return 0, err
	}
	
	userID, ok := sessionData["user_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid session data")
	}
	
	return int64(userID), nil
}

// 删除Session
func (r *RedisDB) DeleteSession(token string) error {
	key := fmt.Sprintf("session:%s", token)
	return r.client.Del(r.ctx, key).Err()
}

// 刷新Session过期时间
func (r *RedisDB) RefreshSession(token string, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", token)
	return r.client.Expire(r.ctx, key, expiration).Err()
}

// 检查Session是否存在
func (r *RedisDB) SessionExists(token string) (bool, error) {
	key := fmt.Sprintf("session:%s", token)
	result := r.client.Exists(r.ctx, key).Val()
	return result > 0, nil
} 