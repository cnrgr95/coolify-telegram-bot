package src

import (
	"coolifymanager/src/config"
	"coolifymanager/src/database"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

func startHandler(c *td.Client, msg *td.Message) error {
	if !config.IsDev(msg.SenderID()) {
		_, err := msg.ReplyText(c, "🚫 Bu botu kullanma yetkiniz yok. Yöneticiden Telegram ID'nizi eklemesini isteyin.", nil)
		return err
	}
	text := "<b>🎛 FL Panel Coolify Yönetimi</b>\n\nUygulamalarınızı Telegram ve web panelinden güvenle yönetebilirsiniz."
	kb := &td.ReplyMarkupInlineKeyboard{Rows: [][]td.InlineKeyboardButton{
		{{Text: "📦 Uygulamalar", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("list_projects")}}},
		{{Text: "📅 Zamanlanmış İşler", Type: &td.InlineKeyboardButtonTypeCallback{Data: []byte("jobs:1")}}},
	}}
	_, err := msg.ReplyText(c, text, &td.SendTextMessageOpts{ParseMode: "HTML", ReplyMarkup: kb})
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
	text := fmt.Sprintf("<b>📊 Sistem Durumu</b>\n\n✅ Bot çalışıyor\n⏱ Gecikme: <code>%d ms</code>\n🕒 Çalışma süresi: <code>%s</code>\n⚙️ Go iş parçacıkları: <code>%d</code>", time.Since(start).Milliseconds(), time.Since(startTime).Truncate(time.Second), runtime.NumGoroutine())
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
	if !config.IsDev(msg.SenderID()) {
		return nil
	}
	rows, err := database.GetAuthorizedUserRecords()
	if err != nil {
		return err
	}
	lines := []string{fmt.Sprintf("👑 <code>%d</code> (Ana yönetici)", config.OwnerID())}
	for _, u := range rows {
		if u.TelegramID != config.OwnerID() {
			lines = append(lines, fmt.Sprintf("👤 <code>%d</code> — %s", u.TelegramID, u.Role))
		}
	}
	_, err = msg.ReplyText(c, "<b>Yetkili Kullanıcılar</b>\n\n"+strings.Join(lines, "\n")+"\n\n<code>/yetki_ekle ID viewer|operator|admin</code>", &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
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
	lines := []string{"<b>Web Panel Kullanıcıları</b>"}
	for _, user := range database.GetWebUsers() {
		lines = append(lines, fmt.Sprintf("👤 <code>%s</code> — %s", user.Username, user.Role))
	}
	lines = append(lines, "", "<code>/web_ekle kullanici parola rol</code>", "<code>/web_sil kullanici</code>")
	_, err := msg.ReplyText(c, strings.Join(lines, "\n"), &td.SendTextMessageOpts{ParseMode: "HTML"})
	return err
}
