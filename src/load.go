package src

import (
	"fmt"
	"time"

	"coolifymanager/src/scheduler"

	td "github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/filters/callbackquery"
	"github.com/AshokShau/gotdbot/filters/message"
)

var (
	startTime = time.Now()
)

func InitFunc(c *td.Client) error {
	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("scheduler start error: %s", err.Error())
	}
	go monitorResourceChanges(c)

	// Commands
	c.OnCommand("start", startHandler)
	c.OnCommand("ping", pingHandler)
	c.OnCommand("jobs", jobsHandler)
	c.OnCommand("job", scheduleHandler)
	c.OnCommand("schedule", scheduleHandler)
	c.OnCommand("unschedule", unscheduleHandler)
	c.OnCommand("rmJob", unscheduleHandler)
	c.OnCommand("yetki_ekle", addAuthorizedHandler)
	c.OnCommand("yetki_sil", removeAuthorizedHandler)
	c.OnCommand("yetkililer", listAuthorizedHandler)
	c.OnCommand("web_ekle", addWebUserHandler)
	c.OnCommand("web_sil", removeWebUserHandler)
	c.OnCommand("web_kullanicilar", listWebUsersHandler)
	c.OnMessage(quickMenuHandler, message.Private)

	// Callbacks
	c.OnUpdateNewCallbackQuery(jobsPaginationHandler, callbackquery.Prefix("jobs:"))
	c.OnUpdateNewCallbackQuery(jobDeleteHandler, callbackquery.Prefix("job_del:"))
	c.OnUpdateNewCallbackQuery(listProjectsHandler, callbackquery.Prefix("list_projects"))
	c.OnUpdateNewCallbackQuery(projectMenuHandler, callbackquery.Prefix("project_menu:"))
	c.OnUpdateNewCallbackQuery(scheduleMenuHandler, callbackquery.Prefix("sch_m:"))
	c.OnUpdateNewCallbackQuery(scheduleActionHandler, callbackquery.Prefix("sch_a:"))
	c.OnUpdateNewCallbackQuery(scheduleTimeHandler, callbackquery.Prefix("sch_t:"))
	c.OnUpdateNewCallbackQuery(scheduleCreateHandler, callbackquery.Prefix("sch_c:"))
	c.OnUpdateNewCallbackQuery(restartHandler, callbackquery.Prefix("restart:"))
	c.OnUpdateNewCallbackQuery(deployHandler, callbackquery.Prefix("deploy:"))
	c.OnUpdateNewCallbackQuery(redeployHandler, callbackquery.Prefix("redeploy:"))
	c.OnUpdateNewCallbackQuery(logsHandler, callbackquery.Prefix("logs:"))
	c.OnUpdateNewCallbackQuery(statusHandler, callbackquery.Prefix("status:"))
	c.OnUpdateNewCallbackQuery(stopHandler, callbackquery.Prefix("stop:"))
	c.OnUpdateNewCallbackQuery(deleteHandler, callbackquery.Prefix("delete:"))
	c.OnUpdateNewCallbackQuery(telegramUserActionHandler, callbackquery.Prefix("tg_role:"))
	c.OnUpdateNewCallbackQuery(telegramUserActionHandler, callbackquery.Prefix("tg_del:"))
	c.OnUpdateNewCallbackQuery(webUserActionHandler, callbackquery.Prefix("web_role:"))
	c.OnUpdateNewCallbackQuery(webUserActionHandler, callbackquery.Prefix("web_del:"))
	c.OnUpdateNewCallbackQuery(newUserRoleHandler, callbackquery.Prefix("new_tg_role:"))
	c.OnUpdateNewCallbackQuery(newUserRoleHandler, callbackquery.Prefix("new_web_role:"))

	return nil
}
