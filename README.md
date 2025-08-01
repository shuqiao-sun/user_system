[ Entry tesk 孙书樵 .md](https://github.com/user-attachments/files/21542046/Entry.tesk.md)

# Entry task 孙书樵

## 1. 项目简介——用户管理系统 (User Management System)

这是一个基于Go语言开发的用户管理系统，采用微服务架构设计，支持用户登录、修改用户名、修改/上传头像图片。系统采用HTTP Server + TCP Server的分离架构，实现了业务逻辑与数据访问的完全隔离，提供了良好的安全性和可扩展性。

## 2. 架构设计
### 设计说明
1. 实现了HTTP Server和TCP Server的分离架构。HTTP Server和TCP Server之间的通信通过自定义的RPC协议实现

2. HTTP Server 仅仅处理API和用户输入，并不直接连接任何数据库。

3. 主要的功能逻辑放在TCP server实现。包括后端校验、连接数据库等业务逻辑。

4. 用户账号信息存储在MySQL数据库。通过MySQL Go client连接数据库，密码采用sha256存储。

5. Session 管理的TokenToken存储在redis，使用当前时间进行加密，保证session唯一。

具体**架构**如下图所示：

```mermaid
graph TB
    %% 用户层
    User[用户浏览器]
    
    %% HTTP Server层
    subgraph "HTTP Server"
        HTTP[HTTP Server]
        Upload[头像文件服务<br/>/uploads/]
        API[API路由<br/>/api/*]
    end
    
    %% RPC通信层
    subgraph "RPC通信"
        RPCClient[RPC Client]
    end
    
    %% TCP Server层
    subgraph "TCP Server"
        TCPServer[TCP Server]
        Business[业务逻辑]
    end
    
    %% 数据库层
    subgraph "数据存储层"
        Redis[(Redis<br/>Session管理)]
        MySQL[(MySQL<br/>用户数据)]
    end
    
    %% 文件存储
    subgraph "文件存储"
        UploadDir[uploads/<br/>头像文件]
    end
    
    %% 连接关系
    User -->|HTTP请求| HTTP
    User -->|头像文件| Upload
    
    HTTP -->|RPC调用| RPCClient
    RPCClient -->|TCP连接| TCPServer
    
    TCPServer -->|Session管理| Redis
    TCPServer -->|用户数据| MySQL
    Upload -->|文件访问| UploadDir
    
    %% 样式
    classDef userLayer fill:#e1f5fe
    classDef httpLayer fill:#f3e5f5
    classDef rpcLayer fill:#fff3e0
    classDef tcpLayer fill:#e8f5e8
    classDef dataLayer fill:#ffebee
    classDef fileLayer fill:#f1f8e9
    
    class User userLayer
    class HTTP,Static,Upload,API,Pages httpLayer
    class RPCClient rpcLayer
    class TCPServer,Auth,Business,Validation tcpLayer
    class Redis,MySQL dataLayer
    class UploadDir fileLayer
```
### 文件目录结构
相关代码及其作用如下：
user_system_/
|── bin/                 
|   └── user_system         # 编译输出的可执行文件，也就是启动系统
|─ client/                 
|   └── rpc_client.go       # RPC通信客户端
|── config/                
|   └── config.go           # 配置管理
|── database/               
|   └── mysql.go            # MySQL连接管理
|   └── redis.go            # Redis连接管理
|── models/                 
|   └── user.go             # 用户模型定义
|── rpc/                    
|   └── protocol.go         # 自定义RPC协议
|── scripts/                
|   └── init_db.go          # 数据库初始化
|── server/               
|   └── http_server.go      # HTTP服务器
|   └── tcp_server.go       # TCP服务器
|── go.mod                  # Go模块文件
|── go.sum                  # 依赖校验文件
|── main.go                 # 主程序入口
|── Makefile                # 构建脚本
|── docker-compose.yml      # Docker配置
└── README.md               # 项目说明

## 3. 前端界面
### 登陆界面
如下图所示：
![登陆界面](assets/登陆界面.png)
### 用户管理界面
如下图所示：
![信息界面](assets/信息界面.png)
### 文件上传的逻辑
```mermaid
sequenceDiagram
    participant User as 用户前端
    participant HTTP as HTTP Server
    participant RPC as RPC Client
    participant TCP as TCP Server
    participant DB as MySQL
    participant FS as 文件系统

    Note over User,FS: 文件上传完整流程

    %% 1. 用户选择文件
    User->>User: 选择头像文件
    User->>User: 输入昵称
    User->>User: 点击"更新信息"按钮

    %% 2. 前端验证
    User->>User: 检查文件大小(≤2MB)
    User->>User: 检查文件类型(JPG/PNG/GIF/WebP)
    User->>User: 创建FormData对象

    %% 3. 发送请求
    User->>HTTP: POST /api/update-info
    Note right of User: Content-Type: multipart/form-data
    Note right of User: Authorization: Bearer token
    Note right of User: Body: FormData(nickname, avatar)

    %% 4. HTTP Server处理
    HTTP->>HTTP: extractToken() - 提取Token
    HTTP->>HTTP: 检查Content-Type
    HTTP->>HTTP: 进入multipart/form-data分支
    HTTP->>HTTP: handleUpdateInfoWithFile()

    %% 5. 解析表单数据
    HTTP->>HTTP: ParseMultipartForm(5MB)
    HTTP->>HTTP: FormValue("nickname")
    HTTP->>HTTP: FormFile("avatar")

    %% 6. 获取当前用户信息
    HTTP->>RPC: GetProfile(token)
    RPC->>TCP: MSG_GET_PROFILE
    TCP->>DB: GetUserByID(userID)
    DB-->>TCP: User数据
    TCP-->>RPC: GetProfileResponse
    RPC-->>HTTP: GetProfileResponse

    %% 7. 文件处理
    HTTP->>HTTP: 验证文件类型
    HTTP->>HTTP: 验证文件大小
    HTTP->>HTTP: 生成唯一文件名
    HTTP->>FS: 保存文件到uploads/
    FS-->>HTTP: 文件保存成功
    HTTP->>HTTP: 生成访问URL

    %% 8. 更新用户信息
    HTTP->>RPC: UpdateProfile(token, nickname, profilePic)
    RPC->>TCP: MSG_UPDATE_PROFILE
    TCP->>DB: UpdateUser(userID, nickname, profilePic)
    DB-->>TCP: 更新成功
    TCP-->>RPC: UpdateProfileResponse
    RPC-->>HTTP: UpdateProfileResponse

    %% 9. 返回响应
    HTTP-->>User: JSON响应 {success, user}
    User->>User: 更新头像预览
    User->>User: 显示成功消息
```

## 4.RPC协议详解
### 协议格式

Rpc协议格式如下：

> \[4字节长度前缀][JSON消息体] 

其中json消息体中封装了RPC 等请求和响应的结构体。对应如下：

```go
// RPC请求结构
type Message struct {
    Type    uint32          `json:"type"`    // 消息类型
    ID      uint32          `json:"id"`      // 消息ID
    Payload json.RawMessage `json:"payload"` // 消息内容
}

// RPC响应结构
type Response struct {
    Type    uint32          `json:"type"`    // 响应类型
    ID      uint32          `json:"id"`      // 响应ID
    Status  uint32          `json:"status"`  // 状态码
    Message string          `json:"message"` // 状态消息
    Payload json.RawMessage `json:"payload"` // 响应数据
}
```



其中的payload来自于对http请求进行处理、封装、json后的数据，其结构在model/user.go中

```go
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	Message string `json:"message"`
	User    *User  `json:"user,omitempty"`
}

type UpdateProfileRequest struct {
	Nickname   string `json:"nickname"`
	ProfilePic string `json:"profile_pic"`
}

type UpdateProfileResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *User  `json:"user,omitempty"`
}

type GetProfileResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	User    *User  `json:"user,omitempty"`
}
```



### 通信过程
时序图如下所示：
```mermaid
sequenceDiagram
    participant Client as RPC客户端
    participant TCP as TCP Server
    participant DB as 数据库层

    Note over Client,DB: RPC通信完整流程

    %% 1. 建立连接
    Client->>TCP: net.Dial("tcp", "localhost:9090")
    TCP-->>Client: 连接建立成功

    %% 2. 准备请求数据
    Note over Client: 准备请求数据
    Client->>Client: 生成消息ID (msgID++)
    Client->>Client: 序列化payload (json.Marshal)
    Client->>Client: 创建Message结构体

    %% 3. 消息序列化
    Note over Client: 消息序列化过程
    Client->>Client: json.Marshal(Message)
    Client->>Client: 添加4字节长度前缀
    Client->>Client: binary.BigEndian.PutUint32(buf[:4], length)
    Client->>Client: copy(buf[4:], jsonData)

    %% 4. 发送请求
    Client->>TCP: conn.Write(serializedMessage)
    Note right of TCP: 接收TCP数据包

    %% 5. TCP Server处理
    Note over TCP: TCP Server处理流程
    TCP->>TCP: handleConnection()
    TCP->>TCP: 读取TCP数据到buffer
    TCP->>TCP: handleMessage(buffer[:n])

    %% 6. 消息反序列化
    Note over TCP: 消息反序列化过程
    TCP->>TCP: 读取4字节长度前缀
    TCP->>TCP: binary.BigEndian.Uint32(lengthBuf)
    TCP->>TCP: 读取指定长度的消息体
    TCP->>TCP: json.Unmarshal(messageBuf, &msg)

    %% 7. 消息类型分发
    Note over TCP: 消息分发处理
    TCP->>TCP: switch msg.Type
    TCP->>TCP: 生成响应ID (responseID++)
    TCP->>TCP: 调用对应的处理函数

    %% 8. 业务逻辑处理
    Note over TCP: 业务逻辑处理
    TCP->>TCP: 解析payload (json.Unmarshal)
    TCP->>TCP: 验证Token
    TCP->>DB: 数据库操作
    DB-->>TCP: 返回数据

    %% 9. 构建响应
    Note over TCP: 构建响应
    TCP->>TCP: 创建Response结构体
    TCP->>TCP: 序列化业务数据 (json.Marshal)
    TCP->>TCP: 设置状态和消息

    %% 10. 响应序列化
    Note over TCP: 响应序列化过程
    TCP->>TCP: json.Marshal(Response)
    TCP->>TCP: 添加4字节长度前缀
    TCP->>TCP: binary.BigEndian.PutUint32(buf[:4], length)

    %% 11. 发送响应
    TCP->>Client: conn.Write(serializedResponse)
    Note right of Client: 接收响应数据

    %% 12. 客户端处理响应
    Note over Client: 响应处理流程
    Client->>Client: readResponseFromConn()
    Client->>Client: 读取4字节长度前缀
    Client->>Client: 读取指定长度的响应体
    Client->>Client: json.Unmarshal(responseBuf, &response)

    %% 13. 关闭连接
    Client->>TCP: conn.Close()
    TCP-->>Client: 连接关闭

    Note over Client,DB: RPC通信完成
```

## 5. 时序图
时序图如下：
### 用户登录流程
```mermaid
sequenceDiagram
    participant U as 用户前端界面
    participant H as HTTP Server
    participant R as RPC客户端
    participant T as TCP Server
    participant D as MySQL
    participant S as Redis

    Note over U,S: 用户登录流程
    U->>H: POST /api/login {username, password}
    H->>R: LoginRequest<br/>(username, password)
    R->>T: MSG_LOGIN {username, password}
    T->>D: GetUserByUsername(username)
    D-->>T: User数据
    T->>T: verifyPassword(password, hash)
    T->>S: GenerateSessionToken(userID)
    T->>S: StoreSession(token, userID, expiration)
    T-->>R: LoginResponse {success, token, user}
    R-->>H: LoginResponse
    H-->>U: JSON响应 {success, token, user}
```

### 获取用户资料
```mermaid
sequenceDiagram
    participant U as 用户前端界面
    participant H as HTTP Server
    participant R as RPC客户端
    participant T as TCP Server
    participant D as MySQL
    participant S as Redis

    Note over U,S: 获取用户资料
    U->>H: GET /api/profile (Authorization: Bearer token)
    H->>R: GetProfile(token)
    R->>T: MSG_GET_PROFILE {token}
    T->>S: ValidateToken(token)
    S-->>T: userID
    T->>D: GetUserByID(userID)
    D-->>T: User数据
    T-->>R: GetProfileResponse {success, user}
    R-->>H: GetProfileResponse
    H-->>U: JSON响应 {success, user}
```
### 更新用户信息
```mermaid
sequenceDiagram
    participant U as 用户前端界面
    participant H as HTTP Server
    participant R as RPC客户端
    participant T as TCP Server
    participant D as MySQL
    participant S as Redis

    Note over U,S: 信息修改流程
    U->>H: POST /api/update-info (multipart/form-data)
    Note right of U: 包含昵称和头像文件
    H->>H: 保存头像文件到uploads/
    H->>R: GetProfile(token)
    R->>T: MSG_GET_PROFILE {token}
    T->>S: ValidateToken(token)
    S-->>T: userID
    T->>D: GetUserByID(userID)
    D-->>T: User数据
    T-->>R: GetProfileResponse {success, user}
    R-->>H: GetProfileResponse
    H->>R: UpdateProfile(token, nickname, profilePic)
    R->>T: MSG_UPDATE_PROFILE {token, nickname, profilePic}
    T->>S: ValidateToken(token)
    S-->>T: userID
    T->>D: UpdateUser(userID, nickname, profilePic)
    D-->>T: 更新成功
    T-->>R: UpdateProfileResponse {success, user}
    R-->>H: UpdateProfileResponse
    H-->>U: JSON响应 {success, user}
```
### 用户登出
```mermaid
sequenceDiagram
    participant U as 用户前端界面
    participant H as HTTP Server
    participant R as RPC客户端
    participant T as TCP Server
    participant D as MySQL
    participant S as Redis

    Note over U,S: 用户登出流程
    U->>H: POST /api/logout (Authorization: Bearer token)
    H->>R: Logout(token)
    R->>T: MSG_LOGOUT {token}
    T->>S: DeleteSession(token)
    S-->>T: 删除成功
    T-->>R: Response {success}
    R-->>H: Response
    H-->>U: JSON响应 {success, message}
```

## 6. API接口

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
curl请求：
```bash
 curl -X POST "http://localhost:8080/api/login" \
-H "Content-Type: application/json" \
-d '{"username": "user_1", "password": "user_1"}'
```

#### 用户登出
```http
POST /api/logout
Authorization: Bearer <token>
```
curl请求：
```bash
curl -X POST http://localhost:8080/api/logout \
  -H "Authorization: Bearer session_1_1754005543196454000" 
```

## 用户信息接口

### 获取个人资料
```http
GET /api/profile
Authorization: Bearer <token>
```
curl请求：
```bash
curl -X GET "http://localhost:8080/api/profile" \
-H "Authorization: Bearer session_1_1754007576785945000"
```

### 更新信息
```http
POST /api/update-info
Authorization: Bearer <token>
Content-Type: multipart/form-data

Form Data:
- nickname: "新昵称" (可选)
- avatar: [图片文件] (可选)
```
例如可以使用curl对如下命令构建请求：
```bash
curl -X POST "http://localhost:8080/api/update-info" \
-H "Authorization: Bearer <token>" \
-H "Content-Type: multipart/form-data" \
-F "nickname=新昵称" \
-F "avatar=@/Users/shuqiao.sun/Pictures/shopee.png"
```
## 响应格式
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
## 7.性能测试
使用压力测试工具wrk，分别测试200和2000并发的情况。
### 200并发
测试结果：
```
➜  wrk_test wrk -t18 -c200 -d30s -s login.lua http://localhost:8080/ 
Running 30s test @ http://localhost:8080/
  18 threads and 200 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   182.62ms   29.63ms 380.58ms   82.91%
    Req/Sec    64.66     28.67   118.00     64.25%
  32523 requests in 30.10s, 12.20MB read
Requests/sec:   1080.44
Transfer/sec:    415.12KB
```
### 2000并发
测试结果：
```
➜  wrk_test wrk -t18 -c2000 -d30s -s login.lua http://localhost:8080/
Running 30s test @ http://localhost:8080/
  18 threads and 2000 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.61s   247.92ms   1.92s    86.85%
    Req/Sec    82.91     75.59   490.00     77.96%
  35311 requests in 30.10s, 13.25MB read
Requests/sec:   1173.02
Transfer/sec:    450.77KB
```

## 8.总结与思考
### 学习心得
熟悉了简单的Web API后台架构：
- 使用Go实现HTTP API（JSON、文件）
- 基于TCP的RPC框架（设计和实现通信协议）
- 基于Token的鉴权机制和流程
- 使用Go对MySQL、Redis进行基本操作
- 了解了wrk性能测试工具的基本使用
### AI的使用(cursor)
可以用AI，但不要直接一键使用AI。
- 架构要自己设计，否则该架构特别痛苦。也可以问ai推荐的架构形式
- 问ai时要确定输入输出，可以让他帮忙写接口。
- 有bug和正常，描述清楚，慢慢持续追问。
-  cursor不会自动保存历史对话的所有代码！**无法回滚代码，一定要及时保存代码！**
-  ai写文档，改代码有时会过度操作，输入提示词的时候，一定要明确更改的位置，或者用ask模式。
### 安全性问题
- SQL注入防护：参数化查询
- token不可预测
- HTTP明文传输问题：密码和Token通过HTTP明文传输。未来应该用SSL/TLS加密
- 密码hash问题：采用固定的sha256哈希算法。未来应该用更安全的加密算法		（未来改进）
- 文件上传与目录遍历：类型校验、内容校验、防止路径遍历。	（未来改进）
