# LongCat API

[![Go 版本](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org)
[![许可证](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![English Docs](https://img.shields.io/badge/docs-English-blue.svg)](README_EN.md)

OpenAI 和 Claude API 兼容的 LongCat 聊天服务。这允许您将 LongCat 与任何 OpenAI 或 Claude API 兼容的客户端一起使用。

本项目基于 https://github.com/JessonChan/longcat-web-api 进行修改

**📖 [中文版本](README.md) | [English Version](README_EN.md)**

## 🚀 功能特性

- ✅ OpenAI API 兼容性 (`/v1/chat/completions`)
- ✅ Claude API 兼容性 (`/v1/messages`)
- ✅ 流式和非流式响应
- ✅ 对话历史管理
- ✅ 交互式 Cookie 配置
- ✅ 安全的 Cookie 存储
- ✅ Web 应用程序的 CORS 支持
- ✅ 详细日志模式

## 📋 目录

- [快速开始](#快速开始)
- [安装](#安装)
- [配置](#配置)
- [API 使用](#api-使用)
- [开发者指南](#开发者指南)
- [故障排除](#故障排除)
- [贡献](#贡献)
- [许可证](#许可证)

## 🚀 快速开始


### 前置要求
- Go 1.21 或更高版本
- LongCat 聊天账户

## 📦 安装


### 使用 Go Install
```bash
go install github.com/JessonChan/longcat-web-api@latest
```

安装后，`longcat-web-api` 二进制文件将在您的 Go bin 目录中可用。您可以直接运行它：

```bash
longcat-web-api
```

**首次运行设置：**
如果没有配置 Cookie，系统会提示您提供它们：
```
=== 需要 Cookie 配置 ===

获取您的 Cookie：
1. 在浏览器中打开 https://longcat.chat 并登录
2. 打开开发者工具 (F12)
3. 转到应用程序/存储 → Cookie → https://longcat.chat
4. 找到这些 Cookie 并复制它们的值

在此处粘贴您的 Cookie 并按 Enter：
> _lxsdk_cuid=xxx; passport_token_key=yyy; _lxsdk_s=zzz
```

服务器将在默认端口 8082 上启动。

## 🤖 客户端配置

### DeepChat 配置

1. **打开 DeepChat**
2. **前往 设置 → 提供商**
3. **添加自定义提供商：**
   - **提供商名称：** `LongCat API`
   - **API 密钥：** `any-code` (或任何您想要的文本)
   - **基础 URL：** `http://localhost:8082/v1`
   - **模型：** `gpt-4` (或任何模型名称)
4. **保存并选择 LongCat API 提供商**

### CherryStudio 配置

1. **打开 CherryStudio**
2. **前往 设置 → API 密钥**
3. **添加自定义 API 配置：**
   - **API 名称：** `LongCat API`
   - **API 密钥：** `any-code` (必需，但可以是任何文本)
   - **API URL：** `http://localhost:8082/v1`
   - **模型：** `gpt-4`
4. **保存并选择 LongCat API 配置**

### Claude Code 


1. **设置环境变量：**
   ```bash
   export ANTHROPIC_BASE_URL=http://localhost:8082
   ```

### 其他 OpenAI 兼容客户端

对于任何 OpenAI 兼容的客户端，使用以下设置：
- **API 密钥：** `any-code` (不验证，但大多数客户端需要)
- **基础 URL：** `http://localhost:8082/v1`
- **模型：** `gpt-4` (或您喜欢的任何模型名称)

### 从源代码安装
```bash
git clone https://github.com/JessonChan/longcat-web-api.git
cd longcat-web-api
go build -o longcat-web-api
```


### 1. 构建应用程序
```bash
go build -o longcat-web-api
```

### 2. 运行服务器
```bash
./longcat-web-api
```

**首次运行设置：**
首次运行时，如果没有配置 Cookie，系统会提示您提供它们：
```
=== 需要 Cookie 配置 ===

获取您的 Cookie：
1. 在浏览器中打开 https://longcat.chat 并登录
2. 打开开发者工具 (F12)
3. 转到应用程序/存储 → Cookie → https://longcat.chat
4. 找到这些 Cookie 并复制它们的值

在此处粘贴您的 Cookie 并按 Enter：
> _lxsdk_cuid=xxx; passport_token_key=yyy; _lxsdk_s=zzz
```

服务器将在默认端口 8082 上启动。


## ⚙️ 配置

### Cookie 配置

#### 方法 1：交互式设置（推荐）
只需运行应用程序并在提示时粘贴您的 Cookie。它们将被安全保存以供将来使用。

#### 方法 2：环境变量
在您的 `.env` 文件或环境中设置：
```bash
COOKIE_LXSDK_CUID=your_cuid_value
COOKIE_PASSPORT_TOKEN=your_token_value  # 必需
COOKIE_LXSDK_S=your_s_value
```

#### 方法 3：保存的配置
当您在交互式设置期间选择保存 Cookie 时，Cookie 会自动保存到 `~/.config/longcat-web-api/config.json`。

### 环境变量

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `SERVER_PORT` | 服务器端口 | 8082 |
| `LONGCAT_API_URL` | LongCat API 端点 | (内置) |
| `TIMEOUT_SECONDS` | 请求超时 | 30 |
| `COOKIE_LXSDK_CUID` | LongCat 会话 Cookie | - |
| `COOKIE_PASSPORT_TOKEN` | LongCat 认证令牌（必需） | - |
| `COOKIE_LXSDK_S` | LongCat 跟踪 Cookie | - |

## 🛠️ 命令行选项

```bash
# 显示帮助
./longcat-web-api -h

# 更新存储的 Cookie
./longcat-web-api -update-cookies

# 清除存储的 Cookie
./longcat-web-api -clear-cookies

# 显示版本
./longcat-web-api -version

# 启用详细日志
./longcat-web-api -verbose
```

## 🔌 API 使用

### OpenAI 兼容 API

#### 基本聊天完成
```bash
curl http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "你好！你好吗？"}
    ],
    "stream": false
  }'
```

#### 流式响应
```bash
curl http://localhost:8082/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "system", "content": "你是一个有帮助的助手。"},
      {"role": "user", "content": "用简单的术语解释量子计算。"}
    ],
    "stream": true
  }'
```

### Claude 兼容 API

#### 基本消息
```bash
curl http://localhost:8082/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3",
    "max_tokens": 1000,
    "messages": [
      {"role": "user", "content": "你好！你好吗？"}
    ]
  }'
```

#### 带系统消息
```bash
curl http://localhost:8082/v1/messages \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3",
    "max_tokens": 1000,
    "system": "你是一个以友好语气回答的有帮助的助手。",
    "messages": [
      {"role": "user", "content": "生命的意义是什么？"}
    ],
    "stream": true
  }'
```

### Python 客户端示例

```python
import openai

# 配置 OpenAI 客户端以使用 LongCat 包装器
client = openai.OpenAI(
    api_key="not-needed",  # 本地包装器不需要 API 密钥
    base_url="http://localhost:8082/v1"
)

# 非流式聊天完成
response = client.chat.completions.create(
    model="gpt-4",
    messages=[
        {"role": "user", "content": "你好！你能帮我学习 Go 编程吗？"}
    ]
)
print(response.choices[0].message.content)

# 流式聊天完成
stream = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "给我讲个故事"}],
    stream=True
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="")
```

### JavaScript/Node.js 示例

```javascript
const OpenAI = require('openai');

const openai = new OpenAI({
  baseURL: 'http://localhost:8082/v1',
  apiKey: 'not-needed' // 本地包装器不需要 API 密钥
});

async function chat() {
  const completion = await openai.chat.completions.create({
    model: 'gpt-4',
    messages: [
      { role: 'user', content: '你好！你怎么能帮助我？' }
    ]
  });
  
  console.log(completion.choices[0].message.content);
}

chat();
```

## 🔑 从浏览器获取 Cookie

1. 打开 https://longcat.chat 并登录
2. 打开开发者工具 (F12)
3. 转到应用程序选项卡 → 存储 → Cookie
4. 查找并复制这些 Cookie 值：
   - `_lxsdk_cuid`
   - `passport_token_key`（必需）
   - `_lxsdk_s`

您可以单独复制它们或作为完整的 Cookie 字符串复制。

## 👨‍💻 开发者指南

### 项目结构

```
longcat-web-api/
├── main.go                 # 主应用程序入口点
├── api/                    # API 服务实现
│   ├── openai.go          # OpenAI API 兼容性
│   ├── claude.go          # Claude API 兼容性
│   └── client.go          # LongCat API 客户端
├── config/                # 配置管理
├── types/                 # 类型定义
├── conversation/          # 对话管理
└── logging/              # 日志工具
```

### 开发设置

1. **克隆仓库：**
   ```bash
   git clone https://github.com/JessonChan/longcat-web-api.git
   cd longcat-web-api
   ```

2. **安装依赖：**
   ```bash
   go mod tidy
   ```

3. **在开发模式下运行：**
   ```bash
   go run main.go -verbose
   ```

### 构建

```bash
# 为当前平台构建
go build -o longcat-web-api

# 为多个平台构建
make build-all
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行详细输出的测试
go test -v ./...

# 运行覆盖率测试
go test -cover ./...
```

### 贡献

1. Fork 仓库
2. 创建功能分支：`git checkout -b feature/amazing-feature`
3. 提交您的更改：`git commit -m 'Add amazing feature'`
4. 推送到分支：`git push origin feature/amazing-feature`
5. 打开 Pull Request

#### 代码风格

- 遵循 Go 标准格式化 (`go fmt`)
- 使用约定式提交
- 为新功能添加测试
- 根据需要更新文档

## 🚨 故障排除

### 常见问题

#### 认证失败
**错误：** `Failed to authenticate with LongCat`

**解决方案：**
1. 更新您的 Cookie：`./longcat-web-api -update-cookies`
2. 确保 Cookie 没有过期（如果需要，重新登录 LongCat）
3. 验证您是否复制了完整的 Cookie 值
4. 检查配置文件是否具有适当的权限

#### 端口已被占用
**错误：** `bind: address already in use`

**解决方案：**
1. 更改端口：`export SERVER_PORT=8083`
2. 杀死使用该端口的进程：`lsof -ti:8082 | xargs kill -9`

#### 构建错误
**错误：** 各种 Go 编译错误

**解决方案：**
1. 确保您有 Go 1.21 或更高版本：`go version`
2. 清理并重新构建：`go clean && go build`
3. 更新依赖：`go mod tidy`

#### Cookie 配置问题
**错误：** 未找到 Cookie 或 Cookie 无效

**解决方案：**
1. 清除保存的 Cookie：`./longcat-web-api -clear-cookies`
2. 重新配置 Cookie：`./longcat-web-api -update-cookies`
3. 检查环境变量是否设置正确

### 常见问题

**问：我需要 API 密钥吗？**
答：不需要，您只需要来自浏览器的 LongCat 会话 Cookie。

**问：我可以将此与任何 OpenAI/Claude 客户端一起使用吗？**
答：是的，它与任何支持 OpenAI 或 Claude API 格式的客户端兼容。

**问：当我的 Cookie 过期时如何更新？**
答：运行 `./longcat-web-api -update-cookies` 并从浏览器提供新的 Cookie。

**问：我的对话历史会被保存吗？**
答：对话历史仅在服务器会话期间在内存中管理。

**问：我可以在不同的端口上运行吗？**
答：是的，设置 `SERVER_PORT` 环境变量：`export SERVER_PORT=3000`

## 🔒 安全说明

- Cookie 以 0600 权限存储（仅所有者读/写）
- Cookie 值在显示时被屏蔽
- `passport_token_key` 是认证所必需的
- 保护您的 Cookie 安全，不要分享它们
- 服务器默认在本地运行 - 在向网络公开时请谨慎

## 🤝 贡献

欢迎贡献！请随时提交 Pull Request。对于重大更改，请先打开 Issue 讨论您想要更改的内容。

## 📄 许可证

本项目采用 MIT 许可证 - 详情请参见 [LICENSE](LICENSE) 文件。