package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"coolifymanager/src/config"
	"coolifymanager/src/coolity"
	"coolifymanager/src/database"
	"coolifymanager/src/scheduler"
	uid "github.com/google/uuid"
)

type panelData struct {
	Apps          []projectGroup
	TelegramUsers []database.AuthorizedUser
	WebUsers      []database.WebUser
	Databases     any
	Servers       any
	OwnerID       int64
	Username      string
	Role          string
	Message       string
	Page          string
	Total         int
	Healthy       int
	Issues        int
	DBCount       int
	HealthPercent int
	Metrics       systemMetrics
	Tasks         []database.ScheduledTask
}

type systemMetrics struct {
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	RAMUsed   uint64  `json:"ram_used"`
	RAMTotal  uint64  `json:"ram_total"`
	Available bool    `json:"available"`
}

func loadSystemMetrics() systemMetrics {
	bases := []string{os.Getenv("SENTINEL_URL")}
	if bases[0] == "" {
		bases = []string{"http://host.docker.internal:8000", "http://host.docker.internal:8888", "http://coolify-sentinel:8888"}
	}
	client := &http.Client{Timeout: 3 * time.Second}
	for _, base := range bases {
		var result systemMetrics
		if response, err := client.Get(strings.TrimRight(base, "/") + "/api/cpu/current"); err == nil {
			var data struct {
				Percent float64 `json:"percent"`
			}
			if json.NewDecoder(response.Body).Decode(&data) == nil {
				result.CPU = data.Percent
				result.Available = true
			}
			response.Body.Close()
		}
		if response, err := client.Get(strings.TrimRight(base, "/") + "/api/memory/current"); err == nil {
			var data struct {
				UsedPercent float64 `json:"usedPercent"`
				Used        uint64  `json:"used"`
				Total       uint64  `json:"total"`
			}
			if json.NewDecoder(response.Body).Decode(&data) == nil {
				result.RAM = data.UsedPercent
				result.RAMUsed = data.Used
				result.RAMTotal = data.Total
				result.Available = true
			}
			response.Body.Close()
		}
		if result.Available {
			return result
		}
	}
	return systemMetrics{}
}

type projectGroup struct {
	Name string
	Apps []coolify.Application
}

func groupApplications(apps []coolify.Application) []projectGroup {
	grouped := map[string][]coolify.Application{}
	for _, app := range apps {
		name := app.Project
		if name == "" {
			name = "Diğer"
		}
		grouped[name] = append(grouped[name], app)
	}
	names := make([]string, 0, len(grouped))
	for name := range grouped {
		names = append(names, name)
	}
	sort.Strings(names)
	groups := make([]projectGroup, 0, len(names))
	for _, name := range names {
		sort.Slice(grouped[name], func(i, j int) bool { return grouped[name][i].Name < grouped[name][j].Name })
		groups = append(groups, projectGroup{Name: name, Apps: grouped[name]})
	}
	return groups
}

func webTaskType(value string) string {
	labels := map[string]string{"restart": "Yeniden Başlat", "redeploy": "Redeploy", "stop": "Durdur", "delete": "Sil"}
	if label := labels[value]; label != "" {
		return label
	}
	return value
}

func webTaskSchedule(task database.ScheduledTask) string {
	if task.OneTime {
		return "Tek seferlik"
	}
	labels := map[string]string{"every_1h": "Saatlik", "every_24h": "Günlük", "every_168h": "Haftalık"}
	if label := labels[task.Schedule]; label != "" {
		return label
	}
	return task.Schedule
}

var panelTemplate = template.Must(template.New("panel").Funcs(template.FuncMap{
	"isService":    func(uuid string) bool { return strings.HasPrefix(uuid, "svc:") },
	"taskType":     webTaskType,
	"taskSchedule": webTaskSchedule,
}).Parse(`<!doctype html>
<html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>FL Panel · Coolify</title><style>
:root{--bg:#070b16;--panel:#10182a;--line:#25314b;--text:#f2f5fb;--muted:#91a0bb;--brand:#7868ff;--ok:#42d392;--danger:#ef5c70}*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at 15% 0,#172447 0,transparent 35%),var(--bg);color:var(--text);font:14px Inter,system-ui,sans-serif}header{position:sticky;top:0;z-index:2;display:flex;justify-content:space-between;align-items:center;padding:18px 5vw;background:#070b16dd;backdrop-filter:blur(16px);border-bottom:1px solid var(--line)}.brand{font-size:19px;font-weight:800}.pill{padding:7px 11px;border:1px solid var(--line);border-radius:99px;color:var(--muted)}main{max-width:1280px;margin:auto;padding:34px 5vw}.hero{display:flex;justify-content:space-between;gap:20px;align-items:end;margin-bottom:28px}h1{font-size:clamp(28px,5vw,48px);margin:0}.muted{color:var(--muted)}nav{display:flex;gap:8px;margin:22px 0;flex-wrap:wrap}nav a,.btn,button{color:white;background:#1a2440;border:1px solid #303d5d;border-radius:10px;padding:10px 13px;text-decoration:none;cursor:pointer}.primary{background:var(--brand);border-color:var(--brand)}.danger{background:#381b27;border-color:#6b2b3d;color:#ff9aaa}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(285px,1fr));gap:15px}.card,.section{background:linear-gradient(145deg,#121b30,#0e1526);border:1px solid var(--line);border-radius:17px;padding:18px;box-shadow:0 16px 50px #0004}.section{margin-top:24px}.app-name{font-size:17px;font-weight:750}.status{color:var(--ok);margin:9px 0}.url{color:#94b8ff;word-break:break-all;min-height:20px}.actions{display:flex;gap:7px;flex-wrap:wrap;margin-top:14px}form{margin:0}.manage{display:grid;grid-template-columns:1fr 1fr;gap:18px}.form-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(150px,1fr));gap:9px;margin:14px 0}input,select{width:100%;background:#0a1120;color:white;border:1px solid var(--line);border-radius:9px;padding:11px}.row{display:flex;justify-content:space-between;align-items:center;border-top:1px solid var(--line);padding:11px 2px}.message{padding:13px 16px;border-radius:10px;background:#1c2945;margin-bottom:18px}@media(max-width:760px){.manage{grid-template-columns:1fr}.hero{display:block}header{padding:14px 4vw}}
</style></head><body><header><div class="brand">⚡ FL Panel</div><div class="pill">👤 {{.Username}} · {{.Role}}</div></header><main>
<div class="hero"><div><div class="muted">COOLIFY CONTROL CENTER</div><h1>Altyapınız tek ekranda.</h1><p class="muted">Uygulamaları ve erişimleri güvenle yönetin.</p></div><div class="pill">● Sistem çevrimiçi</div></div>
<nav><a href="/">📦 Uygulamalar</a><a href="/resources">🗄 Kaynaklar</a><a href="/status">📊 Sistem Durumu</a>{{if ne .Role "viewer"}}<a href="/schedules">📅 Zamanlanmış Görevler</a>{{end}}{{if eq .Role "admin"}}<a href="/access">👥 Erişim Yönetimi</a>{{end}}<a href="/logout">Çıkış</a></nav>
{{if .Message}}<div class="message">{{.Message}}</div>{{end}}
{{if eq .Page "status"}}<section class="grid"><article class="card"><div class="muted">CPU Kullanımı</div><div id="cpu-value" style="font-size:34px;font-weight:800">{{if .Metrics.Available}}{{printf "%.1f" .Metrics.CPU}}%{{else}}N/A{{end}}</div><div class="status-bar"><div id="cpu-bar" style="width:{{.Metrics.CPU}}%"></div></div></article><article class="card"><div class="muted">RAM Kullanımı</div><div id="ram-value" style="font-size:34px;font-weight:800">{{if .Metrics.Available}}{{printf "%.1f" .Metrics.RAM}}%{{else}}N/A{{end}}</div><div class="status-bar"><div id="ram-bar" style="width:{{.Metrics.RAM}}%"></div></div></article><article class="card"><div class="muted">Disk Kullanımı</div><div style="font-size:34px;font-weight:800">N/A</div><small class="muted">Sentinel anık disk verisi sunmuyor.</small></article><article class="card"><div class="muted">Ağ / İnternet</div><div style="font-size:34px;font-weight:800">N/A</div><small class="muted">Sentinel anık ağ verisi sunmuyor.</small></article></section><section class="grid" style="margin-top:16px"><article class="card"><div class="muted">Toplam Kaynak</div><div style="font-size:28px;font-weight:800">{{.Total}}</div></article><article class="card"><div class="muted">Sağlıklı</div><div style="font-size:28px;font-weight:800;color:var(--ok)">{{.Healthy}}</div></article><article class="card"><div class="muted">Sorunlu</div><div style="font-size:28px;font-weight:800;color:var(--danger)">{{.Issues}}</div></article></section>{{end}}
{{if eq .Page "apps"}}<section id="apps">{{range .Apps}}<div class="section"><h2>🗂 {{.Name}}</h2><div class="grid">{{range .Apps}}<article class="card live-resource" data-id="{{.UUID}}"><div class="app-name">{{.Name}}</div><div class="status">● {{.Status}}</div><div class="url">{{.FQDN}}</div><div class="actions">{{if ne $.Role "viewer"}}<form method="post" action="/action"><input type="hidden" name="uuid" value="{{.UUID}}">{{if not (isService .UUID)}}<button class="primary" name="op" value="deploy">🚀 Dağıt</button><button name="op" value="redeploy">♻️ Redeploy</button>{{end}}<button name="op" value="restart">🔄 Yeniden Başlat</button><button class="danger" name="op" value="stop">⏹ Durdur</button>{{if and (eq $.Role "admin") (not (isService .UUID))}}<button class="danger" name="op" value="delete" onclick="return confirm('Bu uygulamayı kalıcı olarak silmek istediğinize emin misiniz?')">🗑 Sil</button>{{end}}</form>{{end}}{{if not (isService .UUID)}}<a class="btn" href="/logs?uuid={{.UUID}}">📜 Loglar</a>{{end}}</div></article>{{end}}</div></div>{{else}}<div class="card">Uygulama bulunamadı.</div>{{end}}</section>{{end}}
{{if eq .Page "schedules"}}<section class="section"><h2>📅 Yeni Zamanlanmış Görev</h2><p class="muted">İşlem, ilk çalışma zamanı ve tekrar biçimini seçin. Durdur ve Sil yalnızca tek sefer çalıştırılabilir.</p><form class="form-grid" method="post" action="/schedules"><select name="uuid" required><option value="">Uygulama seçin</option>{{range .Apps}}{{range .Apps}}{{if not (isService .UUID)}}<option value="{{.UUID}}">{{.Name}}</option>{{end}}{{end}}{{end}}</select><select name="action" required><option value="restart">Yeniden Başlat</option><option value="redeploy">Redeploy</option><option value="stop">Durdur</option>{{if eq .Role "admin"}}<option value="delete">Sil</option>{{end}}</select><select name="repeat" required><option value="once">Tek seferlik</option><option value="hourly">Saatlik</option><option value="daily">Günlük</option><option value="weekly">Haftalık</option></select><input type="datetime-local" name="run_at" required><button class="primary">Görevi Kaydet</button></form></section><section class="section"><h2>Aktif Görevler</h2>{{range .Tasks}}<div class="row"><span><b>{{.Name}}</b> · {{taskType .Type}} · {{taskSchedule .}} · {{.NextRun.Format "02.01.2006 15:04"}}</span><form method="post" action="/schedules"><input type="hidden" name="task_id" value="{{.ID}}"><button class="danger" name="op" value="delete">İptal Et</button></form></div>{{else}}<div class="card muted">Aktif zamanlanmış görev bulunmuyor.</div>{{end}}</section>{{end}}
{{if or (eq .Page "resources") (eq .Page "status")}}<section class="section" id="resources"><h2>🗄 Veritabanları ve Sunucular</h2><div class="grid">{{range .Databases}}<article class="card live-resource" data-id="{{.UUID}}"><div class="app-name">{{.Name}}</div><div class="status">● {{.Status}}</div><div class="muted">{{.DatabaseType}} · {{.Image}}</div><div class="muted">CPU limiti: {{if .LimitsCPUs}}{{.LimitsCPUs}}{{else}}Sınırsız{{end}} · RAM limiti: {{if .LimitsMemory}}{{.LimitsMemory}}{{else}}Sınırsız{{end}}</div></article>{{else}}<div class="card muted">Veritabanı kaydı bulunamadı.</div>{{end}}{{range .Servers}}<article class="card live-resource" data-id="{{.UUID}}"><div class="app-name">🖥 {{.Name}}</div><div class="status">● {{if .Status}}{{.Status}}{{else}}{{.ServerStatus}}{{end}}</div><div class="muted">{{.IP}}</div></article>{{end}}</div><p class="muted">Anlık CPU/RAM grafikleri için Coolify sunucusunda Metrics etkin olmalıdır.</p></section>{{end}}
{{if and (eq .Role "admin") (eq .Page "access")}}<section class="manage" id="access"><div class="section"><h2>📱 Telegram Yetkileri</h2><p class="muted">Ana yönetici: {{.OwnerID}}</p><form class="form-grid" method="post" action="/telegram-users"><input name="id" inputmode="numeric" placeholder="Telegram ID" required><select name="role"><option value="viewer">Görüntüleyici</option><option value="operator">Operatör</option><option value="admin">Yönetici</option></select><button class="primary" name="op" value="save">Ekle / Güncelle</button></form>{{range .TelegramUsers}}<div class="row"><span><b>{{.TelegramID}}</b> · {{.Role}}</span><form method="post" action="/telegram-users"><input type="hidden" name="id" value="{{.TelegramID}}"><button class="danger" name="op" value="delete">Sil</button></form></div>{{end}}</div>
<div class="section"><h2>🖥️ Web Kullanıcıları</h2><p class="muted">Panel hesaplarını ve rollerini yönetin.</p><form class="form-grid" method="post" action="/web-users"><input name="username" placeholder="Kullanıcı adı" required><input type="password" name="password" minlength="8" placeholder="Parola (en az 8)" required><select name="role"><option value="viewer">Görüntüleyici</option><option value="operator">Operatör</option><option value="admin">Yönetici</option></select><button class="primary" name="op" value="save">Ekle / Güncelle</button></form>{{range .WebUsers}}<div class="row"><span><b>{{.Username}}</b> · {{.Role}}</span><form method="post" action="/web-users"><input type="hidden" name="username" value="{{.Username}}"><button class="danger" name="op" value="delete">Sil</button></form></div>{{end}}</div></section>{{end}}
</main><script>let inventory=[...document.querySelectorAll('.live-resource')].map(x=>x.dataset.id).sort().join(',');async function refresh(){try{const r=await fetch('/api/dashboard',{headers:{Accept:'application/json'}});if(!r.ok)return;const d=await r.json();if(d.metrics&&d.metrics.available){const cpu=document.getElementById('cpu-value'),ram=document.getElementById('ram-value');if(cpu)cpu.textContent=d.metrics.cpu.toFixed(1)+'%';if(ram)ram.textContent=d.metrics.ram.toFixed(1)+'%'}const all=[...d.apps,...d.databases,...d.servers],next=all.map(x=>x.uuid).sort().join(',');if(next!==inventory){location.reload();return}for(const x of all){const card=document.querySelector('[data-id="'+CSS.escape(x.uuid)+'"]');if(!card)continue;const status=card.querySelector('.status');const value=x.status||x.server_status||'bilinmiyor';status.textContent='● '+value;status.style.color=value.includes('healthy')||value.includes('running')?'var(--ok)':'var(--danger)'}}catch(e){}}setInterval(refresh,5000);refresh();</script></body></html>`))

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
	type loginAttempt struct {
		Count        int
		BlockedUntil time.Time
	}
	sessions := map[string]session{}
	var sessionsMu sync.Mutex
	loginAttempts := map[string]loginAttempt{}
	var loginAttemptsMu sync.Mutex
	securityHeaders := func(w http.ResponseWriter) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	}
	clientIP := func(r *http.Request) string {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			return host
		}
		return r.RemoteAddr
	}
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
			securityHeaders(w)
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
		securityHeaders(w)
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = loginTemplate.Execute(w, nil)
			return
		}
		ip := clientIP(r)
		loginAttemptsMu.Lock()
		attempt := loginAttempts[ip]
		if time.Now().Before(attempt.BlockedUntil) {
			loginAttemptsMu.Unlock()
			w.WriteHeader(http.StatusTooManyRequests)
			_ = loginTemplate.Execute(w, "Çok fazla başarısız deneme. Lütfen 15 dakika sonra tekrar deneyin.")
			return
		}
		loginAttemptsMu.Unlock()
		username, password := r.FormValue("username"), r.FormValue("password")
		role, ok := credentials(username, password)
		if !ok {
			loginAttemptsMu.Lock()
			attempt = loginAttempts[ip]
			attempt.Count++
			if attempt.Count >= 5 {
				attempt.Count = 0
				attempt.BlockedUntil = time.Now().Add(15 * time.Minute)
			}
			loginAttempts[ip] = attempt
			loginAttemptsMu.Unlock()
			w.WriteHeader(http.StatusUnauthorized)
			_ = loginTemplate.Execute(w, "Kullanıcı adı veya parola hatalı.")
			return
		}
		loginAttemptsMu.Lock()
		delete(loginAttempts, ip)
		loginAttemptsMu.Unlock()
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
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept"), "text/html") {
			http.Redirect(w, r, "/status", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, `{"status":"ok"}`)
	})
	http.HandleFunc("/api/dashboard", wrap("viewer", func(w http.ResponseWriter, _ *http.Request, _, _ string) {
		apps, _ := config.Coolify.ListApplications()
		databases, _ := config.Coolify.ListDatabases()
		servers, _ := config.Coolify.ListServers()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"apps": apps, "databases": databases, "servers": servers, "metrics": loadSystemMetrics(), "updated_at": time.Now()})
	}))
	renderPanel := func(page string) func(http.ResponseWriter, *http.Request, string, string) {
		return func(w http.ResponseWriter, r *http.Request, username, role string) {
			apps, err := config.Coolify.ListApplications()
			databases, _ := config.Coolify.ListDatabases()
			servers, _ := config.Coolify.ListServers()
			message := ""
			if err != nil {
				message = err.Error()
			}
			rows, _ := database.GetAuthorizedUserRecords()
			tasks, _ := database.GetTasks()
			sort.Slice(tasks, func(i, j int) bool { return tasks[i].NextRun.Before(tasks[j].NextRun) })
			total := len(apps) + len(databases)
			healthy := 0
			for _, app := range apps {
				if strings.Contains(app.Status, "healthy") || strings.Contains(app.Status, "running") {
					healthy++
				}
			}
			for _, item := range databases {
				if strings.Contains(item.Status, "healthy") || strings.Contains(item.Status, "running") {
					healthy++
				}
			}
			percent := 0
			if total > 0 {
				percent = healthy * 100 / total
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = panelTemplate.Execute(w, panelData{Apps: groupApplications(apps), Databases: databases, Servers: servers, TelegramUsers: rows, WebUsers: database.GetWebUsers(), OwnerID: config.OwnerID(), Username: username, Role: role, Message: message, Page: page, Total: total, Healthy: healthy, Issues: total - healthy, DBCount: len(databases), HealthPercent: percent, Metrics: loadSystemMetrics(), Tasks: tasks})
		}
	}
	http.HandleFunc("/resources", wrap("viewer", renderPanel("resources")))
	http.HandleFunc("/status", wrap("viewer", renderPanel("status")))
	http.HandleFunc("/schedules", wrap("operator", func(w http.ResponseWriter, r *http.Request, username, role string) {
		if r.Method == http.MethodGet {
			renderPanel("schedules")(w, r, username, role)
			return
		}
		if r.FormValue("op") == "delete" {
			id := r.FormValue("task_id")
			_ = scheduler.RemoveTask(id)
			if err := database.DeleteTask(id); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/schedules", http.StatusSeeOther)
			return
		}
		action, repeat := r.FormValue("action"), r.FormValue("repeat")
		if action == "delete" && role != "admin" {
			http.Error(w, "Silme işlemi için yönetici yetkisi gerekir", http.StatusForbidden)
			return
		}
		if (action == "stop" || action == "delete") && repeat != "once" {
			http.Error(w, "Durdur ve Sil yalnızca tek seferlik zamanlanabilir", http.StatusBadRequest)
			return
		}
		location, locationErr := time.LoadLocation("Europe/Istanbul")
		if locationErr != nil {
			location = time.Local
		}
		runAt, err := time.ParseInLocation("2006-01-02T15:04", r.FormValue("run_at"), location)
		if err != nil || !runAt.After(time.Now()) {
			http.Error(w, "Çalışma zamanı gelecekte olmalıdır", http.StatusBadRequest)
			return
		}
		app, err := config.Coolify.GetApplicationByUUID(r.FormValue("uuid"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		schedules := map[string]string{"hourly": "every_1h", "daily": "every_24h", "weekly": "every_168h"}
		task := database.ScheduledTask{ID: uid.New().String(), Name: app.Name, ProjectUUID: app.UUID, Type: action, NextRun: runAt}
		if repeat == "once" {
			task.OneTime, task.Schedule = true, "one_time"
		} else {
			task.Schedule = schedules[repeat]
		}
		if task.Schedule == "" {
			http.Error(w, "Geçersiz tekrar biçimi", http.StatusBadRequest)
			return
		}
		if err := database.AddTask(task); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := scheduler.ScheduleTask(task); err != nil {
			_ = database.DeleteTask(task.ID)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/schedules", http.StatusSeeOther)
	}))
	http.HandleFunc("/access", wrap("admin", renderPanel("access")))
	http.HandleFunc("/", wrap("viewer", renderPanel("apps")))
	http.HandleFunc("/action", wrap("operator", func(w http.ResponseWriter, r *http.Request, _, role string) {
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
			case "delete":
				if role != "admin" {
					http.Error(w, "Silme işlemi için yönetici yetkisi gerekir", http.StatusForbidden)
					return
				}
				err = config.Coolify.DeleteApplicationByUUID(id)
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
		http.Redirect(w, r, "/access", 303)
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
		http.Redirect(w, r, "/access", 303)
	}))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Web paneli :%s portunda dinleniyor", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
