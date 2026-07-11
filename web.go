package main

import (
	"crypto/subtle"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"coolifymanager/src/config"
	"coolifymanager/src/database"
)

var dashboard = template.Must(template.New("dashboard").Parse(`<!doctype html><html lang="tr"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>FL Panel Coolify</title><style>body{margin:0;background:#0b1020;color:#e8ecf6;font:15px system-ui}main{max-width:1100px;margin:auto;padding:32px}h1{margin-bottom:4px}.sub{color:#95a0b8;margin-bottom:24px}.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(290px,1fr));gap:16px}.card{background:#151c31;border:1px solid #283250;border-radius:15px;padding:18px}.name{font-size:18px;font-weight:700}.status{color:#75e6a4;margin:8px 0}.url{color:#8eb8ff;word-break:break-all}form{display:inline}button,a.btn{border:0;border-radius:8px;padding:9px 12px;margin:12px 5px 0 0;background:#6d5dfc;color:white;text-decoration:none;cursor:pointer}.danger{background:#d44f62}.msg{padding:12px;background:#243052;border-radius:9px;margin-bottom:18px}</style></head><body><main><h1>FL Panel Coolify</h1><div class="sub">Telegram botu ve web yÃ¶netim paneli | <a href="/yetkililer" style="color:#8eb8ff;text-decoration:none">ğŸ‘¤ Yetkili KullanÄ±cÄ±larÄ± YÃ¶net</a></div>{{if .Message}}<div class="msg">{{.Message}}</div>{{end}}<div class="grid">{{range .Apps}}<section class="card"><div class="name">{{.Name}}</div><div class="status">{{.Status}}</div><div class="url">{{.FQDN}}</div><form method="post" action="/action"><input type="hidden" name="uuid" value="{{.UUID}}"><button name="op" value="deploy">Dağıt</button><button name="op" value="restart">Yeniden Başlat</button><button class="danger" name="op" value="stop">Durdur</button></form><a class="btn" href="/logs?uuid={{.UUID}}">Loglar</a></section>{{else}}<p>Uygulama bulunamadÄ±.</p>{{end}}</div></main></body></html>`))

func startWebPanel() {
	user, pass := os.Getenv("WEB_USER"), os.Getenv("WEB_PASSWORD")
	if user == "" { user = "admin" }
	if pass == "" { log.Print("WEB_PASSWORD is required; web panel disabled"); return }
	auth := func(next http.HandlerFunc) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request) { u,p,ok:=r.BasicAuth(); if !ok || subtle.ConstantTimeCompare([]byte(u),[]byte(user))!=1 || subtle.ConstantTimeCompare([]byte(p),[]byte(pass))!=1 { w.Header().Set("WWW-Authenticate", `Basic realm="FL Panel"`); http.Error(w,"Yetkisiz",http.StatusUnauthorized); return }; next(w,r) } }
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "application/json"); fmt.Fprint(w, `{"status":"ok"}`) })
	http.HandleFunc("/", auth(func(w http.ResponseWriter,r *http.Request){ apps,err:=config.Coolify.ListApplications(); msg:=""; if err!=nil {msg=err.Error()}; _=dashboard.Execute(w,map[string]any{"Apps":apps,"Message":msg}) }))
	http.HandleFunc("/action", auth(func(w http.ResponseWriter,r *http.Request){ if r.Method!="POST" {http.Error(w,"Method",405);return}; id,op:=r.FormValue("uuid"),r.FormValue("op"); var err error; switch op {case "deploy": _,err=config.Coolify.StartApplicationDeployment(id,false,false); case "restart": _,err=config.Coolify.RestartApplicationByUUID(id); case "stop": _,err=config.Coolify.StopApplicationByUUID(id); default: err=fmt.Errorf("geÃ§ersiz iÅŸlem")}; if err!=nil {http.Error(w,err.Error(),500);return}; http.Redirect(w,r,"/",303) }))
	http.HandleFunc("/logs", auth(func(w http.ResponseWriter,r *http.Request){ logs,err:=config.Coolify.GetApplicationLogsByUUID(r.URL.Query().Get("uuid")); if err!=nil {http.Error(w,err.Error(),500);return}; w.Header().Set("Content-Type","text/plain; charset=utf-8"); fmt.Fprint(w,strings.TrimSpace(logs)) }))
	http.HandleFunc("/yetkililer", auth(func(w http.ResponseWriter,r *http.Request){
		if r.Method=="POST" { id,err:=strconv.ParseInt(r.FormValue("id"),10,64); if err!=nil{http.Error(w,"GeÃ§ersiz Telegram ID",400);return}; if r.FormValue("op")=="sil" { if id==config.OwnerID(){http.Error(w,"Ana yÃ¶netici silinemez",400);return};err=database.RemoveAuthorizedUser(id) } else { role:=r.FormValue("role");if role!="viewer"&&role!="operator"&&role!="admin"{http.Error(w,"GeÃ§ersiz rol",400);return};err=database.AddAuthorizedUser(id,role) };if err!=nil{http.Error(w,err.Error(),500);return};http.Redirect(w,r,"/yetkililer",303);return }
		rows,err:=database.GetAuthorizedUserRecords();if err!=nil{http.Error(w,err.Error(),500);return};w.Header().Set("Content-Type","text/html; charset=utf-8");fmt.Fprintf(w,`<!doctype html><html lang="tr"><meta name="viewport" content="width=device-width"><style>body{background:#0b1020;color:#eee;font:16px system-ui;max-width:760px;margin:40px auto;padding:20px}form,li{background:#151c31;padding:14px;margin:10px;border-radius:10px}input,select,button{padding:9px;margin:4px}button{background:#6d5dfc;color:white;border:0;border-radius:7px}</style><h1>Yetkili KullanÄ±cÄ±lar</h1><p>Ana yÃ¶netici: %d</p><form method="post"><input name="id" placeholder="Telegram ID" required><select name="role"><option value="viewer">GÃ¶rÃ¼ntÃ¼leyici</option><option value="operator">OperatÃ¶r</option><option value="admin">YÃ¶netici</option></select><button name="op" value="ekle">Ekle / GÃ¼ncelle</button></form><ul>`,config.OwnerID());for _,u:=range rows{fmt.Fprintf(w,`<li>%d â€” %s <form style="display:inline;background:none;padding:0" method="post"><input type="hidden" name="id" value="%d"><button name="op" value="sil">Sil</button></form></li>`,u.TelegramID,u.Role,u.TelegramID)};fmt.Fprint(w,`</ul><a href="/" style="color:#8eb8ff">Panele dÃ¶n</a></html>`)
	}))
	port:=os.Getenv("PORT"); if port=="" {port="8080"}; http.HandleFunc("/debug-mongo", handleDebugMongo); log.Printf("Web panel listening on :%s",port); log.Fatal(http.ListenAndServe(":"+port,nil))
}

