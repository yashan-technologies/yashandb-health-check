package mathutil

import (
	"fmt"
	"math"
	"yhc/i18n"

	"git.yasdb.com/go/yasutil"
)

const (
	thousand        = 1000
	ten_thounsand   = 10 * thousand
	million         = 100 * ten_thounsand
	ten_million     = 10 * million
	hundred_million = 10 * ten_million
)

func Round(num float64, decimal int) float64 {
	pow := math.Pow(10, float64(decimal))
	return math.Round(num*pow) / pow
}

func GenHumanReadableNumber(num float64, decimal int) string {
	if num == 0 || num < thousand {
		return yasutil.FormatFloat(num, decimal)
	}
	
	// 检查当前语言，英文使用K/M/B，中文使用千/万/亿
	currentLang := i18n.GetLanguage()
	
	if currentLang == "en-US" || currentLang == "en" {
		// 英文：使用 K (千), M (百万), B (十亿)
		billion := float64(1000000000)
		if num >= billion {
			return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/billion, decimal), i18n.T("number.billion"))
		}
		if num >= million {
			return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/million, decimal), i18n.T("number.million"))
		}
		return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/thousand, decimal), i18n.T("number.thousand"))
	}
	
	// 中文：使用 千/万/百万/千万/亿
	if num < ten_thounsand {
		return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/thousand, decimal), i18n.T("number.thousand"))
	}
	if num < million {
		return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/ten_thounsand, decimal), i18n.T("number.ten_thousand"))
	}
	if num < ten_million {
		return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/million, decimal), i18n.T("number.million"))
	}
	if num < hundred_million {
		return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/ten_million, decimal), i18n.T("number.ten_million"))
	}
	return fmt.Sprintf("%s%s", yasutil.FormatFloat(num/hundred_million, decimal), i18n.T("number.hundred_million"))
}
