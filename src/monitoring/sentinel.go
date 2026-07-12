package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

type Metrics struct {
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	RAMUsed   uint64  `json:"ram_used"`
	RAMTotal  uint64  `json:"ram_total"`
	Available bool    `json:"available"`
	Source    string  `json:"source,omitempty"`
	Error     string  `json:"error,omitempty"`
}

func Load() Metrics {
	bases := []string{os.Getenv("SENTINEL_URL")}
	if bases[0] == "" {
		bases = []string{"http://host.docker.internal:8000", "http://host.docker.internal:8888", "http://coolify-sentinel:8888"}
	}
	if discovered := discoverSentinelURL(); discovered != "" {
		bases = append(bases, discovered)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	token := os.Getenv("SENTINEL_TOKEN")
	lastError := "Sentinel bağlantısı kurulamadı"
	for _, base := range bases {
		result := Metrics{Source: base}
		cpu := &struct {
			Percent *float64 `json:"percent"`
		}{}
		err := getJSON(client, base, token, "/api/cpu/current", cpu)
		if err != nil {
			lastError = err.Error()
		} else if cpu.Percent != nil {
			result.CPU = *cpu.Percent
			result.Available = true
		}

		memory := &struct {
			UsedPercent *float64 `json:"usedPercent"`
			Used        uint64   `json:"used"`
			Total       uint64   `json:"total"`
		}{}
		err = getJSON(client, base, token, "/api/memory/current", memory)
		if err != nil {
			lastError = err.Error()
		} else if memory.UsedPercent != nil {
			result.RAM = *memory.UsedPercent
			result.RAMUsed = memory.Used
			result.RAMTotal = memory.Total
			result.Available = true
		}
		if result.Available {
			return result
		}
	}
	return Metrics{Error: lastError}
}

func discoverSentinelURL() string {
	if _, err := os.Stat("/var/run/docker.sock"); err != nil {
		return ""
	}
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "unix", "/var/run/docker.sock")
	}}
	client := &http.Client{Transport: transport, Timeout: 3 * time.Second}
	response, err := client.Get("http://docker/containers/coolify-sentinel/json")
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return ""
	}
	var inspect struct {
		NetworkSettings struct {
			Networks map[string]struct {
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	if json.NewDecoder(response.Body).Decode(&inspect) != nil {
		return ""
	}
	for _, network := range inspect.NetworkSettings.Networks {
		if network.IPAddress != "" {
			return "http://" + network.IPAddress + ":8888"
		}
	}
	return ""
}

func getJSON(client *http.Client, base, token, path string, target any) error {
	request, err := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+path, nil)
	if err != nil {
		return err
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("Sentinel endpointine erişilemiyor: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Sentinel kimlik doğrulaması başarısız")
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Sentinel endpointi HTTP %d döndürdü", response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("Sentinel yanıtı okunamadı: %w", err)
	}
	return nil
}
