# openchat-wx

> 一个可自托管、可二次开发的微信机器人客户端。项目基于
> [`hp0912/wechat-robot-client`](https://github.com/hp0912/wechat-robot-client)
> 继续维护，重点增强 OpenAI 兼容模型接入、多模态能力、自然群聊回复和可复现部署。

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 项目状态

`openchat-wx` 目前以 Docker Compose 作为推荐部署方式，管理后台默认地址为
`https://127.0.0.1:8443/`。仓库只保存源码、部署模板和示例配置，不保存微信登录态、
数据库、证书、滑块令牌或模型密钥。

主要能力包括：

- 微信 iPad 协议登录、联系人和群聊管理；
- AI 私聊与群聊回复、上下文对话、系统提示词；
- 群聊日报、排行榜、词云、每日早报和欢迎语；
- 图片、语音、视频、文件和引用消息处理；
- 知识库、向量检索、长期记忆、MCP 和 Skills；
- Docker 数据持久化、HTTPS 管理后台和健康检查；
- 针对协议长连接旧端口的持久化转发侧车。

## 重要说明

本项目使用非官方微信协议。它可能受到微信版本、账号风控、协议服务和网络环境影响。
请仅用于个人学习、测试和合法用途，不要用于骚扰、批量营销、绕过平台限制或其他违规行为。
使用者需要自行承担账号风险。

## 目录结构

```text
openchat-wx/
├── .deploy/
│   ├── local/                # macOS / Windows / 本机 Docker 推荐配置
│   └── server/               # Linux 服务器部署模板
├── admin-frontend/           # 项目内置管理后台前端（React + Ant Design）
├── cmd/admin-proxy/          # 带管理后台登录校验的客户端 API 代理
├── common_cron/              # 定时任务
├── controller/               # HTTP API 控制器
├── model/                    # 数据模型
├── plugin/                   # 消息插件与能力入口
├── pkg/                      # 协议、MCP、Skills、工具等基础包
├── repository/               # 数据访问层
├── service/                  # 业务服务
├── startup/                  # 启动、迁移、种子和依赖初始化
├── Dockerfile                # 多阶段生产镜像
├── Dockerfile.admin-proxy    # 管理后台安全代理镜像
├── go.mod
└── main.go
```

## 环境要求

开始前请确认：

- macOS、Windows 11 或主流 Linux；
- Docker Desktop 4.30+，或 Docker Engine 24+；
- Docker Compose v2；
- Git；
- OpenSSL，用于生成本地 HTTPS 证书；
- 至少 4 核 CPU、8 GB 内存和 15 GB 可用磁盘；
- 一个可正常登录的微信账号；
- 有效的滑块服务令牌。

检查版本：

```bash
docker version
docker compose version
git --version
openssl version
```

## 快速开始

### 1. 拉取源码

```bash
git clone https://github.com/xiaoguiwucan/openchat-wx.git
cd openchat-wx/.deploy/local
```

### 2. 创建本地配置

```bash
cp .env.example .env
```

编辑 `.env`，至少替换所有 `change-me-*` 项，并填写 `SLIDER_TOKEN`。
建议用随机值生成敏感配置：

```bash
openssl rand -hex 24
```

`.env` 已被 Git 忽略。不要把它发到聊天、Issue、日志或提交记录中。

关键变量：

| 变量 | 必填 | 用途 |
| --- | --- | --- |
| `LOGIN_TOKEN` | 是 | 管理后台登录口令 |
| `SLIDER_TOKEN` | 登录时必填 | 微信登录滑块服务令牌 |
| `SESSION_SECRET` | 是 | 管理后台会话签名 |
| `MYSQL_ROOT_PASSWORD` | 是 | MySQL 管理密码 |
| `MYSQL_USER_PASSWORD` | 是 | MySQL 普通用户密码 |
| `REDIS_PASSWORD` | 是 | Redis 密码 |
| `QDRANT_API_KEY` | 是 | Qdrant API 密钥 |
| `THIRD_PARTY_API_KEY` | 是 | 机器人内部接口鉴权 |
| `OPENAI_API_KEY` | 否 | 管理后台可选 AI 功能 |
| `MUSIC_U` | 否 | 网易云音乐 Cookie |

### 3. 创建 Docker 网络

部署文件使用外部网络，首次安装需要创建一次：

```bash
docker network inspect wechat-robot >/dev/null 2>&1 || docker network create wechat-robot
```

### 4. 生成 HTTPS 证书

macOS / Linux：

```bash
chmod +x gen-self-signed-cert.sh
./gen-self-signed-cert.sh
```

Windows PowerShell：

```powershell
./gen-self-signed-cert.ps1
```

如果要从局域网其他设备访问，请把本机局域网 IP 加进证书：

```bash
./gen-self-signed-cert.sh --ip 192.168.1.10
```

### 5. 校验并启动

```bash
docker compose config --quiet
docker compose pull --ignore-buildable

# 管理后台会按这个兼容标签创建客户端。务必在 pull 之后构建并覆盖该标签。
docker build \
  --build-arg VERSION="$(git -C ../.. describe --tags --always --dirty)" \
  -t openchat-wx:local ../..
docker tag openchat-wx:local \
  registry.cn-shenzhen.aliyuncs.com/houhou/wechat-robot-client:latest

docker compose up -d
docker compose ps
```

管理后端当前把动态客户端镜像名固定为
`registry.cn-shenzhen.aliyuncs.com/houhou/wechat-robot-client:latest`。上面的最后一条
`docker tag` 只是提供兼容名称，镜像中的 `/app/openchat-wx` 仍来自当前仓库源码。如果先
构建再执行 `docker compose pull`，本地兼容标签会被官方镜像覆盖，因此顺序不能颠倒。

首次启动 MySQL 可能需要 1 到 3 分钟。查看整体日志：

```bash
docker compose logs -f --tail=200
```

### 6. 打开管理后台

访问：

```text
https://127.0.0.1:8443/
```

本地自签名证书会触发浏览器警告，这是预期行为。确认地址确实是本机后再继续。
使用 `.env` 中的 `LOGIN_TOKEN` 登录。

### 7. 创建并登录机器人

1. 在管理后台进入机器人管理；
2. 创建机器人实例；
3. 推荐选择 iPad 登录类型；
4. 用手机微信扫描二维码；
5. 在手机上确认登录；
6. 等待后台状态变为“在线”；
7. 同步联系人和群聊。

登录成功后的配置入口不是机器人名称或头像：在机器人卡片底部操作区点击最左侧的设备形
图标，打开机器人详情；进入“联系人”，找到目标群并点击“群聊设置”。灯泡图标的作用是
重启客户端，不是设置入口。

机器人实例由管理后端动态创建，容器名称通常为：

```text
client_<机器人编码>
server_<机器人编码>
```

查看实例：

```bash
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}'
```

## AI 配置

管理后台中可为全局、单个好友或单个群聊配置 AI。群聊/好友级设置优先于全局设置；
没有填写的项目会回退到全局配置。

OpenAI 兼容服务通常需要：

```text
Base URL: https://your-provider.example/v1
API Key:  由中转站提供
Model:    中转站实际支持的模型名
```

请先用中转站的 `/models` 或最小对话请求确认密钥和模型可用，再保存到机器人配置。

### 多模型渠道

登录管理后台后，按以下路径进入内置模型渠道页面：

```text
机器人卡片 → 机器人详情 → 全局设置 → 模型渠道
```

该页面是项目内置 React 管理后台的一部分，不再需要打开孤立页面。浏览器请求会通过同源
`/api/v1/openchat/<机器人编码>/...` 路径进入 `openchat-admin-proxy`；代理先验证管理后台
登录态，再按机器人编码转发到对应的动态客户端容器，因此同时支持多个机器人且不会把
客户端配置接口匿名暴露。

仓库不再提供单独的渠道配置页面或 `9001` 回退端口。使用仓库脚本重部署已有客户端时，
会保留原环境、技能挂载和回滚容器：

```bash
cd .deploy/local
./redeploy-openchat-client.sh client_<机器人编码>
```

一个渠道同时保存 Base URL、API Key、对话模型、识图模型、绘图模型和群总结模型。
渠道只是可复用的连接与模型目录，不会强制所有能力使用同一个渠道。全局设置中的 AI 回复、
图像识别、AI 绘图、群聊总结和文本嵌入都提供独立的“渠道 + 模型”下拉框。例如可以让
AI 回复使用 `Bigsea / gpt-5.6-luna`，同时让 AI 绘图使用
`Grok / grok-imagine-image-quality`。
在“全局设置”中可以新增、编辑、停用、删除、测试和切换任意数量的 OpenAI 兼容渠道。
各能力模型可以来自同一中转站的不同模型。底层 API 还支持把所选渠道应用到以下范围：

- `全局默认`：未单独选择渠道的会话使用该渠道；
- `指定群聊`：输入 `ROOM_ID@chatroom`，只切换该群；
- `指定好友`：输入好友微信 ID，只切换该好友。

新增或编辑渠道时，填写 Base URL 和 API Key 后点击“获取渠道模型”，管理后台会调用兼容
接口的 `/models`，将渠道实际返回的模型持久化缓存，并作为对话、识图、生图和群总结模型
的候选项。页面刷新时直接读取缓存，不会重复访问中转站；再次点击获取或刷新模型时会更新
数据库缓存和所有相关下拉菜单。编辑
已有渠道时 API Key 可以留空，服务端会使用已保存的密钥；如果中转站没有实现 `/models`，
仍可手动输入模型名称。

全局设置不再重复展示 API 地址和 API 密钥。每项能力的连接凭据来自它自己选择的模型渠道，
模型下拉列表来自该渠道缓存的 `/models` 结果；切换渠道后会同步切换下拉列表，并优先带出
该渠道对应能力的默认模型。数据库中的旧连接字段继续保留用于无损升级和旧版本兼容，但
增强版管理后台不会再要求重复填写。

能力渠道选择优先于旧版散落在全局、群聊或好友设置中的 Base URL、API Key 和模型字段。
新字段尚未选择时会回退到原来的全局默认渠道，所以升级后原配置立即可用；保存新的全局
设置后，各能力会持久化自己的渠道选择。群聊或好友如果配置了旧版单独渠道，它继续覆盖
该会话的聊天渠道；尚未保存独立渠道的识图和绘图能力也继续沿用该渠道。独立能力渠道一旦
保存，识图、绘图和总结就不再被旧版单渠道覆盖。

对话、识图、绘图、群聊总结和长期记忆提取都会按当前会话范围解析渠道。对于只实现普通
`chat/completions`、不支持 SSE 流式响应或工具调用的第三方中转站，客户端会自动降级为
非流式或无工具对话，优先保证基础回复可用。

管理页只返回脱敏后的 API Key。编辑已有渠道时密钥输入框留空即可保留原密钥。
“测试连接”会向该渠道的 `/chat/completions` 发送一条最小请求，验证地址、密钥和对话
模型是否可用。绘图和识图仍建议在测试群各做一次真实消息验收。

对应 API：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/v1/robot/ai-providers` | 渠道列表及当前选择 |
| `POST` | `/api/v1/robot/ai-providers` | 创建渠道 |
| `POST` | `/api/v1/robot/ai-providers/models` | 使用新渠道参数或已有渠道密钥获取 `/models` |
| `PUT` | `/api/v1/robot/ai-providers/:id` | 更新渠道，空 `api_key` 保留旧密钥 |
| `DELETE` | `/api/v1/robot/ai-providers/:id` | 删除未被使用的渠道 |
| `POST` | `/api/v1/robot/ai-providers/:id/test` | 最小对话连通性测试 |
| `POST` | `/api/v1/robot/ai-providers/select` | 按全局、群聊或好友切换渠道 |

### 模型继承规则

`openchat-wx` 不锁定任何模型厂商。聊天、识图、绘图和群聊总结均可使用第三方
OpenAI 兼容中转站，实际选择顺序如下：

| 能力 | 地址和密钥 | 模型 |
| --- | --- | --- |
| 聊天 | `chat_ai_provider_id` 对应渠道 | `chat_model` |
| 识图与表情理解 | `image_recognition_provider_id` 对应渠道 | `image_recognition_model` |
| 绘图 | `image_generation_provider_id` 对应渠道 | `image_ai_settings.model` |
| 群聊总结 | `summary_ai_provider_id` 对应渠道 | `chat_room_summary_model` |
| 文本嵌入 | `text_embedding_provider_id` 对应渠道 | `text_embedding_model` |

每个能力渠道为空时回退 `ai_provider_id`。群聊或好友设置了自己的 `ai_provider_id` 时，
该会话的聊天仍使用群/好友渠道；识图、绘图和总结仅在对应独立渠道尚未保存时沿用它。

文本嵌入是可选能力。留空时全局设置仍可保存，聊天、识图、绘图和群聊总结不受影响，
但知识库向量检索和长期记忆不会初始化。文本嵌入渠道必须实现 OpenAI 兼容的
`POST /v1/embeddings`，聊天模型不能直接当作嵌入模型使用。常见组合包括
`text-embedding-3-small / 1536`、`text-embedding-3-large / 3072` 和
`bge-m3 / 1024`；最终维度必须以渠道实际响应为准。

兼容接口要求：聊天、识图和群聊总结需要 `/chat/completions`，绘图需要
`/images/generations`。绘图结果同时支持 URL 与 `b64_json`。模型名称必须使用中转站
实际公布的名称，不能只填写网页中的显示名。

### 自定义绘图模型

管理后台会根据“AI 绘图渠道”自动注入 `base_url` 和 `api_key`，通常只需要在生图模型
下拉框中选择模型，并按需调整尺寸、质量等参数。旧配置仍兼容以下平铺格式：

```json
{
  "base_url": "https://your-provider.example/v1",
  "api_key": "your-api-key",
  "model": "your-image-model",
  "size": "1024x1024",
  "quality": "high",
  "response_format": "b64_json"
}
```

也兼容带命名空间的格式，键名可为 `openai_compatible`、`openai-compatible` 或
`custom`：

```json
{
  "openai_compatible": {
    "base_url": "https://your-provider.example/v1",
    "api_key": "your-api-key",
    "model": "your-image-model",
    "size": "1024x1024"
  }
}
```

选择了 AI 绘图渠道后，可以省略 `base_url` 与 `api_key`，运行时始终使用所选渠道的凭据。
启用绘图后，可以发送“帮我画一张雨夜霓虹街道”或“生成图片：极简产品海报”。

### 识图与表情理解

填写 `image_recognition_model` 后，引用一张已上传到 OSS 的图片或表情包并向机器人
提问，客户端会把图片 URL 作为 OpenAI 多模态消息交给该模型，再把识别结果加入当前
对话上下文。识图模型与聊天模型不仅可以不同，也可以分别来自不同模型渠道。

图片和表情理解依赖机器人 OSS 设置中的“自动上传图片”；未启用时，微信加密媒体没有
中转站可访问的 URL，客户端会明确提示上传配置缺失。

### 表情包复用

客户端会把当前会话历史中收到的原生微信表情作为本地素材池，不复制到公共素材库，也不
跨群共享。可发送“来个表情包”“发个开心表情包”“整张梗图”或“斗图”；客户端优先按
表情 XML 自带描述匹配，没有语义描述时会稳定选择最近素材，并通过微信原生表情接口发送。
引用表情向机器人提问时，则继续使用上面的自定义识图模型理解其文字、主体与情绪。

### 群聊自由回复

自由回复用于处理未 @ 机器人、也没有显式关键词的自然群聊消息。它不会覆盖明确触发：
优先级固定为 `@机器人` → `群/全局强制唤醒词` → `自由回复`。强制唤醒词是可选项，
开启自由回复后无需配置该字段；一旦消息以它开头，则不受自由回复频率、冷却和每日上限影响。
全局入口位于：

```text
机器人卡片 → 机器人详情 → 全局设置 → 自由回复
```

页面提供启用开关、疯狂/积极/平衡/谨慎档位、同群冷却秒数和单群每日上限。群聊单独配置时会
覆盖全局值。对应字段如下：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `free_reply_enabled` | boolean | `false` | 是否启用 |
| `free_reply_level` | string | `normal` | `crazy`、`active`、`normal` 或 `cautious` |
| `free_reply_cooldown_seconds` | integer | `60` | 同一群两次自然回复的最短间隔 |
| `free_reply_daily_limit` | integer | `30` | 单群每日上限；`0` 表示不设上限 |

本地评分沿用 LightAgent 的确定性规则，优先识别机器人名称、开放问题、明确能力需求、
群聊记忆、AI 看法、表情包请求、玩梗和多人复读，并抑制媒体 XML、低信息文本、敏感信息、
明显的人类对话承接和重复触发。四档默认阈值依次为 `crazy=20`、`active=35`、`normal=50`、
`cautious=65`。即使选择 `crazy`，完全没有命中任何规则的普通陈述仍为 0 分，不会逐句插话。
自由回复不会自动 @ 发言人，显式触发仍保持原有 @ 回复行为。建议先在测试群使用
`cautious` 或 `normal`，确认效果后再提高参与程度。

普通寒暄和知识问答使用无工具快速通道；只有搜索、群聊总结、记忆、文件、识图等明确需要
外部能力的请求才加载工具。这样可以避免模型在“在吗”等简单消息上误调用工具并产生第二轮
请求。工具请求的耗时仍取决于所选模型、工具执行和可能发生的后续模型调用。

### 手动运行群聊总结

定时总结默认处理昨天的消息。配置完成后，也可以调用客户端内部 API 对最近 24 小时做
一次手动验收；它会复用该群的 Base URL、API Key、`chat_room_summary_model` 和输出模式：

```bash
curl -X POST http://CLIENT_HOST:9000/api/v1/robot/chat-room/summary/run \
  -H 'Content-Type: application/json' \
  -d '{"chat_room_id":"ROOM_ID@chatroom"}'
```

可选传入 Unix 秒级时间戳 `start_time` 与 `end_time`，时间范围最多 7 天。为避免低质量
报告，现有总结逻辑要求时间范围内至少有 100 条消息；调用成功后总结会直接发送到目标群。

### Docker 中的 localhost

机器人客户端运行在容器内时，`127.0.0.1` 和 `localhost` 指的是容器本身，
不是宿主机。传统部署需要把本机中转站地址写为：

```text
http://host.docker.internal:8000/v1
```

Linux Docker 如无法解析该域名，可为容器添加：

```yaml
extra_hosts:
  - "host.docker.internal:host-gateway"
```

项目的增强版客户端会统一处理这类地址，相关行为仍可通过环境变量覆盖。

## 群聊配置

进入“联系人 / 群聊”，选择目标群后打开群配置。常用选项包括：

- 启用 AI 聊天；
- 可选的 AI 强制唤醒词；
- 群级聊天、识图、生图和群聊总结渠道与模型；
- 群级系统提示词；
- 群聊总结模型、模式和定时任务；
- 欢迎新成员、排行榜、每日早报和早安问候；
- 知识库分类与记忆提取黑名单。

保存后建议先发一条带明确触发词的短消息验证，再逐步开启自然回复或定时任务。

## 数据持久化

以下目录包含本机运行数据，并已加入 `.gitignore`：

```text
.deploy/local/wechat_admin_mysql_data/
.deploy/local/wechat_admin_redis_data/
.deploy/local/wechat-server/
.deploy/local/wechat-robot/
.deploy/local/xiaohongshu-mcp/
.deploy/local/secrets/nginx/
```

其中 `.deploy/local/wechat-robot/` 包含机器人技能及实例数据；数据库中保存设置、联系人、
消息和登录相关状态。删除这些目录可能导致配置和登录态丢失。

## 备份与恢复

### MySQL 备份

```bash
cd .deploy/local
set -a
. ./.env
set +a
docker exec wechat-admin-mysql \
  mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --all-databases \
  > "openchat-wx-$(date +%Y%m%d-%H%M%S).sql"
```

备份文件可能包含账号标识、聊天内容和 API 配置，请加密保管。

### 文件数据备份

停止写入后打包数据目录：

```bash
docker compose stop
tar -czf "openchat-wx-data-$(date +%Y%m%d-%H%M%S).tar.gz" \
  wechat_admin_mysql_data \
  wechat_admin_redis_data \
  wechat-server \
  wechat-robot
docker compose start
```

恢复时使用同一版本的 Compose 文件和 `.env`，先还原目录，再启动服务。

## 从源码构建

仓库根目录执行：

```bash
docker build \
  --build-arg VERSION="$(git describe --tags --always --dirty)" \
  -t openchat-wx:local .
```

默认使用兼容的官方客户端运行层来提供 Chromium、字体和时区数据，也可以显式覆盖：

```bash
docker build \
  --build-arg RUNTIME_IMAGE=registry.cn-shenzhen.aliyuncs.com/houhou/wechat-robot-client:latest \
  -t openchat-wx:local .
```

`RUNTIME_IMAGE` 只提供 Chromium、字体、时区等运行依赖，最终运行的仍是本仓库构建出的
`/app/openchat-wx`。

要让管理后台之后新建的机器人自动使用本地源码镜像，还需要添加兼容标签：

```bash
docker tag openchat-wx:local \
  registry.cn-shenzhen.aliyuncs.com/houhou/wechat-robot-client:latest
```

管理后台里的“更新镜像”会重新拉取上游官方镜像并覆盖此标签。若误点，重新执行构建与
`docker tag`；已经运行的客户端需要停止并重新创建，单纯重启不会切换镜像层。数据库、
微信登录数据和 Skills 都在外部服务或挂载目录中，重新创建客户端前仍建议先备份。

检查镜像：

```bash
docker image inspect openchat-wx:local >/dev/null
```

项目使用多阶段构建，最终镜像只包含运行时依赖和 `openchat-wx` 二进制。

## 升级

升级前先备份数据库和 `.deploy/local/wechat-robot/`。

```bash
git pull --ff-only
cd .deploy/local
docker compose config --quiet
docker compose pull --ignore-buildable

docker build \
  --build-arg VERSION="$(git -C ../.. describe --tags --always --dirty)" \
  -t openchat-wx:local ../..
docker tag openchat-wx:local \
  registry.cn-shenzhen.aliyuncs.com/houhou/wechat-robot-client:latest

docker compose up -d
```

客户端启动时会执行数据库自动迁移。已有客户端仍绑定旧镜像层，需要在管理后台先停止并
移除“客户端容器”，再重新启动客户端；不要删除机器人实例或协议服务端。不要跨多个大
版本跳跃升级，升级后先检查日志和机器人在线状态。

## 停止与重启

```bash
cd .deploy/local

# 重启所有基础服务
docker compose restart

# 停止但保留容器和数据
docker compose stop

# 删除容器和网络连接，但保留持久化目录
docker compose down
```

不要在没有备份的情况下执行 `reset.sh`、删除数据目录或删除数据库卷。

## 故障排查

### 管理后台打不开

```bash
docker compose ps
docker compose logs --tail=200 wechat-nginx
docker compose logs --tail=200 wechat-robot-admin-frontend
docker compose logs --tail=200 wechat-robot-admin-backend
```

确认 8443 端口未被占用，证书文件存在于 `.deploy/local/secrets/nginx/`。

### 创建机器人没有反应

检查管理后端是否能访问 Docker Socket：

```bash
docker inspect wechat-robot-admin-backend \
  --format '{{json .Mounts}}'
docker compose logs --tail=300 wechat-robot-admin-backend
```

### 扫码后提示微信版本过低

这通常是协议版本或设备类型被微信拒绝，不是手机微信真的过旧。优先使用项目当前推荐的
iPad 协议镜像和版本，更新基础镜像后重新创建登录二维码。频繁重复扫码可能提高风控风险。

### 手机已确认，后台仍未登录

依次检查协议服务、客户端和长连接转发：

```bash
docker ps --format '{{.Names}}\t{{.Status}}' | grep -E 'client_|server_|wechat-longlink-proxy'
docker logs --tail=200 wechat-longlink-proxy
```

`wechat-longlink-proxy` 用于把旧协议镜像访问的 `long.weixin.qq.com:80` 转发到当前 443
端口。它应保持 `healthy`。本地 Compose 同时启动 `wechat-longlink-watchdog`：协议进程在
长连接 EOF 后仍保持 HTTP 存活时，watchdog 连续三次确认无活动连接，便只重启对应的
`server_<机器人编码>` 容器。Redis 登录缓存和机器人客户端不会被删除。

确认守护任务：

```bash
docker ps --filter name=wechat-longlink-watchdog
docker logs --tail=100 wechat-longlink-watchdog
```

默认每 30 秒检查一次、连续 3 次才恢复，可在 `.deploy/local/.env` 调整：

```text
LONGLINK_CHECK_INTERVAL_SECONDS=30
LONGLINK_FAILURE_THRESHOLD=3
```

### AI 报 connect: connection refused

若错误地址是 `127.0.0.1`，先确认中转站服务本身已监听：

```bash
curl -fsS http://127.0.0.1:8000/v1/models >/dev/null
```

随后从 Docker 网络测试宿主机：

```bash
docker run --rm curlimages/curl:latest \
  -fsS http://host.docker.internal:8000/v1/models >/dev/null
```

本机能访问而容器不能访问时，问题在 Docker 到宿主机的网络解析或防火墙。

### AI 返回 401 / 403

检查 API Key 是否属于当前 Base URL；部分中转站要求额外请求头或限制来源 IP。
不要把完整密钥写进日志或 Issue。

### AI 返回 404 / model not found

检查 Base URL 是否应带 `/v1`，并确认模型名与 `/models` 返回值完全一致。

## 安全清单

- 不提交 `.env`、证书私钥、数据库目录和微信登录数据；
- 不在 Issue、截图或日志中展示 API Key、滑块令牌和 Data62；
- 首次部署后立即替换全部 `change-me-*` 值；
- 不把 3306、6379 和机器人内部端口直接暴露到公网；
- 公网部署使用可信 HTTPS 证书、反向代理和访问控制；
- 定期备份并验证恢复流程；
- 临时测试密钥使用完后立即撤销或轮换。

## 开发与验证

提交前建议至少执行：

```bash
docker run --rm -v "$PWD:/src" -w /src golang:1.25.8 \
  go test ./... -run '^$' -count=0

docker run --rm -v "$PWD:/src" -w /src golang:1.25.8 \
  go test ./service ./pkg/mcp ./utils -count=1

docker build -t openchat-wx:test .
docker compose -f .deploy/local/docker-compose.yml build \
  wechat-robot-admin-frontend openchat-admin-proxy

docker compose -f .deploy/local/docker-compose.yml \
  --env-file .deploy/local/.env.example config --quiet
```

涉及管理页面或消息行为的改动，还应在测试机器人和测试群中完成真实验证，不要直接在生产群试错。

## 二次拉取验收

发布后可在全新目录执行以下检查，确认仓库不依赖本机未提交文件：

```bash
tmp_dir="$(mktemp -d)"
git clone https://github.com/xiaoguiwucan/openchat-wx.git "$tmp_dir/openchat-wx"
cd "$tmp_dir/openchat-wx"

docker build -t openchat-wx:fresh-clone .
docker compose -f .deploy/local/docker-compose.yml build \
  wechat-robot-admin-frontend openchat-admin-proxy
docker tag openchat-wx:fresh-clone \
  registry.cn-shenzhen.aliyuncs.com/houhou/wechat-robot-client:latest
docker compose -f .deploy/local/docker-compose.yml \
  --env-file .deploy/local/.env.example config --quiet
```

真实启动前仍需复制 `.env.example`、替换密钥、生成证书和创建 Docker 网络。

## 上游与致谢

- 上游项目：[`hp0912/wechat-robot-client`](https://github.com/hp0912/wechat-robot-client)
- 能力设计参考：[`yideng966/LightAgent`](https://github.com/yideng966/LightAgent)

本仓库保留上游 MIT License 和版权信息。任何引用项目的名称、商标和服务均归其各自所有者所有。
`admin-frontend/` 基于上游管理前端并保留其 ISC License 文件。

## License

[MIT](LICENSE)
