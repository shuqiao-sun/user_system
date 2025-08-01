package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"user_system_v1/client"
	"user_system_v1/models"
)

type HTTPServer struct {
	rpcClient *client.RPCClient
	router    *mux.Router
	uploadDir string
}

func NewHTTPServer(rpcClient *client.RPCClient) *HTTPServer {
	// 创建上传目录
	uploadDir := "uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Failed to create upload directory: %v", err)
	}

	server := &HTTPServer{
		rpcClient: rpcClient,
		router:    mux.NewRouter(),
		uploadDir: uploadDir,
	}

	server.setupRoutes()
	return server
}

func (s *HTTPServer) setupRoutes() {
	// 静态文件服务
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// 头像文件服务
	s.router.PathPrefix("/uploads/").Handler(http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// API路由
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	api.HandleFunc("/login", s.handleLogin).Methods("POST")
	api.HandleFunc("/profile", s.handleGetProfile).Methods("GET")
	api.HandleFunc("/profile", s.handleUpdateProfile).Methods("PUT")
	api.HandleFunc("/logout", s.handleLogout).Methods("POST")
	api.HandleFunc("/update-info", s.handleUpdateInfo).Methods("POST")

	// 页面路由
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
	s.router.HandleFunc("/login", s.handleLoginPage).Methods("GET")
	s.router.HandleFunc("/profile", s.handleProfilePage).Methods("GET")
}

func (s *HTTPServer) Start(port string) error {
	log.Printf("HTTP Server started on port %s", port)
	return http.ListenAndServe(":"+port, s.router)
}

// 健康检查端点
func (s *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

// 处理首页
func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>用户管理系统</title>
    <meta charset="utf-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 600px; margin: 0 auto; }
        .form-group { margin-bottom: 15px; }
        label { display: block; margin-bottom: 5px; }
        input[type="text"], input[type="password"] { width: 100%; padding: 8px; border: 1px solid #ddd; }
        button { padding: 10px 20px; background: #007bff; color: white; border: none; cursor: pointer; }
        .error { color: red; }
        .success { color: green; }
    </style>
</head>
<body>
    <div class="container">
        <h1>用户管理系统</h1>
        <div id="loginForm">
            <h2>登录</h2>
            <div class="form-group">
                <label>用户名:</label>
                <input type="text" id="username" placeholder="输入用户名">
            </div>
            <div class="form-group">
                <label>密码:</label>
                <input type="password" id="password" placeholder="输入密码">
            </div>
            <button onclick="login()">登录</button>
            <div id="loginMessage"></div>
        </div>
        
        <div id="profileForm" style="display:none;">
            <h2>用户信息</h2>
            <div class="form-group">
                <label>用户名:</label>
                <input type="text" id="displayUsername" readonly>
            </div>
            <div class="form-group">
                <label>昵称:</label>
                <input type="text" id="nickname" placeholder="输入昵称">
            </div>
            <div class="form-group">
                <label>头像:</label>
                <div id="avatarPreview" style="margin: 10px 0;">
                    <img id="avatarImage" src="" alt="头像" style="width: 100px; height: 100px; border-radius: 50%; object-fit: cover; border: 2px solid #ddd; display: none;">
                </div>
                <input type="file" id="avatarFile" accept="image/*" style="margin-bottom: 10px;">
                <div style="font-size: 12px; color: #666; margin-bottom: 10px;">
                    支持格式: JPG, PNG, GIF, WebP (最大2MB)
                </div>
            </div>
            <button onclick="updateInfo()">更新信息</button>
            <button onclick="logout()">登出</button>
            <div id="profileMessage"></div>
        </div>
    </div>
    
    <script>
        let currentToken = '';
        
        async function login() {
            const username = document.getElementById('username').value;
            const password = document.getElementById('password').value;
            
            const response = await fetch('/api/login', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({username, password})
            });
            
            const result = await response.json();
            
            if (result.success) {
                currentToken = result.token;
                document.getElementById('loginForm').style.display = 'none';
                document.getElementById('profileForm').style.display = 'block';
                document.getElementById('displayUsername').value = result.user.username;
                document.getElementById('nickname').value = result.user.nickname || '';
                // 显示现有头像
                if (result.user.profile_pic) {
                    document.getElementById('avatarImage').src = result.user.profile_pic;
                    document.getElementById('avatarImage').style.display = 'block';
                } else {
                    document.getElementById('avatarImage').style.display = 'none';
                }
                document.getElementById('loginMessage').innerHTML = '<span class="success">登录成功!</span>';
            } else {
                document.getElementById('loginMessage').innerHTML = '<span class="error">' + result.message + '</span>';
            }
        }
        
        async function updateInfo() {
            const nickname = document.getElementById('nickname').value;
            const fileInput = document.getElementById('avatarFile');
            const file = fileInput.files[0];
            
            // 如果有选择文件，检查文件
            if (file) {
                // 检查文件大小 (2MB)
                if (file.size > 2 * 1024 * 1024) {
                    document.getElementById('profileMessage').innerHTML = '<span class="error">文件大小不能超过2MB</span>';
                    return;
                }
                
                // 检查文件类型
                const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/gif', 'image/webp'];
                if (!allowedTypes.includes(file.type)) {
                    document.getElementById('profileMessage').innerHTML = '<span class="error">只支持JPG、PNG、GIF、WebP格式的图片</span>';
                    return;
                }
            }
            
            // 创建FormData，包含昵称和文件
            const formData = new FormData();
            formData.append('nickname', nickname);
            if (file) {
                formData.append('avatar', file);
            }
            
            try {
                const response = await fetch('/api/update-info', {
                    method: 'POST',
                    headers: {
                        'Authorization': 'Bearer ' + currentToken
                    },
                    body: formData
                });
                
                const result = await response.json();
                
                if (result.success) {
                    document.getElementById('profileMessage').innerHTML = '<span class="success">信息更新成功!</span>';
                    // 更新头像预览
                    if (result.user && result.user.profile_pic) {
                        document.getElementById('avatarImage').src = result.user.profile_pic;
                        document.getElementById('avatarImage').style.display = 'block';
                    }
                    // 清空文件选择
                    fileInput.value = '';
                } else {
                    document.getElementById('profileMessage').innerHTML = '<span class="error">' + result.message + '</span>';
                }
            } catch (error) {
                document.getElementById('profileMessage').innerHTML = '<span class="error">更新失败: ' + error.message + '</span>';
            }
        }
        
        // 文件选择时预览
        document.getElementById('avatarFile').addEventListener('change', function(e) {
            const file = e.target.files[0];
            if (file) {
                const reader = new FileReader();
                reader.onload = function(e) {
                    document.getElementById('avatarImage').src = e.target.result;
                    document.getElementById('avatarImage').style.display = 'block';
                };
                reader.readAsDataURL(file);
            }
        });
        
        async function logout() {
            await fetch('/api/logout', {
                method: 'POST',
                headers: {
                    'Authorization': 'Bearer ' + currentToken
                }
            });
            
            currentToken = '';
            document.getElementById('loginForm').style.display = 'block';
            document.getElementById('profileForm').style.display = 'none';
            document.getElementById('username').value = '';
            document.getElementById('password').value = '';
            document.getElementById('loginMessage').innerHTML = '';
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// 处理登录页面
func (s *HTTPServer) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// 处理个人资料页面
func (s *HTTPServer) handleProfilePage(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// 处理登录API
func (s *HTTPServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	// 解析HTTP请求
	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		log.Printf("Failed to decode login request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("Login attempt for user: %s", loginReq.Username)

	// 调用RPC服务
	loginResp, err := s.rpcClient.Login(loginReq.Username, loginReq.Password)
	if err != nil {
		log.Printf("RPC login failed: %v", err)

		// 返回更友好的错误信息
		errorResp := map[string]interface{}{
			"success": false,
			"message": "服务器连接失败，请稍后重试",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResp)
		return
	}

	log.Printf("Login result: success=%v, message=%s", loginResp.Success, loginResp.Message)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loginResp)
}

// 处理获取个人资料API
func (s *HTTPServer) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 调用RPC服务
	profileResp, err := s.rpcClient.GetProfile(token)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileResp)
}

// 处理更新个人资料API
func (s *HTTPServer) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var updateReq models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 调用RPC服务
	updateResp, err := s.rpcClient.UpdateProfile(token, updateReq.Nickname, updateReq.ProfilePic)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateResp)
}

// 处理更新信息API
func (s *HTTPServer) handleUpdateInfo(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 检查Content-Type
	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		// 处理文件上传
		s.handleUpdateInfoWithFile(w, r, token)
	} else {
		// 处理JSON数据
		var updateReq models.UpdateProfileRequest
		if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// 先获取用户当前信息
		currentProfileResp, err := s.rpcClient.GetProfile(token)
		if err != nil {
			http.Error(w, "Failed to get current profile", http.StatusInternalServerError)
			return
		}

		if !currentProfileResp.Success {
			http.Error(w, "Failed to get current profile", http.StatusInternalServerError)
			return
		}

		// 使用当前信息作为默认值
		currentNickname := currentProfileResp.User.Nickname
		currentProfilePic := currentProfileResp.User.ProfilePic

		// 如果提供了新昵称，则使用新昵称
		if updateReq.Nickname != "" {
			currentNickname = updateReq.Nickname
		}

		// 如果提供了新头像，则使用新头像
		if updateReq.ProfilePic != "" {
			currentProfilePic = updateReq.ProfilePic
		}

		// 调用RPC服务更新用户信息，使用合并后的值
		updateResp, err := s.rpcClient.UpdateProfile(token, currentNickname, currentProfilePic)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updateResp)
	}
}

// 处理带文件上传的更新信息
func (s *HTTPServer) handleUpdateInfoWithFile(w http.ResponseWriter, r *http.Request, token string) {
	// 限制文件大小 (5MB)
	r.ParseMultipartForm(5 << 20)

	// 获取昵称
	nickname := r.FormValue("nickname")

	// 先获取用户当前信息
	currentProfileResp, err := s.rpcClient.GetProfile(token)
	if err != nil {
		http.Error(w, "Failed to get current profile", http.StatusInternalServerError)
		return
	}

	if !currentProfileResp.Success {
		http.Error(w, "Failed to get current profile", http.StatusInternalServerError)
		return
	}

	// 使用当前信息作为默认值
	currentNickname := currentProfileResp.User.Nickname
	currentProfilePic := currentProfileResp.User.ProfilePic

	// 如果提供了新昵称，则使用新昵称
	if nickname != "" {
		currentNickname = nickname
	}

	// 处理文件上传
	file, handler, err := r.FormFile("avatar")
	if err == nil {
		defer file.Close()

		// 检查文件类型
		allowedTypes := map[string]bool{
			".jpg":  true,
			".jpeg": true,
			".png":  true,
			".gif":  true,
			".webp": true,
		}

		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if !allowedTypes[ext] {
			http.Error(w, "Unsupported file type. Please upload JPG, PNG, GIF, or WebP files.", http.StatusBadRequest)
			return
		}

		// 检查文件大小 (限制为2MB)
		if handler.Size > 2<<20 {
			http.Error(w, "File too large. Please upload files smaller than 2MB.", http.StatusBadRequest)
			return
		}

		// 生成唯一文件名
		timestamp := time.Now().Unix()
		filename := fmt.Sprintf("avatar_%d%s", timestamp, ext)
		filepath := filepath.Join(s.uploadDir, filename)

		// 创建文件
		dst, err := os.Create(filepath)
		if err != nil {
			log.Printf("Failed to create file: %v", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// 复制文件内容
		if _, err := io.Copy(dst, file); err != nil {
			log.Printf("Failed to copy file: %v", err)
			http.Error(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		// 生成访问URL
		currentProfilePic = fmt.Sprintf("/uploads/%s", filename)
	}

	// 调用RPC服务更新用户信息，使用合并后的值
	updateResp, err := s.rpcClient.UpdateProfile(token, currentNickname, currentProfilePic)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updateResp)
}

// 处理登出API
func (s *HTTPServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// 调用RPC服务
	err := s.rpcClient.Logout(token)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "登出成功",
	})
}

// 从请求头中提取Token
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
