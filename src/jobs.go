package src

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"coolifymanager/src/config"
	"coolifymanager/src/database"
	"coolifymanager/src/scheduler"
	td "github.com/AshokShau/gotdbot"
)

const pageSize = 5

func jobsHandler(c *td.Client, msg *td.Message) error {
	if !config.IsDev(msg.SenderID()) {
		_, err := msg.ReplyText(c, "🚫 Bu komutu kullanma yetkiniz yok.", nil)
		return err
	}
	text, keyboard, err := buildJobsMessage(1)
	if err != nil {
		_, err = msg.ReplyText(c, "❌ "+err.Error(), nil)
		return err
	}
	_, err = msg.ReplyText(c, text, &td.SendTextMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func jobsPaginationHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		return cb.Answer(c, 0, true, "Bu işlem için yetkiniz yok.", "")
	}
	page := 1
	if parts := strings.Split(cb.DataString(), ":"); len(parts) > 1 {
		_, _ = fmt.Sscanf(parts[1], "%d", &page)
	}
	text, keyboard, err := buildJobsMessage(page)
	if err != nil {
		return cb.Answer(c, 0, true, "❌ "+err.Error(), "")
	}
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func buildJobsMessage(page int) (string, td.ReplyMarkup, error) {
	tasks, err := database.GetTasks()
	if err != nil {
		return "", nil, fmt.Errorf("görevler alınamadı: %v", err)
	}
	if len(tasks) == 0 {
		return "📭 Henüz zamanlanmış görev yok.\n\nUygulamalar menüsünden bir uygulama seçerek yeni görev oluşturabilirsiniz.", nil, nil
	}
	sort.Slice(tasks, func(i, j int) bool { return nextTaskRun(tasks[i]).Before(nextTaskRun(tasks[j])) })
	start, end, buttons := Paginate(len(tasks), page, pageSize, "jobs:")
	var text strings.Builder
	keyboard := &td.ReplyMarkupInlineKeyboard{}
	text.WriteString(fmt.Sprintf("<b>📅 Zamanlanmış Görevler</b>\nSayfa %d • Toplam %d görev\n\n", page, len(tasks)))
	for _, task := range tasks[start:end] {
		text.WriteString(fmt.Sprintf("<b>📦 %s</b>\n", task.Name))
		text.WriteString(fmt.Sprintf("├ İşlem: <b>%s</b>\n", taskTypeLabel(task.Type)))
		text.WriteString(fmt.Sprintf("├ Tekrar: <b>%s</b>\n", taskScheduleLabel(task)))
		text.WriteString(fmt.Sprintf("├ Sonraki çalışma: <code>%s</code>\n", nextTaskRun(task).Format("02.01.2006 15:04")))
		text.WriteString(fmt.Sprintf("└ ID: <code>%s</code>\n\n", task.ID))
		keyboard.Rows = append(keyboard.Rows, []td.InlineKeyboardButton{{Text: "🗑 İptal et • " + task.Name, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("job_del:" + task.ID)}}})
	}
	if len(buttons) > 0 {
		row := make([]td.InlineKeyboardButton, 0, len(buttons))
		for _, button := range buttons {
			row = append(row, td.InlineKeyboardButton{Text: button.Text, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(button.Data)}})
		}
		keyboard.Rows = append(keyboard.Rows, row)
	}
	return text.String(), keyboard, nil
}

func taskTypeLabel(taskType string) string {
	labels := map[string]string{"restart": "Yeniden Başlat", "stop": "Durdur", "redeploy": "Redeploy", "delete": "Sil"}
	if label := labels[taskType]; label != "" {
		return label
	}
	return taskType
}

func taskScheduleLabel(task database.ScheduledTask) string {
	if task.OneTime {
		return "Tek seferlik"
	}
	labels := map[string]string{"every_1h": "Her saat", "every_24h": "Her gün", "every_168h": "Her hafta", "hourly": "Her saat", "daily": "Her gün", "weekly": "Her hafta"}
	if label := labels[task.Schedule]; label != "" {
		return label
	}
	return task.Schedule
}

func nextTaskRun(task database.ScheduledTask) time.Time {
	if task.OneTime || task.NextRun.After(time.Now()) {
		return task.NextRun
	}
	duration, ok := scheduler.ParseDurationSchedule(task.Schedule)
	if !ok || duration <= 0 || task.NextRun.IsZero() {
		return task.NextRun
	}
	elapsed := time.Since(task.NextRun)
	return task.NextRun.Add((elapsed/duration + 1) * duration)
}

func jobDeleteHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "schedule") {
		return cb.Answer(c, 0, true, "Yetkiniz yok.", "")
	}
	id := strings.TrimPrefix(cb.DataString(), "job_del:")
	_ = scheduler.RemoveTask(id)
	if err := database.DeleteTask(id); err != nil {
		return err
	}
	_ = cb.Answer(c, 0, false, "Görev iptal edildi.", "")
	text, keyboard, err := buildJobsMessage(1)
	if err != nil {
		return err
	}
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}
