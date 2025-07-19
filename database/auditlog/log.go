package auditlog

import (
	"log"
	"time"

	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
)

func Log(ip, uuid, message, msgType string) {
	now := time.Now()
	db := dbcore.GetDBInstance()
	logEntry := &models.Log{
		IP:      ip,
		UUID:    uuid,
		Message: message,
		MsgType: msgType,
		Time:    models.FromTime(now),
	}
	db.Create(logEntry)
}

func EventLog(eventType, message string) {
	Log("", "", message, eventType)
}

// Delete logs older than 30 days
func RemoveOldLogs() {
	db := dbcore.GetDBInstance()
	threshold := time.Now().AddDate(0, 0, -30)
	if err := db.Where("time < ?", threshold).Delete(&models.Log{}).Error; err != nil {
		log.Println("Failed to remove old logs:", err)
	}
}
