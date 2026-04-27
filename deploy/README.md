# Deployment

CI builds `ghcr.io/jackyinpei/magnet2video:latest` on every push to `main`,
then SSHes into the server and restarts. First-time setup is manual; after
that every push to `main` redeploys.

## 1. GitHub repository secrets

Settings → Secrets and variables → Actions → New repository secret:

| Name              | Value                                                       |
| ----------------- | ----------------------------------------------------------- |
| `SSH_HOST`        | server public IP or domain                                  |
| `SSH_USER`        | login user (must be in `docker` group)                      |
| `SSH_PRIVATE_KEY` | private key (whole `-----BEGIN ...` block) for that user    |
| `SSH_PORT`        | optional, defaults to 22                                    |

Generate the key on your laptop:
```bash
ssh-keygen -t ed25519 -f ~/.ssh/magnet2video_deploy -N ""
ssh-copy-id -i ~/.ssh/magnet2video_deploy.pub user@server
# paste contents of ~/.ssh/magnet2video_deploy into SSH_PRIVATE_KEY
```

## 2. Server first-time setup (Ubuntu)

```bash
# install docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker "$USER"   # log out + back in

# create deploy dir
sudo mkdir -p /opt/magnet2video/.logs /opt/magnet2video/download
sudo chown -R "$USER":"$USER" /opt/magnet2video
cd /opt/magnet2video

# 把仓库根目录的 .env.example 内容粘进 .env，改密码
nano .env
```

仅此一个 `.env` 文件即可 —— 不再有 `.docker.env`，也不用编辑 `configs/config.yml`
（默认值已经烤进镜像）。

## 3. First boot

CD job 会把 `docker-compose.server.yml` 拷过来并启动。手动跑第一次：

```bash
cd /opt/magnet2video
docker compose pull
docker compose up -d
docker compose logs -f app
```

## 4. MySQL 1045 on startup

如果 app 日志里出现类似:

```text
Access denied for user 'root'@'172.x.x.x' (using password: YES)
```

说明 app 容器已经连到了 MySQL，但 MySQL 不允许这个账号从 app 容器的地址登录。
旧 compose 让 app 使用 `root` 连库，MySQL 的 root 账号通常只适合本机维护；
现在的 compose 改为使用专用账号 `DB_USER`/`DB_PASSWORD`。

如果是刚部署、没有需要保留的数据，最简单:

```bash
cd /opt/magnet2video
nano .env
# 确认/新增:
#   MYSQL_ROOT_PASSWORD=<root维护密码>
#   DB_USER=magnet2video
#   DB_PASSWORD=<app数据库密码>
#   DB_NAME=magnet2video

docker compose down -v
docker compose pull
docker compose up -d
docker compose logs -f app
```

`docker compose down -v` 会删除 MySQL/Redis/RabbitMQ 数据卷；已经有数据时不要用。
要保留数据，就进 MySQL 手动创建 app 用户:

```bash
cd /opt/magnet2video
docker exec -it magnet2video-mysql mysql -uroot -p
```

然后在 MySQL 里执行:

```sql
CREATE DATABASE IF NOT EXISTS `magnet2video`;
CREATE USER IF NOT EXISTS 'magnet2video'@'%' IDENTIFIED BY '<DB_PASSWORD里的值>';
GRANT ALL PRIVILEGES ON `magnet2video`.* TO 'magnet2video'@'%';
FLUSH PRIVILEGES;
```

再确认 `/opt/magnet2video/.env` 里:

```dotenv
DB_USER=magnet2video
DB_PASSWORD=<上面同一个密码>
DB_NAME=magnet2video
```

最后重启:

```bash
docker compose up -d
docker compose logs -f app
```

如果 `RABBITMQ_DEFAULT_*` 没在第一次启动前就在 `.env` 里，那这些 env 变量
对已经存在的数据卷无效，需要手动加 vhost / user：

```bash
docker exec magnet2video-rabbitmq rabbitmqctl add_vhost magnet
docker exec magnet2video-rabbitmq rabbitmqctl add_user worker "<RABBITMQ_PASS>"
docker exec magnet2video-rabbitmq rabbitmqctl set_permissions -p magnet worker ".*" ".*" ".*"
docker exec magnet2video-rabbitmq rabbitmqctl set_user_tags worker management
docker exec magnet2video-rabbitmq rabbitmqctl delete_user guest
```

## 5. Worker (your laptop or any machine)

```bash
mkdir -p ~/magnet2video-worker/{.logs,download}
cd ~/magnet2video-worker

# 拷 docker-compose.worker.yml 过来当 docker-compose.yml
# 拷 .env.example 过来当 .env，改这几项:
#   WORKER_ID=home-worker-01
#   RABBITMQ_URL=amqp://worker:<RABBITMQ_PASS>@<SERVER_PUBLIC_IP>:5672/magnet
#   S3_*  和 server 完全一致 (worker 直接上传到 S3)
# server 那一摊 (DB/REDIS/SUPER_ADMIN) 留默认空值即可

docker compose pull
docker compose up -d
docker compose logs -f
```

## 6. Verifying

- `https://<server>:8080` — web UI
- `https://<server>:15672` — RabbitMQ management (login with worker user)
- `docker compose ps` on server — all containers `healthy`
- worker logs should show `[worker mode] starting as ...` and a periodic
  heartbeat being published
