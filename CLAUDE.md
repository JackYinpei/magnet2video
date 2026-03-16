# CLAUDE.md

本文档为 Claude Code 提供项目架构、编码规范和开发指南,确保代码一致性和可维护性。

## 项目概览

**项目名称**: magnet-video
**技术栈**: Go 1.25.1 + Gin + GORM + Redis + 消息队列(GoChannel/RabbitMQ)
**核心功能**: 企业级 BT 种子下载服务,支持磁力链解析、P2P 下载管理、视频自动转码和在线播放
**部署方式**: Docker Compose(MySQL + Redis + RabbitMQ(可选) + Go 应用)

### 主要特性

1. **磁力链解析**: 基于 `anacrolix/torrent` 库解析磁力链,获取种子元数据
2. **BT 下载管理**: P2P 下载、进度跟踪、暂停/恢复/删除操作
3. **视频转码**: FFmpeg 自动转码为 H.264/MP4 浏览器兼容格式(Remux 或 Transcode)
4. **异步处理**: 消息队列解耦下载和转码,支持 GoChannel(内存)和 RabbitMQ 两种实现
5. **流式播放**: HTTP Range 请求支持,实现视频拖拽和边下边播
6. **权限控制**: JWT 认证 + 公开/私有种子分离
7. **缓存优化**: Redis 缓存(Cache-Aside 模式) + 异步写入策略
8. **云存储集成**: 支持 GCS/S3 云存储,自动上传转码文件,Signed URL 安全访问

---

## Torrent 文件模型(扁平化)

当前设计统一采用 **扁平 `files` 列表**,不再使用“主文件 + subtitles 嵌套”结构:

- `files[]` 中每条记录都代表一个实体文件(原始/转码/字幕)
- 字段:
  - `type`: `"video" | "subtitle" | "other"`
  - `source`: `"original" | "transcoded" | "extracted"`
  - `parent_path`: 关联到原始文件(转码/字幕文件指向原视频)
- `poster_path` 仍为独立字段(外部导入,不进入 `files`)
  - `local://<relative_path>`: 本地文件(相对下载目录),由服务端解析为 `/api/v1/torrent/file/:hash/*path`
  - `cloud://<object_path>`: 云存储对象路径,由服务端签名为可访问 URL
  - `http(s)://...`: 外部直链,直接透传

**注意**: 当前阶段不做数据迁移,历史记录中的 `subtitles`/`transcoded_path` 不会自动展开为独立条目。

---

## 开发命令

### 运行应用

```bash
# 运行应用(使用 configs/config.yml)
go run main.go

# Docker Compose 部署
docker-compose up -d
```

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定测试
go test ./internal/utils/errorx/...

# 显示详细输出
go test -v ./...

# S3 集成测试(无本机 Go 环境时,通过 Docker 运行)
# 说明:
# - 会执行真实上传 + Signed URL 下载校验
# - 需配置好 configs/config.yml 或相关 S3 环境变量
sudo docker run --rm -it \
  -e RUN_S3_INTEGRATION_TEST=1 \
  -v "$PWD":/workspace \
  -w /workspace \
  -v gomodcache:/go/pkg/mod \
  -v gobuildcache:/root/.cache/go-build \
  golang:1.25-alpine \
  sh -c "/usr/local/go/bin/go test ./internal/cloud/internal -run '^TestS3UploadWithProdConfig$' -v -count=1"
```

### 代码质量工具

```bash
# 格式化代码
gofmt -w .

# 自动导入管理
goimports -w .

# 静态分析
go vet ./...

# 生成 Wire 依赖注入代码
cd pkg/wire && wire
```

---

## 架构设计

### 整体架构

项目采用经典的**分层架构** + **依赖注入**模式,结构清晰,职责分明:

```
magnet-video/
├── cmd/                          # 应用入口
│   └── gin_server.go            # 服务器初始化和生命周期管理
├── configs/                      # 配置文件
│   ├── config.yml               # 应用配置(从 config.example.yml 复制)
│   ├── i18n/                    # 国际化资源文件(zh-CN, en-US)
│   └── prompts/                 # AI 提示词模板(YAML)
├── internal/                     # 核心基础设施层(不可被外部导入)
│   ├── ai/                      # AI 服务集成(OpenAI/Gemini)
│   ├── cache/                   # 缓存抽象层(基于 Redis)
│   ├── cloud/                   # 云存储模块(GCS/S3)
│   │   ├── cloud.go            # 工厂类和接口定义
│   │   ├── internal/           # 具体实现(manager.go=GCS, s3_manager.go=S3)
│   │   ├── handler/            # 云上传消息消费者
│   │   └── types/              # 消息类型定义
│   ├── db/                      # 数据库管理(GORM)
│   ├── i18n/                    # 国际化管理
│   ├── logger/                  # 日志管理(Logrus + 日志轮转)
│   ├── middleware/              # Gin 中间件(CORS/Auth/RequestID)
│   ├── model/                   # 数据模型
│   │   ├── base/               # 基础模型(雪花 ID、软删除)
│   │   ├── user/               # 用户模型
│   │   ├── torrent/            # BT 种子模型
│   │   └── transcode/          # 转码任务模型
│   ├── queue/                   # 消息队列抽象层(GoChannel/RabbitMQ)
│   ├── redis/                   # Redis 客户端管理
│   ├── sse/                     # 服务端推送(Server-Sent Events)
│   ├── torrent/                 # BT 下载核心引擎
│   ├── transcode/               # 视频转码模块
│   │   ├── ffmpeg/             # FFmpeg 命令封装
│   │   ├── handler/            # 消息消费者处理器
│   │   └── types/              # 转码消息类型定义
│   ├── types/                   # 类型定义和常量
│   │   ├── consts/             # 系统常量
│   │   └── errno/              # 错误码定义(分模块管理)
│   └── utils/                   # 工具库
│       ├── email/              # 邮件发送
│       ├── errorx/             # 错误处理框架
│       ├── file/               # 文件操作
│       ├── jwt/                # JWT 认证
│       ├── snowflake/          # 雪花 ID 生成器
│       ├── validator/          # 参数校验
│       └── vo/                 # VO 响应工具
├── pkg/                         # 应用层(MVC 架构)
│   ├── router/                 # 路由注册
│   │   ├── router.go           # 路由主入口
│   │   └── routes/             # 模块化路由配置
│   ├── serve/
│   │   ├── controller/         # HTTP 控制器层
│   │   │   ├── dto/           # 请求 DTO(带验证规则)
│   │   │   ├── torrent_controller.go
│   │   │   ├── user_controller.go
│   │   │   └── admin_controller.go
│   │   └── service/            # 业务逻辑层
│   │       ├── impl/          # 服务实现
│   │       ├── torrent_service.go    # 服务接口
│   │       └── user_service.go
│   ├── vo/                     # 视图对象(响应 VO)
│   └── wire/                   # 依赖注入配置(Wire)
│       ├── wire.go            # 容器定义
│       ├── providers.go       # 依赖提供者
│       └── wire_gen.go        # 自动生成(勿手动编辑)
├── web/                        # 前端静态资源(Go embed)
│   └── static/                # 嵌入式静态文件
├── download/                   # BT 下载存储目录
├── database/                   # SQLite 数据库文件
├── .logs/                      # 日志文件目录
└── docs/                       # 项目文档
    └── coding-standards.en-US.md  # 编码规范
```

### 架构图 (Mermaid)

```mermaid
flowchart TB
  subgraph Clients["客户端"]
    U["用户/浏览器"]
    A["管理员"]
  end

  subgraph App["magnet-video (Gin)"]
    Entry["main.go → cmd.Start"]
    Config["Viper 配置 (configs/*.yml + ENV)"]
    Wire["Wire 容器"]
    Gin["gin.Engine"]
    MW["中间件\nRecovery / RequestID / CORS / Logger / JWT / Admin"]
    Router["Router /api/v1 & /api/v2"]
    Controllers["Controllers\nUser / Torrent / Admin / Test"]
    Services["Services\nUser / Torrent / Transcode / Admin / Test"]
    Web["Embedded Web UI\nweb/static"]
  end

  subgraph Infra["基础设施层"]
    DB[(GORM DB\nMySQL/Postgres/SQLite)]
    Redis[(Redis)]
    Cache["Cache Manager"]
    Cloud["Cloud Storage\nGCS / S3"]
    CloudConsumer["Cloud Upload Consumer"]
    TorrentMgr["Torrent Manager\nanacrolix/torrent"]
    Queue["Queue Producer\nGoChannel / RabbitMQ"]
    Consumer["Transcode Consumer\n(Handler)"]
    FFmpeg["FFmpeg / FFprobe"]
    SSE["SSE Manager"]
    AI["AI Manager\nOpenAI / Gemini"]
    Logger["Logrus + Rotatelogs"]
  end

  subgraph External["外部系统"]
    P2P[(BT Swarm / Trackers)]
    MQ[(RabbitMQ 可选)]
    Disk[(Download Dir / Transcoded Files)]
    CloudStorage[(GCS / S3 云存储)]
  end

  U --> Gin
  A --> Gin

  Entry --> Config --> Wire --> Gin
  Gin --> MW --> Router --> Controllers --> Services
  Gin --> Web

  Services --> DB
  Services --> Cache --> Redis
  Services --> TorrentMgr --> P2P
  Services --> Queue --> MQ
  Queue --> Consumer --> FFmpeg --> Disk
  Consumer --> Cloud --> CloudConsumer --> CloudStorage
  TorrentMgr --> Disk
  Services --> SSE
  Services --> AI
  Services --> Logger
```

### 分层职责

#### 1. 基础设施层 (`internal/`)

基础设施层提供技术组件和底层服务,不涉及业务逻辑。

##### 数据库模块 (`internal/db/`)
- **职责**: GORM 封装,支持 PostgreSQL/MySQL/SQLite,自动迁移
- **核心接口**: `DatabaseManager`
  - `DB()`: 获取 GORM 实例
  - `Initialize()`: 初始化数据库连接
  - `Close()`: 关闭连接
- **配置**: `configs/config.*.yml` 中的 `DATABASE` 配置
- **自动迁移**: 启动时自动迁移 `internal/model/get_all_models.go` 中注册的模型

##### Redis 模块 (`internal/redis/`)
- **职责**: Redis 客户端管理,连接池配置
- **核心接口**: `RedisManager`
  - `GetClient()`: 获取 redis.Client 实例
- **配置**: 支持单机/哨兵/集群模式,连接池参数可配置

##### 缓存模块 (`internal/cache/`)
- **职责**: Redis 缓存抽象,实现 Cache-Aside 模式
- **核心功能**:
  - `Get/Set/Delete`: 基础缓存操作
  - `GetOrSet`: 缓存未命中时自动加载并缓存
  - `DeleteByPattern`: 批量删除(支持通配符)
- **依赖**: RedisManager, LoggerManager

##### BT 下载引擎 (`internal/torrent/`)
- **职责**: 基于 `anacrolix/torrent` 库实现磁力链解析和 P2P 下载
- **核心文件**: `internal/torrent/internal/manager.go` (18KB)
- **核心功能**:
  - `ParseMagnet`: 解析磁力链,获取文件列表
  - `StartDownload`: 启动下载任务(支持文件选择)
  - `GetProgress`: 查询下载进度
  - `GetFileReader`: 获取文件流(支持模糊匹配)
  - `PauseDownload/ResumeDownload/RemoveDownload`: 管理下载生命周期
- **配置**: 下载目录、监听端口(42069)、DHT 配置等

##### 视频转码模块 (`internal/transcode/`)
- **职责**: FFmpeg 视频转码为浏览器兼容格式(H.264/MP4)
- **核心组件**:
  - `ffmpeg/`: FFmpeg/FFprobe 命令封装
  - `handler/`: 消息消费者,异步处理转码任务
  - `types/`: 转码消息和类型定义
- **转码策略**:
  - **Remux**: 仅容器转换(快速,无需重编码)
  - **Transcode**: 完全重编码为 H.264(适用于不兼容编码)
- **工作流程**:
  1. FFprobe 分析视频编码格式
  2. 判断是否需要转码(仅 H.264 才可 Remux)
  3. FFmpeg 执行转码(实时进度回调)
  4. 更新数据库状态

##### 消息队列 (`internal/queue/`)
- **职责**: 消息队列抽象层,支持多种实现
- **实现方式**:
  - **GoChannel**: 内存队列,无需外部服务,适合开发/测试/单机部署
  - **RabbitMQ**: 分布式消息队列,适合生产环境
- **核心接口**:
  - `Producer`: 消息生产者接口
  - `Consumer`: 消息消费者接口
  - `Handler`: 消息处理器接口
  - `Message`: 通用消息结构(与实现无关)
- **使用场景**: 转码任务和云上传任务异步处理
- **Topic**: `transcode-jobs`, `cloud-upload-jobs`
- **消息格式**: 见 `internal/transcode/types/message.go`
- **配置切换**: 通过 `QUEUE.TYPE` 配置选择实现(`channel` 或 `rabbitmq`)
- **RabbitMQ 消费者自动重连**: 当 RabbitMQ 连接断开或 delivery channel 关闭时,消费者 goroutine 会自动检测并重连,使用指数退避重试(5s → 10s → 20s → ... → 最大 60s),重连成功后自动重新订阅 topic 并恢复消费。通过 `sync.Mutex` 保护并发重连安全。

##### 日志系统 (`internal/logger/`)
- **职责**: Logrus + file-rotatelogs 实现日志轮转
- **特性**: 分级日志(Debug/Info/Warn/Error)、自动轮转、结构化输出
- **配置**: 日志路径、保留天数、轮转周期

##### AI 集成 (`internal/ai/`)
- **职责**: 多 AI 提供商集成(OpenAI/Gemini)
- **特性**:
  - 负载均衡: Round-robin 跨 API Key
  - 速率限制: 每实例独立速率限制(如 "60/min")
  - 重试机制: 可配置最大重试次数
  - 提示词模板: YAML 模板(支持变量替换)
- **配置**: `configs/config.*.yml` 中的 `AI.PROVIDERS`

##### 中间件 (`internal/middleware/`)
- **中间件栈** (注册顺序,见 `pkg/router/router.go`):
  1. `gin.Recovery()` - Panic 恢复
  2. `requestid.New()` - 请求追踪 ID
  3. `cors.New()` - 跨域支持(可配置)
  4. `gin.Logger()` - 日志记录
  5. `auth.JWTMiddleware()` - JWT 认证(保护路由)
  6. `auth.AdminMiddleware()` - 管理员权限校验

##### 云存储模块 (`internal/cloud/`)
- **职责**: 文件上传到云存储(GCS/S3),生成 Signed URL 安全访问
- **支持的云服务商**:
  - **Google Cloud Storage (GCS)**: `internal/cloud/internal/manager.go`
  - **AWS S3 / MinIO**: `internal/cloud/internal/s3_manager.go`
- **核心接口**: `CloudStorageManager`
  - `Upload(ctx, objectPath, reader, contentType)`: 上传文件
  - `UploadWithProgress(ctx, objectPath, reader, size, contentType, progressFn)`: 带进度回调上传,大文件(>100MB)自动使用 multipart upload,上传期间每 5 分钟输出 `[upload-status]` 日志显示当前上传文件和已用时间
  - `GenerateSignedURL(ctx, objectPath, expiration)`: 生成临时访问链接
  - `Delete(ctx, objectPath)`: 删除云端文件
  - `Exists(ctx, objectPath)`: 检查文件是否存在
  - `IsEnabled()`: 检查云存储是否启用
  - `GetSignedURLExpiration()`: 获取 Signed URL 过期时间
- **消息队列**:
  - **Topic**: `cloud-upload-jobs`
  - **消息格式**: `internal/cloud/types/message.go`
  - **消费者**: `internal/cloud/handler/cloud_upload_handler.go`
- **工作流程**:
  1. 转码完成(或非转码文件下载完成)
  2. 发送 `CloudUploadMessage` 到消息队列
  3. `CloudUploadHandler` 消费消息,调用 `CloudStorageManager.UploadWithProgress()`
  4. 更新 `TorrentFile.CloudUploadStatus` 和 `CloudPath`
  5. 前端请求 `/api/v1/torrent/cloud-url/:hash/:index` 获取 Signed URL
  6. 客户端直接访问云存储播放视频
- **配置**: 见下方 `CLOUD_STORAGE` 配置项

#### 2. 数据模型层 (`internal/model/`)

##### Base 模型 (`internal/model/base/`)
```go
type Base struct {
    ID        int64   // 雪花 ID 主键(唯一、有序、高性能)
    CreatedAt int64   // 创建时间戳(毫秒)
    UpdatedAt int64   // 更新时间戳(毫秒)
    Ext       JSONMap // JSON 扩展字段(灵活存储自定义数据)
    Deleted   bool    // 软删除标志
}
```
- 自动生成雪花 ID(分布式唯一)
- GORM 钩子自动管理时间戳
- 所有业务模型继承 `Base`

##### Torrent 模型 (`internal/model/torrent/`)
```go
type Torrent struct {
    Base
    InfoHash            string       // 种子哈希(唯一索引)
    Name                string       // 种子名称
    Files               TorrentFiles // 文件列表(JSON 序列化)
    TotalSize           int64        // 总大小(字节)
    Status              int          // 下载状态: 0=待下载,1=下载中,2=完成,3=失败,4=暂停
    Progress            int          // 下载进度(0-100)
    TranscodeStatus     int          // 转码状态: 0=无需转码,1=待转码,2=转码中,3=完成,4=失败
    TranscodeProgress   int          // 转码进度(0-100)
    IsPublic            bool         // 是否公开分享
    UserID              int64        // 所属用户 ID
    CloudUploadStatus   int          // 云上传整体状态
    CloudUploadProgress int          // 云上传进度(0-100)
    CloudUploadedCount  int          // 已上传文件数
    TotalCloudUpload    int          // 总上传文件数
}

type TorrentFile struct {
    Path              string // 文件相对路径
    Size              int64  // 文件大小
    IsSelected        bool   // 是否下载
    IsStreamable      bool   // 是否可流式播放(视频/音频)
    TranscodeStatus   int    // 文件转码状态: 0=无需转码,1=待转码,2=转码中,3=完成,4=失败
    TranscodedPath    string // 转码后文件路径
    CloudUploadStatus int    // 云上传状态: 0=无, 1=待上传, 2=上传中, 3=完成, 4=失败
    CloudPath         string // 云存储对象路径
    CloudUploadError  string // 云上传失败错误信息
}
```

##### TranscodeJob 模型 (`internal/model/transcode/`)
```go
type TranscodeJob struct {
    Base
    TorrentID     int64  // 关联 Torrent ID
    InputPath     string // 输入文件路径
    OutputPath    string // 输出文件路径
    Status        int    // 任务状态: 0=待处理,1=处理中,2=完成,3=失败
    Progress      int    // 进度百分比(0-100)
    TranscodeType string // "remux" 或 "transcode"
    ErrorMsg      string // 错误信息(失败时)
}
```

##### User 模型 (`internal/model/user/`)
```go
type User struct {
    Base
    Username     string // 用户名(唯一索引)
    PasswordHash string // 密码哈希(bcrypt)
    Email        string // 邮箱
    IsActive     bool   // 是否激活
    IsAdmin      bool   // 是否管理员
}
```

##### 抽象图 (Mermaid)

> 说明: Torrent.Files / Trackers 在数据库中以 JSON 字段存储,图中以实体关系抽象表达。

```mermaid
erDiagram
  USER ||--o{ TORRENT : owns
  TORRENT ||--o{ TORRENT_FILE : files
  TORRENT ||--o{ TRANSCODE_JOB : jobs

  USER {
    bigint id PK
    string email
    string nickname
    string role
    bool is_super_admin
  }

  TORRENT {
    bigint id PK
    string info_hash
    string name
    bigint total_size
    int status
    float progress
    bool is_public
    int transcode_status
    int transcode_progress
    bigint creator_id FK
  }

  TORRENT_FILE {
    string path
    bigint size
    bool is_selected
    bool is_streamable
    int transcode_status
    string transcoded_path
    int cloud_upload_status
    string cloud_path
  }

  TRANSCODE_JOB {
    bigint id PK
    bigint torrent_id FK
    string info_hash
    int file_index
    int status
    int progress
    string transcode_type
    string input_path
    string output_path
  }
```

#### 3. 应用层 (`pkg/`)

应用层实现业务逻辑,遵循 MVC 三层架构。

##### 三层架构模式

```
Controller 层 (pkg/serve/controller/)
    ↓ 调用
Service 层 (pkg/serve/service/impl/)
    ↓ 调用
Database/TorrentManager/QueueProducer
```

- **Controller 层**:
  - 处理 HTTP 请求,验证 DTO
  - 调用 Service 层业务逻辑
  - 返回 VO 响应
  - 无业务逻辑

- **Service 层**:
  - 核心业务逻辑实现
  - 事务管理
  - 调用基础设施层组件(Database/TorrentManager/QueueProducer)
  - 错误处理

##### 核心控制器

**TorrentController** (`pkg/serve/controller/torrent_controller.go`)
- `ParseMagnet`: 解析磁力链,返回文件列表
- `StartDownload`: 启动下载(支持文件选择)
- `GetProgress`: 查询下载进度
- `PauseDownload/ResumeDownload/RemoveDownload`: 管理下载
- `ListTorrents`: 查询用户种子列表
- `GetDetail`: 获取种子详情
- `ListPublicTorrents`: 浏览公开种子
- `SetPublic`: 设置种子公开/私有
- `ServeFile`: 流式传输文件(支持 Range 请求,实现视频拖拽)
- `ServeTranscodedFile`: 服务转码后文件
- `RequestTranscode`: 手动触发转码
- `GetCloudURL`: 获取云存储 Signed URL

**UserController** (`pkg/serve/controller/user_controller.go`)
- `Register`: 用户注册
- `Login`: 用户登录(返回 JWT Token)
- `GetInfo`: 获取用户信息
- `UpdateProfile`: 更新用户资料

**AdminController** (`pkg/serve/controller/admin_controller.go`)
- `ListUsers`: 查询所有用户
- `UpdateUser`: 修改用户信息
- `DeleteUser`: 删除用户
- `ListTranscodeJobs`: 查询转码任务
- `RetryTranscode`: 重试失败的转码任务

##### 核心服务

**TorrentServiceImpl** (`pkg/serve/service/impl/torrent_service_impl.go`)
- 648 行,核心业务逻辑实现
- 主要功能:
  - 磁力链解析和元数据获取
  - 下载任务生命周期管理
  - 文件流读取(支持模糊匹配)
  - 转码任务调度(消息队列)
  - 权限验证(公开/私有种子)

##### 路由设计

**公开路由** (无需认证)
```
GET  /api/v1/torrent/public               # 浏览公开种子
GET  /api/v1/torrent/detail/:hash         # 种子详情
GET  /api/v1/torrent/file/:hash/*path     # 流式播放(支持 Range)
GET  /api/v1/torrent/transcoded/*path     # 转码文件服务
GET  /api/v1/torrent/cloud-url/:hash/:idx # 获取云存储 Signed URL
POST /api/v1/user/register                # 用户注册
POST /api/v1/user/login                   # 用户登录
```

**保护路由** (需要 JWT Token)
```
POST /api/v1/torrent/parse                # 解析磁力链
POST /api/v1/torrent/download             # 启动下载
GET  /api/v1/torrent/progress/:hash       # 查询进度
POST /api/v1/torrent/pause/:hash          # 暂停下载
POST /api/v1/torrent/resume/:hash         # 恢复下载
POST /api/v1/torrent/remove/:hash         # 删除下载
GET  /api/v1/torrent/list                 # 用户种子列表
POST /api/v1/torrent/poster               # 设置海报(从已有图片文件)
POST /api/v1/torrent/poster/upload        # 上传海报到云存储
POST /api/v1/torrent/public/:hash         # 设置公开/私有
POST /api/v1/torrent/transcode            # 手动触发转码
GET  /api/v1/user/info                    # 用户信息
PUT  /api/v1/user/update                  # 更新资料
```

**管理员路由** (需要管理员权限)
```
GET  /api/v1/admin/users                  # 查询所有用户
PUT  /api/v1/admin/user/:id               # 修改用户
DELETE /api/v1/admin/user/:id             # 删除用户
GET  /api/v1/admin/transcode              # 查询转码任务
POST /api/v1/admin/transcode/retry/:id    # 重试转码
```

##### DTO 和 VO

**DTO** (`pkg/serve/controller/dto/`)
- 请求数据传输对象,带验证规则
- 使用 `binding` 标签定义验证规则
```go
type StartDownloadDTO struct {
    InfoHash       string `json:"infoHash" binding:"required"`
    SelectedFiles  []int  `json:"selectedFiles" binding:"required,min=1"`
}
```

**VO** (`pkg/vo/`)
- 视图对象,API 响应格式
- 使用 `internal/utils/vo/vo_utils.go` 工具构建统一响应
```go
type TorrentVO struct {
    InfoHash  string        `json:"infoHash"`
    Name      string        `json:"name"`
    Files     []TorrentFile `json:"files"`
    Status    int           `json:"status"`
    Progress  int           `json:"progress"`
}
```

---

## 依赖注入 (Wire)

项目使用 **Google Wire** 实现编译时依赖注入,避免反射开销。

### Wire 文件结构

```
pkg/wire/
├── wire.go           # 容器定义(wireinject 构建标签)
├── wire_gen.go       # 自动生成(勿手动编辑)
└── providers.go      # 依赖提供者函数
```

### Container 结构

```go
type Container struct {
    // 基础设施管理器
    DatabaseManager      db.DatabaseManager
    RedisManager         redis.RedisManager
    LoggerManager        logger.LoggerManager
    TorrentManager       torrent.TorrentManager
    QueueProducer        queue.QueueProducer
    CloudStorageManager  cloud.CloudStorageManager

    // 业务控制器
    TorrentController  controller.TorrentController
    UserController     controller.UserController
    AdminController    controller.AdminController

    // 业务服务
    TorrentService     service.TorrentService
    UserService        service.UserService
    AdminService       service.AdminService
}
```

### 添加新依赖的步骤

1. 在 `providers.go` 中添加 Provider 函数:
```go
func ProvideFooService(db db.DatabaseManager) service.FooService {
    return impl.NewFooServiceImpl(db)
}
```

2. 将 Provider 添加到 `AllProviders` 集合:
```go
var AllProviders = wire.NewSet(
    InfrastructureProviders,
    ServiceProviders,
    ControllerProviders,
    ProvideFooService, // 新增
)
```

3. 在 `wire.go` 的 `Container` 结构体中添加字段:
```go
type Container struct {
    // ...
    FooService service.FooService
}
```

4. 重新生成依赖注入代码:
```bash
cd pkg/wire && wire
```

---

## 应用生命周期

应用启动流程详见 `cmd/gin_server.go:25`。

### 启动流程

1. **加载配置** (`configs/config.yml`)

2. **Wire 依赖注入** → 创建 `Container`

3. **初始化管理器** (按顺序):
   - LoggerManager
   - DatabaseManager (自动迁移数据库模型)
   - RedisManager
   - TorrentManager

4. **创建超级管理员** (如果配置了 `APP.USER`)

5. **启动转码消费者** (后台 goroutine)

6. **启动云上传消费者** (如果启用云存储,后台 goroutine)

7. **注册中间件和路由**

8. **启动 HTTP 服务器** (端口 8080)

9. **监听信号优雅关闭** (SIGINT/SIGTERM)
   - 15 秒超时
   - 关闭所有管理器(释放资源)

### 优雅关闭

```go
defer func() {
    container.DatabaseManager.Close()
    container.RedisManager.Close()
    container.TorrentManager.Close()
    container.CloudStorageManager.Close()
}()
```

---

## 核心业务流程

### 数据流图 (Mermaid)

```mermaid
sequenceDiagram
  participant U as 用户
  participant UI as Web UI
  participant API as Gin API
  participant TM as TorrentManager
  participant DB as 数据库
  participant Q as 队列(GoChannel/RabbitMQ)
  participant C as Transcode Consumer
  participant CC as Cloud Consumer
  participant FF as FFmpeg/FFprobe
  participant FS as 下载目录
  participant Cloud as 云存储(GCS/S3)

  U->>UI: 提交磁力链
  UI->>API: POST /api/v1/torrent/parse
  API->>TM: ParseMagnet
  TM-->>API: 元数据/文件列表
  API-->>UI: 解析结果

  U->>UI: 选择文件并开始下载
  UI->>API: POST /api/v1/torrent/download
  API->>TM: StartDownload
  API->>DB: 创建/更新 Torrent 记录
  TM->>FS: 下载文件

  UI->>API: GET /api/v1/torrent/list 或 /progress (轮询)
  API->>TM: GetProgress
  API->>DB: 更新进度/状态
  API-->>UI: 下载状态

  Note over API,DB: 下载完成后触发转码检测
  API->>DB: 创建 TranscodeJob & 更新文件状态
  API->>Q: 发送 TranscodeMessage
  Q->>C: 消费转码任务
  C->>FF: Remux / Transcode
  FF->>FS: 输出 _transcoded.mp4
  C->>DB: 更新转码进度/状态

  Note over C,Cloud: 转码完成后触发云上传
  C->>Q: 发送 CloudUploadMessage
  Q->>CC: 消费云上传任务
  CC->>Cloud: 上传文件
  CC->>DB: 更新云上传状态

  U->>UI: 播放视频
  UI->>API: GET /api/v1/torrent/cloud-url/:hash/:idx
  API->>Cloud: GenerateSignedURL
  Cloud-->>API: Signed URL
  API-->>UI: 返回云链接
  UI->>Cloud: 直接访问云存储播放
```

### BT 下载流程

```
1. 用户提交磁力链(ParseMagnet)
   ↓
2. TorrentManager 解析元数据 → 返回文件列表
   ↓
3. 用户选择文件(StartDownload)
   ↓
4. 创建 Torrent 记录 → 启动下载
   ↓
5. 后台监控下载进度 → 更新数据库
   ↓
6. 下载完成 → 判断是否需要转码(仅视频文件)
   ↓
7. 发送转码消息到消息队列
```

### 视频转码流程

```
1. Consumer 接收转码消息
   ↓
2. TranscodeHandler 处理任务
   ↓
3. FFprobe 分析视频编码格式
   ↓
4. 决策: Remux(快速) 或 Transcode(完全重编码)
   ↓
5. FFmpeg 执行转码(实时进度回调)
   ↓
6. 更新 TranscodeJob 和 Torrent 状态
   ↓
7. 转码完成 → 触发云上传(如果启用)
```

### 云上传流程

```
1. 转码完成(或非转码文件下载完成)
   ↓
2. 发送 CloudUploadMessage 到消息队列 (cloud-upload-jobs)
   ↓
3. CloudUploadHandler 消费消息
   ↓
4. 更新文件状态为 uploading
   ↓
5. CloudStorageManager.UploadWithProgress() 上传到 GCS/S3
   ↓
6. 更新 TorrentFile.CloudUploadStatus = completed, 保存 CloudPath
   ↓
7. 更新 Torrent 整体上传进度
```

### 文件播放流程

```
1. 前端请求视频文件
   ↓
2. 检查文件是否已上传到云存储
   ↓
3a. 已上传云 → 请求 /cloud-url/:hash/:idx → 获取 Signed URL → 直接访问云存储
3b. 未上传云 → 使用本地文件(ServeFile 或 ServeTranscodedFile)
   ↓
4. 验证权限(公开种子或本人种子)
   ↓
5. 获取文件 Reader(支持模糊匹配)
   ↓
6. http.ServeContent 处理 Range 请求
   ↓
7. 流式传输视频数据
```

---

## 配置管理

配置文件使用 **Viper** 管理,支持热重载。

### 配置文件路径

- 配置文件: `configs/config.yml`（从 `config.example.yml` 复制后修改）

### 主要配置项

```yaml
APP:
  APP_PORT: "8080"          # HTTP 端口
  CORS:                     # 跨域配置
    ALLOW_ORIGINS: ["*"]
  JWT:                      # JWT 配置
    SECRET: "your-secret"
    EXPIRE_HOURS: 168       # 7 天
  USER:                     # 超级管理员配置
    USERNAME: "admin"
    PASSWORD: "admin123"

DATABASE:
  DB_DIALECT: "mysql"       # 支持 mysql/postgres/sqlite
  DB_NAME: "magnet"
  DB_HOST: "localhost"
  DB_PORT: "3306"

REDIS:
  HOST: "localhost:6379"
  POOL_SIZE: 10
  MIN_IDLE_CONNS: 5

QUEUE:
  TYPE: "channel"             # 队列类型: "channel"(内存) 或 "rabbitmq"
  RABBITMQ:                   # RabbitMQ 配置(仅当 TYPE=rabbitmq 时需要)
    URL: "amqp://guest:guest@localhost:5672/"
    EXCHANGE: "transcode"
    EXCHANGE_TYPE: "direct"
    PREFETCH_COUNT: 1

TORRENT:
  DOWNLOAD_DIR: "download"  # 下载目录
  LISTEN_PORT: 42069        # P2P 监听端口

AI:
  PROVIDERS:
    openai:
      - INSTANCE_NAME: "default"
        API_KEY: "sk-..."
        MODEL: "gpt-4"
        RATE_LIMIT: "60/min"

CLOUD_STORAGE:
  ENABLED: true                           # 是否启用云存储上传
  PROVIDER: "gcs"                         # 云存储提供商: "gcs" 或 "s3"
  BUCKET_NAME: "my-bucket"                # Bucket 名称
  SIGNED_URL_EXPIRE_HOURS: 3              # Signed URL 过期时间(小时)
  PATH_PREFIX: "torrents"                 # 对象路径前缀
  # GCS 专用
  CREDENTIALS_FILE: "./gcs-sa.json"       # GCS Service Account JSON 文件
  # S3 专用
  REGION: "us-east-1"                     # AWS 区域
  ACCESS_KEY_ID: "AKIA..."                # AWS Access Key
  SECRET_ACCESS_KEY: "..."                # AWS Secret Key
  ENDPOINT: ""                            # 自定义端点(MinIO 等)
```

### 访问配置

```go
import "github.com/Done-0/gin-scaffold/internal/configs"

config := configs.GetConfig()
port := config.APP.APP_PORT
```

### 动态修改配置

```go
configs.UpdateField("APP.APP_PORT", "9090")
```

---

## 错误处理系统

### 错误码设计

错误码使用 **errorx** 框架,见 `internal/utils/errorx/`。

#### 错误码范围

| 模块       | 错误码范围       | 当前最大值 |
|----------|--------------|--------|
| 系统错误   | 10000-19999  | 10008  |
| 用户模块   | 20000-20099  | 20006  |
| Torrent 模块 | 20100-20199 | 20110 |
| 转码模块   | 20200-20299  | 20203  |
| 管理员模块 | 20300-20399  | 20301  |
| 云存储模块 | 20400-20499  | 20404  |

详见 `internal/types/errno/` 目录下的错误码定义。

### 注册错误码

```go
// internal/types/errno/system.go
func init() {
    errorx.Register(10001, "database connection failed")
    errorx.Register(10002, "redis connection failed")
}
```

### 创建错误

```go
// 简单错误
return errorx.New(20001) // "user not found"

// 带参数的错误
return errorx.New(20002, errorx.KV("field", "email")) // "invalid {field}"
```

### 错误响应格式

```json
{
  "code": 20001,
  "message": "user not found",
  "data": null
}
```

### i18n 支持

错误信息自动根据请求头 `Accept-Language` 返回对应语言(支持 `zh-CN`, `en-US`)。

---

## 编码规范

项目严格遵循 `docs/coding-standards.en-US.md` 中的编码规范。

### 命名规范

| 类型            | 规范                                    | 示例                          |
|---------------|---------------------------------------|------------------------------|
| 包名           | 全小写,与目录名一致                      | `package torrent`            |
| 文件名         | 小写 + 下划线                           | `torrent_service_impl.go`    |
| 结构体/接口    | CamelCase(首字母大写导出,小写私有)      | `TorrentManager`             |
| 变量           | camelCase                              | `userName`, `userID`         |
| 常量           | CamelCase + 类别前缀                   | `MethodGET`, `StatusPending` |
| 布尔变量       | 必须以 `Has/Is/Can/Allow` 开头         | `isActive`, `hasPermission`  |
| 特殊缩写       | 保持原大小写(非首词时小写)              | `APIClient`, `apiKey`        |

### 注释规范

**必须使用英文注释**。

#### 包注释
```go
// Package torrent provides BT download management.
// Creator: Done-0
// Created: 2025-01-20
```

#### 结构体注释
```go
// Torrent, represents a BT download task.
type Torrent struct {
    InfoHash string // Torrent info hash
    Name     string // Torrent name
}
```

#### 函数注释 (可选但推荐)
```go
// StartDownload starts a BT download task.
// Parameters:
//     infoHash: Torrent info hash
//     selectedFiles: File indices to download
//
// Returns:
//     error: Error if any
func StartDownload(infoHash string, selectedFiles []int) error {
    // ...
}
```

### 代码风格

#### 导入分组
```go
import (
    "fmt"
    "net/http"

    "github.com/gin-gonic/gin"

    "github.com/Done-0/gin-scaffold/internal/logger"
    "github.com/Done-0/gin-scaffold/pkg/vo"
)
```

#### 错误处理
```go
// ✅ 正确: 早返回
if err != nil {
    return err
}
// 正常流程

// ❌ 错误: 使用 else
if err != nil {
    return err
} else {
    // 正常流程
}
```

#### 测试文件
- 文件名: `*_test.go`
- 测试函数: `TestXxx`

---

## 常见开发任务

### 添加新的 API 接口

1. **定义错误码** (`internal/types/errno/`)
```go
// internal/types/errno/foo.go
func init() {
    errorx.Register(20400, "foo not found")
}
```

2. **创建数据模型** (`internal/model/foo/`)
```go
// internal/model/foo/foo.go
type Foo struct {
    base.Base
    Name string `gorm:"not null"`
}
```

3. **注册模型迁移** (`internal/model/get_all_models.go`)
```go
func GetAllModels() []interface{} {
    return []interface{}{
        &user.User{},
        &foo.Foo{}, // 新增
    }
}
```

4. **创建 DTO** (`pkg/serve/controller/dto/foo_dto.go`)
```go
type CreateFooDTO struct {
    Name string `json:"name" binding:"required,min=1,max=100"`
}
```

5. **创建 VO** (`pkg/vo/foo_vo.go`)
```go
type FooVO struct {
    ID   int64  `json:"id"`
    Name string `json:"name"`
}
```

6. **创建 Service 接口和实现**
```go
// pkg/serve/service/foo_service.go
type FooService interface {
    CreateFoo(dto dto.CreateFooDTO) (*vo.FooVO, error)
}

// pkg/serve/service/impl/foo_service_impl.go
type FooServiceImpl struct {
    db db.DatabaseManager
}
```

7. **创建 Controller** (`pkg/serve/controller/foo_controller.go`)
```go
type FooController struct {
    fooService service.FooService
}

func (c *FooController) Create(ctx *gin.Context) {
    var dto dto.CreateFooDTO
    if err := ctx.ShouldBindJSON(&dto); err != nil {
        ctx.JSON(http.StatusBadRequest, vo_utils.Error(10003))
        return
    }
    result, err := c.fooService.CreateFoo(dto)
    // ...
}
```

8. **注册路由** (`pkg/router/routes/foo_routes.go`)
```go
func RegisterFooRoutes(container *wire.Container, v1 *gin.RouterGroup) {
    fooGroup := v1.Group("/foo")
    {
        fooGroup.POST("", container.FooController.Create)
    }
}
```

9. **添加 Wire Provider** (`pkg/wire/providers.go`)
```go
func ProvideFooService(db db.DatabaseManager) service.FooService {
    return impl.NewFooServiceImpl(db)
}

func ProvideFooController(fooService service.FooService) controller.FooController {
    return controller.NewFooController(fooService)
}

var AllProviders = wire.NewSet(
    // ...
    ProvideFooService,
    ProvideFooController,
)
```

10. **更新 Container** (`pkg/wire/wire.go`)
```go
type Container struct {
    // ...
    FooController  controller.FooController
    FooService     service.FooService
}
```

11. **重新生成 Wire 代码**
```bash
cd pkg/wire && wire
```

### 使用 Redis 缓存

```go
import "github.com/Done-0/gin-scaffold/internal/cache"

// 直接操作 Redis
rdb := container.RedisManager.GetClient()
rdb.Set(ctx, "key", "value", time.Hour)

// 使用缓存抽象层
cacheService := cache.NewCache(container.RedisManager, container.LoggerManager)
value, err := cacheService.Get(ctx, "key")

// Cache-Aside 模式
value, err := cacheService.GetOrSet(ctx, "key", time.Hour, func() (string, error) {
    // 缓存未命中,从数据库加载
    return loadFromDB()
})
```

### 添加消息队列消费

```go
// 实现 queue.Handler 接口
type MyHandler struct{}

func (h *MyHandler) Handle(ctx context.Context, msg *queue.Message) error {
    // 处理消息
    fmt.Println("Received:", string(msg.Value))
    return nil
}

// 创建消费者
consumer, err := queue.NewConsumer(config, &MyHandler{})
if err != nil {
    log.Fatal(err)
}

// 订阅 topic
consumer.Subscribe([]string{"my-topic"})
```

### 发送消息到队列

```go
// 获取生产者
producer, err := queue.NewProducer(config)
if err != nil {
    log.Fatal(err)
}

// 发送消息
err = producer.Send(ctx, "my-topic", nil, []byte("hello"))
```

---

## Docker 部署

### docker-compose.yml

```yaml
services:
  # RabbitMQ 消息队列 (可选，仅当 QUEUE.TYPE=rabbitmq 时需要)
  rabbitmq:
    image: rabbitmq:3-management-alpine
    ports:
      - "5672:5672"
      - "15672:15672"

  mysql:
    image: mysql:8.0
    ports:
      - "3306:3306"
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: magnet

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  app:
    build: .
    ports:
      - "8080:8080"       # HTTP 端口
      - "42069:42069"     # BT P2P 端口
    volumes:
      - ./download:/app/download
      - ./configs:/app/configs
      - ./.logs:/app/.logs
    depends_on:
      - mysql
      - redis
      # - rabbitmq  # 如果使用 rabbitmq，取消注释
```

### 部署步骤

```bash
# 启动所有服务
docker-compose up -d

# 查看日志
docker-compose logs -f app

# 停止服务
docker-compose down
```

---

## 项目亮点

1. **完整的企业级架构**: 分层清晰,高内聚低耦合
2. **视频在线播放**: 自动转码为浏览器兼容格式(类 Netflix 体验)
3. **异步转码系统**: 消息队列解耦,支持 GoChannel(开发) 和 RabbitMQ(生产)
4. **云存储集成**: 支持 GCS/S3,自动上传转码文件,Signed URL 安全访问
5. **依赖注入**: Wire 编译时注入,避免反射开销
6. **缓存策略**: Cache-Aside 模式 + 异步写入
7. **流式传输**: 支持 HTTP Range 请求,实现视频拖拽和边下边播
8. **权限系统**: 公开/私有种子分离,JWT 保护
9. **雪花 ID**: 分布式 ID 生成,支持高并发
10. **优雅关闭**: 15 秒超时,释放所有资源
11. **完善文档**: CLAUDE.md、coding-standards、API 文档

---

## 开发建议

### 原则

- **奥卡姆剃刀**: 如无必要,勿增实体
- **高内聚,低耦合**: 模块职责单一,依赖关系清晰
- **早返回**: 错误处理使用早返回,避免嵌套
- **测试覆盖**: 重要功能必须有单元测试

### 反模式(避免)

- ❌ 过度设计: 不要为假设的未来需求设计
- ❌ 重复代码: 三次以上重复考虑抽象
- ❌ 忽略错误: 所有错误必须处理,不允许使用 `_` 忽略
- ❌ 裸返回: 函数返回值必须明确命名

---

## 相关文档

- [编码规范](docs/coding-standards.en-US.md)
- [错误码说明](internal/types/errno/README.md)
- [Wire 官方文档](https://github.com/google/wire)
- [Gin 框架文档](https://gin-gonic.com/docs/)
- [GORM 文档](https://gorm.io/docs/)

---

**最后更新**: 2026-02-14
**维护者**: Done-0
