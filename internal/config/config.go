package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Addr                   string
	APIKey                 string
	BaseURL                string
	PrimaryModel           string
	FallbackModels         []string
	PerModelTimeoutSec     int
	ReadingTimeoutSec      int
	MaxBodyBytes           int64
}

func Load() Config {
	return Config{
		Addr:               getenv("GATEWAY_ADDR", ":8080"),
		APIKey:             os.Getenv("OPENAI_API_KEY"),
		BaseURL:            getenv("OPENAI_BASE_URL", "https://openrouter.ai/api/v1"),
		PrimaryModel:       getenv("OPENAI_MODEL", "openai/gpt-oss-120b:free"),
		FallbackModels:     splitCSV(getenv("OPENAI_MODEL_FALLBACKS", "poolside/laguna-xs.2:free,nex-agi/nex-n2-pro:free")),
		PerModelTimeoutSec: getenvInt("LLM_PER_MODEL_TIMEOUT_SEC", 50),
		ReadingTimeoutSec:  getenvInt("LLM_READING_TIMEOUT_SEC", 120),
		MaxBodyBytes:       1 << 20,
	}
}

func (c Config) Models() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 1+len(c.FallbackModels))
	for _, m := range append([]string{c.PrimaryModel}, c.FallbackModels...) {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		if _, ok := seen[m]; ok {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	return out
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
