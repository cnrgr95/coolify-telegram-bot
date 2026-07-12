package src

import (
	"coolifymanager/src/config"
	"coolifymanager/src/coolity"
	"coolifymanager/src/database"
	"coolifymanager/src/scheduler"
	"fmt"
	"os"
	"strings"

	td "github.com/AshokShau/gotdbot"
	"github.com/google/uuid"
)

func listProjectsHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}

	_ = cb.Answer(c, 0, false, "İşleniyor...", "")
	apps, err := config.Coolify.ListApplications()
	if err != nil {
		_, _ = cb.EditMessageText(c, "Projeler alınamadı: "+err.Error(), nil)
		return nil
	}

	if len(apps) == 0 {
		_, _ = cb.EditMessageText(c, "ğŸ˜¶ Uygulama bulunamadı.", nil)
		return nil
	}

	page := 1
	cbData := cb.DataString()
	if strings.Contains(cbData, ":") {
		parts := strings.Split(cbData, ":")
		if len(parts) > 1 {
			fmt.Sscanf(parts[1], "%d", &page)
		}
	}

	start, end, paginationButtons := Paginate(len(apps), page, 7, "list_projects:")

	kb := &td.ReplyMarkupInlineKeyboard{}
	for _, app := range apps[start:end] {
		text := fmt.Sprintf("ğŸ“¦ %s (%s)", app.Name, app.Status)
		data := "project_menu:" + app.UUID

		kb.Rows = append(kb.Rows, []td.InlineKeyboardButton{
			{
				Text: text,
				Type: &td.InlineKeyboardButtonTypeCallback{
					Data: []byte(data),
				},
			},
		})
	}

	if len(paginationButtons) > 0 {
		row := make([]td.InlineKeyboardButton, 0, len(paginationButtons))

		for _, btn := range paginationButtons {
			row = append(row, td.InlineKeyboardButton{
				Text: btn.Text,
				Type: &td.InlineKeyboardButtonTypeCallback{
					Data: []byte(btn.Data),
				},
			})
		}

		kb.Rows = append(kb.Rows, row)
	}

	_, err = cb.EditMessageText(c, "<b>ğŸ“‹ Bir uygulama seçin:</b>", &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func projectMenuHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}

	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "project_menu:")
	if strings.HasPrefix(uuid, "svc:") {
		var found *coolify.Application
		apps, _ := config.Coolify.ListApplications()
		for i := range apps {
			if apps[i].UUID == uuid {
				found = &apps[i]
				break
			}
		}
		if found == nil {
			_, e := cb.EditMessageText(c, "❌ Servis bulunamadı.", nil)
			return e
		}
		kb := &td.ReplyMarkupInlineKeyboard{
			Rows: [][]td.InlineKeyboardButton{
				{
					{Text: "🔄 Yeniden Başlat", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("restart:" + uuid)}},
					{Text: "🛑 Durdur", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("stop:" + uuid)}},
				},
				{
					{Text: "🔙 Geri", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("list_projects:")}},
				},
			},
		}
		_, e := cb.EditMessageText(c, fmt.Sprintf("<b>%s</b>\nDurum: <code>%s</code>\n\nBu kaynak bir Docker Compose servisidir.", found.Name, found.Status), &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
		return e
	}
	app, err := config.Coolify.GetApplicationByUUID(uuid)
	if err != nil {
		_, err = cb.EditMessageText(c, "âŒ Proje yüklenemedi: "+err.Error(), nil)
		return err
	}

	text := fmt.Sprintf("<b>ğŸ“¦ %s</b>\nğŸŒ %s\nğŸ“„ Durum: <code>%s</code>", app.Name, app.FQDN, app.Status)
	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”„ Yeniden Başlat",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("restart:" + uuid),
					},
				},
				{
					Text: "ğŸš€ Dağıt",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("deploy:" + uuid),
					},
				},
			},
			{
				{
					Text: "ğŸ“œ Loglar",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("logs:" + uuid),
					},
				},
				{
					Text: "â„¹ï¸ Durum",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("status:" + uuid),
					},
				},
			},
			{
				{
					Text: "ğŸ“… Zamanla",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("sch_m:" + uuid),
					},
				},
			},
			{
				{
					Text: "ğŸ›‘ Durdur",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("stop:" + uuid),
					},
				},
				{
					Text: "âŒ Sil",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("delete:" + uuid),
					},
				},
			},
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("list_projects:"),
					},
				},
			},
		},
	}

	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})

	return err
}

func restartHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "restart:")
	if strings.HasPrefix(uuid, "svc:") {
		err := config.Coolify.ServiceAction(uuid, "restart")
		if err != nil {
			_, _ = cb.EditMessageText(c, "❌ "+err.Error(), nil)
		} else {
			_, _ = cb.EditMessageText(c, "✅ Servis yeniden başlatma kuyruğuna alındı.", nil)
		}
		return nil
	}

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	res, err := config.Coolify.RestartApplicationByUUID(uuid)
	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Yeniden Başlat failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	text := fmt.Sprintf("âœ… Yeniden Başlat queued!\nDağıtım UUID: <code>%s</code>", res.DeploymentUUID)
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func deployHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}

	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "deploy:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	res, err := config.Coolify.StartApplicationDeployment(uuid, false, false)
	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Dağıt failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return err
	}

	text := fmt.Sprintf("âœ… Dağıtım queued!\nDağıtım UUID: <code>%s</code>", res.DeploymentUUID)
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func logsHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	uuid := strings.TrimPrefix(cb.DataString(), "logs:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	_, _ = cb.EditMessageText(c, "İşleniyor...", nil)
	logsData, err := config.Coolify.GetApplicationLogsByUUID(uuid)
	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Loglar error: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	tmpFile, err := os.CreateTemp("", "logs-*.txt")
	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Failed to create temp file: "+err.Error(), nil)
		return err
	}

	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte(logsData)); err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Failed to write logs: "+err.Error(), nil)
		return err
	}

	tmpFile.Close()

	file := tmpFile.Name()
	_, err = c.EditMessageMedia(cb.ChatId, &td.InputMessageDocument{Document: td.GetInputFile(file)}, cb.MessageId, &td.EditMessageMediaOpts{ReplyMarkup: kb})
	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Failed to send logs file: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return fmt.Errorf("edit message media error: %s", err.Error())
	}

	return nil
}

func statusHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, true, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "status:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	app, err := config.Coolify.GetApplicationByUUID(uuid)
	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Durum hatası: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	text := fmt.Sprintf("ğŸ“¦ <b>%s</b>\nGüncel Durum: <code>%s</code>", app.Name, app.Status)
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func stopHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "stop:")
	if strings.HasPrefix(uuid, "svc:") {
		err := config.Coolify.ServiceAction(uuid, "stop")
		if err != nil {
			_, _ = cb.EditMessageText(c, "❌ "+err.Error(), nil)
		} else {
			_, _ = cb.EditMessageText(c, "✅ Servis durdurma kuyruğuna alındı.", nil)
		}
		return nil
	}

	res, err := config.Coolify.StopApplicationByUUID(uuid)
	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Durdur failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	_, err = cb.EditMessageText(c, "ğŸ›‘ "+res.Message, &td.EditTextMessageOpts{ReplyMarkup: kb, ParseMode: "HTML"})
	return err
}

func deleteHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "delete:")

	err := config.Coolify.DeleteApplicationByUUID(uuid)
	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	if err != nil {
		_, err = cb.EditMessageText(c, "âŒ Sil failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	_, err = cb.EditMessageText(c, "âœ… Application deleted successfully.", &td.EditTextMessageOpts{ReplyMarkup: kb})
	return err
}

func scheduleMenuHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "sch_m:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”„ Yeniden Başlat",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("sch_a:" + uuid + ":restart"),
					},
				},
			},
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	_, err := cb.EditMessageText(c, "<b>ğŸ“… İşlem türünü seçin:</b>", &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err
}

func scheduleActionHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	// Format: sch_a:uuid:actionType
	cbData := cb.DataString()
	data := strings.TrimPrefix(cbData, "sch_a:")
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}
	uuid := parts[0]
	actionType := parts[1]

	// Common intervals
	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "Hourly",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_1h", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "Daily",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_1d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "Every 2 Days",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_2d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "Every 3 Days",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_3d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "Weekly",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_7d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("sch_m:" + uuid),
					},
				},
			},
		},
	}

	_, err := cb.EditMessageText(c, "<b>â° Select Zamanla:</b>", &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err
}

func scheduleCreateHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "ğŸš« Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	// Format: sch_c:uuid:actionType:schedule
	data := strings.TrimPrefix(cb.DataString(), "sch_c:")

	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return nil
	}
	uuid := parts[0]
	actionType := parts[1]
	schedule := parts[2]

	app, err := config.Coolify.GetApplicationByUUID(uuid)
	if err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Failed to get application: "+err.Error(), nil)
		return nil
	}

	task := database.ScheduledTask{
		ID:          uuid.New().String(),
		Name:        app.Name,
		ProjectUUID: uuid,
		Type:        actionType,
		Schedule:    schedule,
	}

	if err := database.AddTask(task); err != nil {
		_, _ = cb.EditMessageText(c, "âŒ Failed to save task: "+err.Error(), nil)
		return nil
	}

	if err := scheduler.ScheduleTask(task); err != nil {
		_ = database.DeleteTask(task.ID)
		_, _ = cb.EditMessageText(c, "âŒ Failed to schedule task: "+err.Error(), nil)
		return nil
	}

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "ğŸ”™ Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	_, err = cb.EditMessageText(c, fmt.Sprintf("âœ… Görev başarıyla zamanlandı!\n\nID: <code>%s</code>\nType: %s\nZamanla: %s", task.ID, actionType, schedule), &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err
}
