package notifier

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"time"

	"github.com/komari-monitor/komari/database/clients"
	"github.com/komari-monitor/komari/database/dbcore"
	"github.com/komari-monitor/komari/database/models"
	"github.com/komari-monitor/komari/database/records"
	"github.com/komari-monitor/komari/utils/messageSender"
)

// LoadNotificationService 管理定时器和任务
type LoadNotificationService struct {
	mu       sync.Mutex
	tickers  map[int]*time.Ticker
	tasks    map[int][]models.LoadNotification
	stopChan chan struct{}
}

var LoadNotificationManager = &LoadNotificationService{
	tickers:  make(map[int]*time.Ticker),
	tasks:    make(map[int][]models.LoadNotification),
	stopChan: make(chan struct{}),
}

// Reload 重载时间表
func (m *LoadNotificationService) Reload(loadNotifications []models.LoadNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 停止所有现有定时器
	for _, ticker := range m.tickers {
		ticker.Stop()
	}
	m.tickers = make(map[int]*time.Ticker)
	m.tasks = make(map[int][]models.LoadNotification)

	// 按Interval分组任务
	taskGroups := make(map[int][]models.LoadNotification)
	for _, task := range loadNotifications {
		taskGroups[task.Interval] = append(taskGroups[task.Interval], task)
	}

	// 为每个唯一的Interval创建定时器
	for interval, tasks := range taskGroups {
		ticker := time.NewTicker(time.Duration(interval) * time.Minute)
		m.tickers[interval] = ticker
		m.tasks[interval] = tasks

		go func(ticker *time.Ticker, tasks []models.LoadNotification) {
			for {
				select {
				case <-ticker.C:
					for _, task := range tasks {
						go executeLoadNotificationTask(task)
					}
				case <-m.stopChan:
					return
				}
			}
		}(ticker, tasks)
	}

	return nil
}

// executeLoadNotificationTask 执行单个LoadNotificationTask
func executeLoadNotificationTask(task models.LoadNotification) {
	// 检查是否在冷却期内
	if shouldSkipNotification(task) {
		return
	}

	now := time.Now()
	windowStart := now.Add(-time.Duration(task.Interval) * time.Minute)
	overloadClients := make([]string, 0)
	for _, clientUUID := range task.Clients {
		// 获取客户端在时间窗口内的记录
		records, err := getRecordsForClient(clientUUID, windowStart, now)
		if err != nil {
			continue
		}

		// 检查指标是否达到阈值
		if checkMetricThreshold(records, task) {
			overloadClients = append(overloadClients, clientUUID)
		}

	}
	sendLoadNotification(overloadClients, task)
	updateLastNotified(task.Id, now)
}

// shouldSkipNotification 检查是否应该跳过通知（冷却期检查）
func shouldSkipNotification(task models.LoadNotification) bool {
	if task.LastNotified.ToTime().IsZero() {
		return false
	}

	// 计算冷却期（使用 interval 作为冷却期）
	cooldownPeriod := time.Duration(task.Interval) * time.Minute
	timeSinceLastNotified := time.Since(task.LastNotified.ToTime())

	return timeSinceLastNotified < cooldownPeriod
}

// getRecordsForClient 获取指定客户端在时间窗口内的记录
func getRecordsForClient(clientUUID string, start, end time.Time) ([]models.Record, error) {
	return records.GetRecordsByClientAndTime(clientUUID, start, end)
}

// checkMetricThreshold 检查指标是否达到阈值
func checkMetricThreshold(records []models.Record, task models.LoadNotification) bool {
	if len(records) == 0 {
		return false
	}

	// 计算需要达标的最小记录数
	minRequiredRecords := int(float32(len(records)) * task.Ratio)
	if minRequiredRecords == 0 {
		minRequiredRecords = 1
	}

	exceededCount := 0

	for _, record := range records {
		metricValue := getMetricValue(record, task.Metric)
		if metricValue >= task.Threshold {
			exceededCount++
		}
	}

	return exceededCount >= minRequiredRecords
}

// getMetricValue 根据指标名称获取记录中的对应值
func getMetricValue(record models.Record, metric string) float32 {
	client, err := clients.GetClientByUUID(record.Client) // 确保客户端信息已加载
	if err != nil {
		log.Printf("Failed to get client info for %s: %v", record.Client, err)
		return 0
	}
	switch metric {
	case "cpu":
		return record.Cpu
	case "gpu":
		return record.Gpu
	case "ram":
		if record.RamTotal > 0 {
			return float32(record.Ram) / float32(client.MemTotal) * 100
		}
		return 0
	case "swap":
		if record.SwapTotal > 0 {
			return float32(record.Swap) / float32(client.SwapTotal) * 100
		}
		return 0
	case "load":
		return record.Load
	case "temp":
		return record.Temp
	case "disk":
		if record.DiskTotal > 0 {
			return float32(record.Disk) / float32(client.DiskTotal) * 100
		}
		return 0
	default:
		// 尝试通过反射获取字段值
		v := reflect.ValueOf(record)
		field := v.FieldByName(metric)
		if field.IsValid() && field.CanInterface() {
			switch field.Kind() {
			case reflect.Float32:
				return float32(field.Float())
			case reflect.Float64:
				return float32(field.Float())
			case reflect.Int, reflect.Int32, reflect.Int64:
				return float32(field.Int())
			}
		}
		return 0
	}
}

// sendLoadNotification 发送负载通知
func sendLoadNotification(clientUUIDs []string, task models.LoadNotification) {
	message := "The following clients have exceeded the load threshold:\n\n"
	for _, clientUUID := range clientUUIDs {
		client, err := clients.GetClientByUUID(clientUUID)
		if err != nil {
			log.Printf("Failed to get client info for %s: %v", clientUUID, err)
			continue
		}
		message += fmt.Sprintf("%s\n", client.Name)
	}
	if len(clientUUIDs) == 0 {
		return
	}
	go func() {
		messageSender.SendTextMessage(message, fmt.Sprintf("%s - Komari Load Alert", task.Name))
	}()
}

// updateLastNotified 更新最后通知时间
func updateLastNotified(taskId uint, notifyTime time.Time) {
	db := dbcore.GetDBInstance()
	if err := db.Model(&models.LoadNotification{}).Where("id = ?", taskId).Update("last_notified", notifyTime).Error; err != nil {
		log.Printf("Failed to update last_notified for task %d: %v", taskId, err)
	}
}

// ReloadLoadNotificationSchedule 加载或重载时间表
func ReloadLoadNotificationSchedule(loadNotifications []models.LoadNotification) error {
	return LoadNotificationManager.Reload(loadNotifications)
}
