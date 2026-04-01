package i18n

import (
	"embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

//go:embed *.json
var translationsFS embed.FS

var (
	langMap map[string]string
	mu      sync.RWMutex
)

func Init(lang string) error {
	mu.Lock()
	defer mu.Unlock()

	fileName := lang + ".json"
	data, err := translationsFS.ReadFile(fileName)
	if err != nil {
		data, err = translationsFS.ReadFile("en.json")
		if err != nil {
			return err
		}
	}

	langMap = make(map[string]string)
	if err := json.Unmarshal(data, &langMap); err != nil {
		return err
	}

	return nil
}

func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if val, ok := langMap[key]; ok {
		return val
	}

	if langMap != nil {
		enData, _ := translationsFS.ReadFile("en.json")
		var enMap map[string]string
		_ = json.Unmarshal(enData, &enMap)
		if val, ok := enMap[key]; ok {
			return val
		}
	}

	return key
}

func AvailableLanguages() []string {
	entries, err := translationsFS.ReadDir(".")
	if err != nil {
		return []string{"en"}
	}

	var langs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			lang := strings.TrimSuffix(e.Name(), ".json")
			langs = append(langs, lang)
		}
	}

	sort.Strings(langs)
	return langs
}
