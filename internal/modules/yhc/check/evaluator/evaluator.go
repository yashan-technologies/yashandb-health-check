package evaluator

import (
	"yhc/defs/confdef"
	"yhc/internal/modules/yhc/check/define"
	"yhc/utils/jsonutil"

	"git.yasdb.com/go/yaslog"
)

type Evaluator struct {
	log           yaslog.YasLog
	result        map[define.MetricName][]*define.YHCItem
	failedItem    map[define.MetricName][]*define.YHCItem
	evaluateModel *confdef.EvaluateModel
}

func NewEvaluator(log yaslog.YasLog, result map[define.MetricName][]*define.YHCItem, failedItem map[define.MetricName][]*define.YHCItem) *Evaluator {
	return &Evaluator{
		result:        result,
		failedItem:    failedItem,
		log:           log,
		evaluateModel: confdef.GetEvaluateModel(),
	}
}

func (e *Evaluator) Evaluate() *define.EvaluateResult {
	score := e.getScore()
	healthStatus := e.getHealthStatus(score)
	alertSummary := e.getAlertSummary()
	return &define.EvaluateResult{
		EvaluateModel: confdef.GetEvaluateModel(),
		Score:         score,
		HealthStatus:  healthStatus,
		AlertSummary:  alertSummary,
	}
}

func (e *Evaluator) getScore() float64 {
	totalWeight, metricWeight := e.getMetricWeight()
	var score float64
	for metric, item := range e.result {
		weight := metricWeight[string(metric)]
		metricScore := e.evaluateModel.TotalScore * (weight) / totalWeight
		alertTotalWeight, _ := e.getAlertWeight(item)
		if alertTotalWeight >= e.evaluateModel.MaxAlertTotalWeight {
			alertTotalWeight = e.evaluateModel.MaxAlertTotalWeight
		}
		score += metricScore * (1 - alertTotalWeight/e.evaluateModel.MaxAlertTotalWeight)
	}
	return score
}

func (e *Evaluator) getHealthStatus(score float64) string {
	healthStatusList := []string{confdef.HL_EXCELLENT, confdef.HL_GOOD, confdef.HL_Fair, confdef.HL_POOR, confdef.HL_CRITACAL}
	for _, healthStatus := range healthStatusList {
		healthModel, ok := e.evaluateModel.HealthModel[healthStatus]
		if !ok {
			continue
		}
		if score < healthModel.Min || score > healthModel.Max {
			continue
		}
		return confdef.GetHealthStatusAlias(healthStatus)
	}
	return confdef.GetHealthStatusAlias(confdef.HL_UNKNOW)
}

func (e *Evaluator) getAlertSummary() *define.AlertSummary {
	res := &define.AlertSummary{}
	for _, result := range e.result {
		for _, item := range result {
			for level, alerts := range item.Alerts {
				switch level {
				case confdef.AL_INFO:
					res.InfoCount += len(alerts)
				case confdef.AL_WARNING:
					res.WarningCount += len(alerts)
				case confdef.AL_CRITICAL:
					res.CriticalCount += len(alerts)
				default:
					e.log.Debugf("invalid alert level %s, skip", level)
				}
			}
		}
	}
	return res
}

func (e *Evaluator) getAlertWeight(items []*define.YHCItem) (float64, map[string]float64) {
	var totalWeight float64
	res := map[string]float64{}
	for _, item := range items {
		for alertLevel, alertDetail := range item.Alerts {
			weight, ok := e.evaluateModel.AlertsWeight[alertLevel]
			if !ok {
				e.log.Debugf("failed to find alert weight of %s, skip alert %s", alertLevel, jsonutil.ToJSONString(alertDetail))
			}
			totalWeight += weight
			res[alertLevel] = totalWeight
			if e.evaluateModel.IgnoreSameAlert {
				continue
			}
		}
	}
	return totalWeight, res
}

func (e *Evaluator) getMetricWeight() (float64, map[string]float64) {
	res := map[string]float64{}
	var totalWeight float64
	allMetrics := e.getAllMetric()
	for metric, weight := range e.evaluateModel.MetricsWeight {
		if _, ok := allMetrics[metric]; ok {
			totalWeight += weight
			delete(allMetrics, metric)
			res[metric] = weight
		}
	}
	for module, weight := range e.evaluateModel.ModuleWeight {
		moduleMetrics := confdef.GetModuleMetrics()
		metrics, ok := moduleMetrics[module]
		if !ok {
			continue
		}
		for _, metric := range metrics {
			if _, ok := allMetrics[metric]; ok {
				totalWeight += weight
				delete(allMetrics, metric)
				res[metric] = weight
			}
		}
	}
	for metric := range allMetrics {
		totalWeight += e.evaluateModel.DefaultMetricWeight
		res[metric] = e.evaluateModel.DefaultMetricWeight
	}
	return totalWeight, res
}

func (e *Evaluator) getAllMetric() map[string]struct{} {
	res := map[string]struct{}{}
	for metric := range e.result {
		res[string(metric)] = struct{}{}
	}
	if e.evaluateModel.IgnoreFailedMetric {
		return res
	}
	for metric := range e.failedItem {
		res[string(metric)] = struct{}{}
	}
	return res
}
