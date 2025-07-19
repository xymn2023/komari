package notifier

import (
	"fmt"
	"math"
	"time"

	"github.com/komari-monitor/komari/database/clients"
	"github.com/komari-monitor/komari/database/config"
	"github.com/komari-monitor/komari/utils/messageSender"
	"github.com/komari-monitor/komari/utils/renewal"
)

func CheckExpireScheduledWork() {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()) // UTC 9AM = CST 17PM
		if now.After(next) {
			next = next.Add(24 * time.Hour)
		}
		duration := next.Sub(now)
		time.Sleep(duration)

		cfg, err := config.Get()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		clients_all, err := clients.GetAllClientBasicInfo()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		checkTime := time.Now()

		// 过期提醒检查（仅当启用过期通知时）
		if cfg.ExpireNotificationEnabled {
			notificationLeadDays := cfg.ExpireNotificationLeadDays

			type clientToExpireInfo struct {
				Name     string
				DaysLeft int
			}

			var clientLeadToExpire []clientToExpireInfo

			for _, client := range clients_all {
				clientExpireTime := client.ExpiredAt.ToTime()

				if clientExpireTime.Before(checkTime) {
					continue
				}

				notificationThreshold := checkTime.Add(time.Duration(notificationLeadDays) * 24 * time.Hour)

				if clientExpireTime.Before(notificationThreshold) || clientExpireTime.Equal(notificationThreshold) {
					remainingDuration := clientExpireTime.Sub(checkTime)
					daysLeft := int(math.Ceil(remainingDuration.Hours() / 24))

					clientLeadToExpire = append(clientLeadToExpire, clientToExpireInfo{
						Name:     client.Name,
						DaysLeft: daysLeft,
					})
				}
			}

			if len(clientLeadToExpire) > 0 {
				message := "The following clients are about to expire: \n\n"
				for _, clientInfo := range clientLeadToExpire {
					message += fmt.Sprintf("• %s in %d days\n", clientInfo.Name, clientInfo.DaysLeft)
				}
				messageSender.SendTextMessage(message, "Komari Expiration Notification")
			}
		}

		// 等待1秒，防止多次触发
		time.Sleep(time.Second)
		for _, client := range clients_all {
			renewal.CheckAndAutoRenewal(client)
		}
	}

}
