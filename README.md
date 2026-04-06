# magnet2video

## 简介

magnet2video 是一个企业级 BT 种子下载与视频转码服务，支持磁力链解析、P2P 下载管理、视频自动转码和在线播放。

## 技术栈

- **Go** + **Gin**：Web 框架
- **GORM**：ORM，支持 MySQL / PostgreSQL / SQLite，自动迁移
- **anacrolix/torrent**：BT 下载引擎
- **FFmpeg / FFprobe**：视频转码（Remux / H.264）
- **Redis**：缓存（Cache-Aside 模式）
- **GoChannel / RabbitMQ**：异步消息队列
- **GCS / S3**：云存储，Signed URL 访问
- **Wire**：编译时依赖注入
- **JWT**：认证鉴权
- **Logrus** + **file-rotatelogs**：结构化日志与轮转
- **bwmarrin/snowflake**：雪花 ID

## 主要特性

- 磁力链解析，获取种子元数据和文件列表
- P2P 下载，支持暂停 / 恢复 / 删除
- 视频自动转码为浏览器兼容格式（H.264/MP4）
- HTTP Range 请求，支持视频拖拽和边下边播
- 云存储自动上传，Signed URL 安全播放
- JWT 认证 + 公开 / 私有种子权限控制
- 国际化（zh-CN / en-US）

## 快速开始

1. 复制配置文件并修改：

```bash
cp configs/config.example.yml configs/config.yml
```

2. 启动服务：

```bash
go run main.go
```

3. Docker 部署：

```bash
docker compose up -d
```

## 许可证

Apache License 2.0
