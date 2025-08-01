package rpc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const (
	// 消息类型
	MSG_LOGIN          = 1
	MSG_GET_PROFILE    = 2
	MSG_UPDATE_PROFILE = 3
	MSG_LOGOUT         = 4
	MSG_HEARTBEAT      = 5

	// 响应状态
	STATUS_SUCCESS = 0
	STATUS_ERROR   = 1
)

// RPC协议层的结构体
// RPC消息结构——封装所有HTTP server -> TCP server的请求
type Message struct {
	Type    uint32          `json:"type"`    // 确定消息类型，路由到对应的处理函数
	ID      uint32          `json:"id"`      // 通过ID字段标识每个请求，支持并发处理
	Payload json.RawMessage `json:"payload"` // Payload字段携带具体的业务数据
}

// RPC响应结构: TCP Server返回给HTTP Server的响应，准确的说应该是返回给rpc client的响应
type Response struct {
	Type    uint32          `json:"type"`
	ID      uint32          `json:"id"`
	Status  uint32          `json:"status"`  // 表示操作成功或失败
	Message string          `json:"message"` // 提供详细的错误描述
	Payload json.RawMessage `json:"payload"` // 返回业务数据
}

// 序列化消息
func (m *Message) Serialize() ([]byte, error) {
	// 协议的格式 [4字节长度前缀][JSON消息体]

	// 1. 将消息序列化为JSON
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	// 2. 添加4字节长度前缀（大端序）
	length := uint32(len(data))
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[:4], length)
	copy(buf[4:], data)

	return buf, nil
}

// 反序列化消息
func DeserializeMessage(data []byte) (*Message, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("invalid message length")
	}

	length := binary.BigEndian.Uint32(data[:4])
	if len(data) < int(4+length) {
		return nil, fmt.Errorf("incomplete message")
	}

	var msg Message
	err := json.Unmarshal(data[4:4+length], &msg)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// 序列化响应
func (r *Response) Serialize() ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}

	// 添加长度前缀
	length := uint32(len(data))
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[:4], length)
	copy(buf[4:], data)

	return buf, nil
}

// 反序列化响应
func DeserializeResponse(data []byte) (*Response, error) {
	var resp Response
	err := json.Unmarshal(data, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
