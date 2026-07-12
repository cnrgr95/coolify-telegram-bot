package src

import (
	"fmt"
	"time"

	"coolifymanager/src/config"
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
						notifyAdmins(client, fmt.Sprintf("🆕 <b>%s</b> eklendi.\nDurum: <code>%s</code>", names[id], status))
					} else if old != status {
						notifyAdmins(client, fmt.Sprintf("🔔 <b>%s</b> durum değiştirdi.\n<code>%s</code> → <code>%s</code>", names[id], old, status))
					}
				}
				for id, old := range previous {
					if _, exists := current[id]; !exists {
						notifyAdmins(client, fmt.Sprintf("🗑 Bir kaynak silindi.\nUUID: <code>%s</code>\nSon durum: <code>%s</code>", id, old))
					}
				}
			}
			previous = current
			initialized = true
		}
		time.Sleep(30 * time.Second)
	}
}

func notifyAdmins(client *td.Client, text string) {
	for _, id := range config.AdminIDs() {
		_, _ = client.SendTextMessage(id, text, &td.SendTextMessageOpts{ParseMode: "HTML"})
	}
}
