package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"coolifymanager/src/config"
	"coolifymanager/src/database"
)

type panelData struct {
	Apps          any
	TelegramUsers []database.AuthorizedUser
	WebUsers      []database.WebUser
	Databases     any
	Servers       any
	OwnerID       int64
	Username      string
	Role          string
	Message       string
}

var panelTemplate = template.Must(template.New("panel").Funcs(template.FuncMap{"isService": func(uuid string) bool { return strings.HasPrefix(uuid, "svc:") }}).Parse(`<!doctype html>
<html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>FL Panel · Coolify</title><style>
:root{--bg:#070b16;--panel:#10182a;--line:#25314b;--text:#f2f5fb;--muted:#91a0bb;--brand:#7868ff;--ok:#42d392;--danger:#ef5c70}*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at 15% 0,#172447 0,transparent 35%),var(--bg);color:var(--text);font:14px Inter,system-ui,sans-serif}header{position:sticky;top:0;z-index:2;display:flex;justify-content:space-between;align-items:center;padding:18px 5vw;background:#070b16dd;backdrop-filter:blur(16px);border-bottom:1px solid var(--line)}.brand{font-size:19px;font-weight:800}.pill{padding:7px 11px;border:1px solid var(--line);border-radius:99px;color:var(--muted)}main{max-width:1280px;margin:auto;padding:34px 5vw}.hero{display:flex;justify-content:space-between;gap:20px;align-items:end;margin-bottom:28px}h1{font-size:clamp(28px,5vw,48px);margin:0}.muted{color:var(--muted)}nav{display:flex;gap:8px;margin:22px 0;flex-wrap:wrap}nav a,.btn,button{color:white;background:#1a2440;border:1px solid #303d5d;border-radius:10px;padding:10px 13px;text-decoration:none;cursor:pointer}.primary{background:var(--brand);border-color:var(--brand)}.danger{background:#381b27;border-color:#6b2b3d;color:#ff9aaa}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(285px,1fr));gap:15px}.card,.section{background:linear-gradient(145deg,#121b30,#0e1526);border:1px solid var(--line);border-radius:17px;padding:18px;box-shadow:0 16px 50px #0004}.section{margin-top:24px}.app-name{font-size:17px;font-weight:750}.status{color:var(--ok);margin:9px 0}.url{color:#94b8ff;word-break:break-all;min-height:20px}.actions{display:flex;gap:7px;flex-wrap:wrap;margin-top:14px}form{margin:0}.manage{display:grid;grid-template-columns:1fr 1fr;gap:18px}.form-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:9px;margin:14px 0}input,select{width:100%;background:#0a1120;color:white;border:1px solid var(--line);border-radius:9px;padding:11px}.row{display:flex;justify-content:space-between;align-items:center;border-top:1px solid var(--line);padding:11px 2px}.message{padding:13px 16px;border-radius:10px;background:#1c2945;margin-bottom:18px}@media(max-width:760px){.manage{grid-template-columns:1fr}.hero{display:block}header{padding:14px 4vw}}
</style></head><body><header><div class="brand">⚡ FL Panel</div><div class="pill">👤 {{.Username}} · {{.Role}}</div></header><main>
<div class="hero"><div><div class="muted">COOLIFY CONTROL CENTER</div><h1>Altyapınız tek ekranda.</h1><p class="muted">Uygulamaları ve erişimleri güvenle yönetin.</p></div><div class="pill">● Sistem çevrimiçi</div></div>
<nav><a href="#apps">📦 Uygulamalar</a><a href="#resources">🗄 Kaynaklar</a>{{if ne .Role "viewer"}}<a href="#access">👥 Erişim Yönetimi</a>{{end}}<a href="/health">♡ Sağlık</a><a href="/logout">Çıkış</a></nav>
{{if .Message}}<div class="message">{{.Message}}</div>{{end}}
<section id="apps"><div class="grid">{{range .Apps}}<article class="card"><div class="app-name">{{.Name}}</div><div class="status">● {{.Status}}</div><div class="url">{{.FQDN}}</div><div class="actions">{{if ne $.Role "viewer"}}<form method="post" action="/action"><input type="hidden" name="uuid" value="{{.UUID}}">{{if not (isService .UUID)}}<button class="primary" name="op" value="deploy">Dağıt</button><button name="op" value="redeploy">♻️ Redeploy</button>{{end}}<button name="op" value="restart">Yeniden Başlat</button><button class="danger" name="op" value="stop">Durdur</button></form>{{end}}{{if not (isService .UUID)}}<a class="btn" href="/logs?uuid={{.UUID}}">Loglar</a>{{end}}</div></article>{{else}}<div class="card">Uygulama bulunamadı.</div>{{end}}</div></section>
<section class="section" id="resources"><h2>🗄 Veritabanları ve Sunucular</h2><div class="grid">{{range .Databases}}<article class="card"><div class="app-name">{{.Name}}</div><div class="status">● {{.Status}}</div><div class="muted">{{.DatabaseType}} · {{.Image}}</div><div class="muted">CPU limiti: {{if .LimitsCPUs}}{{.LimitsCPUs}}{{else}}Sınırsız{{end}} · RAM limiti: {{if .LimitsMemory}}{{.LimitsMemory}}{{else}}Sınırsız{{end}}</div></article>{{else}}<div class="card muted">Veritabanı kaydı bulunamadı.</div>{{end}}{{range .Servers}}<article class="card"><div class="app-name">🖥 {{.Name}}</div><div class="status">● {{if .Status}}{{.Status}}{{else}}{{.ServerStatus}}{{end}}</div><div class="muted">{{.IP}}</div></article>{{end}}</div><p class="muted">Disk boyutu ve anlık CPU/RAM tüketimi mevcut Coolify genel API yanıtında bulunmuyor; API desteklediğinde bu kartlara eklenebilir.</p></section>
{{if eq .Role "admin"}}<section class="manage" id="access"><div class="section"><h2>📱 Telegram Yetkileri</h2><p class="muted">Ana yönetici: {{.OwnerID}}</p><form class="form-grid" method="post" action="/telegram-users"><input name="id" inputmode="numeric" placeholder="Telegram ID" required><select name="role"><option value="viewer">Görüntüleyici</option><option value="operator">Operatör</option><option value="admin">Yönetici</option></select><button class="primary" name="op" value="save">Ekle / Güncelle</button></form>{{range .TelegramUsers}}<div class="row"><span><b>{{.TelegramID}}</b> · {{.Role}}</span><form method="post" action="/telegram-users"><input type="hidden" name="id" value="{{.TelegramID}}"><button class="danger" name="op" value="delete">Sil</button></form></div>{{end}}</div>
<div class="section"><h2>🖥️ Web Kullanıcıları</h2><p class="muted">Panel hesaplarını ve rollerini yönetin.</p><form class="form-grid" method="post" action="/web-users"><input name="username" placeholder="Kullanıcı adı" required><input type="password" name="password" minlength="8" placeholder="Parola (en az 8)" required><select name="role"><option value="viewer">Görüntüleyici</option><option value="operator">Operatör</option><option value="admin">Yönetici</option></select><button class="primary" name="op" value="save">Ekle / Güncelle</button></form>{{range .WebUsers}}<div class="row"><span><b>{{.Username}}</b> · {{.Role}}</span><form method="post" action="/web-users"><input type="hidden" name="username" value="{{.Username}}"><button class="danger" name="op" value="delete">Sil</button></form></div>{{end}}</div></section>{{end}}
</main></body></html>`))

var loginTemplate = template.Must(template.New("login").Parse(`<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>FL Panel Giriş</title><style>body{margin:0;min-height:100vh;display:grid;place-items:center;background:radial-gradient(circle at 20% 0,#263b76,transparent 40%),#070b16;color:#f4f7ff;font:15px system-ui}.box{width:min(420px,90vw);padding:32px;background:#10182add;border:1px solid #2d3b5c;border-radius:22px;box-shadow:0 30px 90px #0008}h1{font-size:32px;margin:8px 0}p{color:#9ba9c4}input,button{width:100%;box-sizing:border-box;padding:13px;margin-top:11px;border-radius:11px;border:1px solid #334263;background:#0a1120;color:white}button{background:#7868ff;border:0;font-weight:700;cursor:pointer}.error{color:#ff91a2}</style></head><body><form class="box" method="post"><div>⚡ FL PANEL</div><h1>Tekrar hoş geldiniz.</h1><p>Coolify yönetim merkezine giriş yapın.</p>{{if .}}<div class="error">{{.}}</div>{{end}}<input name="username" autocomplete="username" placeholder="Kullanıcı adı" required autofocus><input type="password" name="password" autocomplete="current-password" placeholder="Parola" required><button>Giriş Yap</button></form></body></html>`))

func startWebPanel() {
	bootstrapUser, bootstrapPassword := os.Getenv("WEB_USER"), os.Getenv("WEB_PASSWORD")
	if bootstrapUser == "" {
		bootstrapUser = "admin"
	}
	if bootstrapPassword == "" {
		log.Print("WEB_PASSWORD gerekli; web paneli devre dışı")
		return
	}

	type session struct {
		Username, Role string
		Expires        time.Time
	}
	sessions := map[string]session{}
	var sessionsMu sync.Mutex
	credentials := func(username, password string) (string, bool) {
		if subtle.ConstantTimeCompare([]byte(username), []byte(bootstrapUser)) == 1 && subtle.ConstantTimeCompare([]byte(password), []byte(bootstrapPassword)) == 1 {
			return "admin", true
		}
		return database.AuthenticateWebUser(username, password)
	}
	authenticate := func(r *http.Request) (string, string, bool) {
		if cookie, err := r.Cookie("flpanel_session"); err == nil {
			sessionsMu.Lock()
			current, ok := sessions[cookie.Value]
			sessionsMu.Unlock()
			if ok && time.Now().Before(current.Expires) {
				return current.Username, current.Role, true
			}
		}
		username, password, ok := r.BasicAuth()
		if !ok {
			return "", "", false
		}
		role, ok := credentials(username, password)
		return username, role, ok
	}
	wrap := func(minRole string, next func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			username, role, ok := authenticate(r)
			allowed := ok && (minRole == "viewer" || role == "admin" || (minRole == "operator" && role == "operator"))
			if !allowed {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next(w, r, username, role)
		}
	}
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = loginTemplate.Execute(w, nil)
			return
		}
		username, password := r.FormValue("username"), r.FormValue("password")
		role, ok := credentials(username, password)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			_ = loginTemplate.Execute(w, "Kullanıcı adı veya parola hatalı.")
			return
		}
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		token := hex.EncodeToString(b)
		sessionsMu.Lock()
		sessions[token] = session{Username: username, Role: role, Expires: time.Now().Add(12 * time.Hour)}
		sessionsMu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "flpanel_session", Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 43200})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie("flpanel_session"); err == nil {
			sessionsMu.Lock()
			delete(sessions, cookie.Value)
			sessionsMu.Unlock()
		}
		http.SetCookie(w, &http.Cookie{Name: "flpanel_session", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true})
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	http.HandleFunc("/", wrap("viewer", func(w http.ResponseWriter, r *http.Request, username, role string) {
		apps, err := config.Coolify.ListApplications()
		databases, _ := config.Coolify.ListDatabases()
		servers, _ := config.Coolify.ListServers()
		message := ""
		if err != nil {
			message = err.Error()
		}
		rows, _ := database.GetAuthorizedUserRecords()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = panelTemplate.Execute(w, panelData{Apps: apps, Databases: databases, Servers: servers, TelegramUsers: rows, WebUsers: database.GetWebUsers(), OwnerID: config.OwnerID(), Username: username, Role: role, Message: message})
	}))
	http.HandleFunc("/action", wrap("operator", func(w http.ResponseWriter, r *http.Request, _, _ string) {
		if r.Method != http.MethodPost {
			http.Error(w, "Geçersiz yöntem", 405)
			return
		}
		id, op := r.FormValue("uuid"), r.FormValue("op")
		var err error
		if strings.HasPrefix(id, "svc:") {
			if op == "deploy" {
				op = "restart"
			}
			err = config.Coolify.ServiceAction(id, op)
		} else {
			switch op {
			case "deploy":
				_, err = config.Coolify.StartApplicationDeployment(id, false, false)
			case "redeploy":
				_, err = config.Coolify.StartApplicationDeployment(id, true, false)
			case "restart":
				_, err = config.Coolify.RestartApplicationByUUID(id)
			case "stop":
				_, err = config.Coolify.StopApplicationByUUID(id)
			default:
				err = fmt.Errorf("geçersiz işlem")
			}
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/", 303)
	}))
	http.HandleFunc("/logs", wrap("viewer", func(w http.ResponseWriter, r *http.Request, _, _ string) {
		logs, err := config.Coolify.GetApplicationLogsByUUID(r.URL.Query().Get("uuid"))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, strings.TrimSpace(logs))
	}))
	http.HandleFunc("/telegram-users", wrap("admin", func(w http.ResponseWriter, r *http.Request, _, _ string) {
		id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
		if err != nil {
			http.Error(w, "Geçersiz Telegram ID", 400)
			return
		}
		if r.FormValue("op") == "delete" {
			if id == config.OwnerID() {
				http.Error(w, "Ana yönetici silinemez", 400)
				return
			}
			err = database.RemoveAuthorizedUser(id)
		} else {
			err = database.AddAuthorizedUser(id, r.FormValue("role"))
		}
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		http.Redirect(w, r, "/#access", 303)
	}))
	http.HandleFunc("/web-users", wrap("admin", func(w http.ResponseWriter, r *http.Request, _, _ string) {
		var err error
		if r.FormValue("op") == "delete" {
			err = database.RemoveWebUser(r.FormValue("username"))
		} else {
			err = database.AddWebUser(r.FormValue("username"), r.FormValue("password"), r.FormValue("role"))
		}
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		http.Redirect(w, r, "/#access", 303)
	}))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Web paneli :%s portunda dinleniyor", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
