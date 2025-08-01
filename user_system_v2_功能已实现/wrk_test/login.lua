-- Lua script for wrk with JSON data
math.randomseed(os.time())

-- 用户名列表，前10000个用户
local users = {}
for i = 1, 10000 do
    table.insert(users, "user_" .. i)
end

-- 生成随机用户
local function get_random_user()
    local idx = math.random(1, #users)
    local username = users[idx]
    return username, username  -- 假设密码与用户名相同
end

-- 设置请求方法和参数
wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"

-- 定义请求体
function request()
    local username, password = get_random_user()
    local body = '{"username": "' .. username .. '", "password": "' .. password .. '"}'
    return wrk.format("POST", "/api/login", wrk.headers, body)
end
