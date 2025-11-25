package i18n

import (
	"embed"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.toml
var localeFS embed.FS

var (
	bundle          *i18n.Bundle
	localizer       *i18n.Localizer
	currentLanguage Language = ZhCN
	mu              sync.RWMutex
)

type Language string

const (
	ZhCN Language = "zh-CN"
	EnUS Language = "en-US"
)

func Init() error {
	bundle = i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	for _, lang := range []string{"zh-CN", "en-US"} {
		filename := "locales/active." + lang + ".toml"
		data, err := localeFS.ReadFile(filename)
		if err != nil {
			continue
		}
		if _, err := bundle.ParseMessageFileBytes(data, filename); err != nil {
			return err
		}
	}

	SetLanguage("zh-CN")
	return nil
}

func SetLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()

	normalizedLang := normalizeLanguage(lang)
	currentLanguage = Language(normalizedLang)
	localizer = i18n.NewLocalizer(bundle, normalizedLang)
}

func GetLanguage() Language {
	mu.RLock()
	defer mu.RUnlock()
	return currentLanguage
}

func normalizeLanguage(lang string) string {
	switch lang {
	case "zh", "zh-CN", "zh_CN", "chinese":
		return "zh-CN"
	case "en", "en-US", "en_US", "english":
		return "en-US"
	default:
		return "zh-CN"
	}
}

func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if localizer == nil {
		return key
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: key,
	})
	if err != nil {
		return key
	}
	return msg
}

func Tf(key string, templateData map[string]interface{}) string {
	mu.RLock()
	defer mu.RUnlock()

	if localizer == nil {
		return key
	}

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    key,
		TemplateData: templateData,
	})
	if err != nil {
		return key
	}
	return msg
}
