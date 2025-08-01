# 用户管理系统 (User Management System)

## 项目简介

这是一个基于Go语言开发的高性能用户管理系统，采用微服务架构设计，支持用户注册、登录、个人资料管理、头像上传等功能。系统采用HTTP Server + TCP Server的分离架构，实现了业务逻辑与数据访问的完全隔离，提供了良好的安全性和可扩展性。

## 核心特性

### 🏗️ 架构设计
- **微服务架构**：HTTP Server与TCP Server分离
- **安全隔离**：HTTP Server不直接连接数据库
- **高性能**：支持高并发访问，QPS > 3000
- **可扩展**：支持水平扩展和负载均衡

### 🔐 安全特性
- **Token认证**：基于Redis的Session管理
- **密码加密**：bcrypt哈希存储
- **SQL注入防护**：参数化查询
- **XSS防护**：输入验证和清理

### 📁 文件管理
- **头像上传**：支持JPG、PNG、GIF、WebP格式
- **文件验证**：大小限制2MB，格式验证
- **静态服务**：自动生成唯一文件名

### 🎯 用户体验
- **统一更新**：一个按钮完成昵称和头像更新
- **智能保持**：只更新有变化的字段
- **实时预览**：文件选择时即时预览
- **友好反馈**：统一的成功/失败提示

## 运行环境要求

### 系统要求
- **操作系统**：Linux/macOS/Windows
- **Go版本**：1.19+
- **MySQL**：8.0+
- **Redis**：6.0+
- **内存**：最少2GB
- **磁盘**：最少1GB可用空间

### 依赖服务
```bash
# MySQL配置
- 端口：3306
- 数据库：user_system
- 字符集：utf8mb4

# Redis配置
- 端口：6379
- 数据库：0
- 密码：可选
```

## 安装和运行

### 1. 克隆项目
```bash
git clone <repository-url>
cd user_system_v1_副本
```

### 2. 安装依赖
```bash
# 安装Go依赖
go mod download

# 安装Docker（可选）
# 用于容器化部署
```

### 3. 配置环境
```bash
# 复制配置文件
cp config/config.example.yaml config/config.yaml

# 编辑配置文件
vim config/config.yaml
```

### 4. 启动服务

#### 方式一：直接运行
```bash
# 快速模式（使用内存数据库）
QUICK_MODE=true ./bin/user_system

# 生产模式
./bin/user_system
```

#### 方式二：使用Makefile
```bash
# 开发模式
make dev

# 构建应用
make build

# 运行应用
make run
```

#### 方式三：Docker部署
```bash
# 启动所有服务
docker-compose up -d

# 停止服务
docker-compose down
```

### 5. 验证安装
```bash
# 访问Web界面
http://localhost:8080

# 测试API
curl http://localhost:8080/api/profile
```

## API接口

### 认证接口

#### 用户登录
```http
POST /api/login
Content-Type: application/json

{
    "username": "user_1",
    "password": "user_1"
}
```

#### 用户登出
```http
POST /api/logout
Authorization: Bearer <token>
```

### 用户信息接口

#### 获取个人资料
```http
GET /api/profile
Authorization: Bearer <token>
```

#### 更新信息
```http
POST /api/update-info
Authorization: Bearer <token>
Content-Type: multipart/form-data

Form Data:
- nickname: "新昵称" (可选)
- avatar: [图片文件] (可选)
```

### 响应格式
```json
{
    "success": true,
    "message": "操作成功",
    "user": {
        "id": 1,
        "username": "user_1",
        "nickname": "用户昵称",
        "profile_pic": "/uploads/avatar_1234567890.png",
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    }
}
```

## 技术实现

### 架构设计

#### 系统架构
```
┌─────────────────┐    RPC     ┌─────────────────┐
│   HTTP Server   │ ────────── │   TCP Server    │
│                 │            │                 │
│ - API处理       │            │ - 业务逻辑      │
│ - 文件上传      │            │ - 数据验证      │
│ - 静态服务      │            │ - 数据库操作    │
│ - 无数据库连接  │            │ - 直接连接数据库│
└─────────────────┘            └─────────────────┘
                                        │
                                        ▼
                        ┌─────────────────┐
                        │   MySQL + Redis │
                        │                 │
                        │ - 用户数据      │
                        │ - Session管理   │
                        └─────────────────┘
```

#### 数据流
1. **HTTP请求** → HTTP Server
2. **API处理** → 参数验证、文件处理
3. **RPC调用** → TCP Server
4. **业务逻辑** → 数据验证、业务处理
5. **数据库操作** → MySQL/Redis
6. **响应返回** → 用户

### 核心组件

#### 1. HTTP Server (`server/http_server.go`)
- **路由管理**：使用Gorilla Mux
- **文件上传**：multipart/form-data处理
- **静态服务**：头像文件服务
- **中间件**：认证、日志、错误处理

#### 2. TCP Server (`server/tcp_server.go`)
- **RPC协议**：自定义协议实现
- **业务逻辑**：用户认证、资料管理
- **数据库操作**：MySQL和Redis连接
- **并发处理**：Goroutine池管理

#### 3. 数据库层
- **MySQL**：用户数据存储
- **Redis**：Session管理和缓存
- **连接池**：高效的连接管理

#### 4. 客户端 (`client/rpc_client.go`)
- **RPC客户端**：与TCP Server通信
- **连接管理**：连接池和重试机制
- **协议处理**：请求/响应序列化

### 关键技术

#### 1. 并发处理
```go
// Goroutine池管理
type TCPServer struct {
    workerPool chan struct{}
    // ...
}

// 并发处理请求
func (s *TCPServer) handleConnection(conn net.Conn) {
    select {
    case s.workerPool <- struct{}{}:
        defer func() { <-s.workerPool }()
        s.processRequest(conn)
    default:
        // 拒绝连接
    }
}
```

#### 2. 文件上传
```go
// 文件验证和处理
func (s *HTTPServer) handleUpdateInfoWithFile(w http.ResponseWriter, r *http.Request, token string) {
    // 文件大小限制
    r.ParseMultipartForm(5 << 20)
    
    // 文件类型验证
    allowedTypes := map[string]bool{".jpg": true, ".png": true, ".gif": true, ".webp": true}
    
    // 生成唯一文件名
    timestamp := time.Now().Unix()
    filename := fmt.Sprintf("avatar_%d%s", timestamp, ext)
}
```

#### 3. 智能数据合并
```go
// 保持现有信息，只更新变化字段
func (s *HTTPServer) handleUpdateInfoWithFile(w http.ResponseWriter, r *http.Request, token string) {
    // 获取当前信息
    currentProfileResp, err := s.rpcClient.GetProfile(token)
    
    // 使用当前信息作为默认值
    currentNickname := currentProfileResp.User.Nickname
    currentProfilePic := currentProfileResp.User.ProfilePic
    
    // 智能合并
    if nickname != "" {
        currentNickname = nickname
    }
    if file != nil {
        currentProfilePic = newAvatarURL
    }
}
```

## 性能特性

### 并发性能
- **200并发固定用户**：QPS > 3000
- **200并发随机用户**：QPS > 1000
- **2000并发固定用户**：QPS > 1500
- **2000并发随机用户**：QPS > 800

### 内存使用
- **HTTP Server**：~50MB
- **TCP Server**：~100MB
- **总内存**：~150MB

### 响应时间
- **登录请求**：< 10ms
- **资料获取**：< 5ms
- **信息更新**：< 20ms
- **文件上传**：< 100ms

## 开发指南

### 项目结构
```
user_system_v1_副本/
├── auth/              # 认证服务（备选方案）
├── bin/               # 编译输出
├── client/            # RPC客户端
├── config/            # 配置文件
├── database/          # 数据库连接
├── models/            # 数据模型
├── rpc/               # RPC协议定义
├── scripts/           # 脚本文件
├── server/            # 服务器实现
├── test_*.go          # 测试文件
├── go.mod             # Go模块文件
├── go.sum             # 依赖校验
├── main.go            # 主程序入口
├── Makefile           # 构建脚本
├── docker-compose.yml # Docker配置
└── README.md          # 项目说明
```

### 开发命令
```bash
# 运行测试
go test ./...

# 性能测试
make benchmark

# 代码格式化
go fmt ./...

# 代码检查
go vet ./...

# 构建
go build -o bin/user_system .
```

### 测试数据
系统自动创建10,000,000个测试用户：
- 用户名：user_1, user_2, ..., user_10000000
- 密码：user_1（所有用户使用相同密码）
- 昵称：用户1, 用户2, ..., 用户10000000

## 部署指南

### 生产环境配置
```yaml
# config/config.yaml
server:
  http_port: 8080
  tcp_port: 9090
  worker_pool_size: 100

database:
  mysql:
    host: localhost
    port: 3306
    username: root
    password: password
    database: user_system
  redis:
    host: localhost
    port: 6379
    password: ""
    database: 0

upload:
  max_file_size: 2097152  # 2MB
  allowed_types: [".jpg", ".jpeg", ".png", ".gif", ".webp"]
  upload_dir: "uploads"
```

### 监控和日志
```bash
# 查看日志
tail -f logs/app.log

# 监控进程
ps aux | grep user_system

# 检查端口
netstat -tlnp | grep :8080
```

## 故障排除

### 常见问题

#### 1. 连接数据库失败
```bash
# 检查MySQL服务
sudo systemctl status mysql

# 检查连接配置
mysql -u root -p -h localhost
```

#### 2. Redis连接失败
```bash
# 检查Redis服务
sudo systemctl status redis

# 测试连接
redis-cli ping
```

#### 3. 文件上传失败
```bash
# 检查上传目录权限
ls -la uploads/

# 创建上传目录
mkdir -p uploads && chmod 755 uploads
```

### 性能优化
```bash
# 调整系统参数
echo 'net.core.somaxconn = 65535' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_max_syn_backlog = 65535' >> /etc/sysctl.conf
sysctl -p
```

## 贡献指南

### 开发流程
1. Fork项目
2. 创建功能分支
3. 提交代码
4. 创建Pull Request

### 代码规范
- 使用Go官方代码规范
- 添加必要的注释
- 编写单元测试
- 更新相关文档

## 许可证

本项目采用MIT许可证，详见LICENSE文件。

## 联系方式

如有问题或建议，请通过以下方式联系：
- 提交Issue
- 发送邮件
- 参与讨论

---

**注意**：本项目仅供学习和研究使用，请勿用于生产环境。 