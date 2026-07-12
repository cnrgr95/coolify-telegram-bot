package src

import (
	"coolifymanager/src/config"
	"coolifymanager/src/database"
	"coolifymanager/src/scheduler"
	"fmt"
	"strings"

	td "github.com/AshokShau/gotdbot"
)

func unscheduleHandler(c *td.Client, msg *td.Message) error {
	if !config.IsDev(msg.SenderID()) {
		_, err := msg.ReplyText(c, "🚫 Bu komutu kullanma yetkiniz yok.", nil)
		return err
	}

	args := strings.Fields(msg.Text())
	if len(args) < 2 {
		_, err := msg.ReplyText(c, "Kullanım: /unschedule <görev_id>", nil)
		return err
	}
	taskID := args[1]

	if err := scheduler.RemoveTask(taskID); err != nil {
		_, err = msg.ReplyText(c, fmt.Sprintf("⚠️ Görev zamanlayıcıdan kaldırılamadı: %v", err), nil)
	}

	if err := database.DeleteTask(taskID); err != nil {
		_, err = msg.ReplyText(c, fmt.Sprintf("❌ Görev veritabanından silinemedi: %v", err), nil)
		return err
	}

	_, err := msg.ReplyText(c, fmt.Sprintf("✅ <code>%s</code> görevi kaldırıldı.", taskID), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}
