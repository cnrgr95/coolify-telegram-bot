package src

import (
	"fmt"
	"time"

	"coolifymanager/src/scheduler"

	td "github.com/AshokShau/gotdbot"
	"github.com/AshokShau/gotdbot/filters/callbackquery"
)

var (
	startTime = time.Now()
)

func InitFunc(c *td.Client) error {
	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("scheduler start error: %s", err.Error())
	}

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

	// Callbacks
	c.OnUpdateNewCallbackQuery(jobsPaginationHandler, callbackquery.Prefix("jobs:"))
	c.OnUpdateNewCallbackQuery(listProjectsHandler, callbackquery.Prefix("list_projects"))
	c.OnUpdateNewCallbackQuery(projectMenuHandler, callbackquery.Prefix("project_menu:"))
	c.OnUpdateNewCallbackQuery(scheduleMenuHandler, callbackquery.Prefix("sch_m:"))
	c.OnUpdateNewCallbackQuery(scheduleActionHandler, callbackquery.Prefix("sch_a:"))
	c.OnUpdateNewCallbackQuery(scheduleCreateHandler, callbackquery.Prefix("sch_c:"))
	c.OnUpdateNewCallbackQuery(restartHandler, callbackquery.Prefix("restart:"))
	c.OnUpdateNewCallbackQuery(deployHandler, callbackquery.Prefix("deploy:"))
	c.OnUpdateNewCallbackQuery(logsHandler, callbackquery.Prefix("logs:"))
	c.OnUpdateNewCallbackQuery(statusHandler, callbackquery.Prefix("status:"))
	c.OnUpdateNewCallbackQuery(stopHandler, callbackquery.Prefix("stop:"))
	c.OnUpdateNewCallbackQuery(deleteHandler, callbackquery.Prefix("delete:"))

	return nil
}
