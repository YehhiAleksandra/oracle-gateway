package config

import (
	"os"
	"strings"
)

// Task names match digital-oracle reading kinds.
const (
	TaskTarot       = "tarot"
	TaskRunes       = "runes"
	TaskPalmistry   = "palmistry"
	TaskGraphology  = "graphology"
	TaskNumerology  = "numerology"
	TaskAstrology   = "astrology"
	TaskHoroscope   = "horoscope"
	TaskCardOfDay   = "card_of_day"
	TaskMatrix      = "matrix"
	TaskSynastry    = "synastry"
	TaskPing        = "ping"
)

func (c Config) ModelsForTask(task string) []string {
	task = strings.ToLower(strings.TrimSpace(task))
	if task == "" {
		return c.Models()
	}
	if chain, ok := c.TaskModels[task]; ok && len(chain) > 0 {
		return chain
	}
	if task == TaskPalmistry || task == TaskGraphology {
		if len(c.VisionModels) > 0 {
			return c.VisionModels
		}
	}
	return c.Models()
}

func (c Config) VisionModelsForTask(task string) []string {
	task = strings.ToLower(strings.TrimSpace(task))
	if chain, ok := c.VisionTaskModels[task]; ok && len(chain) > 0 {
		return chain
	}
	if len(c.VisionModels) > 0 {
		return c.VisionModels
	}
	return c.Models()
}

func loadTaskModels(defaults []string) map[string][]string {
	out := map[string][]string{}
	prefix := "MODEL_TASK_"
	for _, env := range os.Environ() {
		key, value, ok := strings.Cut(env, "=")
		if !ok || !strings.HasPrefix(key, prefix) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(key, prefix)))
		if name == "" {
			continue
		}
		chain := splitCSV(value)
		if len(chain) == 0 {
			continue
		}
		out[name] = chain
	}
	for _, task := range []string{
		TaskTarot, TaskRunes, TaskPalmistry, TaskGraphology,
		TaskNumerology, TaskAstrology, TaskHoroscope, TaskCardOfDay,
		TaskMatrix, TaskSynastry, TaskPing,
	} {
		if _, ok := out[task]; !ok {
			out[task] = append([]string(nil), defaults...)
		}
	}
	return out
}

func loadVisionTaskModels(defaults []string) map[string][]string {
	out := map[string][]string{}
	prefix := "MODEL_VISION_TASK_"
	for _, env := range os.Environ() {
		key, value, ok := strings.Cut(env, "=")
		if !ok || !strings.HasPrefix(key, prefix) {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(key, prefix)))
		if name == "" {
			continue
		}
		chain := splitCSV(value)
		if len(chain) == 0 {
			continue
		}
		out[name] = chain
	}
	for _, task := range []string{TaskPalmistry, TaskGraphology} {
		if _, ok := out[task]; !ok {
			out[task] = append([]string(nil), defaults...)
		}
	}
	return out
}
