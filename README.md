
# Komari 
![Badge](https://hitscounter.dev/api/hit?url=https%3A%2F%2Fgithub.com%2Fkomari-monitor%2Fkomari&label=&icon=github&color=%23a370f7&message=&style=flat&tz=UTC)

![komari](https://socialify.git.ci/komari-monitor/komari/image?description=1&font=Inter&forks=1&issues=1&language=1&logo=https%3A%2F%2Fraw.githubusercontent.com%2Fkomari-monitor%2Fkomari-web%2Fd54ce1288df41ead08aa19f8700186e68028a889%2Fpublic%2Ffavicon.png&name=1&owner=1&pattern=Plus&pulls=1&stargazers=1&theme=Auto)

Komari 是一款轻量级的自托管服务器监控工具，旨在提供简单、高效的服务器性能监控解决方案。它支持通过 Web 界面查看服务器状态，并通过轻量级 Agent 收集数据。

[文档](https://komari-monitor.github.io/komari-document/)

本仓库存在的意义：提供一个更全面的完整的前后端集合体，后续更新内容请前往原项目地址[Komari](https://github.com/komari-monitor/komari)进行反馈。

## 特性
- **轻量高效**：低资源占用，适合各种规模的服务器。
- **自托管**：完全掌控数据隐私，部署简单。
- **Web 界面**：直观的监控仪表盘，易于使用。

## 快速开始

### 依赖
- Docker（快速部署）
- 或者 Go 1.18+ 和 Node.js 20+（手工构建）

### Docker 部署
1. 创建`docker-compose.yml`：
   ```bash
   touch docker-compose.yml
   ```
2. 复制以下内容并保存：
   ```bash
   version: '3.8'
   services:
     komari:
       image: smhw3565/komari:latest
       container_name: komari
       ports:
         - "25774:25774"
       volumes:
         - ./data:/app/data
       environment:
         - GIN_MODE=release
         - KOMARI_DB_TYPE=sqlite
         - KOMARI_DB_FILE=/app/data/komari.db
         # 可选：自定义初始管理员账号密码
         - ADMIN_USERNAME=admin
         - ADMIN_PASSWORD=yourpassword
       restart: unless-stopped 
   ```
3. 一键运行：
   ```bash
   docker compose up -d
   ```
4. 在浏览器中访问 `http://<your_server_ip>:25774`。

> [!NOTE]
> 你也可以通过修改`docker-compose.yml配置 ``ADMIN_USERNAME` 和 `ADMIN_PASSWORD` 自定义初始用户名和密码。

原有 Dockerfile 用的是 alpine:3.21，但由于 go-sqlite3 依赖 CGO 和动态库，Alpine 镜像不适合直接运行 go-sqlite3 相关的 Go 程序。

你需要切换到 debian 或 ubuntu 这类有完整 libc 和 sqlite3 动态库的镜像，并且在 Linux 下用 CGO_ENABLED=1 编译。所以打包了最为全面完整的底层环境进行打包。

## 前端开发指南
[Komari 主题开发指南 | Komari](https://komari-monitor.github.io/komari-document/dev/theme.html)

## 客户端 Agent 开发指南
[Komari Agent 信息上报与事件处理文档](https://komari-monitor.github.io/komari-document/dev/agent.html)

## 贡献
欢迎提交 Issue 或 Pull Request！

## 鸣谢
 - [DreamCloud - 极高性价比解锁直连亚太高防](https://as211392.com/)
 - 感谢我自己能这么闲
 - 提交PR、制作主题的各位开发者

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=komari-monitor/komari&type=Date)](https://www.star-history.com/#komari-monitor/komari&Date)
