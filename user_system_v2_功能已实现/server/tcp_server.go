package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"user_system_v1/database"
	"user_system_v1/models"
	"user_system_v1/rpc"
)

type TCPServer struct {
	mysqlDB    *database.MySQLDB
	redisDB    *database.RedisDB
	listener   net.Listener
	clients    map[net.Conn]bool
	mutex      sync.RWMutex
	msgID      uint32
	msgIDMutex sync.Mutex
}

func NewTCPServer(mysqlDB *database.MySQLDB, redisDB *database.RedisDB) *TCPServer {
	return &TCPServer{
		mysqlDB: mysqlDB,
		redisDB: redisDB,
		clients: make(map[net.Conn]bool),
	}
}

func (s *TCPServer) Start(port string) error {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	s.listener = listener
	log.Printf("TCP Server started on port %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// 添加客户端连接
		s.mutex.Lock()
		s.clients[conn] = true
		s.mutex.Unlock()

		go s.handleConnection(conn)
	}
}

func (s *TCPServer) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 关闭所有客户端连接
	for conn := range s.clients {
		conn.Close()
		delete(s.clients, conn)
	}

	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		s.mutex.Lock()
		delete(s.clients, conn)
		s.mutex.Unlock()
	}()

	// 设置连接超时
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	buffer := make([]byte, 4096)

	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from connection: %v", err)
			}
			break
		}

		// 处理消息
		response, err := s.handleMessage(buffer[:n])
		if err != nil {
			log.Printf("Error handling message: %v", err)
			continue
		}

		// 发送响应
		if response != nil {
			responseData, err := response.Serialize()
			if err != nil {
				log.Printf("Error serializing response: %v", err)
				continue
			}

			_, err = conn.Write(responseData)
			if err != nil {
				log.Printf("Error writing response: %v", err)
				break
			}

			// 发送响应后关闭连接（每个请求一个连接）
			break
		}

		// 重置超时
		conn.SetDeadline(time.Now().Add(30 * time.Second))
	}
}

func (s *TCPServer) handleMessage(data []byte) (*rpc.Response, error) {
	// 解析消息
	msg, err := rpc.DeserializeMessage(data)
	if err != nil {
		return nil, err
	}

	// 生成响应ID
	s.msgIDMutex.Lock()
	s.msgID++
	responseID := s.msgID
	s.msgIDMutex.Unlock()

	// 根据消息类型处理
	switch msg.Type {
	case rpc.MSG_LOGIN:
		return s.handleLogin(msg, responseID)
	case rpc.MSG_GET_PROFILE:
		return s.handleGetProfile(msg, responseID)
	case rpc.MSG_UPDATE_PROFILE:
		return s.handleUpdateProfile(msg, responseID)
	case rpc.MSG_LOGOUT:
		return s.handleLogout(msg, responseID)
	case rpc.MSG_HEARTBEAT:
		return s.handleHeartbeat(msg, responseID)
	default:
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Unknown message type",
		}, nil
	}
}

func (s *TCPServer) handleLogin(msg *rpc.Message, responseID uint32) (*rpc.Response, error) {
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.Unmarshal(msg.Payload, &loginReq); err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Invalid request format",
		}, nil
	}

	// 获取用户信息
	user, err := s.mysqlDB.GetUserByUsername(loginReq.Username)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "用户名或密码错误",
		}, nil
	}

	// 验证密码
	if !s.verifyPassword(loginReq.Password, user.PasswordHash) {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "用户名或密码错误",
		}, nil
	}

	// 生成Session Token
	token, err := s.redisDB.GenerateSessionToken(user.ID)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "登录失败，请重试",
		}, err
	}

	// 存储Session
	expiration := time.Duration(3600) * time.Second // 1小时
	err = s.redisDB.StoreSession(token, user.ID, expiration)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "登录失败，请重试",
		}, err
	}

	loginResp := &models.LoginResponse{
		Success: true,
		Token:   token,
		Message: "登录成功",
		User:    user,
	}

	// 序列化响应数据
	payload, err := json.Marshal(loginResp)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Response serialization failed",
		}, err
	}

	return &rpc.Response{
		Type:    msg.Type,
		ID:      responseID,
		Status:  rpc.STATUS_SUCCESS,
		Message: loginResp.Message,
		Payload: payload,
	}, nil
}

func (s *TCPServer) handleGetProfile(msg *rpc.Message, responseID uint32) (*rpc.Response, error) {
	var profileReq struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(msg.Payload, &profileReq); err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Invalid request format",
		}, nil
	}

	// 验证Token
	userID, err := s.validateToken(profileReq.Token)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Invalid session",
		}, nil
	}

	// 获取用户信息
	user, err := s.mysqlDB.GetUserByID(userID)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "获取用户信息失败",
		}, err
	}

	profileResp := &models.GetProfileResponse{
		Success: true,
		Message: "获取成功",
		User:    user,
	}

	// 序列化响应数据
	payload, err := json.Marshal(profileResp)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Response serialization failed",
		}, err
	}

	return &rpc.Response{
		Type:    msg.Type,
		ID:      responseID,
		Status:  rpc.STATUS_SUCCESS,
		Message: profileResp.Message,
		Payload: payload,
	}, nil
}

func (s *TCPServer) handleUpdateProfile(msg *rpc.Message, responseID uint32) (*rpc.Response, error) {
	var updateReq struct {
		Token      string `json:"token"`
		Nickname   string `json:"nickname"`
		ProfilePic string `json:"profile_pic"`
	}

	if err := json.Unmarshal(msg.Payload, &updateReq); err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Invalid request format",
		}, nil
	}

	// 验证Token
	userID, err := s.validateToken(updateReq.Token)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Invalid session",
		}, nil
	}

	// 更新用户信息
	err = s.mysqlDB.UpdateUser(userID, updateReq.Nickname, updateReq.ProfilePic)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "更新失败",
		}, err
	}

	// 获取更新后的用户信息
	user, err := s.mysqlDB.GetUserByID(userID)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "获取更新后的信息失败",
		}, err
	}

	updateResp := &models.UpdateProfileResponse{
		Success: true,
		Message: "更新成功",
		User:    user,
	}

	// 序列化响应数据
	payload, err := json.Marshal(updateResp)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Response serialization failed",
		}, err
	}

	return &rpc.Response{
		Type:    msg.Type,
		ID:      responseID,
		Status:  rpc.STATUS_SUCCESS,
		Message: updateResp.Message,
		Payload: payload,
	}, nil
}

func (s *TCPServer) handleLogout(msg *rpc.Message, responseID uint32) (*rpc.Response, error) {
	var logoutReq struct {
		Token string `json:"token"`
	}

	if err := json.Unmarshal(msg.Payload, &logoutReq); err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Invalid request format",
		}, nil
	}

	// 删除Session
	err := s.redisDB.DeleteSession(logoutReq.Token)
	if err != nil {
		return &rpc.Response{
			Type:    msg.Type,
			ID:      responseID,
			Status:  rpc.STATUS_ERROR,
			Message: "Logout failed",
		}, err
	}

	return &rpc.Response{
		Type:    msg.Type,
		ID:      responseID,
		Status:  rpc.STATUS_SUCCESS,
		Message: "Logout successful",
	}, nil
}

func (s *TCPServer) handleHeartbeat(msg *rpc.Message, responseID uint32) (*rpc.Response, error) {
	return &rpc.Response{
		Type:    msg.Type,
		ID:      responseID,
		Status:  rpc.STATUS_SUCCESS,
		Message: "Heartbeat received",
	}, nil
}

// 验证Token
func (s *TCPServer) validateToken(token string) (int64, error) {
	// 检查Session是否存在
	exists, err := s.redisDB.SessionExists(token)
	if err != nil {
		return 0, err
	}

	if !exists {
		return 0, fmt.Errorf("invalid session")
	}

	// 获取用户ID
	userID, err := s.redisDB.GetSession(token)
	if err != nil {
		return 0, err
	}

	// 刷新Session过期时间
	expiration := time.Duration(3600) * time.Second
	err = s.redisDB.RefreshSession(token, expiration)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

// 验证密码（简化版本）
func (s *TCPServer) verifyPassword(password, hash string) bool {
	// 实际应用中应该使用bcrypt
	hashedPassword := s.hashPassword(password)

	// 添加调试信息
	fmt.Printf("DEBUG: Comparing password '%s' with hash '%s' vs stored hash '%s'\n",
		password, hashedPassword, hash)

	return hashedPassword == hash
}

// 哈希密码（简化版本）
func (s *TCPServer) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
