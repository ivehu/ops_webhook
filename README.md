# Ops Webhook HTTP 服务

Ops Webhook HTTP 服务是一个使用 Go 语言编写的轻量级 HTTP 服务，用于接收特定请求并执行预定义的命令。该服务支持基本的认证机制，确保只有授权的请求才能触发命令执行。

## 功能特性
- **配置化管理**：通过 `config.yaml` 文件配置端口、认证字符串和允许执行的命令。
- **认证机制**：使用 `Bearer Token` 进行请求认证，确保服务安全。
- **命令执行**：支持执行预定义的命令，并返回执行结果，未指定的命令不可执行。

## 环境要求
- Go 1.16 或更高版本

## 安装与运行

### 1. 克隆项目
```bash
git clone <项目仓库地址>
cd <项目目录>
```

### 2. 安装依赖
```bash
go mod tidy
```

### 3. 配置文件
在项目根目录下创建 config.yaml 文件，示例内容如下：

```yaml
port: 10020
authstr: your_auth_string
commands:  
    - "ls *"  
    - "pwd"  
    - "df -hT"
```
port：服务监听的端口号。
authstr：用于请求认证的字符串。
commands：允许执行的命令列表，支持通配符 *。

### 4. 运行服务
```bash
go run main.go
```

## 使用方法
### 1. 健康检查
发送 GET 请求到 /ping 端点，服务将返回 pong。
```bash
curl -X GET http://localhost:10020/ping
```
### 2. 执行命令
发送 POST 请求到 /run 端点，请求头中需要包含 Authorization 字段，值为 Bearer <authstr>。请求体中包含要执行的命令。
```bash
curl -X POST -H "Authorization: Bearer your_auth_string" -d "ls /tmp" http://localhost:10020/run
```
### 3. 错误处理
如果请求未通过认证，服务将返回 401 Unauthorized。
如果请求体为空或包含非法字符，服务将返回 400 Bad Request。
如果命令执行失败，服务将返回 500 Internal Server Error。

## 日志记录
服务使用 Go 标准库的 log 包进行日志记录，日志信息包含日期、时间、文件名和行号。

## 注意事项
请确保 config.yaml 文件中的 commands 列表只包含安全的命令，避免执行危险操作。
服务使用 sh -c 执行命令，请确保系统中存在 sh 解释器。

## supervisord运行服务
```bash
[program:ops_webhook]
directory=/root/ops_webhook
command=/root/ops_webhook/ops_webhook
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/webser/logs/ops_webhook/ops_webhook.log
stdout_logfile_maxbytes=50MB
stdout_logfile_backups=5
stdout_capture_maxbytes=1MB
stdout_events_enabled=false
stopsignal=QUIT
```