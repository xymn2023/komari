
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


一键命令

```
wget https://raw.githubusercontent.com/xymn2023/komari/main/install.sh && chmod +x install.sh && sudo ./install.sh
```


### 依赖
- Docker（快速部署）


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

## 显示访问者IP小工具
食用方法：登入后台--设置--站点--自定义 Body--在页面底部添加自定义内容--粘贴即可
主页刷新显示效果


```
<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>IP 地址查询</title>
  <style>
    /* 基础浮动面板样式（展开态） */
    #ip-query-widget {
      position: fixed;
      bottom: 60px;
      left: 50%;
      transform: translateX(-50%);
      background-color: rgba(255, 255, 255, 0.95);
      border: 1px solid #e0e6ed;
      border-radius: 12px;
      box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
      padding: 15px 25px;
      box-sizing: border-box;
      z-index: 1000;
      font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
      color: #333;
      display: flex;
      align-items: center;
      justify-content: center;
      cursor: pointer;
      transition: 
        bottom 0.3s ease,
        left 0.3s ease,
        right 0.3s ease,
        width 0.3s ease,
        height 0.3s ease,
        padding 0.3s ease,
        transform 0.3s ease;
    }

    /* 文本容器，展开态下留出箭头空间 */
    #result-container {
      padding-right: 40px;
      font-size: 18px;
      line-height: 1.8;
      display: inline-block;
    }

    /* 渐变文字 */
    #result {
      white-space: nowrap;
      font-weight: 600;
      background: linear-gradient(90deg, #6a82fb, #fc5c7d);
      -webkit-background-clip: text;
      background-clip: text;
      -webkit-text-fill-color: transparent;
      color: transparent;
      animation: textGradientShift 4s ease-in-out infinite alternate;
      background-size: 200% auto;
    }
    @keyframes textGradientShift {
      0% { background-position: 0% 50%; }
      100% { background-position: 100% 50%; }
    }

    /* 箭头，同文字渐变色 */
    #toggle-icon {
      position: absolute;
      right: 12px;
      top: 50%;
      transform: translateY(-50%);
      font-size: 16px;
      font-weight: bold;
      background: linear-gradient(90deg, #6a82fb, #fc5c7d);
      -webkit-background-clip: text;
      background-clip: text;
      color: transparent;
      pointer-events: none;
      user-select: none;
      background-size: 200% auto;
      animation: textGradientShift 4s ease-in-out infinite alternate;
    }

    /* 折叠态：圆形白底，和放大镜水平对齐 */
    #ip-query-widget.collapsed {
      width: 30px;            /* 与放大镜背景直径一致 */
      height: 30px;
      padding: 0;
      border-radius: 50%;
      background-color: #fff;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
      left: auto;
      right: 32px;            /* 根据放大镜位置微调 */
      bottom: 80px;           /* 上移避开底部按钮 */
      transform: none;
      display: flex;
      align-items: center;
      justify-content: center;
    }

    /* 折叠时隐藏文字 */
    #ip-query-widget.collapsed #result-container {
      display: none;
    }
    /* 折叠态箭头：居中显示 */
    #ip-query-widget.collapsed #toggle-icon {
      position: static;
      transform: none;
      font-size: 14px;
    }

    /* 隐藏页眉／页脚（如有） */
    .header-branding, .footer-info {
      display: none;
    }

    /* 小屏响应 */
    @media (max-width: 768px) {
      #ip-query-widget {
        bottom: 40px;
        padding: 10px 15px;
      }
      #ip-query-widget.collapsed {
        bottom: 60px;
        right: 40px;
      }
      #result-container {
        font-size: 15px;
      }
    }
  </style>
</head>
<body>

  <div id="ip-query-widget">
    <div id="result-container">
      <div id="result">正在查询您的IP信息...</div>
    </div>
    <span id="toggle-icon">»»</span>

    <script>
      function callback(ip, location, asn, org) {
        const loc = location || '未知地区';
        const au  = asn      || '未知ASN';
        document.getElementById('result')
                .textContent = `访问IP: ${ip} | ${loc} | ${au}`;
      }
    </script>
    <script src="https://ping0.cc/geo/jsonp/callback"></script>
  </div>

  <script>
    (function(){
      const widget = document.getElementById('ip-query-widget');
      const icon   = document.getElementById('toggle-icon');
      widget.addEventListener('click', function(e){
        e.stopPropagation();
        this.classList.toggle('collapsed');
        icon.textContent = this.classList.contains('collapsed') ? '««' : '»»';
      });
    })();
  </script>

</body>
</html>
```






































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
