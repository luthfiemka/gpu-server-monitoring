package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type QuestDB struct {
	Host     string
	Port     int
	ILPPort  int
	ILPAuth  string
	User     string
	Password string
	Protocol string
	Interval int
}

type Agent struct {
	QuestDB  QuestDB
	Hostname string
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func Load(path string) (*Agent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	sections := map[string]map[string]string{}
	current := ""

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = line[1 : len(line)-1]
			if sections[current] == nil {
				sections[current] = map[string]string{}
			}
			continue
		}
		if idx := strings.IndexByte(line, '='); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			if current != "" {
				sections[current][key] = val
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	q, ok := sections["questdb"]
	if !ok {
		return nil, fmt.Errorf("missing [questdb] section")
	}

	get := func(key, fallback string) string {
		if v, ok := q[key]; ok && v != "" {
			return v
		}
		return fallback
	}

	GetInt := func(key string, fallback int) int {
		if v, ok := q[key]; ok && v != "" {
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
		return fallback
	}

	hostname := ""
	if a, ok := sections["agent"]; ok {
		hostname = a["hostname"]
	}

	return &Agent{
		QuestDB: QuestDB{
			Host:     envOr("GPU_DASH_QUESTDB_HOST", get("host", "localhost")),
			Port:     envOrInt("GPU_DASH_QUESTDB_PORT", GetInt("port", 9000)),
			ILPPort:  envOrInt("GPU_DASH_QUESTDB_ILP_PORT", GetInt("ilp_port", 9009)),
			ILPAuth:  envOr("GPU_DASH_QUESTDB_ILP_AUTH", get("ilp_auth", "")),
			User:     envOr("GPU_DASH_QUESTDB_USER", get("user", "admin")),
			Password: envOr("GPU_DASH_QUESTDB_PASSWORD", get("password", "quest")),
			Protocol: envOr("GPU_DASH_QUESTDB_PROTOCOL", get("protocol", "http")),
			Interval: envOrInt("GPU_DASH_QUESTDB_INTERVAL", GetInt("interval", 5)),
		},
		Hostname: hostname,
	}, nil
}
