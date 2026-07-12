package src

import (
	"fmt"
	"html"
	"strings"
	"time"

	"coolifymanager/src/config"
	"coolifymanager/src/database"
	td "github.com/AshokShau/gotdbot"
)

func monitorResourceChanges(client *td.Client) {
	previous := map[string]string{}
	initialized := false
	for {
		apps, err := config.Coolify.ListApplications()
		if err == nil {
			current := map[string]string{}
			names := map[string]string{}
			for _, app := range apps {
				current[app.UUID] = app.Status
				names[app.UUID] = app.Name
			}
			if initialized {
				for id, status := range current {
					if old, exists := previous[id]; !exists {
						notifyAdmins(client, fmt.Sprintf("🆕 <b>%s</b> eklendi.\nDurum: <code>%s</code>", html.EscapeString(names[id]), statusLabel(status)))
					} else if old != status {
						notifyAdmins(client, fmt.Sprintf("🔔 <b>%s</b> durum değiştirdi.\n<code>%s</code> → <code>%s</code>", html.EscapeString(names[id]), statusLabel(old), statusLabel(status)))
					}
				}
				for id, old := range previous {
					if _, exists := current[id]; !exists {
						notifyAdmins(client, fmt.Sprintf("🗑 Bir kaynak silindi.\nUUID: <code>%s</code>\nSon durum: <code>%s</code>", id, statusLabel(old)))
					}
				}
			}
			previous = current
			initialized = true
		}
		time.Sleep(30 * time.Second)
	}
}

func statusLabel(status string) string {
	labels := map[string]string{
		"running:healthy":   "Çalışıyor • Sağlıklı",
		"running:unhealthy": "Çalışıyor • Sağlıksız",
		"running:unknown":   "Çalışıyor • Sağlık bilinmiyor",
		"exited:unhealthy":  "Durduruldu • Sağlıksız",
		"exited":            "Durduruldu",
		"restarting":        "Yeniden başlatılıyor",
	}
	if label, ok := labels[strings.ToLower(status)]; ok {
		return label
	}
	return status
}

func notifyScheduledTaskResult(client *td.Client, task database.ScheduledTask, executionErr error) {
	operation := taskTypeLabel(task.Type)
	if executionErr == nil {
		notifyAdmins(client, fmt.Sprintf("✅ <b>Zamanlanmış görev başarılı</b>\n\nUygulama: <b>%s</b>\nİşlem: <b>%s</b>\nGörev ID: <code>%s</code>", html.EscapeString(task.Name), operation, task.ID))
		return
	}
	notifyAdmins(client, fmt.Sprintf("❌ <b>Zamanlanmış görev başarısız</b>\n\nUygulama: <b>%s</b>\nİşlem: <b>%s</b>\nHata: <code>%s</code>\nGörev ID: <code>%s</code>", html.EscapeString(task.Name), operation, html.EscapeString(executionErr.Error()), task.ID))
}

func notifyAdmins(client *td.Client, text string) {
	for _, id := range config.AdminIDs() {
		_, _ = client.SendTextMessage(id, text, &td.SendTextMessageOpts{ParseMode: "HTML"})
	}
}
