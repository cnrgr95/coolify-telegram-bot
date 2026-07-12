package coolify

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// NewClient creates a new Coolify API client with optional caching
// ttl is the cache time-to-live duration. If 0, caching is disabled.
func NewClient(baseURL, token string, httpClient *http.Client, ttl time.Duration) *Client {
	c := &Client{
		BaseURL: baseURL,
		Token:   token,
		Client:  httpClient,
	}
	if ttl > 0 {
		c.cache = newCache(ttl)
	}
	return c
}

type Client struct {
	BaseURL         string
	Token           string
	Client          *http.Client
	cache           *cache
	projectMu       sync.Mutex
	projectCache    map[int64]string
	projectCachedAt time.Time
}

func (c *Client) projectNames() map[int64]string {
	c.projectMu.Lock()
	defer c.projectMu.Unlock()
	if c.projectCache != nil && time.Since(c.projectCachedAt) < 5*time.Minute {
		return c.projectCache
	}
	names := map[int64]string{}
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/projects", nil)
	if err != nil {
		return names
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.Client.Do(req)
	if err != nil {
		return names
	}
	defer resp.Body.Close()
	var projects []struct{ UUID, Name string }
	if json.NewDecoder(resp.Body).Decode(&projects) != nil {
		return names
	}
	for _, project := range projects {
		detailReq, _ := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/projects/"+project.UUID, nil)
		detailReq.Header.Set("Authorization", "Bearer "+c.Token)
		detailResp, e := c.Client.Do(detailReq)
		if e != nil {
			continue
		}
		var detail struct {
			Environments []struct {
				ID int64 `json:"id"`
			} `json:"environments"`
		}
		_ = json.NewDecoder(detailResp.Body).Decode(&detail)
		detailResp.Body.Close()
		for _, environment := range detail.Environments {
			names[environment.ID] = project.Name
		}
	}
	c.projectCache = names
	c.projectCachedAt = time.Now()
	return names
}

func (c *Client) listResources(path string) ([]Resource, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("kaynaklar alınamadı: %s", resp.Status)
	}
	var resources []Resource
	if err := json.NewDecoder(resp.Body).Decode(&resources); err != nil {
		return nil, err
	}
	return resources, nil
}

func (c *Client) ListDatabases() ([]Resource, error) {
	resources, err := c.listResources("/api/v1/databases")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/api/v1/services", nil)
	if err != nil {
		return resources, nil
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.Client.Do(req)
	if err != nil {
		return resources, nil
	}
	defer resp.Body.Close()
	var services []struct {
		Databases []Resource `json:"databases"`
	}
	if json.NewDecoder(resp.Body).Decode(&services) == nil {
		for _, service := range services {
			resources = append(resources, service.Databases...)
		}
	}
	projects := c.projectNames()
	for i := range resources {
		resources[i].Project = projects[resources[i].EnvironmentID]
	}
	return resources, nil
}
func (c *Client) ListServers() ([]Resource, error) { return c.listResources("/api/v1/servers") }

func (c *Client) ListApplications() ([]Application, error) {
	// Check cache first
	if c.cache != nil {
		if cached, found := c.cache.Get("applications"); found {
			return cached.([]Application), nil
		}
	}

	// If not in cache or cache miss, make the API call
	req, err := http.NewRequest("GET", c.BaseURL+"/api/v1/applications", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New("invalid token (400)")
	}

	var apps []Application
	err = json.NewDecoder(resp.Body).Decode(&apps)
	if err != nil {
		return nil, err
	}

	// Docker Compose servisleri /applications sonucunda yer almaz; bunları da menüye ekle.
	if req2, e := http.NewRequest("GET", c.BaseURL+"/api/v1/services", nil); e == nil {
		req2.Header.Set("Authorization", "Bearer "+c.Token)
		if sr, e := c.Client.Do(req2); e == nil {
			defer sr.Body.Close()
			var services []struct {
				UUID          string `json:"uuid"`
				Name          string `json:"name"`
				Status        string `json:"status"`
				EnvironmentID int64  `json:"environment_id"`
				Applications  []struct {
					FQDN string `json:"fqdn"`
				} `json:"applications"`
			}
			if json.NewDecoder(sr.Body).Decode(&services) == nil {
				for _, s := range services {
					fqdn := ""
					if len(s.Applications) > 0 {
						fqdn = s.Applications[0].FQDN
					}
					apps = append(apps, Application{UUID: "svc:" + s.UUID, Name: s.Name, FQDN: fqdn, Status: s.Status, EnvironmentID: s.EnvironmentID})
				}
			}
		}
	}
	projects := c.projectNames()
	for i := range apps {
		apps[i].Project = projects[apps[i].EnvironmentID]
		if apps[i].Project == "" {
			apps[i].Project = "Diğer"
		}
	}

	// Cache the result if cache is enabled
	if c.cache != nil {
		c.cache.Set("applications", apps)
	}

	return apps, nil
}

func (c *Client) ServiceAction(uuid, action string) error {
	uuid = strings.TrimPrefix(uuid, "svc:")
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/v1/services/%s/%s", c.BaseURL, uuid, action), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("servis işlemi başarısız: %s", resp.Status)
	}
	return nil
}

func (c *Client) GetApplicationByUUID(uuid string) (*ApplicationDetail, error) {
	cacheKey := fmt.Sprintf("app_%s", uuid)
	if c.cache != nil {
		if cached, found := c.cache.Get(cacheKey); found {
			v := cached.(ApplicationDetail)
			return &v, nil
		}
	}

	url := fmt.Sprintf("%s/api/v1/applications/%s", c.BaseURL, uuid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New("invalid token (400)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("application not found")
	}

	var app ApplicationDetail
	err = json.NewDecoder(resp.Body).Decode(&app)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if c.cache != nil {
		c.cache.Set(cacheKey, app)
	}

	return &app, nil
}

func (c *Client) DeleteApplicationByUUID(uuid string) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s", c.BaseURL, uuid)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return errors.New("invalid token (400)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return errors.New("application not found")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected response: %s", resp.Status)
	}

	// Clear relevant cache entries
	if c.cache != nil {
		c.cache.Delete(fmt.Sprintf("app_%s", uuid))
		c.cache.Delete(fmt.Sprintf("app_envs_%s", uuid))
		c.cache.Delete(fmt.Sprintf("app_start_%s", uuid))
		c.cache.Delete(fmt.Sprintf("app_start_%s_true_true", uuid))
		c.cache.Delete(fmt.Sprintf("app_start_%s_true_false", uuid))
		c.cache.Delete(fmt.Sprintf("app_start_%s_false_true", uuid))
		c.cache.Delete(fmt.Sprintf("app_start_%s_false_false", uuid))
		c.cache.Delete(fmt.Sprintf("app_stop_%s", uuid))
		c.cache.Delete(fmt.Sprintf("app_restart_%s", uuid))
		c.cache.Delete("applications")
	}

	return nil
}

func (c *Client) GetApplicationLogsByUUID(uuid string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s/logs?lines=-1", c.BaseURL, uuid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return "", errors.New("invalid token (400)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("application logs not found")
	}

	var logs ApplicationLogs
	err = json.NewDecoder(resp.Body).Decode(&logs)
	if err != nil {
		return "", err
	}

	return logs.Logs, nil
}

func (c *Client) GetApplicationEnvsByUUID(uuid string) ([]EnvironmentVariable, error) {
	cacheKey := fmt.Sprintf("app_envs_%s", uuid)
	if c.cache != nil {
		if cached, found := c.cache.Get(cacheKey); found {
			return cached.([]EnvironmentVariable), nil
		}
	}

	url := fmt.Sprintf("%s/api/v1/applications/%s/envs", c.BaseURL, uuid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New("invalid token (400)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("application environment variables not found")
	}

	var envs []EnvironmentVariable
	err = json.NewDecoder(resp.Body).Decode(&envs)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if c.cache != nil {
		c.cache.Set(cacheKey, envs)
	}

	return envs, nil
}

func (c *Client) StartApplicationDeployment(uuid string, force, instantDeploy bool) (*StartDeploymentResponse, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s/start", c.BaseURL, uuid)
	// Build query parameters
	query := url + "?"
	if force {
		query += "force=true&"
	}
	if instantDeploy {
		query += "instant_deploy=true"
	}

	req, err := http.NewRequest("GET", query, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New("invalid token (400)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("application not found")
	}

	var deployment StartDeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}

func (c *Client) StopApplicationByUUID(uuid string) (*StopApplicationResponse, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s/stop", c.BaseURL, uuid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New("invalid token (400)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("application not found")
	}

	var stopResponse StopApplicationResponse
	err = json.NewDecoder(resp.Body).Decode(&stopResponse)
	if err != nil {
		return nil, err
	}

	return &stopResponse, nil
}

func (c *Client) RestartApplicationByUUID(uuid string) (*StartDeploymentResponse, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s/restart", c.BaseURL, uuid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthenticated: invalid or missing token (401)")
	}
	if resp.StatusCode == http.StatusBadRequest {
		return nil, errors.New("invalid token (400)")
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("application not found")
	}

	var deployment StartDeploymentResponse
	err = json.NewDecoder(resp.Body).Decode(&deployment)
	if err != nil {
		return nil, err
	}

	return &deployment, nil
}
