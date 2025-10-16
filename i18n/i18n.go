package i18n

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

var translations = make(map[string]map[string]string)

func LoadTranslations(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			lang := file.Name()[:len(file.Name())-len(".json")]
			filePath := filepath.Join(dir, file.Name())

			fileData, err := os.ReadFile(filePath)
			if err != nil {
				log.Printf("failed to read translation file %s: %v", filePath, err)
				continue
			}

			var langMap map[string]string
			if err := json.Unmarshal(fileData, &langMap); err != nil {
				log.Printf("failed to parse translation file %s: %v", filePath, err)
				continue
			}

			translations[lang] = langMap
			log.Printf("loaded translations for language: %s", lang)
		}
	}
	return nil
}

func GetMessage(lang, key string, data interface{}) string {
	langMap, ok := translations[lang]
	if !ok {
		langMap = translations["en"]
	}

	messageTemplate, ok := langMap[key]
	if !ok {
		return key
	}
	
	tmpl, err := template.New("message").Parse(messageTemplate)
    if err != nil {
        log.Printf("failed to parse template for key %s: %v", key, err)
        return messageTemplate
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        log.Printf("failed to execute template for key %s: %v", key, err)
        return messageTemplate
    }

	return buf.String()
}