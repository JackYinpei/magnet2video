# DDD Migration Plan for magnet-video

## Context

当前项目采用经典 MVC 分层架构（Controller → Service → DB），存在以下核心问题：
- **贫血模型**：`Torrent`/`TorrentFile`/`TranscodeJob` 纯数据容器，无行为
- **无 Repository 抽象**：Service 层直接调用 `dbManager.DB()`（60+ 处 GORM 调用）
- **业务逻辑泄漏**：`TorrentController`（1160行）包含云上传重试、状态聚合等业务逻辑；`TranscodeHandler`（575行）和 `CloudUploadHandler`（288行）包含状态计算和重试策略
- **重复逻辑**：云路径构建、内容类型判断、状态聚合计算在 3+ 处重复
- **不可单测**：Service 直接依赖 GORM，无法不用数据库测试业务逻辑

迁移目标：引入 DDD 分层，领域层纯 Go 可独立单测，通过 Repository 解耦持久化，保持所有 API 行为不变。

---

## Target Directory Structure

```
domain/                           # 领域层（纯 Go，无框架依赖）
├── torrent/                      # Torrent 限界上下文
│   ├── torrent.go               # 聚合根 + 行为方法
│   ├── torrent_file.go          # 实体 + 行为方法
│   ├── value_objects.go         # DownloadStatus, Visibility, TranscodeStatus, CloudUploadStatus
│   ├── repository.go            # TorrentRepository 接口
│   ├── errors.go                # 领域错误
│   ├── torrent_test.go          # 聚合根单测
│   └── torrent_file_test.go     # 实体单测
├── transcode/                    # 转码限界上下文
│   ├── job.go                   # 聚合根 + 行为方法
│   ├── policy.go                # 领域服务：转码策略
│   ├── repository.go            # JobRepository 接口
│   ├── errors.go
│   ├── job_test.go
│   └── policy_test.go
├── cloud/                        # 云上传限界上下文
│   ├── upload.go                # UploadSpec 值对象
│   ├── policy.go                # 领域服务：上传策略（重试、路径构建、ContentType）
│   ├── errors.go
│   └── policy_test.go
└── user/                         # 用户限界上下文
    ├── user.go                  # 聚合根
    ├── repository.go            # UserRepository 接口
    └── user_test.go

internal/
├── repository/                   # 新增：Repository GORM 实现
│   ├── mapper/                  # 领域模型 <-> GORM 模型 映射器
│   │   ├── torrent_mapper.go
│   │   ├── transcode_mapper.go
│   │   └── user_mapper.go
│   ├── torrent_repo.go
│   ├── transcode_repo.go
│   ├── user_repo.go
│   ├── torrent_repo_test.go     # SQLite 集成测试
│   ├── transcode_repo_test.go
│   └── user_repo_test.go
├── model/                        # 保持不变（GORM 持久化模型）
├── transcode/handler/            # 瘦化：仅反序列化消息 → 委托 AppService
├── cloud/handler/                # 瘦化：仅反序列化消息 → 委托 AppService
└── (其余 internal/ 不变)

pkg/
├── app/                          # 新增：应用服务层
│   ├── torrent_app_service.go   # 编排 Torrent 领域 + 基础设施
│   ├── transcode_app_service.go
│   ├── cloud_app_service.go
│   ├── user_app_service.go
│   └── admin_app_service.go
├── serve/controller/             # 瘦化：纯 HTTP 处理，委托 AppService
├── vo/                          # 保持不变
└── wire/                        # 更新 Provider
```

---

## Phase 0: Value Objects & Domain Errors (~200 lines)

**目标**：建立领域词汇表，零风险，不改现有代码。

### 创建文件

**`domain/torrent/value_objects.go`**
- `DownloadStatus` (int) + 常量 Pending/Downloading/Completed/Failed/Paused
- `Visibility` (int) + 常量 Private/Internal/Public
- `TranscodeStatus` (int) + 常量 None/Pending/Processing/Completed/Failed
- `CloudUploadStatus` (int) + 常量 None/Pending/Uploading/Completed/Failed
- 每个类型加 `String()` 方法和 `IsValid()` 校验

**`domain/torrent/errors.go`**
- `ErrInvalidStateTransition`
- `ErrNotOwner`
- `ErrCannotDeleteWhileDownloading`
- `ErrCannotDeleteWhileUploading`

**`domain/transcode/errors.go`**
- `ErrJobAlreadyRunning`
- `ErrJobCannotRetry`

**`domain/cloud/errors.go`**
- `ErrMaxRetriesExceeded`

**`domain/user/errors.go`**
- `ErrCannotDeleteSuperAdmin`
- `ErrCannotDeleteSelf`

### 测试
每个值对象的 `IsValid()` 和 `String()` 测试。

---

## Phase 1: Rich Domain Entities (~1200 lines)

**目标**：创建带行为的领域模型，封装业务规则。与现有代码并行存在。

### 创建文件

**`domain/torrent/torrent.go`** (~200 lines) — 聚合根
- 字段：与 `internal/model/torrent/torrent.go` 对应，但使用值对象类型
- `NewTorrent(infoHash, name string, totalSize int64, creatorID int64) *Torrent` — 工厂方法
- `StartDownload(files []TorrentFile, downloadPath string) error` — 校验状态 → 设置 Downloading
- `MarkCompleted() error` — 校验 Downloading → 设置 Completed
- `MarkFailed() error`
- `Pause() error` — 校验 Downloading → Paused
- `Resume() error` — 校验 Paused → Downloading
- `SetVisibility(v Visibility) error`
- `SetPoster(path string)`
- `MarkLocalFilesDeleted() error` — 校验无进行中下载/上传
- `IsOwnedBy(userID int64) bool`
- `IsVisibleTo(userID int64, isAuthenticated bool) bool`
- `GetTranscodableFiles() []*TorrentFile` — 返回 IsSelected && IsOriginal && TranscodeStatus==None 的视频文件
- `GetCloudUploadableFiles() []*TorrentFile`
- `RecalculateTranscodeSummary()` — 从 Files 重算聚合计数器
- `RecalculateCloudSummary()`
- `UpdateProgress(progress float64, status DownloadStatus)`
- `FileByIndex(index int) *TorrentFile`

**`domain/torrent/torrent_file.go`** (~150 lines) — 实体
- `MarkTranscodePending()` / `MarkTranscoding()` / `MarkTranscodeCompleted(path string)` / `MarkTranscodeFailed(err string)`
- `MarkCloudPending()` / `MarkCloudUploading()` / `MarkCloudCompleted(cloudPath string)` / `MarkCloudFailed(err string)`
- `ResetTranscodeStatus()` — 用于 admin 重置
- `ResetCloudStatus()` — 用于重试
- `IsOriginal() bool` — Source == "" || Source == "original"
- `IsVideo() bool` — Type == "video"
- `CanRetryCloudUpload() bool` — CloudUploadStatus == Failed

**`domain/transcode/job.go`** (~100 lines) — 聚合根
- `NewJob(torrentID int64, infoHash string, fileIndex int, inputPath, outputPath string, transcodeType string, creatorID int64) *Job`
- `Start() error` — Pending → Processing，设置 StartedAt
- `UpdateProgress(progress int)`
- `Complete(outputPath string) error` — Processing → Completed，设置 CompletedAt
- `Fail(errMsg string) error` — → Failed
- `CanRetry() bool` — Status == Failed

**`domain/transcode/policy.go`** (~60 lines) — 领域服务
- `IsVideoFile(path string) bool` — 抽取自 `internal/model/torrent/file_type.go`
- `DetermineTranscodeType(codec, containerFormat string) string` — "remux" 或 "transcode"

**`domain/cloud/upload.go`** (~40 lines) — 值对象
- `UploadSpec` struct

**`domain/cloud/policy.go`** (~80 lines) — 领域服务
- `NewUploadPolicy(maxRetries int, pathPrefix string) *UploadPolicy`
- `BuildCloudPath(infoHash, fileName string) string`
- `DetermineContentType(filePath string) string`
- `ShouldRetry(retryCount int) bool`
- `BackoffDuration(retryCount int) time.Duration`
- `NewUploadSpec(torrentID int64, ...) UploadSpec`

**`domain/user/user.go`** (~50 lines) — 聚合根

### Mapper 文件

**`internal/repository/mapper/torrent_mapper.go`**
**`internal/repository/mapper/transcode_mapper.go`**
**`internal/repository/mapper/user_mapper.go`**

### 测试 — 100% 纯单测，无外部依赖

---

## Phase 2: Repository Interfaces & Implementations (~900 lines)

**目标**：在领域层定义 Repository 接口，在 `internal/repository/` 用 GORM 实现。

### Repository 接口

**`domain/torrent/repository.go`**
**`domain/transcode/repository.go`**
**`domain/user/repository.go`**

### GORM 实现

**`internal/repository/torrent_repo.go`**
**`internal/repository/transcode_repo.go`**
**`internal/repository/user_repo.go`**

### 测试 — SQLite 集成测试

---

## Phase 3: Application Services (~1500 lines)

**目标**：创建应用服务编排领域对象和基础设施。与旧 Service 并行存在。

**`pkg/app/torrent_app_service.go`**
**`pkg/app/transcode_app_service.go`**
**`pkg/app/cloud_app_service.go`**
**`pkg/app/user_app_service.go`**
**`pkg/app/admin_app_service.go`**

---

## Phase 4: Wiring & Controller Migration

**目标**：更新 Wire 容器，Controller 委托 AppService，Handler 瘦化。

---

## Phase 5: Cleanup

- 删除旧 Service 实现和接口文件
- 更新 Wire providers

---

## 实施顺序与依赖关系

```
Phase 0 (值对象) → Phase 1 (领域实体) → Phase 2 (Repository) → Phase 3 (Application Services) → Phase 4 (Wire + 迁移) → Phase 5 (清理)
```

每个 Phase 独立可编译、可测试。Phase 0-3 不改动任何现有文件，零风险。

---

## 关键设计决策

1. **领域层不依赖 `gin.Context`**：Application Service 接收 `context.Context` + 显式 `userID int64` 参数
2. **保持 GORM 模型不变**：`internal/model/` 作为持久化模型保留，通过 Mapper 与领域模型互转
3. **不引入领域事件（本次）**：用直接方法调用代替事件总线，降低复杂度
4. **值对象用 int 类型**：与数据库 schema 保持兼容，通过方法封装校验逻辑
5. **CloudUploadPolicy 作为领域服务**：路径构建和重试策略是领域知识，不属于基础设施
