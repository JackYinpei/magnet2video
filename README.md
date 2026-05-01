# magnet2video

## 简介

magnet2video 是一个企业级 BT 种子下载与视频转码服务，支持磁力链解析、P2P 下载、视频自动转码、云存储上传和在线播放。

当前版本采用 **Server / Worker 拆分架构**，适合 Server 跑在公网小配置机器上、Worker 跑在家里/内网大硬盘机器上的场景：

- **Server**：对外暴露 API，负责数据库、Redis、JWT、UI、签名 URL 等轻量工作。
- **Worker**：执行磁力链下载、FFmpeg 转码、S3 上传等重 I/O 工作，无需公网 IP。
- 两者通过 **RabbitMQ** 通信（事件 + 心跳 + 任务），Worker 离线时任务会在队列里堆积，上线后自动消费。

也保留了单机 `all` 模式（开发/小规模部署一键跑起来）。

## 技术栈

- **Go** + **Gin**：Web 框架
- **GORM**：ORM，支持 MySQL / PostgreSQL / SQLite，自动迁移
- **anacrolix/torrent**：BT 下载引擎
- **FFmpeg / FFprobe**：视频转码（Remux / H.264）
- **Redis**：缓存（Cache-Aside）、Worker 心跳 TTL、事件幂等 SETNX
- **RabbitMQ / GoChannel**：异步消息队列，支持跨进程事件与任务派发
- **S3 / S3 兼容**（AWS S3、MinIO、Ceph、自托管 S3 网关）：云存储，Signed URL 访问
- **Wire**（包名沿用，但 `pkg/wire/container.go` 实际为手工维护的容器构造器）
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

整体由三方组成 —— **前端**(浏览器)、**Server**(对外 API,持有 DB/Redis)、**Worker**(执行 P2P 下载/转码/上云)。三方通过 RabbitMQ 解耦:Server 把任务下发到 worker,Worker 异步上报事件回 Server。

```mermaid
flowchart TB
  subgraph FE["🖥️ 前端 (浏览器)"]
    Browser["app.js + index.html<br/>JWT 鉴权 / detailCache / 3s 轮询"]
  end

  subgraph SVR["🌐 Server (mode=server / mode=all)"]
    direction TB
    HTTP["Gin :8080<br/>Controller + Service 层"]
    EvProc["WorkerEventProcessor<br/>消费 worker-events,<br/>幂等去重 + DB 写入 +<br/>触发后续队列消息"]
    HBC["HeartbeatConsumer<br/>消费 worker-heartbeat → Redis"]
    Reaper["Reaper goroutine<br/>定期扫中间态僵尸行 → Failed"]
    LTM["Local TorrentManager<br/>(仅用于 ParseMagnet 解析磁链)"]
  end

  subgraph WK["⚙️ Worker (mode=worker / mode=all)"]
    direction TB
    JobDL["DownloadJobHandler"]
    JobTC["TranscodeHandler"]
    JobCU["CloudUploadHandler"]
    P2PMgr["TorrentManager<br/>(anacrolix/torrent)"]
    FFmpeg["FFmpeg / FFprobe"]
    CloudMgr["Cloud Manager<br/>(S3 / S3-compatible)"]
    HBPub["HeartbeatPublisher 30s"]
    ProgRpt["ProgressReporter 5s"]
  end

  subgraph INFRA["🗄️ 共享基础设施"]
    direction LR
    DB[("MySQL<br/>users / torrents /<br/>torrent_files / transcode_jobs")]
    Redis[("Redis<br/>缓存 / 幂等键 /<br/>下载进度 / 心跳状态")]
    MQ{{"RabbitMQ<br/>5 个 topic"}}
  end

  subgraph EXT["🌍 外部"]
    P2P[(BT Swarm)]
    LocalDisk[(Worker 本地磁盘<br/>/downloads /transcoded)]
    Cloud[(S3 / S3-compatible 对象存储)]
  end

  %% Frontend ↔ Server (HTTP)
  Browser <==>|"REST /api/v1/*<br/>auth · parse · download · list · detail<br/>transcode/requeue · cloud-upload/retry-file<br/>cloud-url · file (Range fallback)"| HTTP

  %% Frontend ↔ Cloud (直连)
  Browser -.->|"GET 视频<br/>(Signed URL,<br/>客户端直连)"| Cloud
  HTTP -->|"GenerateSignedURL"| Cloud

  %% Server ↔ 共享
  HTTP --> DB
  HTTP --> Redis
  HTTP --> LTM
  EvProc --> DB
  EvProc --> Redis
  HBC --> Redis
  Reaper --> DB

  %% Server → MQ (job 下行)
  HTTP ==>|"download-jobs<br/>start / pause / resume / remove"| MQ
  HTTP ==>|"transcode-jobs<br/>(下载完成 / 用户 requeue)"| MQ
  EvProc ==>|"cloud-upload-jobs<br/>(转码完成自动触发)"| MQ

  %% MQ → Worker (job 派发)
  MQ ==>|"download-jobs"| JobDL
  MQ ==>|"transcode-jobs"| JobTC
  MQ ==>|"cloud-upload-jobs"| JobCU

  %% Worker 内部
  JobDL --> P2PMgr
  JobTC --> FFmpeg
  JobCU --> CloudMgr
  P2PMgr <-->|"piece 交换"| P2P
  P2PMgr --> LocalDisk
  FFmpeg --> LocalDisk
  CloudMgr -->|"对象上传<br/>multipart >100MB"| Cloud

  %% Worker → MQ (event 上行)
  JobDL -.->|"download.completed<br/>+ files[] 元数据"| MQ
  JobTC -.->|"transcode.started/progress/<br/>completed/failed +<br/>subtitle.extracted"| MQ
  JobCU -.->|"cloud.upload.started/<br/>completed/failed +<br/>cloud_path"| MQ
  ProgRpt -.->|"download.progress (高频)"| MQ
  HBPub -.->|"worker-heartbeat<br/>{worker_id, ts, version}"| MQ

  %% MQ → Server (event 消费)
  MQ -.->|"worker-events"| EvProc
  MQ -.->|"worker-heartbeat"| HBC
```

**线型说明**

| 线型 | 含义 |
| --- | --- |
| `==>` 粗实线 | Job(命令)下行,Server → Worker |
| `-.->` 虚线 | Event(遥测)上行 / 客户端直连 Cloud |
| `-->` 普通实线 | 同进程函数调用 / DB / Redis / 文件系统 |
| `<==>` 双向粗线 | HTTP 请求/响应 |

### RabbitMQ Topic

| Topic | 方向 | 载荷 |
| --- | --- | --- |
| `download-jobs` | Server → Worker | `DownloadJobMessage`(start / pause / resume / remove + magnet/info_hash + selected_files) |
| `transcode-jobs` | Server → Worker | `TranscodeMessage`(torrent_id, file_index, input_path, job_id) |
| `cloud-upload-jobs` | Server(EventProcessor)→ Worker | `CloudUploadMessage`(torrent_id, file_index, local_path, object_path, is_transcoded) |
| `worker-events` | Worker → Server | `WorkerEvent` 包络 + payload(`download.progress` / `download.completed` / `transcode.job.{started,progress,completed,failed}` / `subtitle.extracted` / `cloud.upload.{started,completed,failed}`) |
| `worker-heartbeat` | Worker → Server | `{worker_id, timestamp, version}` |

### 三类典型链路

**① 用户提交磁链 → 播放**

```
Browser → POST /torrent/parse → Server (LocalTM 解析元数据) → 返回 files[]
Browser → POST /torrent/download → Server 写 DB + 发 download-jobs
MQ → Worker JobDL → P2PMgr ↔ P2P,写 LocalDisk
Worker → ProgRpt 高频发 download.progress → Server 写 Redis(30s TTL,不进 DB)
Worker → JobDL 发 download.completed → Server 写 DB,自动发 transcode-jobs
MQ → Worker JobTC → FFmpeg 转码,写 LocalDisk
Worker → 发 transcode.job.completed → Server 写 DB + 自动发 cloud-upload-jobs
MQ → Worker JobCU → CloudMgr 上传到 S3
Worker → 发 cloud.upload.completed → Server 写 DB
Browser → GET /torrent/cloud-url/:hash/:idx → Server 生成 Signed URL → Browser 直连 Cloud 播放
```

**② 用户在前端点"重新转码"/"强制转码"**

```
Browser → POST /transcode/requeue {info_hash, file_index, force} → Server
Server (TranscodeService) → 写 DB(状态置 Pending)→ 发 transcode-jobs
↓ 之后流程同 ① 的转码段
```

**③ Worker 崩溃 / 队列消息丢失,文件卡死**

```
某文件 transcode_status=Processing,但 worker 挂了,事件不再来
... 4h 后(可配)...
Reaper goroutine 扫描 → updated_at < now-threshold → 降级为 Failed
+ recomputeTranscodeStatus → 种子级状态同步
→ 前端 detail 端点显示 Failed → 用户点"重新转码"或"强制转码"恢复
```

### 核心设计决策

- **Server 不持有大文件**: 拆分部署时 worker 才有 `/downloads`,Server 通过 Signed URL 让前端直接访问 Cloud,避免代理大流量。
- **Outbox-style 派发**: Service 层先发 MQ,成功才写 DB(防止 DB 已 Pending 但消息丢了 → 卡死)。
- **幂等键 + Redis**: 同一 `event_id` 5min 内只处理一次(`worker:event:seen:<event_id>`),防止 ack 失败导致重放。
- **Reaper 兜底**: Pending/Processing/Uploading 超时降级 Failed,补全 worker 崩溃 / 消息丢失 等"事件丢失"场景。配置见 `EVENTS.REAPER`。
- **DB 唯一索引 + 重试**: `torrent_files (torrent_id, index)` 唯一索引 + `MAX(index)+1` 重试,防止并发 transcode-completed 写丢行。
- **Producer 自动重连**: RabbitMQ Producer 在 channel 死掉后会自动重连并重试一次,避免短期网络抖动雪崩成 504 风暴。

## 快速开始

### 1. 准备依赖

至少需要：

- MySQL / PostgreSQL / SQLite
- Redis
- RabbitMQ（Server / Worker 拆分模式必需；`all` 模式可用内存 GoChannel）
- FFmpeg / FFprobe（Worker 机器上）
- S3 / S3 兼容 Bucket（可选，开启云上传时使用；MinIO、Ceph、自托管 S3 网关均可）

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
│   └── wire/                    # 容器与依赖注入（手写 container.go）
├── web/static/                  # 前端资源（含 Worker 状态 Banner）
├── build.sh                     # bash 构建脚本
└── main.go
```

## 开发提示

- 事件幂等：`internal/events/processor` 使用 Redis key `worker:event:seen:<eventID>` 做 SETNX，TTL 5 分钟。
- 心跳存活：Worker 每 10 秒推送一次心跳，Server 侧 Redis key `worker:status:<id>` TTL 30 秒，超过即判定离线。
- 进度节流：下载 / 转码 / 上传进度默认 2 秒上报一次，避免刷爆队列。
- 本仓库 `pkg/wire/container.go` 为**手工维护**的容器构造器（包名 `wire` 是历史遗留）。修改依赖时直接编辑 `buildShared / buildServerInfra / buildWorkerInfra / buildServerEventPlumbing / buildWorkerEventPlumbing / buildWorkerHandlers / buildServicesAndControllers` 即可，无需运行 `wire` CLI。

## 许可证

Apache License 2.0
