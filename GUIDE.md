systemctl start docker

docker ps -a

docker start 名字 例如 my-redis

netstat -tulpn|grep 3000


nginx -s reload 不顶用的话用下面的
systemctl stop nginx
systemctl start nginx

### 如何ssh 远程登录
1. 在本地电脑ssh-keygen -t rsa
2. 打开id_rsa.pub 把里面的内容复制到剪切板
3. 打开remote client 的~/.ssh/authorized_keys 如果没有就新建一个这个文件
4. 把剪切板里的内容复制到authorized_keys 里然后重启ssh 服务就可以了 centos 使用sudo systemctl reload sshd 其他服务器使用（sudo service ssh reload）
