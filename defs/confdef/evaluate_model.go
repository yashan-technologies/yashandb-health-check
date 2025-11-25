package confdef

import (
	"path"

	"yhc/defs/errdef"
	"yhc/defs/runtimedef"
	"yhc/i18n"

	"git.yasdb.com/go/yasutil/fs"
	"github.com/BurntSushi/toml"
)

const (
	HL_EXCELLENT = "excellent"
	HL_GOOD      = "good"
	HL_Fair      = "fair"
	HL_POOR      = "poor"
	HL_CRITACAL  = "critical"
	HL_UNKNOW    = "unknow"
)

var _evaluateModel *EvaluateModel

type EvaluateModel struct {
	TotalScore          float64                  `toml:"total_score"`            // 总分数
	MetricsWeight       map[string]float64       `toml:"metrics_weight"`         // 指标权重
	ModuleWeight        map[string]float64       `toml:"modele_weight"`          // 用于批量指定该模块下所有指标的权重
	DefaultMetricWeight float64                  `toml:"default_metric_weight"`  // 指标的默认权重，如果某个指标未在metricsWeight中定义权重，将会使用默认权重
	AlertsWeight        map[string]float64       `toml:"alerts_weight"`          // 不同级别告警占对应指标总权重的百分比，告警采用扣分制，最多扣除该项指标的总分数
	MaxAlertTotalWeight float64                  `toml:"max_alert_total_weight"` // 同一指标项的告警总权重，通常定义为100
	IgnoreSameAlert     bool                     `toml:"ignore_same_alert"`      // 同一指标同一级别的告警可能有多条，为true表示同一指标，同一级别的告警只扣一次分数
	IgnoreFailedMetric  bool                     `toml:"ignore_failed_metric"`   // 部分指标如果检查失败，将不会展示在报告中。为true表示不将检查失败的指标权重纳入计算
	HealthModel         map[string]ScoreInterval `toml:"health_model"`           // 不同健康状态对应的分数范围
	HealthStatusAlias   map[string]string        `toml:"health_status_alias"`    // 健康状态的别名，用于报告展示
}

type ScoreInterval struct {
	Min float64 `toml:"min"`
	Max float64 `toml:"max"`
}

func loadEvaluateModel(p string) (*EvaluateModel, error) {
	conf := &EvaluateModel{}
	if !fs.IsFileExist(p) {
		return conf, &errdef.ErrFileNotFound{FName: p}
	}
	if _, err := toml.DecodeFile(p, conf); err != nil {
		return conf, &errdef.ErrFileParseFailed{FName: p, Err: err}
	}
	return conf, nil
}

func initEvaluateModel(p string) error {
	if !path.IsAbs(p) {
		p = path.Join(runtimedef.GetYHCHome(), p)
	}
	conf, err := loadEvaluateModel(p)
	if err != nil {
		return err
	}
	_evaluateModel = conf
	return nil
}

func GetEvaluateModel() *EvaluateModel {
	return _evaluateModel
}

// GetHealthStatusAlias 根据当前语言获取健康状态别名
func GetHealthStatusAlias(level string) string {
	switch level {
	case HL_EXCELLENT:
		return i18n.T("health.excellent")
	case HL_GOOD:
		return i18n.T("health.good")
	case HL_Fair:
		return i18n.T("health.fair")
	case HL_POOR:
		return i18n.T("health.poor")
	case HL_CRITACAL:
		return i18n.T("health.critical")
	case HL_UNKNOW:
		return i18n.T("health.unknown")
	default:
		return level
	}
}
