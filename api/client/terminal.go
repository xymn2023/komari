package client

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/komari-monitor/komari/api"
)

func EstablishConnection(c *gin.Context) {
	session_id := c.Query("id")
	session, exists := api.TerminalSessions[session_id]
	if !exists || session == nil || session.Browser == nil {
		c.JSON(404, gin.H{"status": "error", "error": "Session not found"})
		return
	}
	// Upgrade the connection to WebSocket
	if !websocket.IsWebSocketUpgrade(c.Request) {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "error": "Require WebSocket upgrade"})
		return
	}
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // TODO: 检查源站
		},
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		api.TerminalSessionsMutex.Lock()
		if session.Browser != nil {
			session.Browser.Close()
		}
		delete(api.TerminalSessions, session_id)
		api.TerminalSessionsMutex.Unlock()
		return
	}
	session.Agent = conn
	conn.SetCloseHandler(func(code int, text string) error {
		delete(api.TerminalSessions, session_id)
		// 通知 Browser 关闭终端连接
		if session.Browser != nil {
			session.Browser.Close()
		}
		return nil
	})
	go api.ForwardTerminal(session_id)
}
