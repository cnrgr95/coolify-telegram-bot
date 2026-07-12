package src

import (
	"coolifymanager/src/config"
	"coolifymanager/src/database"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func startHandler(c *td.Client, msg *td.Message) error {
	if !config.IsDev(msg.SenderID()) {
		_, err := msg.ReplyText(c, "🚫 Bu botu kullanma yetkiniz yok. Yöneticiden Telegram ID'nizi eklemesini isteyin.", nil)
		return err
	}
	menu := &td.ReplyMarkupShowKeyboard{IsPersistent: true, ResizeKeyboard: true, InputFieldPlaceholder: "Hızlı menüden bir işlem seçin", Rows: [][]td.KeyboardButton{
		{{Text: "📦 Uygulamalar", Type: &td.KeyboardButtonTypeText{}}, {Text: "📊 Sistem Durumu", Type: &td.KeyboardButtonTypeText{}}},
		{{Text: "👥 Telegram Yetkileri", Type: &td.KeyboardButtonTypeText{}}, {Text: "🖥 Web Kullanıcıları", Type: &td.KeyboardButtonTypeText{}}},
		{{Text: "➕ Telegram Kullanıcısı", Type: &td.KeyboardButtonTypeText{}}, {Text: "➕ Web Kullanıcısı", Type: &td.KeyboardButtonTypeText{}}},
		{{Text: "📅 Zamanlanmış İşler", Type: &td.KeyboardButtonTypeText{}}, {Text: "🌐 Web Panel", Type: &td.KeyboardButtonTypeText{}}},
	}}
	_, err := msg.ReplyText(c, "Hoş geldiniz. Yapmak istediğiniz işlemi aşağıdaki hızlı menüden seçin.", &td.SendTextMessageOpts{ReplyMarkup: menu})
	return err
}

var pendingInputs = struct {
	sync.Mutex
	values map[int64]string
}{values: map[int64]string{}}

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
	if pending != "" {
		delete(pendingInputs.values, msg.SenderID())
	}
	pendingInputs.Unlock()
	if pending == "telegram_add" {
		parts := strings.Fields(text)
		if len(parts) != 2 {
			_, err := msg.ReplyText(c, "Geçersiz format. Örnek: <code>123456789 viewer</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
			return err
		}
		id, err := strconv.ParseInt(parts[0], 10, 64)
		if err == nil {
			err = database.AddAuthorizedUser(id, parts[1])
		}
		if err != nil {
			_, e := msg.ReplyText(c, "❌ "+err.Error(), nil)
			return e
		}
		_, err = msg.ReplyText(c, "✅ Telegram kullanıcısı eklendi.", nil)
		return err
	}
	if pending == "web_add" {
		parts := strings.Fields(text)
		if len(parts) != 3 {
			_, err := msg.ReplyText(c, "Geçersiz format. Örnek: <code>kullanici GucluParola123 viewer</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
			return err
		}
		err := database.AddWebUser(parts[0], parts[1], parts[2])
		if err != nil {
			_, e := msg.ReplyText(c, "❌ "+err.Error(), nil)
			return e
		}
		_, err = msg.ReplyText(c, "✅ Web kullanıcısı eklendi.", nil)
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
		pendingInputs.values[msg.SenderID()] = "telegram_add"
		pendingInputs.Unlock()
		_, err := msg.ReplyText(c, "Telegram ID ve rolü yazın.\nÖrnek: <code>123456789 viewer</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	case "➕ Web Kullanıcısı":
		if !config.Can(msg.SenderID(), "users") {
			return nil
		}
		pendingInputs.Lock()
		pendingInputs.values[msg.SenderID()] = "web_add"
		pendingInputs.Unlock()
		_, err := msg.ReplyText(c, "Kullanıcı adı, parola ve rolü yazın.\nÖrnek: <code>caner GucluParola123 operator</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
		return err
	case "📅 Zamanlanmış İşler":
		return jobsHandler(c, msg)
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
		serverStatus = servers[0].ServerStatus
	}
	text := fmt.Sprintf("<b>📊 Sistem Durumu</b>\n\n🤖 Bot: <b>Çalışıyor</b>\n🖥 Sunucu: <b>%s</b>\n✅ Sağlıklı kaynak: <b>%d</b>\n📦 Uygulama/servis: <b>%d</b>\n🗄 Veritabanı: <b>%d</b>\n⏱ Gecikme: <code>%d ms</code>\n🕒 Çalışma süresi: <code>%s</code>\n⚙️ İş parçacıkları: <code>%d</code>", serverStatus, healthy, len(apps), len(databases), time.Since(start).Milliseconds(), time.Since(startTime).Truncate(time.Second), runtime.NumGoroutine())
	if len(unhealthy) > 0 {
		text += "\n\n<b>⚠️ Sorunlu Kaynaklar</b>\n• " + strings.Join(unhealthy, "\n• ")
	}
	_, err = m.EditText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML"})
	return err
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
