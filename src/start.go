package src

import (
	"coolifymanager/src/config"
	"coolifymanager/src/database"
	"coolifymanager/src/monitoring"
	"coolifymanager/src/scheduler"
	"fmt"
	"html"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
	uuidpkg "github.com/google/uuid"
)

func startHandler(c *td.Client, msg *td.Message) error {
	if !config.IsDev(msg.SenderID()) {
		_, err := msg.ReplyText(c, "🚫 Bu botu kullanma yetkiniz yok. Yöneticiden Telegram ID'nizi eklemesini isteyin.", nil)
		return err
	}
	pendingInputs.Lock()
	delete(pendingInputs.values, msg.SenderID())
	pendingInputs.Unlock()
	menu := &td.ReplyMarkupShowKeyboard{IsPersistent: true, ResizeKeyboard: true, InputFieldPlaceholder: "Hızlı menüden bir işlem seçin", Rows: [][]td.KeyboardButton{
		{{Text: "📦 Uygulamalar", Type: &td.KeyboardButtonTypeText{}}, {Text: "📊 Sistem Durumu", Type: &td.KeyboardButtonTypeText{}}},
		{{Text: "🗄 Veritabanları", Type: &td.KeyboardButtonTypeText{}}, {Text: "📅 Zamanlanmış İşler", Type: &td.KeyboardButtonTypeText{}}},
		{{Text: "👥 Telegram Yetkileri", Type: &td.KeyboardButtonTypeText{}}, {Text: "🖥 Web Kullanıcıları", Type: &td.KeyboardButtonTypeText{}}},
		{{Text: "➕ Telegram Kullanıcısı", Type: &td.KeyboardButtonTypeText{}}, {Text: "➕ Web Kullanıcısı", Type: &td.KeyboardButtonTypeText{}}},
		{{Text: "🌐 Web Panel", Type: &td.KeyboardButtonTypeText{}}},
	}}
	_, err := msg.ReplyText(c, "Hoş geldiniz. Yapmak istediğiniz işlemi aşağıdaki hızlı menüden seçin.", &td.SendTextMessageOpts{ReplyMarkup: menu})
	return err
}

var pendingInputs = struct {
	sync.Mutex
	values map[int64]pendingInput
}{values: map[int64]pendingInput{}}

type pendingInput struct{ Kind, First, Second string }

func cancelKeyboard() *td.ReplyMarkupInlineKeyboard {
	return &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{
		{Text: "✖️ İşlemi İptal Et", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("flow_cancel")}},
	}}}
}

func detailsFromPending(value string) string {
	parts := strings.SplitN(value, "|", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func scheduleInputKeyboard(uuid, action string) *td.ReplyMarkupInlineKeyboard {
	return &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{
		{{Text: "⬅️ Geri", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("flow_schedule_back:" + uuid + ":" + action)}}},
		{{Text: "✖️ İşlemi İptal Et", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("flow_cancel")}}},
	}}
}

func cancelCommandHandler(c *td.Client, msg *td.Message) error {
	pendingInputs.Lock()
	_, existed := pendingInputs.values[msg.SenderID()]
	delete(pendingInputs.values, msg.SenderID())
	pendingInputs.Unlock()
	if existed {
		_, err := msg.ReplyText(c, "✅ İşlem iptal edildi. Hızlı menüden devam edebilirsiniz.", nil)
		return err
	}
	_, err := msg.ReplyText(c, "Bekleyen bir işlem bulunmuyor.", nil)
	return err
}

func flowCancelHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	pendingInputs.Lock()
	delete(pendingInputs.values, cb.SenderUserId)
	pendingInputs.Unlock()
	_ = cb.Answer(c, 0, false, "İşlem iptal edildi", "")
	_, err := cb.EditMessageText(c, "✅ İşlem iptal edildi.\n\nHızlı menüden başka bir işlem seçebilirsiniz.", nil)
	return err
}

func scheduleBackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	pendingInputs.Lock()
	delete(pendingInputs.values, cb.SenderUserId)
	pendingInputs.Unlock()
	data := strings.TrimPrefix(cb.DataString(), "flow_schedule_back:")
	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return flowCancelHandler(c, cb)
	}
	_ = cb.Answer(c, 0, false, "Önceki adıma dönüldü", "")
	return showScheduleModes(c, cb, parts[0], parts[1])
}

func roleKeyboard(prefix string) *td.ReplyMarkupInlineKeyboard {
	return &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{
		{
			{Text: "👁 Görüntüleyici", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(prefix + ":viewer")}},
			{Text: "🛠 Operatör", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(prefix + ":operator")}},
		},
		{
			{Text: "🛡 Yönetici", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(prefix + ":admin")}},
		},
		{{Text: "✖️ İşlemi İptal Et", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("flow_cancel")}}},
	}}
}

func isQuickMenuText(text string) bool {
	items := []string{"📊 Sistem Durumu", "👥 Telegram Yetkileri", "🖥 Web Kullanıcıları", "➕ Telegram Kullanıcısı", "➕ Web Kullanıcısı", "📅 Zamanlanmış İşler", "🗄 Veritabanları", "🌐 Web Panel", "📦 Uygulamalar"}
	for _, item := range items {
		if text == item {
			return true
		}
	}
	return false
}

func quickMenuHandler(c *td.Client, msg *td.Message) error {
	if !config.IsDev(msg.SenderID()) {
		return nil
	}
	text := strings.TrimSpace(msg.Text())
	if strings.HasPrefix(text, "/") {
		return nil
	}
	pendingInputs.Lock()
	pending := pendingInputs.values[msg.SenderID()]
	if pending.Kind != "" && isQuickMenuText(text) {
		delete(pendingInputs.values, msg.SenderID())
		pending = pendingInput{}
	}
	pendingInputs.Unlock()
	if pending.Kind == "telegram_id" {
		id, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			_, e := msg.ReplyText(c, "Geçerli bir Telegram ID girin veya işlemi iptal edin.", &td.SendTextMessageOpts{ReplyMarkup: cancelKeyboard()})
			return e
		}
		pendingInputs.Lock()
		pendingInputs.values[msg.SenderID()] = pendingInput{Kind: "telegram_role", First: strconv.FormatInt(id, 10)}
		pendingInputs.Unlock()
		_, err = msg.ReplyText(c, "Bu kullanıcının rolünü seçin:", &td.SendTextMessageOpts{ReplyMarkup: roleKeyboard("new_tg_role")})
		return err
	}
	if pending.Kind == "web_credentials" {
		parts := strings.Fields(text)
		if len(parts) != 2 {
			_, err := msg.ReplyText(c, "Kullanıcı adı ve parolayı arada boşlukla yazın veya işlemi iptal edin.", &td.SendTextMessageOpts{ReplyMarkup: cancelKeyboard()})
			return err
		}
		pendingInputs.Lock()
		pendingInputs.values[msg.SenderID()] = pendingInput{Kind: "web_role", First: parts[0], Second: parts[1]}
		pendingInputs.Unlock()
		_, err := msg.ReplyText(c, "Web kullanıcısının rolünü seçin:", &td.SendTextMessageOpts{ReplyMarkup: roleKeyboard("new_web_role")})
		return err
	}
	if pending.Kind == "schedule_time" {
		location, err := time.LoadLocation("Europe/Istanbul")
		if err != nil {
			location = time.Local
		}
		runAt, err := time.ParseInLocation("02.01.2006 15:04", text, location)
		if err != nil || !runAt.After(time.Now()) {
			_, replyErr := msg.ReplyText(c, "Geçerli ve gelecekte bir tarih girin. Örnek: <code>15.07.2026 03:30</code>", &td.SendTextMessageOpts{ParseMode: "HTML", ReplyMarkup: scheduleInputKeyboard(pending.First, detailsFromPending(pending.Second))})
			return replyErr
		}
		details := strings.Split(pending.Second, "|")
		if len(details) != 2 {
			pendingInputs.Lock()
			delete(pendingInputs.values, msg.SenderID())
			pendingInputs.Unlock()
			_, replyErr := msg.ReplyText(c, "İşlem bilgileri geçersiz kaldı. Lütfen hızlı menüden yeniden başlayın.", nil)
			return replyErr
		}
		app, err := config.Coolify.GetApplicationByUUID(pending.First)
		if err != nil {
			_, replyErr := msg.ReplyText(c, "Uygulama bilgisi alınamadı: "+err.Error(), nil)
			return replyErr
		}
		schedules := map[string]string{"hourly": "every_1h", "daily": "every_24h", "weekly": "every_168h"}
		task := database.ScheduledTask{ID: uuidpkg.New().String(), Name: app.Name, ProjectUUID: pending.First, Type: details[0], NextRun: runAt}
		if details[1] == "once" {
			task.OneTime, task.Schedule = true, "one_time"
		} else {
			task.Schedule = schedules[details[1]]
		}
		if task.Schedule == "" {
			return nil
		}
		if err := database.AddTask(task); err != nil {
			_, replyErr := msg.ReplyText(c, "Görev kaydedilemedi: "+err.Error(), nil)
			return replyErr
		}
		if err := scheduler.ScheduleTask(task); err != nil {
			_ = database.DeleteTask(task.ID)
			_, replyErr := msg.ReplyText(c, "Görev zamanlanamadı: "+err.Error(), nil)
			return replyErr
		}
		pendingInputs.Lock()
		delete(pendingInputs.values, msg.SenderID())
		pendingInputs.Unlock()
		_, err = msg.ReplyText(c, fmt.Sprintf("✅ Görev kaydedildi\n\nUygulama: %s\nİşlem: %s\nBaşlangıç: %s\nTekrar: %s", app.Name, taskTypeLabel(details[0]), runAt.Format("02.01.2006 15:04"), taskScheduleLabel(task)), nil)
		return err
	}
	switch text {
	case "📊 Sistem Durumu":
		return pingHandler(c, msg)
	case "👥 Telegram Yetkileri":
		return listAuthorizedHandler(c, msg)
	case "🖥 Web Kullanıcıları":
		return listWebUsersHandler(c, msg)
	case "➕ Telegram Kullanıcısı":
		if !config.Can(msg.SenderID(), "users") {
			return nil
		}
		pendingInputs.Lock()
		pendingInputs.values[msg.SenderID()] = pendingInput{Kind: "telegram_id"}
		pendingInputs.Unlock()
		_, err := msg.ReplyText(c, "Eklemek istediğiniz kullanıcının Telegram ID'sini yazın.", &td.SendTextMessageOpts{ReplyMarkup: cancelKeyboard()})
		return err
	case "➕ Web Kullanıcısı":
		if !config.Can(msg.SenderID(), "users") {
			return nil
		}
		pendingInputs.Lock()
		pendingInputs.values[msg.SenderID()] = pendingInput{Kind: "web_credentials"}
		pendingInputs.Unlock()
		_, err := msg.ReplyText(c, "Kullanıcı adı ve parolayı yazın.\nÖrnek: <code>caner GucluParola123</code>", &td.SendTextMessageOpts{ParseMode: "HTML", ReplyMarkup: cancelKeyboard()})
		return err
	case "📅 Zamanlanmış İşler":
		return jobsHandler(c, msg)
	case "🗄 Veritabanları":
		return databasesHandler(c, msg)
	case "🌐 Web Panel":
		_, err := msg.ReplyText(c, "Web paneli: https://tg.flpanel.cloud", nil)
		return err
	case "📦 Uygulamalar":
		kb := &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{{{Text: "📦 Uygulamaları Aç", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("list_projects")}}}}}
		_, err := msg.ReplyText(c, "Uygulama listesini açmak için dokunun.", &td.SendTextMessageOpts{ReplyMarkup: kb})
		return err
	}
	return nil
}

func databasesHandler(c *td.Client, msg *td.Message) error {
	items, err := config.Coolify.ListDatabases()
	if err != nil {
		_, e := msg.ReplyText(c, "❌ Veritabanları alınamadı: "+err.Error(), nil)
		return e
	}
	if len(items) == 0 {
		_, e := msg.ReplyText(c, "📭 Veritabanı bulunamadı.", nil)
		return e
	}
	lines := []string{"<b>🗄 Veritabanları</b>"}
	for _, item := range items {
		project := item.Project
		if project == "" {
			project = "Diğer"
		}
		lines = append(lines, fmt.Sprintf("\n<b>%s › %s</b>\nDurum: <code>%s</code>\nİmaj: <code>%s</code>", project, item.Name, item.Status, item.Image))
	}
	_, err = msg.ReplyText(c, strings.Join(lines, "\n"), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}

func pingHandler(c *td.Client, msg *td.Message) error {
	if !config.IsDev(msg.SenderID()) {
		return nil
	}
	start := time.Now()
	m, err := msg.ReplyText(c, "⏱ Kontrol ediliyor...", nil)
	if err != nil {
		return err
	}
	apps, _ := config.Coolify.ListApplications()
	databases, _ := config.Coolify.ListDatabases()
	servers, _ := config.Coolify.ListServers()
	healthy, unhealthy := 0, []string{}
	for _, app := range apps {
		if strings.Contains(app.Status, "healthy") || strings.Contains(app.Status, "running") {
			healthy++
		} else {
			unhealthy = append(unhealthy, app.Name+" ("+app.Status+")")
		}
	}
	for _, db := range databases {
		if strings.Contains(db.Status, "healthy") || strings.Contains(db.Status, "running") {
			healthy++
		} else {
			unhealthy = append(unhealthy, db.Name+" ("+db.Status+")")
		}
	}
	serverStatus := "erişilebilir"
	if len(servers) > 0 && servers[0].ServerStatus != "" {
		serverStatus = string(servers[0].ServerStatus)
	}
	metrics := monitoring.Load()
	metricText := "📊 <b>Kaynak Kullanımı</b>\n"
	if metrics.Available {
		metricText += fmt.Sprintf("├ CPU: <b>%.1f%%</b>\n├ RAM: <b>%.1f%%</b> (%s / %s)\n", metrics.CPU, metrics.RAM, formatBytes(metrics.RAMUsed), formatBytes(metrics.RAMTotal))
	} else {
		metricText += "├ CPU: <b>N/A</b>\n├ RAM: <b>N/A</b>\n"
		if metrics.Error != "" {
			metricText += "├ Açıklama: <code>" + html.EscapeString(metrics.Error) + "</code>\n"
		}
	}
	metricText = strings.TrimSuffix(metricText, "\n")
	text := fmt.Sprintf("<b>📊 Sistem Durumu</b>\n\n%s\n\n🤖 Bot: <b>Çalışıyor</b>\n🖥 Sunucu: <b>%s</b>\n✅ Sağlıklı kaynak: <b>%d</b>\n📦 Uygulama/servis: <b>%d</b>\n🗄 Veritabanı: <b>%d</b>\n⏱ Gecikme: <code>%d ms</code>\n🕒 Çalışma süresi: <code>%s</code>\n⚙️ İş parçacıkları: <code>%d</code>", metricText, serverStatus, healthy, len(apps), len(databases), time.Since(start).Milliseconds(), time.Since(startTime).Truncate(time.Second), runtime.NumGoroutine())
	if len(unhealthy) > 0 {
		text += "\n\n<b>⚠️ Sorunlu Kaynaklar</b>\n• " + strings.Join(unhealthy, "\n• ")
	}
	_, err = m.EditText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML"})
	return err
}

func formatBytes(value uint64) string {
	const gib = 1024 * 1024 * 1024
	if value >= gib {
		return fmt.Sprintf("%.2f GB", float64(value)/gib)
	}
	const mib = 1024 * 1024
	return fmt.Sprintf("%.1f MB", float64(value)/mib)
}

func commandID(msg *td.Message) (int64, error) {
	parts := strings.Fields(msg.Text())
	if len(parts) < 2 {
		return 0, fmt.Errorf("kullanım: komut <telegram_id>")
	}
	return strconv.ParseInt(parts[1], 10, 64)
}
func addAuthorizedHandler(c *td.Client, msg *td.Message) error {
	if !config.Can(msg.SenderID(), "users") {
		_, e := msg.ReplyText(c, "🚫 Bu işlemi yapma yetkiniz yok.", nil)
		return e
	}
	parts := strings.Fields(msg.Text())
	id, err := commandID(msg)
	role := "operator"
	if len(parts) > 2 {
		role = parts[2]
	}
	if role != "viewer" && role != "operator" && role != "admin" {
		err = fmt.Errorf("rol viewer, operator veya admin olmalı")
	}
	if err == nil {
		err = database.AddAuthorizedUser(id, role)
	}
	if err != nil {
		_, e := msg.ReplyText(c, "❌ "+err.Error(), nil)
		return e
	}
	_, err = msg.ReplyText(c, fmt.Sprintf("✅ <code>%d</code> kullanıcısına <b>%s</b> rolü verildi.", id, role), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}
func removeAuthorizedHandler(c *td.Client, msg *td.Message) error {
	if !config.IsOwner(msg.SenderID()) {
		return nil
	}
	id, err := commandID(msg)
	if err == nil && id == config.OwnerID() {
		err = fmt.Errorf("ana yönetici silinemez")
	}
	if err == nil {
		err = database.RemoveAuthorizedUser(id)
	}
	if err != nil {
		_, e := msg.ReplyText(c, "❌ "+err.Error(), nil)
		return e
	}
	_, err = msg.ReplyText(c, "✅ Yetki kaldırıldı.", nil)
	return err
}
func listAuthorizedHandler(c *td.Client, msg *td.Message) error {
	if !config.Can(msg.SenderID(), "users") {
		return nil
	}
	text, keyboard, err := telegramUsersMenu()
	if err != nil {
		return err
	}
	_, err = msg.ReplyText(c, text, &td.SendTextMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func nextRole(role string) string {
	if role == "viewer" {
		return "operator"
	}
	if role == "operator" {
		return "admin"
	}
	return "viewer"
}

func telegramUsersMenu() (string, *td.ReplyMarkupInlineKeyboard, error) {
	rows, err := database.GetAuthorizedUserRecords()
	if err != nil {
		return "", nil, err
	}
	lines := []string{fmt.Sprintf("👑 <code>%d</code> — owner", config.OwnerID())}
	keyboard := &td.ReplyMarkupInlineKeyboard{}
	for _, user := range rows {
		if user.TelegramID == config.OwnerID() {
			continue
		}
		lines = append(lines, fmt.Sprintf("👤 <code>%d</code> — %s", user.TelegramID, user.Role))
		keyboard.Rows = append(keyboard.Rows, []td.InlineKeyboardButton{{Text: fmt.Sprintf("🔄 %d: %s", user.TelegramID, user.Role), Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("tg_role:%d:%s", user.TelegramID, nextRole(user.Role)))}}, {Text: "🗑 Sil", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte(fmt.Sprintf("tg_del:%d", user.TelegramID))}}})
	}
	return "<b>Telegram Yetkileri</b>\n\n" + strings.Join(lines, "\n") + "\n\nRol düğmesine basarak viewer → operator → admin geçişi yapın.\nYeni hesap: <code>/yetki_ekle ID rol</code>", keyboard, nil
}

func addWebUserHandler(c *td.Client, msg *td.Message) error {
	if !config.Can(msg.SenderID(), "users") {
		return nil
	}
	parts := strings.Fields(msg.Text())
	if len(parts) != 4 {
		_, err := msg.ReplyText(c, "Kullanım: <code>/web_ekle kullanici parola viewer|operator|admin</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}
	if err := database.AddWebUser(parts[1], parts[2], parts[3]); err != nil {
		_, e := msg.ReplyText(c, "❌ "+err.Error(), nil)
		return e
	}
	_, err := msg.ReplyText(c, "✅ Web kullanıcısı eklendi veya güncellendi: <b>"+parts[1]+"</b>", &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}

func removeWebUserHandler(c *td.Client, msg *td.Message) error {
	if !config.Can(msg.SenderID(), "users") {
		return nil
	}
	parts := strings.Fields(msg.Text())
	if len(parts) != 2 {
		_, err := msg.ReplyText(c, "Kullanım: <code>/web_sil kullanici</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	}
	if err := database.RemoveWebUser(parts[1]); err != nil {
		return err
	}
	_, err := msg.ReplyText(c, "✅ Web kullanıcısı silindi.", nil)
	return err
}

func listWebUsersHandler(c *td.Client, msg *td.Message) error {
	if !config.Can(msg.SenderID(), "users") {
		return nil
	}
	text, keyboard := webUsersMenu()
	_, err := msg.ReplyText(c, text, &td.SendTextMessageOpts{ParseMode: "HTML", ReplyMarkup: keyboard})
	return err
}

func webUsersMenu() (string, *td.ReplyMarkupInlineKeyboard) {
	lines := []string{"<b>Web Panel Kullanıcıları</b>"}
	keyboard := &td.ReplyMarkupInlineKeyboard{}
	for _, user := range database.GetWebUsers() {
		lines = append(lines, fmt.Sprintf("👤 <code>%s</code> — %s", user.Username, user.Role))
		keyboard.Rows = append(keyboard.Rows, []td.InlineKeyboardButton{{Text: "🔄 " + user.Username + ": " + user.Role, Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("web_role:" + user.Username + ":" + nextRole(user.Role))}}, {Text: "🗑 Sil", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("web_del:" + user.Username)}}})
	}
	lines = append(lines, "", "Yeni hesap: <code>/web_ekle kullanici parola rol</code>")
	return strings.Join(lines, "\n"), keyboard
}

func telegramUserActionHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "users") {
		return cb.Answer(c, 0, true, "Yetkiniz yok.", "")
	}
	parts := strings.Split(cb.DataString(), ":")
	if len(parts) < 2 {
		return nil
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}
	if parts[0] == "tg_del" {
		if id == config.OwnerID() {
			return cb.Answer(c, 0, true, "Ana yönetici silinemez.", "")
		}
		err = database.RemoveAuthorizedUser(id)
	} else if len(parts) == 3 {
		err = database.AddAuthorizedUser(id, parts[2])
	}
	if err != nil {
		return err
	}
	text, kb, err := telegramUsersMenu()
	if err != nil {
		return err
	}
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func webUserActionHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "users") {
		return cb.Answer(c, 0, true, "Yetkiniz yok.", "")
	}
	parts := strings.Split(cb.DataString(), ":")
	if len(parts) < 2 {
		return nil
	}
	username := parts[1]
	var err error
	if parts[0] == "web_del" {
		err = database.RemoveWebUser(username)
	} else if len(parts) == 3 {
		err = database.UpdateWebUserRole(username, parts[2])
	}
	if err != nil {
		return err
	}
	text, kb := webUsersMenu()
	_, err = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
	return err
}

func newUserRoleHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	if !config.Can(cb.SenderUserId, "users") {
		return cb.Answer(c, 0, true, "Yetkiniz yok.", "")
	}
	parts := strings.Split(cb.DataString(), ":")
	if len(parts) != 2 {
		return nil
	}
	role := parts[1]
	pendingInputs.Lock()
	pending := pendingInputs.values[cb.SenderUserId]
	delete(pendingInputs.values, cb.SenderUserId)
	pendingInputs.Unlock()
	var err error
	if parts[0] == "new_tg_role" && pending.Kind == "telegram_role" {
		id, e := strconv.ParseInt(pending.First, 10, 64)
		if e != nil {
			return e
		}
		err = database.AddAuthorizedUser(id, role)
	} else if parts[0] == "new_web_role" && pending.Kind == "web_role" {
		err = database.AddWebUser(pending.First, pending.Second, role)
	} else {
		return cb.Answer(c, 0, true, "Oturum süresi doldu, ekleme işlemini yeniden başlatın.", "")
	}
	if err != nil {
		_, _ = cb.EditMessageText(c, "❌ "+err.Error(), nil)
		return nil
	}
	_, err = cb.EditMessageText(c, "✅ Kullanıcı başarıyla eklendi.", nil)
	return err
}
