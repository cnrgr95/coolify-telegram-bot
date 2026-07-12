package coolify

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type FlexibleString string

func (value *FlexibleString) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*value = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*value = FlexibleString(text)
		return nil
	}
	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		*value = FlexibleString(fmt.Sprintf("%t", boolean))
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		*value = FlexibleString(number.String())
		return nil
	}
	return fmt.Errorf("desteklenmeyen metin değeri: %s", data)
}

type Application struct {
	ID            int64  `json:"id"`
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	FQDN          string `json:"fqdn"`
	Status        string `json:"status"`
	EnvironmentID int64  `json:"environment_id"`
	Project       string `json:"-"`
	LimitsCPUs    any    `json:"limits_cpus"`
	LimitsMemory  any    `json:"limits_memory"`
}

type ApplicationDetail struct {
	ID                      int64  `json:"id"`
	UUID                    string `json:"uuid"`
	Name                    string `json:"name"`
	FQDN                    string `json:"fqdn"`
	Status                  string `json:"status"`
	Description             string `json:"description"`
	GitRepository           string `json:"git_repository"`
	GitBranch               string `json:"git_branch"`
	DockerRegistryImageName string `json:"docker_registry_image_name"`
	Dockerfile              string `json:"dockerfile"`
	BuildPack               string `json:"build_pack"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
	// TODO: Add more fields as needed...
}

type ApplicationLogs struct {
	Logs string `json:"logs"`
}

type EnvironmentVariable struct {
	ID               int64  `json:"id"`
	UUID             string `json:"uuid"`
	ResourceableType string `json:"resourceable_type"`
	ResourceableID   int64  `json:"resourceable_id"`
	IsBuildTime      bool   `json:"is_build_time"`
	IsLiteral        bool   `json:"is_literal"`
	IsMultiline      bool   `json:"is_multiline"`
	IsPreview        bool   `json:"is_preview"`
	IsShared         bool   `json:"is_shared"`
	IsShownOnce      bool   `json:"is_shown_once"`
	Key              string `json:"key"`
	Value            string `json:"value"`
	RealValue        string `json:"real_value"`
	Version          string `json:"version"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type StartDeploymentResponse struct {
	Message        string `json:"message"`
	DeploymentUUID string `json:"deployment_uuid"`
}

type StopApplicationResponse struct {
	Message string `json:"message"`
}

type Resource struct {
	UUID          string         `json:"uuid"`
	Name          string         `json:"name"`
	Type          string         `json:"type"`
	DatabaseType  string         `json:"database_type"`
	Status        string         `json:"status"`
	ServerStatus  FlexibleString `json:"server_status"`
	Image         string         `json:"image"`
	FQDN          string         `json:"fqdn"`
	IP            string         `json:"ip"`
	LimitsCPUs    any            `json:"limits_cpus"`
	LimitsMemory  any            `json:"limits_memory"`
	EnvironmentID int64          `json:"environment_id"`
	Project       string         `json:"-"`
}
