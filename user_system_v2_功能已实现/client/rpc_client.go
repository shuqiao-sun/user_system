package client

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"user_system_v1/models"
	"user_system_v1/rpc"
)

type RPCClient struct {
	serverAddr string // 此处是TCP服务器的地址 即 localhost:9090
	mutex      sync.Mutex
	msgID      uint32
	msgIDMutex sync.Mutex
}

func NewRPCClient(serverAddr string) (*RPCClient, error) {
	return &RPCClient{
		serverAddr: serverAddr,
	}, nil
}

func (c *RPCClient) Close() error {
	return nil
}

// 发送RPC请求并等待响应
func (c *RPCClient) sendRequest(msgType uint32, payload interface{}) (*rpc.Response, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 每次请求建立新TCP连接
	conn, err := net.Dial("tcp", c.serverAddr) // 此处就是和localhost:9090建立连接
	if err != nil {
		return nil, fmt.Errorf("failed to connect to server: %v", err)
	}
	defer conn.Close()

	// 设置连接超时
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// 生成消息ID
	c.msgIDMutex.Lock()
	c.msgID++
	msgID := c.msgID
	c.msgIDMutex.Unlock()

	// 序列化payload
	payloadData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// 创建消息
	msg := &rpc.Message{
		Type:    msgType,
		ID:      msgID,
		Payload: payloadData,
	}

	// 序列化消息
	msgData, err := msg.Serialize()
	if err != nil {
		return nil, err
	}

	// 发送消息
	_, err = conn.Write(msgData)
	if err != nil {
		return nil, fmt.Errorf("failed to write message: %v", err)
	}

	// 读取响应
	response, err := c.readResponseFromConn(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	return response, nil
}

// 从指定连接读取响应
func (c *RPCClient) readResponseFromConn(conn net.Conn) (*rpc.Response, error) {
	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	// 读取长度前缀
	lengthBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, lengthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read length prefix: %v", err)
	}

	length := binary.BigEndian.Uint32(lengthBuf)

	// 读取消息体
	messageBuf := make([]byte, length)
	_, err = io.ReadFull(conn, messageBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read message body: %v", err)
	}

	// 解析响应
	response, err := rpc.DeserializeResponse(messageBuf)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// 登录
func (c *RPCClient) Login(username, password string) (*models.LoginResponse, error) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}

	response, err := c.sendRequest(rpc.MSG_LOGIN, payload)
	if err != nil {
		return nil, err
	}

	if response.Status != rpc.STATUS_SUCCESS {
		return &models.LoginResponse{
			Success: false,
			Message: response.Message,
		}, nil
	}

	var loginResp models.LoginResponse
	if err := json.Unmarshal(response.Payload, &loginResp); err != nil {
		return nil, err
	}

	return &loginResp, nil
}

// 获取用户信息
func (c *RPCClient) GetProfile(token string) (*models.GetProfileResponse, error) {
	payload := map[string]string{
		"token": token,
	}

	response, err := c.sendRequest(rpc.MSG_GET_PROFILE, payload)
	if err != nil {
		return nil, err
	}

	if response.Status != rpc.STATUS_SUCCESS {
		return &models.GetProfileResponse{
			Success: false,
			Message: response.Message,
		}, nil
	}

	var profileResp models.GetProfileResponse
	if err := json.Unmarshal(response.Payload, &profileResp); err != nil {
		return nil, err
	}

	return &profileResp, nil
}

// 更新用户信息
func (c *RPCClient) UpdateProfile(token, nickname, profilePic string) (*models.UpdateProfileResponse, error) {
	payload := map[string]string{
		"token":       token,
		"nickname":    nickname,
		"profile_pic": profilePic,
	}

	response, err := c.sendRequest(rpc.MSG_UPDATE_PROFILE, payload)
	if err != nil {
		return nil, err
	}

	if response.Status != rpc.STATUS_SUCCESS {
		return &models.UpdateProfileResponse{
			Success: false,
			Message: response.Message,
		}, nil
	}

	var updateResp models.UpdateProfileResponse
	if err := json.Unmarshal(response.Payload, &updateResp); err != nil {
		return nil, err
	}

	return &updateResp, nil
}

// 登出
func (c *RPCClient) Logout(token string) error {
	payload := map[string]string{
		"token": token,
	}

	response, err := c.sendRequest(rpc.MSG_LOGOUT, payload)
	if err != nil {
		return err
	}

	if response.Status != rpc.STATUS_SUCCESS {
		return fmt.Errorf("logout failed: %s", response.Message)
	}

	return nil
}

// 心跳
func (c *RPCClient) Heartbeat() error {
	payload := map[string]string{}

	response, err := c.sendRequest(rpc.MSG_HEARTBEAT, payload)
	if err != nil {
		return err
	}

	if response.Status != rpc.STATUS_SUCCESS {
		return fmt.Errorf("heartbeat failed: %s", response.Message)
	}

	return nil
}
