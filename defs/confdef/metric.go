package confdef

import (
	"path"

	"yhc/defs/errdef"
	"yhc/defs/runtimedef"
	"yhc/i18n"

	"git.yasdb.com/go/yasutil/fs"
	"github.com/BurntSushi/toml"
)

var _metricConfig *YHCMetricConfig

const (
	M_HOST     = "host"
	M_DATABASE = "database"
	M_OBJECTS  = "objects"
	M_SAFETY   = "safety"
	M_CUSTOM   = "custom"
)

const (
	MT_INVALID MetricType = "invalid"
	MT_SQL     MetricType = "sql"
	MT_BASH    MetricType = "bash"
)

type YHCMetricConfig struct {
	Metrics []*YHCMetric `toml:"metrics"`
}

type YHCMetric struct {
	Name           string                    `toml:"name"`
	NameAlias      string                    `toml:"name_alias,omitempty"`
	NameAliasEn    string                    `toml:"name_alias_en,omitempty"`
	ModuleName     string                    `toml:"module_name"`
	MetricType     MetricType                `toml:"metric_type"`
	Hidden         bool                      `toml:"hidden"`
	Default        bool                      `toml:"default"`
	Enabled        bool                      `toml:"enabled"`
	ColumnAlias    map[string]string         `toml:"column_alias,omitempty"`
	ColumnAliasEn  map[string]string         `toml:"column_alias_en,omitempty"`
	ColumnOrder    []string                  `toml:"column_order,omitempty"`
	HiddenColumns  []string                  `toml:"hidden_columns,omitempty"`  // hide column in table, only used in alert expression
	ByteColumns    []string                  `toml:"byte_columns,omitempty"`    // convert byte columns to human readable size
	PercentColumns []string                  `toml:"percent_columns,omitempty"` // convert percent columns to number + '%'
	ItemNames      map[string]string         `toml:"item_names,omitempty"`
	NumberColumns  []string                  `toml:"number_columns,omitempty"`
	Labels         []string                  `toml:"labels,omitempty"`
	AlertRules     map[string][]AlertDetails `toml:"alert_rules,omitempty"`
	SQL            string                    `toml:"sql,omitempty"`     // SQL类型的指标的sql语句
	Command        string                    `toml:"command,omitempty"` // bash类型指标的bash命令
}

type AlertDetails struct {
	Expression    string `toml:"expression"`
	Description   string `toml:"description,omitempty"`
	DescriptionEn string `toml:"description_en,omitempty"`
	Suggestion    string `toml:"suggestion,omitempty"`
	SuggestionEn  string `toml:"suggestion_en,omitempty"`
}

type MetricType string

const (
	AL_INVALID  = "invalid"
	AL_INFO     = "info"
	AL_WARNING  = "warning"
	AL_CRITICAL = "critical"
)

// GetAlertLevelText 根据当前语言获取告警级别文本
func GetAlertLevelText(level string) string {
	switch level {
	case AL_INVALID:
		return i18n.T("alert.error")
	case AL_INFO:
		return i18n.T("alert.info")
	case AL_WARNING:
		return i18n.T("alert.warning")
	case AL_CRITICAL:
		return i18n.T("alert.critical")
	default:
		return level
	}
}

func InitMetricConf(paths []string) error {
	conf := YHCMetricConfig{}
	for _, p := range paths {
		if !path.IsAbs(p) {
			p = path.Join(runtimedef.GetYHCHome(), p)
		}
		c, err := loadMetricConf(p)
		if err != nil {
			return err
		}
		for _, metric := range c.Metrics {
			if len(metric.ModuleName) == 0 {
				metric.ModuleName = M_CUSTOM
			}
			conf.Metrics = append(conf.Metrics, metric)
		}
	}
	_metricConfig = &conf
	return nil
}

func loadMetricConf(p string) (config *YHCMetricConfig, err error) {
	config = new(YHCMetricConfig)
	if !fs.IsFileExist(p) {
		return config, &errdef.ErrFileNotFound{FName: p}
	}
	if _, err := toml.DecodeFile(p, config); err != nil {
		return config, &errdef.ErrFileParseFailed{FName: p, Err: err}
	}
	return config, nil
}

func GetMetricConf() *YHCMetricConfig {
	return _metricConfig
}

// GetMetricAlias 根据当前语言获取指标别名
func (m *YHCMetric) GetMetricAlias() string {
	lang := i18n.GetLanguage()
	switch lang {
	case i18n.EnUS:
		if m.NameAliasEn != "" {
			return m.NameAliasEn
		}
	}
	return m.NameAlias // 默认中文
}

// GetColumnAlias 根据当前语言获取列别名
func (m *YHCMetric) GetColumnAlias(columnName string) string {
	lang := i18n.GetLanguage()
	var aliasMap map[string]string
	
	switch lang {
	case i18n.EnUS:
		aliasMap = m.ColumnAliasEn
	default:
		aliasMap = m.ColumnAlias
	}
	
	if aliasMap != nil {
		if alias, ok := aliasMap[columnName]; ok {
			return alias
		}
	}
	
	// 如果当前语言没有翻译，尝试使用中文
	if lang != i18n.ZhCN && m.ColumnAlias != nil {
		if alias, ok := m.ColumnAlias[columnName]; ok {
			return alias
		}
	}
	
	return columnName
}

// GetAllColumnAliases 获取当前语言的所有列别名映射
func (m *YHCMetric) GetAllColumnAliases() map[string]string {
	lang := i18n.GetLanguage()
	
	switch lang {
	case i18n.EnUS:
		if m.ColumnAliasEn != nil && len(m.ColumnAliasEn) > 0 {
			return m.ColumnAliasEn
		}
	}
	
	return m.ColumnAlias // 默认中文
}

// GetAlertDescription 根据当前语言获取告警描述
func (a *AlertDetails) GetAlertDescription() string {
	lang := i18n.GetLanguage()
	
	switch lang {
	case i18n.EnUS:
		if a.DescriptionEn != "" {
			return a.DescriptionEn
		}
	}
	
	return a.Description
}

// GetAlertSuggestion 根据当前语言获取告警建议
func (a *AlertDetails) GetAlertSuggestion() string {
	lang := i18n.GetLanguage()
	
	switch lang {
	case i18n.EnUS:
		if a.SuggestionEn != "" {
			return a.SuggestionEn
		}
	}
	
	return a.Suggestion
}
