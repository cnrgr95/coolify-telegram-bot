package database

import (
	"encoding/json"
	"os"
	"sync"
	"time"
    "fmt"
)

type ScheduledTask struct {
	ID          string    `json:"_id"`
	Name        string    `json:"name"`
	ProjectUUID string    `json:"project_uuid"`
	Schedule    string    `json:"schedule"`
	Type        string    `json:"type"`
	OneTime     bool      `json:"one_time"`
	NextRun     time.Time `json:"next_run"`
}

type AuthorizedUser struct {
	TelegramID int64     `json:"telegram_id"`
	Role       string    `json:"role"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type DataStore struct {
	Tasks []ScheduledTask  `json:"tasks"`
	Users []AuthorizedUser `json:"users"`
}

var store DataStore
var mu sync.Mutex
var dbPath = "/app/data/bot_data.json"

func Connect(uri string) error {
	mu.Lock(); defer mu.Unlock()
	os.MkdirAll("/app/data", 0755)
	b, err := os.ReadFile(dbPath)
	if err == nil {
		json.Unmarshal(b, &store)
	}
	if store.Tasks == nil { store.Tasks = []ScheduledTask{} }
	if store.Users == nil { store.Users = []AuthorizedUser{} }
	return nil
}

func save() error {
	b, err := json.MarshalIndent(store, "", "  ")
	if err != nil { return err }
	return os.WriteFile(dbPath, b, 0644)
}

func AddAuthorizedUser(id int64, role ...string) error {
	mu.Lock(); defer mu.Unlock()
	r := "operator"; if len(role)>0 && role[0]!="" { r=role[0] }
	found := false
	for i, u := range store.Users {
		if u.TelegramID == id {
			store.Users[i].Role = r
			store.Users[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}
	if !found {
		store.Users = append(store.Users, AuthorizedUser{TelegramID: id, Role: r, UpdatedAt: time.Now()})
	}
	return save()
}

func RemoveAuthorizedUser(id int64) error {
	mu.Lock(); defer mu.Unlock()
	var newUsers []AuthorizedUser
	for _, u := range store.Users {
		if u.TelegramID != id { newUsers = append(newUsers, u) }
	}
	store.Users = newUsers
	return save()
}

func IsAuthorizedUser(id int64) bool {
	mu.Lock(); defer mu.Unlock()
	for _, u := range store.Users {
		if u.TelegramID == id { return true }
	}
	return false
}

func AuthorizedRole(id int64) string {
	mu.Lock(); defer mu.Unlock()
	for _, u := range store.Users {
		if u.TelegramID == id {
			if u.Role == "" { return "operator" }
			return u.Role
		}
	}
	return ""
}

func GetAuthorizedUserRecords() ([]AuthorizedUser, error) {
	mu.Lock(); defer mu.Unlock()
	return append([]AuthorizedUser(nil), store.Users...), nil
}

func GetAuthorizedUsers() ([]int64, error) {
	mu.Lock(); defer mu.Unlock()
	var ids []int64
	for _, u := range store.Users { ids = append(ids, u.TelegramID) }
	return ids, nil
}

func AddTask(task ScheduledTask) error {
	mu.Lock(); defer mu.Unlock()
	store.Tasks = append(store.Tasks, task)
	return save()
}

func GetTasks() ([]ScheduledTask, error) {
	mu.Lock(); defer mu.Unlock()
	return append([]ScheduledTask(nil), store.Tasks...), nil
}

func DeleteTask(id string) error {
	mu.Lock(); defer mu.Unlock()
	var newTasks []ScheduledTask
	for _, t := range store.Tasks {
		if t.ID != id { newTasks = append(newTasks, t) }
	}
	store.Tasks = newTasks
	return save()
}

func GetDueOneTimeTasks() ([]ScheduledTask, error) {
	mu.Lock(); defer mu.Unlock()
	var due []ScheduledTask
	now := time.Now()
	for _, t := range store.Tasks {
		if t.OneTime && (t.NextRun.Before(now) || t.NextRun.Equal(now)) {
			due = append(due, t)
		}
	}
	return due, nil
}

func RemoveOneTimeTask(id string) error {
	return DeleteTask(id)
}

func UpdateTaskNextRun(id string, nextRun time.Time) error {
	mu.Lock(); defer mu.Unlock()
	for i, t := range store.Tasks {
		if t.ID == id {
			store.Tasks[i].NextRun = nextRun
			break
		}
	}
	return save()
}

type LogEntry struct {
	Timestamp string
	Message   string
}
func GetLogs() ([]LogEntry, error) {
    return []LogEntry{{Timestamp: time.Now().Format(time.RFC3339), Message: "No MongoDB logs because it is JSON backend"}}, nil
}
func DebugInfo() string {
    return "Using JSON backend"
}
