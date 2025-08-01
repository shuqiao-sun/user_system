package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
	
	_ "github.com/go-sql-driver/mysql"
	"user_system_v1/config"
	"user_system_v1/models"
)

type MySQLDB struct {
	db *sql.DB
}

func NewMySQLDB(cfg *config.Config) (*MySQLDB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.MySQLUser,
		cfg.MySQLPassword,
		cfg.MySQLHost,
		cfg.MySQLPort,
		cfg.MySQLDatabase,
	)
	
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)
	
	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, err
	}
	
	return &MySQLDB{db: db}, nil
}

func (m *MySQLDB) Close() error {
	return m.db.Close()
}

// 创建用户表
func (m *MySQLDB) CreateTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		nickname VARCHAR(100) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
		profile_pic TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		INDEX idx_username (username),
		INDEX idx_id (id)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	
	_, err := m.db.Exec(query)
	return err
}

// 根据用户名获取用户
func (m *MySQLDB) GetUserByUsername(username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, nickname, profile_pic, created_at, updated_at 
			  FROM users WHERE username = ?`
	
	var user models.User
	err := m.db.QueryRow(query, username).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Nickname,
		&user.ProfilePic,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// 根据ID获取用户
func (m *MySQLDB) GetUserByID(id int64) (*models.User, error) {
	query := `SELECT id, username, password_hash, nickname, profile_pic, created_at, updated_at 
			  FROM users WHERE id = ?`
	
	var user models.User
	err := m.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Nickname,
		&user.ProfilePic,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// 更新用户信息
func (m *MySQLDB) UpdateUser(id int64, nickname, profilePic string) error {
	query := `UPDATE users SET nickname = ?, profile_pic = ?, updated_at = CURRENT_TIMESTAMP 
			  WHERE id = ?`
	
	_, err := m.db.Exec(query, nickname, profilePic, id)
	return err
}

// 批量插入测试用户数据
func (m *MySQLDB) InsertTestUsers(count int) error {
	// 优化MySQL配置
	_, err := m.db.Exec("SET autocommit = 0")
	if err != nil {
		return err
	}
	
	_, err = m.db.Exec("SET unique_checks = 0")
	if err != nil {
		return err
	}
	
	_, err = m.db.Exec("SET foreign_key_checks = 0")
	if err != nil {
		return err
	}
	
	// 使用事务批量插入
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	
	// 准备插入语句
	stmt, err := tx.Prepare(`
		INSERT INTO users (username, password_hash, nickname, profile_pic) 
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	// 批量插入，增加批次大小
	batchSize := 10000 // 增加到10000条一批
	
	// 生成正确的密码哈希
	password := "password"
	passwordHash := ""
	hash := sha256.Sum256([]byte(password))
	passwordHash = hex.EncodeToString(hash[:])
	
	for i := 1; i <= count; i++ {
		username := fmt.Sprintf("user_%d", i)
		nickname := fmt.Sprintf("用户%d", i)
		profilePic := fmt.Sprintf("https://example.com/avatar/%d.jpg", i)
		
		_, err := stmt.Exec(username, passwordHash, nickname, profilePic)
		if err != nil {
			return err
		}
		
		// 每10000条提交一次
		if i%batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return err
			}
			tx, err = m.db.Begin()
			if err != nil {
				return err
			}
			stmt, err = tx.Prepare(`
				INSERT INTO users (username, password_hash, nickname, profile_pic) 
				VALUES (?, ?, ?, ?)
			`)
			if err != nil {
				return err
			}
			
			// 打印进度
			fmt.Printf("Inserted %d users...\n", i)
		}
	}
	
	// 提交剩余的事务
	if err := tx.Commit(); err != nil {
		return err
	}
	
	// 恢复MySQL配置
	_, err = m.db.Exec("SET autocommit = 1")
	if err != nil {
		return err
	}
	
	_, err = m.db.Exec("SET unique_checks = 1")
	if err != nil {
		return err
	}
	
	_, err = m.db.Exec("SET foreign_key_checks = 1")
	if err != nil {
		return err
	}
	
	return nil
}

// 获取用户总数
func (m *MySQLDB) GetUserCount() (int, error) {
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// 随机获取用户（用于性能测试）
func (m *MySQLDB) GetRandomUser() (*models.User, error) {
	query := `SELECT id, username, password_hash, nickname, profile_pic, created_at, updated_at 
			  FROM users ORDER BY RAND() LIMIT 1`
	
	var user models.User
	err := m.db.QueryRow(query).Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Nickname,
		&user.ProfilePic,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &user, nil
}

// 更新密码哈希
func (m *MySQLDB) UpdatePasswordHashes() error {
	// 生成正确的密码哈希
	password := "password"
	hash := sha256.Sum256([]byte(password))
	passwordHash := hex.EncodeToString(hash[:])
	
	// 更新所有用户的密码哈希
	query := `UPDATE users SET password_hash = ? WHERE password_hash LIKE 'hash_%'`
	
	result, err := m.db.Exec(query, passwordHash)
	if err != nil {
		return err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	
	if rowsAffected > 0 {
		fmt.Printf("Updated %d user password hashes\n", rowsAffected)
	}
	
	return nil
} 