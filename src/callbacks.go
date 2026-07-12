package src

import (
	"coolifymanager/src/config"
	"coolifymanager/src/coolity"
	"coolifymanager/src/database"
	"coolifymanager/src/scheduler"
	"fmt"
	"os"
	"sort"
	"strings"

	td "github.com/AshokShau/gotdbot"
	uuidpkg "github.com/google/uuid"
)

func listProjectsHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
		return nil
	}

	_ = cb.Answer(c, 0, false, "İşleniyor...", "")
	apps, err := config.Coolify.ListApplications()
	if err != nil {
		_, _ = cb.EditMessageText(c, "Projeler alınamadı: "+err.Error(), nil)
		return nil
	}

	if len(apps) == 0 {
		_, _ = cb.EditMessageText(c, "😶 Uygulama bulunamadı.", nil)
		return nil
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Project == apps[j].Project {
			return apps[i].Name < apps[j].Name
		}
		return apps[i].Project < apps[j].Project
	})

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
		text := fmt.Sprintf("📦 %s › %s (%s)", app.Project, app.Name, app.Status)
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

	_, err = cb.EditMessageText(c, "<b>📋 Bir uygulama seçin:</b>", &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func projectMenuHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.IsDev(cb.SenderUserId) {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
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
		if !config.Can(cb.SenderUserId, "restart") {
			kb.Rows = kb.Rows[1:]
		}
		_, e := cb.EditMessageText(c, fmt.Sprintf("<b>%s</b>\nDurum: <code>%s</code>\n\nBu kaynak bir Docker Compose servisidir.", found.Name, found.Status), &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
		return e
	}
	app, err := config.Coolify.GetApplicationByUUID(uuid)
	if err != nil {
		_, err = cb.EditMessageText(c, "❌ Proje yüklenemedi: "+err.Error(), nil)
		return err
	}

	text := fmt.Sprintf("<b>📦 %s</b>\n🌐 %s\n📄 Durum: <code>%s</code>", app.Name, app.FQDN, app.Status)
	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "🔄 Yeniden Başlat",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("restart:" + uuid),
					},
				},
				{
					Text: "🚀 Dağıt",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("deploy:" + uuid),
					},
				},
			},
			{
				{
					Text: "📜 Loglar",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("logs:" + uuid),
					},
				},
				{
					Text: "ℹ️ Durum",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("status:" + uuid),
					},
				},
			},
			{
				{
					Text: "📅 Zamanla",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("sch_m:" + uuid),
					},
				},
			},
			{
				{Text: "♻️ Redeploy", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("redeploy:" + uuid)}},
			},
			{
				{
					Text: "🛑 Durdur",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("stop:" + uuid),
					},
				},
				{
					Text: "❌ Sil",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("delete:" + uuid),
					},
				},
			},
			{
				{
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("list_projects:"),
					},
				},
			},
		},
	}
	if config.Role(cb.SenderUserId) == "viewer" {
		kb.Rows = [][]td.InlineKeyboardButton{kb.Rows[1], kb.Rows[len(kb.Rows)-1]}
	} else if !config.Can(cb.SenderUserId, "delete") {
		for rowIndex := range kb.Rows {
			filtered := kb.Rows[rowIndex][:0]
			for _, button := range kb.Rows[rowIndex] {
				if button.Text != "❌ Sil" {
					filtered = append(filtered, button)
				}
			}
			kb.Rows[rowIndex] = filtered
		}
	}

	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})

	return err
}

func restartHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "restart") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
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
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	res, err := config.Coolify.RestartApplicationByUUID(uuid)
	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ Yeniden Başlat failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	text := fmt.Sprintf("✅ Yeniden Başlat queued!\nDağıtım UUID: <code>%s</code>", res.DeploymentUUID)
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func deployHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "deploy") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
		return nil
	}

	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "deploy:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	res, err := config.Coolify.StartApplicationDeployment(uuid, false, false)
	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ Dağıt failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return err
	}

	text := fmt.Sprintf("✅ Dağıtım queued!\nDağıtım UUID: <code>%s</code>", res.DeploymentUUID)
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func redeployHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "deploy") {
		_ = cb.Answer(c, 0, true, "Bu işlem için yetkiniz yok.", "")
		return nil
	}
	resourceID := strings.TrimPrefix(cb.DataString(), "redeploy:")
	res, err := config.Coolify.StartApplicationDeployment(resourceID, true, false)
	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ Redeploy başlatılamadı: "+err.Error(), nil)
		return nil
	}
	_, err = cb.EditMessageText(c, fmt.Sprintf("✅ Redeploy kuyruğa alındı.\nDağıtım: <code>%s</code>", res.DeploymentUUID), &td.EditTextMessageOpts{ParseMode: "HTML"})
	return err
}

func logsHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "logs") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	uuid := strings.TrimPrefix(cb.DataString(), "logs:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "🔙 Geri",
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
		_, _ = cb.EditMessageText(c, "❌ Loglar error: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	tmpFile, err := os.CreateTemp("", "logs-*.txt")
	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ Failed to create temp file: "+err.Error(), nil)
		return err
	}

	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write([]byte(logsData)); err != nil {
		_, _ = cb.EditMessageText(c, "❌ Failed to write logs: "+err.Error(), nil)
		return err
	}

	tmpFile.Close()

	file := tmpFile.Name()
	_, err = c.EditMessageMedia(cb.ChatId, &td.InputMessageDocument{Document: td.GetInputFile(file)}, cb.MessageId, &td.EditMessageMediaOpts{ReplyMarkup: kb})
	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ Failed to send logs file: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return fmt.Errorf("edit message media error: %s", err.Error())
	}

	return nil
}

func statusHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "view") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, true, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "status:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	app, err := config.Coolify.GetApplicationByUUID(uuid)
	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ Durum hatası: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	text := fmt.Sprintf("📦 <b>%s</b>\nGüncel Durum: <code>%s</code>", app.Name, app.Status)
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func stopHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "stop") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
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
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ Durdur failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	_, err = cb.EditMessageText(c, "🛑 "+res.Message, &td.EditTextMessageOpts{ReplyMarkup: kb, ParseMode: "HTML"})
	return err
}

func deleteHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "delete") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
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
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	if err != nil {
		_, err = cb.EditMessageText(c, "❌ Sil failed: "+err.Error(), &td.EditTextMessageOpts{ReplyMarkup: kb})
		return nil
	}

	_, err = cb.EditMessageText(c, "✅ Application deleted successfully.", &td.EditTextMessageOpts{ReplyMarkup: kb})
	return err
}

func scheduleMenuHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "schedule") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	cbData := cb.DataString()
	uuid := strings.TrimPrefix(cbData, "sch_m:")

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "🔄 Yeniden Başlat",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("sch_a:" + uuid + ":restart"),
					},
				},
			},
			{{Text: "⏹ Durdur", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("sch_a:" + uuid + ":stop")}}},
			{{Text: "♻️ Redeploy", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("sch_a:" + uuid + ":redeploy")}}},
			{{Text: "🗑 Sil", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("sch_a:" + uuid + ":delete")}}},
			{
				{
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}
	if !config.Can(cb.SenderUserId, "delete") {
		filtered := kb.Rows[:0]
		for _, row := range kb.Rows {
			if len(row) > 0 && row[0].Text == "🗑 Sil" {
				continue
			}
			filtered = append(filtered, row)
		}
		kb.Rows = filtered
	}

	_, err := cb.EditMessageText(c, "<b>📅 İşlem türünü seçin:</b>", &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err
}

func scheduleActionHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "schedule") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
		return nil
	}
	_ = cb.Answer(c, 0, false, "İşleniyor...", "")

	parts := strings.Split(strings.TrimPrefix(cb.DataString(), "sch_a:"), ":")
	if len(parts) < 2 {
		return nil
	}
	uuid, actionType := parts[0], parts[1]
	if actionType == "delete" && !config.Can(cb.SenderUserId, "delete") {
		_ = cb.Answer(c, 0, true, "Silme işlemi için admin yetkisi gerekir.", "")
		return nil
	}
	return showScheduleModes(c, cb, uuid, actionType)

	/* Eski Telegram mesajlarıyla uyumluluk için bırakılan görünüm.
	rows := [][]td.InlineKeyboardButton{
		{{Text: "🕐 Tek seferlik", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:once", uuid, actionType))}}},
	}
	if actionType == "restart" || actionType == "redeploy" {
		rows = append(rows,
			[]td.InlineKeyboardButton{{Text: "🔁 Saatlik", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:hourly", uuid, actionType))}}},
			[]td.InlineKeyboardButton{{Text: "🔁 Günlük", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:daily", uuid, actionType))}}},
			[]td.InlineKeyboardButton{{Text: "🔁 Haftalık", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:weekly", uuid, actionType))}}},
		)
	}
	rows = append(rows, []td.InlineKeyboardButton{{Text: "🔙 Geri", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("sch_m:" + uuid)}}})
	_, err := cb.EditMessageText(c, "<b>⏰ Çalışma biçimini seçin:</b>\n\nDurdur ve Sil işlemleri yalnızca tek sefer çalıştırılabilir.", &td.EditTextMessageOpts{
		ParseMode: "HTML", ReplyMarkup: &td.ReplyMarkupInlineKeyboard{Rows: rows},
	})
	return err */

	/* Legacy callback format retained for existing Telegram messages.
	// Format: sch_a:uuid:actionType
	cbData := cb.DataString()
	data := strings.TrimPrefix(cbData, "sch_a:")
	parts := strings.Split(data, ":")
	if len(parts) < 2 {
		return nil
	}
	uuid := parts[0]
	actionType := parts[1]
	if actionType == "delete" && !config.Can(cb.SenderUserId, "delete") {
		_ = cb.Answer(c, 0, true, "Silme işlemi için admin yetkisi gerekir.", "")
		return nil
	}

	// Common intervals
	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "Her Saat",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_1h", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "Her Gün",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_1d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "2 Günde Bir",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_2d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "3 Günde Bir",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_3d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "Haftalık",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte(fmt.Sprintf("sch_c:%s:%s:every_7d", uuid, actionType)),
					},
				},
			},
			{
				{
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("sch_m:" + uuid),
					},
				},
			},
		},
	}

	_, err := cb.EditMessageText(c, "<b>⏰ Zamanlama aralĿını seçin:</b>", &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err */
}

func scheduleCreateHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "schedule") {
		_ = cb.Answer(c, 0, true, "🚫 Bu işlem için yetkiniz yok.", "")
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
		_, _ = cb.EditMessageText(c, "❌ Failed to get application: "+err.Error(), nil)
		return nil
	}

	task := database.ScheduledTask{
		ID:          uuidpkg.New().String(),
		Name:        app.Name,
		ProjectUUID: uuid,
		Type:        actionType,
		Schedule:    schedule,
	}

	if err := database.AddTask(task); err != nil {
		_, _ = cb.EditMessageText(c, "❌ Failed to save task: "+err.Error(), nil)
		return nil
	}

	if err := scheduler.ScheduleTask(task); err != nil {
		_ = database.DeleteTask(task.ID)
		_, _ = cb.EditMessageText(c, "❌ Failed to schedule task: "+err.Error(), nil)
		return nil
	}

	kb := &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{
				{
					Text: "🔙 Geri",
					Type: &td.InlineKeyboardButtonTypeCallback{
						Data: []byte("project_menu:" + uuid),
					},
				},
			},
		},
	}

	_, err = cb.EditMessageText(c, fmt.Sprintf("✅ Görev başarıyla zamanlandı!\n\nID: <code>%s</code>\nType: %s\nZamanla: %s", task.ID, actionType, schedule), &td.EditTextMessageOpts{
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	return err
}

func showScheduleModes(c *td.Client, cb *td.UpdateNewCallbackQuery, uuid, actionType string) error {
	rows := [][]td.InlineKeyboardButton{
		{{Text: "🕐 Tek seferlik", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:once", uuid, actionType))}}},
	}
	if actionType == "restart" || actionType == "redeploy" {
		rows = append(rows,
			[]td.InlineKeyboardButton{{Text: "🔁 Saatlik", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:hourly", uuid, actionType))}}},
			[]td.InlineKeyboardButton{{Text: "🔁 Günlük", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:daily", uuid, actionType))}}},
			[]td.InlineKeyboardButton{{Text: "🔁 Haftalık", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("sch_t:%s:%s:weekly", uuid, actionType))}}},
		)
	}
	rows = append(rows,
		[]td.InlineKeyboardButton{{Text: "⬅️ İşlem Türüne Dön", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("sch_m:" + uuid)}}},
		[]td.InlineKeyboardButton{{Text: "✖️ İşlemi İptal Et", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("flow_cancel")}}},
	)
	_, err := cb.EditMessageText(c, "<b>⏰ Çalışma biçimini seçin</b>\n\nDurdur ve Sil yalnızca tek seferlik çalıştırılabilir.", &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: &td.ReplyMarkupInlineKeyboard{Rows: rows}})
	return err
}

func scheduleTimeHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "schedule") {
		_ = cb.Answer(c, 0, true, "Bu işlem için yetkiniz yok.", "")
		return nil
	}
	parts := strings.Split(strings.TrimPrefix(cb.DataString(), "sch_t:"), ":")
	if len(parts) != 3 {
		return nil
	}
	if (parts[1] == "stop" || parts[1] == "delete") && parts[2] != "once" {
		_ = cb.Answer(c, 0, true, "Bu işlem yalnızca tek seferlik zamanlanabilir.", "")
		return nil
	}
	pendingInputs.Lock()
	pendingInputs.values[cb.SenderUserId] = pendingInput{Kind: "schedule_time", First: parts[0], Second: parts[1] + "|" + parts[2]}
	pendingInputs.Unlock()
	_ = cb.Answer(c, 0, false, "Tarih ve saati yazın", "")
	_, err := cb.EditMessageText(c, "<b>📅 İlk çalışma tarihini ve saatini yazın</b>\n\nBiçim: <code>GG.AA.YYYY SS:DD</code>\nÖrnek: <code>15.07.2026 03:30</code>\n\nSaat dilimi: Europe/Istanbul\n\nYanlış seçim yaptıysanız geri dönebilir veya işlemi iptal edebilirsiniz.", &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: scheduleInputKeyboard(parts[0], parts[1])})
	return err
}
