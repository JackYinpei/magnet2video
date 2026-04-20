# magnet2video

## 简介

magnet2video 是一个企业级 BT 种子下载与视频转码服务，支持磁力链解析、P2P 下载、视频自动转码、云存储上传和在线播放。

当前版本采用 **Server / Worker 拆分架构**，适合 Server 跑在公网小配置机器上、Worker 跑在家里/内网大硬盘机器上的场景：

- **Server**：对外暴露 API，负责数据库、Redis、JWT、UI、签名 URL 等轻量工作。
- **Worker**：执行磁力链下载、FFmpeg 转码、S3/GCS 上传等重 I/O 工作，无需公网 IP。
- 两者通过 **RabbitMQ** 通信（事件 + 心跳 + 任务），Worker 离线时任务会在队列里堆积，上线后自动消费。

也保留了单机 `all` 模式（开发/小规模部署一键跑起来）。

## 技术栈

- **Go** + **Gin**：Web 框架
- **GORM**：ORM，支持 MySQL / PostgreSQL / SQLite，自动迁移
- **anacrolix/torrent**：BT 下载引擎
- **FFmpeg / FFprobe**：视频转码（Remux / H.264）
- **Redis**：缓存（Cache-Aside）、Worker 心跳 TTL、事件幂等 SETNX
- **RabbitMQ / GoChannel**：异步消息队列，支持跨进程事件与任务派发
- **GCS / S3**：云存储，Signed URL 访问
- **Wire**：编译时依赖注入（本仓库手工维护 `wire_gen.go`）
- **JWT**：认证鉴权
- **Logrus** + **file-rotatelogs**：结构化日志与轮转
- **bwmarrin/snowflake**：分布式 ID

## 主要特性

- 磁力链解析、P2P 下载，支持暂停 / 恢复 / 删除
- 视频自动转码为浏览器兼容格式（H.264/MP4），实时进度回报
- HTTP Range 请求，支持视频拖拽和边下边播
- 云存储自动上传，Signed URL 安全播放
- JWT 认证 + 公开 / 私有种子权限控制
- **Server / Worker 拆分**：Worker 离线时任务进入队列排队，上线自动恢复
- **Worker 状态面板**：前端顶部 Banner 展示在线 / 离线、当前任务、磁盘剩余
- **事件幂等**：基于 Redis SETNX（TTL 5 分钟），重连不重复处理
- **进度节流**：下载 / 转码 / 上传进度每 2 秒上报一次
- 国际化（zh-CN / en-US）

## 架构

```
                       ┌──────────────────────┐
                       │   浏览器 / 客户端     │
                       └──────────┬───────────┘
                                  │ HTTPS
                                  ▼
                       ┌──────────────────────┐
                       │      Server (公网)    │
                       │  Gin API / DB / Redis │
                       │  JWT / UI / Signed URL│
                       └──────┬────────┬──────┘
                              │        │
                          任务 │        │ 事件 / 心跳
                   (download- │        │ (worker-events,
                       jobs,  │        │  worker-heartbeat)
                    transcode-│        │
                       jobs,  │        │
                cloud-upload- │        │
                       jobs)  ▼        │
                       ┌──────────────────────┐
                       │       RabbitMQ        │
                       └──────┬────────┬──────┘
                              │        │
                              ▼        ▲
                       ┌──────────────────────┐
                       │     Worker (内网)     │
                       │ Torrent / FFmpeg / S3 │
                       └──────────────────────┘
```

- 任务类 Topic：`download-jobs`、`transcode-jobs`、`cloud-upload-jobs`（Server → Worker）
- 事件类 Topic：`worker-events`、`worker-heartbeat`（Worker → Server）

## 快速开始

### 1. 准备依赖

至少需要：

- MySQL / PostgreSQL / SQLite
- Redis
- RabbitMQ（Server / Worker 拆分模式必需；`all` 模式可用内存 GoChannel）
- FFmpeg / FFprobe（Worker 机器上）
- S3 / GCS Bucket（可选，开启云上传时使用）

### 2. 构建

使用仓库自带的 bash 构建脚本：

```bash
# 默认：构建 server 和 worker 两个二进制，输出到 bin/
./build.sh

# 等价写法
./build.sh all

# 只构建其中一个
./build.sh server
./build.sh worker

# 构建单体二进制（默认 -mode=all）
./build.sh mono

# 其它子命令
./build.sh run-server      # go run . -mode=server
./build.sh run-worker      # go run . -mode=worker
./build.sh run-all         # go run . -mode=all
./build.sh test
./build.sh vet
./build.sh fmt
./build.sh clean
./build.sh help
```

产物：

- `bin/magnet2video-server`
- `bin/magnet2video-worker`
- `bin/magnet2video`（mono 模式，单体）

### 3. 配置

项目提供三份模板：

| 模板 | 用途 |
| --- | --- |
| `configs/config.example.yml`        | 单机 `all` 模式 |
| `configs/config.server.example.yml` | 公网 Server |
| `configs/config.worker.example.yml` | 内网 Worker |

拆分部署时：

```bash
# Server 机器
cp configs/config.server.example.yml configs/config.yml
# 填写数据库 / Redis / RabbitMQ / S3 配置

# Worker 机器
cp configs/config.worker.example.yml configs/config.yml
# 填写 RabbitMQ / S3 / 下载目录 / FFmpeg 路径
```

关键配置：

```yaml
APP:
  MODE: "server"        # server / worker / all
  WORKER_ID: ""         # 留空则使用 hostname-pid-ts

QUEUE:
  TYPE: "rabbitmq"      # 拆分模式必须使用 rabbitmq
  RABBITMQ:
    URL: "amqp://user:pass@mq.example.com:5672/"
```

### 4. 启动

```bash
# Server 机器
./bin/magnet2video-server -mode=server

# Worker 机器
./bin/magnet2video-worker -mode=worker

# 单机一体
./bin/magnet2video -mode=all
```

命令行 `-mode` 参数优先级高于配置文件中的 `APP.MODE`。

### 5. Docker（可选）

目前仓库的 `docker-compose.yml` 演示的是单机 `all` 部署：

```bash
docker compose up -d
```

拆分架构下建议自行把 Server / Worker 分别打成镜像，指向同一个 RabbitMQ / 数据库实例。

## 目录结构

```
magnet2video/
├── cmd/                         # 入口 & 三种模式启动逻辑
│   ├── all.go                   # mode=all
│   ├── server.go                # mode=server
│   ├── worker.go                # mode=worker
│   ├── shared.go                # 模式间共享的 helper（超管创建等）
│   └── gin_server.go            # -mode 分发器
├── configs/                     # 配置与模板
├── internal/
│   ├── events/                  # 跨进程事件协议
│   │   ├── types/               # WorkerEvent 信封 / 负载定义
│   │   ├── gateway/             # Worker 侧事件网关（MQ 实现）
│   │   ├── processor/           # Server 侧事件处理器（含 SETNX 幂等）
│   │   └── heartbeat/           # Publisher / Consumer / StatusStore
│   ├── torrent/handler/         # Worker 侧下载任务消费者 & 进度上报
│   ├── transcode/handler/       # Worker 侧转码消费者（事件化）
│   ├── cloud/handler/           # Worker 侧云上传消费者（事件化）
│   └── ...                      # DB / Redis / Cache / SSE / AI / i18n ...
├── pkg/
│   ├── router/                  # 路由注册（含 worker 状态接口）
│   ├── serve/                   # controller + service (DDD 三层)
│   └── wire/                    # 容器与依赖注入（手写 wire_gen.go）
├── web/static/                  # 前端资源（含 Worker 状态 Banner）
├── build.sh                     # bash 构建脚本
└── main.go
```

## 开发提示

- 事件幂等：`internal/events/processor` 使用 Redis key `worker:event:seen:<eventID>` 做 SETNX，TTL 5 分钟。
- 心跳存活：Worker 每 10 秒推送一次心跳，Server 侧 Redis key `worker:status:<id>` TTL 30 秒，超过即判定离线。
- 进度节流：下载 / 转码 / 上传进度默认 2 秒上报一次，避免刷爆队列。
- 本仓库 `pkg/wire/wire_gen.go` 为**手工维护**，不要直接重跑 `wire`；修改 Provider 后请同步手改容器构造器。

## 许可证

Apache License 2.0
