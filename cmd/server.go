package cmd

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/komari-monitor/komari/api"
	"github.com/komari-monitor/komari/api/admin"
	"github.com/komari-monitor/komari/api/admin/clipboard"
	log_api "github.com/komari-monitor/komari/api/admin/log"
	"github.com/komari-monitor/komari/api/admin/notification"
	"github.com/komari-monitor/komari/api/admin/test"
	"github.com/komari-monitor/komari/api/admin/update"
	"github.com/komari-monitor/komari/api/client"
	"github.com/komari-monitor/komari/api/record"
	"github.com/komari-monitor/komari/cmd/flags"
	"github.com/komari-monitor/komari/database/accounts"
	"github.com/komari-monitor/komari/database/auditlog"
	"github.com/komari-monitor/komari/database/config"
	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	d_notification "github.com/komari-monitor/komari/database/notification"
	"github.com/komari-monitor/komari/database/records"
	"github.com/komari-monitor/komari/database/tasks"
	"github.com/komari-monitor/komari/public"
	"github.com/komari-monitor/komari/utils"
	"github.com/komari-monitor/komari/utils/cloudflared"
	"github.com/komari-monitor/komari/utils/geoip"
	"github.com/komari-monitor/komari/utils/messageSender"
	"github.com/komari-monitor/komari/utils/notifier"
	"github.com/komari-monitor/komari/ws"
	"github.com/spf13/cobra"
)

var (
	DynamicCorsEnabled bool = false
)

var ServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the server",
	Long:  `Start the server`,
	Run: func(cmd *cobra.Command, args []string) {
		RunServer()
	},
}

func init() {
	// 从环境变量获取监听地址
	listenAddr := GetEnv("KOMARI_LISTEN", "0.0.0.0:25774")
	ServerCmd.PersistentFlags().StringVarP(&flags.Listen, "listen", "l", listenAddr, "监听地址 [env: KOMARI_LISTEN]")
	RootCmd.AddCommand(ServerCmd)
}

func RunServer() {
	// #region 初始化
	if err := os.MkdirAll("./data", os.ModePerm); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}
	// 创建主题目录
	if err := os.MkdirAll("./data/theme", os.ModePerm); err != nil {
		log.Fatalf("Failed to create theme directory: %v", err)
	}
	InitDatabase()
	if utils.VersionHash != "unknown" {
		gin.SetMode(gin.ReleaseMode)
	}
	go geoip.InitGeoIp()
	go DoScheduledWork()
	go messageSender.Initialize()

	if strings.ToLower(GetEnv("KOMARI_ENABLE_CLOUDFLARED", "false")) == "true" {
		err := cloudflared.RunCloudflared() // 阻塞，确保cloudflared跑起来
		if err != nil {
			log.Fatalf("Failed to run cloudflared: %v", err)
		}
	}

	r := gin.Default()

	// 动态 CORS 中间件
	conf, err := config.Get()
	if err != nil {
		log.Fatal(err)
	}
	DynamicCorsEnabled = conf.AllowCors
	config.Subscribe(func(event config.ConfigEvent) {
		DynamicCorsEnabled = event.New.AllowCors
		if event.New.GeoIpProvider != event.Old.GeoIpProvider {
			go geoip.InitGeoIp()
		}
		if event.New.NotificationMethod != event.Old.NotificationMethod {
			go messageSender.Initialize()
		}
	})
	r.Use(func(c *gin.Context) {
		if DynamicCorsEnabled {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Length, Content-Type, Authorization, Accept, X-CSRF-Token, X-Requested-With, Set-Cookie")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Authorization, Set-Cookie")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "43200") // 12 hours
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
		}
		c.Next()
	})

	r.Use(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	})

	r.Any("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	// #region 公开路由
	r.POST("/api/login", api.Login)
	r.GET("/api/me", api.GetMe)
	r.GET("/api/clients", ws.GetClients)
	r.GET("/api/nodes", api.GetNodesInformation)
	r.GET("/api/public", api.GetPublicSettings)
	r.GET("/api/oauth", api.OAuth)
	r.GET("/api/oauth_callback", api.OAuthCallback)
	r.GET("/api/logout", api.Logout)
	r.GET("/api/version", api.GetVersion)
	r.GET("/api/recent/:uuid", api.GetClientRecentRecords)

	r.GET("/api/records/load", record.GetRecordsByUUID)
	r.GET("/api/records/ping", record.GetPingRecords)
	// #region Agent
	tokenAuthrized := r.Group("/api/clients", api.TokenAuthMiddleware())
	{
		tokenAuthrized.GET("/report", client.WebSocketReport) // websocket
		tokenAuthrized.POST("/uploadBasicInfo", client.UploadBasicInfo)
		tokenAuthrized.POST("/report", client.UploadReport)
		tokenAuthrized.GET("/terminal", client.EstablishConnection)
		tokenAuthrized.POST("/task/result", client.TaskResult)
	}
	// #region 管理员
	adminAuthrized := r.Group("/api/admin", api.AdminAuthMiddleware())
	{
		adminAuthrized.GET("/download/backup", admin.DownloadBackup)
		// test
		testGroup := adminAuthrized.Group("/test")
		{
			testGroup.GET("/geoip", test.TestGeoIp)
			testGroup.POST("/sendMessage", test.TestSendMessage)
		}
		// update
		updateGroup := adminAuthrized.Group("/update")
		{
			updateGroup.POST("/mmdb", update.UpdateMmdbGeoIP)
			updateGroup.POST("/user", update.UpdateUser)
			updateGroup.PUT("/favicon", update.UploadFavicon)
			updateGroup.POST("/favicon", update.DeleteFavicon)
		}
		// tasks
		taskGroup := adminAuthrized.Group("/task")
		{
			taskGroup.GET("/all", admin.GetTasks)
			taskGroup.POST("/exec", admin.Exec)
			taskGroup.GET("/:task_id", admin.GetTaskById)
			taskGroup.GET("/:task_id/result", admin.GetTaskResultsByTaskId)
			taskGroup.GET("/:task_id/result/:uuid", admin.GetSpecificTaskResult)
			taskGroup.GET("/client/:uuid", admin.GetTasksByClientId)
		}
		// settings
		adminAuthrized.GET("/settings", admin.GetSettings)
		adminAuthrized.POST("/settings", admin.EditSettings)

		// themes
		themeGroup := adminAuthrized.Group("/theme")
		{
			themeGroup.PUT("/upload", admin.UploadTheme)
			themeGroup.GET("/list", admin.ListThemes)
			themeGroup.POST("/delete", admin.DeleteTheme)
			themeGroup.GET("/set", admin.SetTheme)
		}
		// clients
		clientGroup := adminAuthrized.Group("/client")
		{
			clientGroup.POST("/add", admin.AddClient)
			clientGroup.GET("/list", admin.ListClients)
			clientGroup.GET("/:uuid", admin.GetClient)
			clientGroup.POST("/:uuid/edit", admin.EditClient)
			clientGroup.POST("/:uuid/remove", admin.RemoveClient)
			clientGroup.GET("/:uuid/token", admin.GetClientToken)
			clientGroup.POST("/order", admin.OrderWeight)
			// client terminal
			clientGroup.GET("/:uuid/terminal", api.RequestTerminal)
		}

		// records
		recordGroup := adminAuthrized.Group("/record")
		{
			recordGroup.POST("/clear", admin.ClearRecord)
			recordGroup.POST("/clear/all", admin.ClearAllRecords)
		}
		// oauth2
		oauth2Group := adminAuthrized.Group("/oauth2")
		{
			oauth2Group.GET("/bind", admin.BindingExternalAccount)
			oauth2Group.POST("/unbind", admin.UnbindExternalAccount)
		}
		sessionGroup := adminAuthrized.Group("/session")
		{
			sessionGroup.GET("/get", admin.GetSessions)
			sessionGroup.POST("/remove", admin.DeleteSession)
			sessionGroup.POST("/remove/all", admin.DeleteAllSession)
		}
		two_factorGroup := adminAuthrized.Group("/2fa")
		{
			two_factorGroup.GET("/generate", admin.Generate2FA)
			two_factorGroup.POST("/enable", admin.Enable2FA)
			two_factorGroup.POST("/disable", admin.Disable2FA)
		}
		adminAuthrized.GET("/logs", log_api.GetLogs)

		// clipboard
		clipboardGroup := adminAuthrized.Group("/clipboard")
		{
			clipboardGroup.GET("/:id", clipboard.GetClipboard)
			clipboardGroup.GET("", clipboard.ListClipboard)
			clipboardGroup.POST("", clipboard.CreateClipboard)
			clipboardGroup.POST("/:id", clipboard.UpdateClipboard)
			clipboardGroup.POST("/remove", clipboard.BatchDeleteClipboard)
			clipboardGroup.POST("/:id/remove", clipboard.DeleteClipboard)
		}

		notificationGroup := adminAuthrized.Group("/notification")
		{
			// offline notifications
			notificationGroup.GET("/offline", notification.ListOfflineNotifications)
			notificationGroup.POST("/offline/edit", notification.EditOfflineNotification)
			notificationGroup.POST("/offline/enable", notification.EnableOfflineNotification)
			notificationGroup.POST("/offline/disable", notification.DisableOfflineNotification)
			loadAlertGroup := notificationGroup.Group("/load")
			{
				loadAlertGroup.GET("/", notification.GetAllLoadNotifications)
				loadAlertGroup.POST("/add", notification.AddLoadNotification)
				loadAlertGroup.POST("/delete", notification.DeleteLoadNotification)
				loadAlertGroup.POST("/edit", notification.EditLoadNotification)
			}
		}

		pingTaskGroup := adminAuthrized.Group("/ping")
		{
			pingTaskGroup.GET("/", admin.GetAllPingTasks)
			pingTaskGroup.POST("/add", admin.AddPingTask)
			pingTaskGroup.POST("/delete", admin.DeletePingTask)
			pingTaskGroup.POST("/edit", admin.EditPingTask)

		}

	}

	public.Static(r.Group("/"), func(handlers ...gin.HandlerFunc) {
		r.NoRoute(handlers...)
	})
	// #region 静态文件服务
	public.UpdateIndex(conf)
	config.Subscribe(func(event config.ConfigEvent) {
		public.UpdateIndex(event.New)
	})

	srv := &http.Server{
		Addr:    flags.Listen,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			OnFatal(err)
			log.Fatalf("listen: %s\n", err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	OnShutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

}

func InitDatabase() {
	// // 打印数据库类型和连接信息
	// if flags.DatabaseType == "mysql" {
	// 	log.Printf("使用 MySQL 数据库连接: %s@%s:%s/%s",
	// 		flags.DatabaseUser, flags.DatabaseHost, flags.DatabasePort, flags.DatabaseName)
	// 	log.Printf("环境变量配置: [KOMARI_DB_TYPE=%s] [KOMARI_DB_HOST=%s] [KOMARI_DB_PORT=%s] [KOMARI_DB_USER=%s] [KOMARI_DB_NAME=%s]",
	// 		os.Getenv("KOMARI_DB_TYPE"), os.Getenv("KOMARI_DB_HOST"), os.Getenv("KOMARI_DB_PORT"),
	// 		os.Getenv("KOMARI_DB_USER"), os.Getenv("KOMARI_DB_NAME"))
	// } else {
	// 	log.Printf("使用 SQLite 数据库文件: %s", flags.DatabaseFile)
	// 	log.Printf("环境变量配置: [KOMARI_DB_TYPE=%s] [KOMARI_DB_FILE=%s]",
	// 		os.Getenv("KOMARI_DB_TYPE"), os.Getenv("KOMARI_DB_FILE"))
	// }
	var count int64 = 0
	if dbcore.GetDBInstance().Model(&models.User{}).Count(&count); count == 0 {
		user, passwd, err := accounts.CreateDefaultAdminAccount()
		if err != nil {
			panic(err)
		}
		log.Println("Default admin account created. Username:", user, ", Password:", passwd)
	}
}

// #region 定时任务
func DoScheduledWork() {
	tasks.ReloadPingSchedule()
	d_notification.ReloadLoadNotificationSchedule()
	ticker := time.NewTicker(time.Minute * 30)
	minute := time.NewTicker(60 * time.Second)
	//records.DeleteRecordBefore(time.Now().Add(-time.Hour * 24 * 30))
	records.CompactRecord()
	cfg, _ := config.Get()
	go notifier.CheckExpireScheduledWork()
	for {
		select {
		case <-ticker.C:
			records.DeleteRecordBefore(time.Now().Add(-time.Hour * time.Duration(cfg.RecordPreserveTime)))
			records.CompactRecord()
			tasks.ClearTaskResultsByTimeBefore(time.Now().Add(-time.Hour * time.Duration(cfg.RecordPreserveTime)))
			tasks.DeletePingRecordsBefore(time.Now().Add(-time.Hour * time.Duration(cfg.PingRecordPreserveTime)))
			auditlog.RemoveOldLogs()
		case <-minute.C:
			api.SaveClientReportToDB()
			if !cfg.RecordEnabled {
				records.DeleteAll()
				tasks.DeleteAllPingRecords()
			}
		}
	}

}

func OnShutdown() {
	auditlog.Log("", "", "server is shutting down", "info")
	cloudflared.Kill()
}

func OnFatal(err error) {
	auditlog.Log("", "", "server encountered a fatal error: "+err.Error(), "error")
	cloudflared.Kill()
}
