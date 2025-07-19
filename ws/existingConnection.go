package ws

import (
	"sync"

	"github.com/gorilla/websocket"
	"github.com/komari-monitor/komari/common"
)

var (
	connectedClients = make(map[string]*SafeConn)
	ConnectedUsers   = []*websocket.Conn{}
	latestReport     = make(map[string]*common.Report)
	mu               = sync.RWMutex{}
)

func GetConnectedClients() map[string]*SafeConn {
	mu.RLock()
	defer mu.RUnlock()
	clientsCopy := make(map[string]*SafeConn)
	for k, v := range connectedClients {
		clientsCopy[k] = v
	}
	return clientsCopy
}

func SetConnectedClients(uuid string, conn *SafeConn) {
	mu.Lock()
	defer mu.Unlock()
	connectedClients[uuid] = conn
}
func DeleteConnectedClients(uuid string) {
	mu.Lock()
	defer mu.Unlock()
	if conn, exists := connectedClients[uuid]; exists {
		conn.Close()
	}
	delete(connectedClients, uuid)
}
func GetLatestReport() map[string]*common.Report {
	mu.RLock()
	defer mu.RUnlock()
	reportCopy := make(map[string]*common.Report)
	for k, v := range latestReport {
		reportCopy[k] = v
	}
	return reportCopy
}
func SetLatestReport(uuid string, report *common.Report) {
	mu.Lock()
	defer mu.Unlock()
	latestReport[uuid] = report
}
func DeleteLatestReport(uuid string) {
	mu.Lock()
	defer mu.Unlock()
	delete(latestReport, uuid)
}
