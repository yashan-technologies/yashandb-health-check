package jsonparser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"yhc/defs/compiledef"
	"yhc/defs/confdef"
	"yhc/defs/timedef"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/stringutil"

	"git.yasdb.com/go/yaslog"
)

const (
	_REPORT_TITLE = "YashanDB 深度巡检报告"
	_FILE_CONTROL = "此文档仅供崖山科技有限公司与最终用户审阅，不得向与此无关的个人或机构传阅或复制。"
	_AUTHOR       = "Yashan Health Check"
	_CHANGE_LOG   = "生成巡检报告"

	_metric_name  = "metricName"
	_alert_number = "alertNumber"

	_module_name       = "moduleName"
	_alert_level       = "alertLevel"
	_alert_labels      = "alertLabels"
	_alert_expersion   = "alertExpresion"
	_alert_value       = "alertValue"
	_alert_suggestion  = "alertSuggestion"
	_alert_description = "alertDescription"

	_node_id       = "nodeID"
	_listen_addr   = "listenAddr"
	_node_role     = "nodeRole"
	_database_name = "databaseName"
	_yasdb_user    = "yasdbUser"
)

// 将不同指标的数据合并到一个map中，只支持map之间的合并
var _mergeMetricMap = map[define.MetricName][]define.MetricName{
	define.METRIC_HOST_INFO: {
		define.METRIC_HOST_INFO,
		define.METRIC_HOST_CPU_INFO,
	},
	define.METRIC_YASDB_DATABASE: {
		define.METRIC_YASDB_DATABASE,
		define.METRIC_YASDB_INSTANCE,
		define.METRIC_YASDB_LISTEN_ADDR,
		define.METRIC_YASDB_DEPLOYMENT_ARCHITECTURE,
	},
	define.METRIC_YASDB_TABLE_LOCK_WAIT: {
		define.METRIC_YASDB_TABLE_LOCK_WAIT,
		define.METRIC_YASDB_ROW_LOCK_WAIT,
	},
}

var _fixedTableLayoutMetrics = map[define.MetricName]struct{}{
	define.METRIC_YASDB_SLOW_LOG: {},
}

type merge struct {
	parentModule  string
	originMetrics []string
	targetTitle   string
}

var _mergeOldMenuToNew []merge = []merge{
	{
		parentModule: string(define.MODULE_HOST_WORKLOAD),
		targetTitle:  "CPU使用情况",
		originMetrics: []string{
			string(define.METRIC_HOST_CURRENT_CPU_USAGE),
			string(define.METRIC_HOST_HISTORY_CPU_USAGE),
		},
	},
	{
		parentModule: string(define.MODULE_HOST_WORKLOAD),
		targetTitle:  "内存使用情况",
		originMetrics: []string{
			string(define.METRIC_HOST_CURRENT_MEMORY_USAGE),
			string(define.METRIC_HOST_HISTORY_MEMORY_USAGE),
		},
	},
	{
		parentModule: string(define.MODULE_HOST_WORKLOAD),
		targetTitle:  "网络使用情况",
		originMetrics: []string{
			string(define.METRIC_HOST_CURRENT_NETWORK_IO),
			string(define.METRIC_HOST_HISTORY_NETWORK_IO),
		},
	},
	{
		parentModule: string(define.MODULE_HOST_WORKLOAD),
		targetTitle:  "磁盘使用情况",
		originMetrics: []string{
			string(define.METRIC_HOST_CURRENT_DISK_IO),
			string(define.METRIC_HOST_HISTORY_DISK_IO),
		},
	},
	{
		parentModule: string(define.MODULE_OVERVIEW_HOST),
		targetTitle:  "主机信息",
		originMetrics: []string{
			string(define.METRIC_HOST_INFO),
			string(define.METRIC_HOST_CPU_INFO),
			string(define.METRIC_HOST_DISK_INFO),
			string(define.METRIC_HOST_DISK_BLOCK_INFO),
			string(define.METRIC_HOST_MEMORY_INFO),
			string(define.METRIC_HOST_NETWORK_INFO),
			string(define.METRIC_HOST_FIREWALLD),
			string(define.METRIC_HOST_IPTABLES),
		},
	},
	{
		parentModule: string(define.MODULE_OVERVIEW_YASDB),
		targetTitle:  "数据库信息",
		originMetrics: []string{
			string(define.METRIC_YASDB_DATABASE),
			string(define.METRIC_YASDB_ARCHIVE_THRESHOLD),
			string(define.METRIC_YASDB_FILE_PERMISSION),
		},
	},
	{
		parentModule: string(define.MODULE_OBJECT_NUMBER),
		targetTitle:  "对象总数",
		originMetrics: []string{
			string(define.METRIC_YASDB_OBJECT_COUNT),
			string(define.METRIC_YASDB_SEGMENTS_COUNT),
			string(define.METRIC_YASDB_SEGMENTS_SUMMARY),
			string(define.METRIC_YASDB_OBJECT_SUMMARY),
		},
	},
	{
		parentModule: string(define.MODULE_YASDB_CONTROLFILE),
		targetTitle:  "控制文件",
		originMetrics: []string{
			string(define.METRIC_YASDB_CONTROLFILE),
			string(define.METRIC_YASDB_CONTROLFILE_COUNT),
		},
	},
	{
		parentModule: string(define.MODULE_LOG),
		targetTitle:  "REDO日志分析",
		originMetrics: []string{
			string(define.METRIC_YASDB_REDO_LOG),
			string(define.METRIC_YASDB_REDO_LOG_COUNT),
		},
	},
	{
		parentModule: string(define.MODULE_YASDB_PERFORMANCE),
		targetTitle:  "内存池命中率",
		originMetrics: []string{
			string(define.METRIC_YASDB_BUFFER_HIT_RATE),
			string(define.METRIC_YASDB_HISTORY_BUFFER_HIT_RATE),
		},
	},
	{
		parentModule: string(define.MODULE_YASDB_PERFORMANCE),
		targetTitle:  "TOP10 SQL",
		originMetrics: []string{
			string(define.METRIC_YASDB_TOP_SQL_BY_CPU_TIME),
			string(define.METRIC_YASDB_TOP_SQL_BY_BUFFER_GETS),
			string(define.METRIC_YASDB_TOP_SQL_BY_DISK_READS),
			string(define.METRIC_YASDB_TOP_SQL_BY_PARSE_CALLS),
		},
	},
	{
		parentModule: string(define.MODULE_YASDB_PERFORMANCE),
		targetTitle:  "性能配置检查",
		originMetrics: []string{
			string(define.METRIC_HOST_HUGE_PAGE),
			string(define.METRIC_HOST_SWAP_MEMORY),
		},
	},
	{
		parentModule: string(define.MODULE_LOG),
		targetTitle:  "慢日志分析",
		originMetrics: []string{
			string(define.METRIC_YASDB_SLOW_LOG_PARAMETER),
			string(define.METRIC_YASDB_SLOW_LOG),
			string(define.METRIC_YASDB_SLOW_LOG_FILE),
		},
	},
	{
		parentModule: string(define.MODULE_LOG),
		targetTitle:  "UNDO日志分析",
		originMetrics: []string{
			string(define.METRIC_YASDB_UNDO_LOG_SIZE),
			string(define.METRIC_YASDB_UNDO_LOG_TOTAL_BLOCK),
			string(define.METRIC_YASDB_UNDO_LOG_RUNNING_TRANSACTIONS),
		},
	},
}

type MetricParseFunc func(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error

type JsonParser struct {
	log            yaslog.YasLog
	base           define.CheckerBase
	startCheckTime time.Time
	endCheckTime   time.Time
	metrics        []*confdef.YHCMetric
	results        map[define.MetricName][]*define.YHCItem
	evaluateResult *define.EvaluateResult
}

func NewJsonParser(log yaslog.YasLog, base define.CheckerBase, startCheck, endCheck time.Time, metrics []*confdef.YHCMetric, results map[define.MetricName][]*define.YHCItem, evaluateResult *define.EvaluateResult) *JsonParser {
	parser := &JsonParser{
		log:            log,
		metrics:        metrics,
		results:        results,
		startCheckTime: startCheck,
		endCheckTime:   endCheck,
		base:           base,
		evaluateResult: evaluateResult,
	}
	return parser
}

// todo: 这个parse函数各个模块之间的关系处理有点问题，需要优化
// todo: 包括wordgenner的模块处理也有问题，后续优化！
func (j *JsonParser) Parse() *define.PandoraReport {
	report := &define.PandoraReport{
		ReportTitle: _REPORT_TITLE,
		FileControl: _FILE_CONTROL,
		Author:      _AUTHOR,
		ChangeLog:   _CHANGE_LOG,
		Time:        j.startCheckTime.Format(timedef.TIME_FORMAT),
		CostTime:    int(j.endCheckTime.Sub(j.startCheckTime).Seconds()),
		Version:     compiledef.GetAPPVersion(),
	}
	j.mergeMetrics()
	j.addCheckSummary(report)
	for i, module := range confdef.GetModuleConf().Modules {
		menu := &define.PandoraMenu{IsMenu: true, Title: confdef.GetModuleAlias(module.Name), TitleEn: module.Name, MenuIndex: i}
		report.ReportData = append(report.ReportData, menu)
		j.dealYHCModule(module, menu)
	}
	j.mergeElements(report)
	j.filterSingleElementTitle(report)
	j.addElementToEmptyMenus(report)
	j.countAlerts(report)
	return report
}

func (j *JsonParser) countAlerts(report *define.PandoraReport) {
	var fn func(menu *define.PandoraMenu)
	fn = func(menu *define.PandoraMenu) {
		for _, child := range menu.Children {
			fn(child)
		}
		// count alert in current menu
		for _, child := range menu.Children {
			menu.InfoCount += child.InfoCount
			menu.WarningCount += child.WarningCount
			menu.CriticalCount += child.CriticalCount
		}
		for _, element := range menu.Elements {
			if element.ElementType == define.ET_ALERT {
				attributes, ok := element.Attributes.(define.AlertAttributes)
				if !ok {
					j.log.Errorf("attributes type of element type %s is not %T but %T", define.ET_ALERT, define.AlertAttributes{}, element.Attributes)
					continue
				}
				switch attributes.AlertType {
				case define.AT_INFO:
					menu.InfoCount++
				case define.AT_WARNING:
					menu.WarningCount++
				case define.AT_CRITICAL:
					menu.CriticalCount++
				default:
					j.log.Errorf("unknown alert type %s", attributes.AlertType)
				}
			}
		}
	}

	for _, menu := range report.ReportData {
		fn(menu)
	}
}

func (j *JsonParser) addElementToEmptyMenus(report *define.PandoraReport) {
	for _, menu := range report.ReportData {
		j.addElementToEmptyMenu(menu)
	}
}

func (j *JsonParser) addElementToEmptyMenu(menu *define.PandoraMenu) {
	emptyElement := &define.PandoraElement{
		ElementType: define.ET_PRE,
		InnerText:   "当前模块无指标",
	}
	if len(menu.Children) == 0 && len(menu.Elements) == 0 {
		menu.Elements = append(menu.Elements, emptyElement)
	}
	for _, child := range menu.Children {
		j.addElementToEmptyMenu(child)
	}
}

func (j *JsonParser) addCheckSummary(report *define.PandoraReport) {
	menu := &define.PandoraMenu{IsMenu: false, Title: "健康检查概览"}
	j.checkSummary(report.Time, report.CostTime, menu)
	j.checkNodesSummary(menu)
	j.evaluateSummary(menu)
	j.alertSummary(menu)
	j.moduleSummary(menu)
	report.ReportData = append(report.ReportData, menu)
}

func (j *JsonParser) evaluateSummary(menu *define.PandoraMenu) {
	descAttr := &define.DescriptionAttributes{}
	data := []*define.DescriptionData{
		{Label: "健康检查总分", Value: fmt.Sprintf("%.2f", j.evaluateResult.EvaluateModel.TotalScore)},
		{Label: "本次健康检查得分", Value: fmt.Sprintf("%.2f", j.evaluateResult.Score)},
		{Label: "本次巡检健康状况", Value: j.evaluateResult.HealthStatus},
		{Label: "本次巡检告警统计", Value: fmt.Sprintf("严重级别告警%d个，警告级别告警%d个，提示级别告警%d个，建议查看【告警详情】模块确认并处理相关问题",
			j.evaluateResult.AlertSummary.CriticalCount,
			j.evaluateResult.AlertSummary.WarningCount,
			j.evaluateResult.AlertSummary.InfoCount)},
		{Label: "得分评估模型", Value: j.getScoreModelString()},
		{Label: "告警权重", Value: j.getAlertWeightString()},
	}
	descAttr.Data = data
	menu.Elements = append(menu.Elements, &define.PandoraElement{
		ElementType:  define.ET_DESCRIPTION,
		Attributes:   descAttr,
		ElementTitle: "健康检查得分详情",
	})
}

func (j *JsonParser) getScoreModelString() string {
	var buf bytes.Buffer
	healthStatusList := []string{confdef.HL_EXCELLENT, confdef.HL_GOOD, confdef.HL_Fair, confdef.HL_POOR, confdef.HL_CRITACAL}
	for _, healthStatus := range healthStatusList {
		interval, ok := j.evaluateResult.EvaluateModel.HealthModel[healthStatus]
		if !ok {
			continue
		}
		healthStatusAlias, ok := j.evaluateResult.EvaluateModel.HealthStatusAlias[healthStatus]
		if !ok {
			healthStatusAlias = healthStatus
		}
		buf.WriteString(fmt.Sprintf("%s(%.2f<=分数<=%.2f)  ", healthStatusAlias, interval.Min, interval.Max))
	}
	return buf.String()
}

func (j *JsonParser) getAlertWeightString() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("单项指标告警总权重：%.2f，", j.evaluateResult.EvaluateModel.MaxAlertTotalWeight))
	buf.WriteString(fmt.Sprintf("相同指标忽略相同级别告警：%v，", j.evaluateResult.EvaluateModel.IgnoreSameAlert))
	buf.WriteString("单项告警权重：")
	alertList := []string{confdef.AL_CRITICAL, confdef.AL_WARNING, confdef.AL_INFO}
	for _, alert := range alertList {
		weight, ok := j.evaluateResult.EvaluateModel.AlertsWeight[alert]
		if !ok {
			continue
		}
		alertLevelAlisa, ok := confdef.AlertLevelMap[alert]
		if !ok {
			alertLevelAlisa = alert
		}
		buf.WriteString(fmt.Sprintf("%s(%.2f)  ", alertLevelAlisa, weight))
	}
	return buf.String()
}

func (j *JsonParser) checkSummary(checkTime string, costTime int, menu *define.PandoraMenu) {
	descAttr := &define.DescriptionAttributes{}
	existAlertItems := 0
	for _, items := range j.results {
		for _, item := range items {
			if len(item.Alerts) != 0 {
				existAlertItems += 1
			}
		}
	}
	data := []*define.DescriptionData{
		{Label: "健康检查开始时间", Value: checkTime},
		{Label: "健康检查花费时间", Value: fmt.Sprintf("%d 秒", costTime)},
		{Label: "检查项共计", Value: fmt.Sprintf("%d 个", len(j.metrics))},
		{Label: "存在告警的检查项", Value: fmt.Sprintf("%d 个", existAlertItems)},
		{Label: "YashanDB Home目录", Value: j.base.DBInfo.YasdbHome},
		{Label: "YashanDB Data目录", Value: j.base.DBInfo.YasdbData},
		{Label: "YashanDB用户", Value: j.base.DBInfo.YasdbUser},
		{Label: "数据库名称", Value: j.base.DBInfo.DatabaseName},
	}
	if !j.base.MultipleNodes {
		data = append(data, &define.DescriptionData{
			Label: "监听地址",
			Value: j.base.DBInfo.ListenAddr,
		})
	}
	descAttr.Data = data
	menu.Elements = append(menu.Elements, &define.PandoraElement{
		ElementType:  define.ET_DESCRIPTION,
		Attributes:   descAttr,
		ElementTitle: "检查概览信息",
	})
}

func (j *JsonParser) checkNodesSummary(menu *define.PandoraMenu) {
	if !j.base.MultipleNodes {
		return
	}

	element := &define.PandoraElement{
		ElementType:  define.ET_TABLE,
		ElementTitle: "节点信息",
	}
	res := make([]map[string]interface{}, 0)
	for _, node := range j.base.NodeInfos {
		data := map[string]interface{}{
			_database_name: node.DatabaseName,
			_node_id:       node.NodeID,
			_listen_addr:   node.ListenAddr,
			_node_role:     node.Role,
			_yasdb_user:    node.User,
		}
		res = append(res, data)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i][_node_id].(string) < res[j][_node_id].(string)
	})
	tabAttr := define.TableAttributes{
		TableColumns: []*define.TableColumn{
			{Title: "数据库名称", DataIndex: _database_name},
			{Title: "节点ID", DataIndex: _node_id},
			{Title: "监听地址", DataIndex: _listen_addr},
			{Title: "数据库角色", DataIndex: _node_role},
			{Title: "用户", DataIndex: _yasdb_user},
		},
		DataSource: res,
	}
	element.Attributes = tabAttr
	menu.Elements = append(menu.Elements, element)
}

func (j *JsonParser) alertSummary(menu *define.PandoraMenu) {
	res := make([]map[string]interface{}, 0)
	for _, metricName := range confdef.GetMetricOrder() {
		result, ok := j.results[define.MetricName(metricName)]
		if !ok {
			continue
		}
		if len(result) == 0 {
			continue
		}
		metric, err := j.getMetric(metricName)
		if err != nil {
			j.log.Debugf("failed to get metric by %s, err: %v", metricName, err)
			continue
		}
		moduleNameAlias := []string{}
		for _, module := range confdef.GetMetricModules(metricName) {
			moduleNameAlias = append(moduleNameAlias, confdef.GetModuleAlias(module))
		}
		for _, item := range result {
			for level, alerts := range item.Alerts {
				for _, alert := range alerts {
					var labels []string
					for key, value := range alert.Labels {
						labels = append(labels, fmt.Sprintf("{%s:%s}", j.getColumnAlias(metric, key), value))
					}
					m := map[string]interface{}{
						_alert_level:       define.AlertTypeAliasMap[define.AlertType(level)],
						_alert_labels:      strings.Join(labels, stringutil.STR_NEWLINE),
						_alert_expersion:   alert.Expression,
						_alert_description: alert.Description,
						_alert_suggestion:  alert.Suggestion,
						_alert_value:       alert.Value,
						_module_name:       strings.Join(moduleNameAlias, "-->"),
						_metric_name:       metric.NameAlias,
					}
					res = append(res, m)
				}
			}
		}
	}
	tabAttr := define.TableAttributes{
		TableColumns: []*define.TableColumn{
			{Title: "指标名称", DataIndex: _metric_name},
			{Title: "模块", DataIndex: _module_name},
			{Title: "告警级别", DataIndex: _alert_level},
			{Title: "告警描述", DataIndex: _alert_description},
			{Title: "表达式", DataIndex: _alert_expersion},
			{Title: "值", DataIndex: _alert_value},
			{Title: "告警建议", DataIndex: _alert_suggestion},
			{Title: "告警标签", DataIndex: _alert_labels},
		},
		DataSource:  res,
		TableLayout: define.TABLE_LAYOUT_FIXED,
	}
	element := &define.PandoraElement{
		ElementType:  define.ET_TABLE,
		ElementTitle: "告警详情",
		Attributes:   tabAttr,
	}
	menu.Elements = append(menu.Elements, element)
}

func (j *JsonParser) moduleSummary(menu *define.PandoraMenu) {
	modules := []string{
		string(define.MODULE_OVERVIEW),
		string(define.MODULE_HOST),
		string(define.MODULE_YASDB),
		string(define.MODULE_OBJECT),
		string(define.MODULE_SECURITY),
		string(define.MODULE_LOG),
		string(define.MODULE_CUSTOM),
	}
	for _, module := range modules {
		element := j.genModuleElement(module)
		if element != nil {
			menu.Elements = append(menu.Elements, element)
		}
	}
}

func (j *JsonParser) genModuleElement(module string) *define.PandoraElement {
	element := &define.PandoraElement{
		ElementType:  define.ET_TABLE,
		ElementTitle: fmt.Sprintf("%s模块检查项列表", confdef.GetModuleAlias(module)),
	}
	res := make([]map[string]interface{}, 0)
	for _, metric := range j.metrics {
		if metric.ModuleName == module {
			itemResults, ok := j.results[define.MetricName(metric.Name)]
			if !ok {
				continue
			}
			alertCount := 0
			for _, item := range itemResults {
				for _, alerts := range item.Alerts {
					alertCount += len(alerts)
				}
			}
			data := map[string]interface{}{
				_metric_name:  metric.NameAlias,
				_alert_number: alertCount,
			}
			res = append(res, data)
		}
	}
	if len(res) == 0 {
		return nil
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i][_alert_number].(int) > res[j][_alert_number].(int)
	})
	tabAttr := define.TableAttributes{
		TableColumns: []*define.TableColumn{
			{Title: "指标名称", DataIndex: _metric_name},
			{Title: "告警数量", DataIndex: _alert_number},
		},
		DataSource: res,
	}
	element.Attributes = tabAttr
	return element
}

func (j *JsonParser) dealYHCModule(module *confdef.YHCModuleNode, menu *define.PandoraMenu) {
	if module == nil {
		return
	}
	if len(module.Children) != 0 {
		for i, childModule := range module.Children {
			childMenu := &define.PandoraMenu{IsMenu: true, Title: childModule.NameAlias, TitleEn: childModule.Name, MenuIndex: i}
			menu.Children = append(menu.Children, childMenu)
			j.dealYHCModule(childModule, childMenu)
		}
	}
	for i, metricName := range module.MetricNames {
		results, ok := j.results[define.MetricName(metricName)]
		if !ok {
			continue
		}
		metric, err := j.getMetric(metricName)
		if err != nil {
			continue
		}
		fn, err := j.genMetricParseFunc(metric)
		if err != nil {
			j.log.Errorf("failed to gen parse func of metric %s", metricName)
			continue
		}
		childMenu := &define.PandoraMenu{Title: metric.NameAlias, TitleEn: metricName, MenuIndex: len(module.Children) + i}
		for _, result := range results {
			if err := fn(childMenu, result, metric); err != nil {
				j.log.Errorf("failed to parse metric %s, err: %v", metricName, err)
				continue
			}
		}
		menu.Children = append(menu.Children, childMenu)
	}
}

func (j *JsonParser) filterSingleElementTitle(report *define.PandoraReport) {
	for _, menu := range report.ReportData {
		j.filterElementTitle(menu)
	}
}

func (j *JsonParser) filterElementTitle(menu *define.PandoraMenu) {
	if menu == nil {
		return
	}
	for _, child := range menu.Children {
		j.filterElementTitle(child)
	}
	if len(menu.Elements) == 0 || len(menu.Elements) > 1 {
		return
	}
	menu.Elements[0].ElementTitle = ""

}

func (j *JsonParser) genMetricParseFunc(metric *confdef.YHCMetric) (MetricParseFunc, error) {
	if !metric.Default {
		switch metric.MetricType {
		case confdef.MT_SQL:
			return j.genCustomSqlParseFunc(metric)
		case confdef.MT_BASH:
			return j.genCustomBashParseFunc(metric)
		default:
			return nil, fmt.Errorf("invalid metric type %s", metric.MetricType)
		}
	}
	return j.genDefaultMetricParseFunc(metric)
}

func (j *JsonParser) genDefaultMetricParseFunc(metric *confdef.YHCMetric) (MetricParseFunc, error) {
	parseFuncMap := map[define.MetricName]MetricParseFunc{
		define.METRIC_YASDB_INSTANCE:                                                               j.parseMap,
		define.METRIC_YASDB_DATABASE:                                                               j.parseMap,
		define.METRIC_YASDB_FILE_PERMISSION:                                                        j.parseTable,
		define.METRIC_YASDB_LISTEN_ADDR:                                                            j.parseMap,
		define.METRIC_YASDB_ARCHIVE_THRESHOLD:                                                      j.parseTable,
		define.METRIC_YASDB_OS_AUTH:                                                                j.parseMap,
		define.METRIC_HOST_INFO:                                                                    j.parseMap,
		define.METRIC_HOST_FIREWALLD:                                                               j.parseMap,
		define.METRIC_HOST_IPTABLES:                                                                j.parseCode,
		define.METRIC_HOST_CPU_INFO:                                                                j.parseMap,
		define.METRIC_HOST_DISK_INFO:                                                               j.parseTable,
		define.METRIC_HOST_DISK_BLOCK_INFO:                                                         j.parseTable,
		define.METRIC_HOST_BIOS_INFO:                                                               j.parseCode,
		define.METRIC_HOST_MEMORY_INFO:                                                             j.parseTable,
		define.METRIC_HOST_NETWORK_INFO:                                                            j.parseTable,
		define.METRIC_HOST_HISTORY_CPU_USAGE:                                                       j.parseHostWorkload,
		define.METRIC_HOST_CURRENT_CPU_USAGE:                                                       j.parseHostWorkload,
		define.METRIC_HOST_CURRENT_DISK_IO:                                                         j.parseHostWorkload,
		define.METRIC_HOST_HISTORY_DISK_IO:                                                         j.parseHostWorkload,
		define.METRIC_HOST_CURRENT_MEMORY_USAGE:                                                    j.parseHostWorkload,
		define.METRIC_HOST_HISTORY_MEMORY_USAGE:                                                    j.parseHostWorkload,
		define.METRIC_HOST_CURRENT_NETWORK_IO:                                                      j.parseHostWorkload,
		define.METRIC_HOST_HISTORY_NETWORK_IO:                                                      j.parseHostWorkload,
		define.METRIC_YASDB_ARCHIVE_DEST_STATUS:                                                    j.parseTable,
		define.METRIC_YASDB_ARCHIVE_LOG:                                                            j.parseTable,
		define.METRIC_YASDB_ARCHIVE_LOG_SPACE:                                                      j.parseMap,
		define.METRIC_YASDB_PARAMETER:                                                              j.parseMap,
		define.METRIC_YASDB_TABLESPACE:                                                             j.parseTable,
		define.METRIC_YASDB_CONTROLFILE_COUNT:                                                      j.parseMap,
		define.METRIC_YASDB_DEPLOYMENT_ARCHITECTURE:                                                j.parseMap,
		define.METRIC_YASDB_CONTROLFILE:                                                            j.parseTable,
		define.METRIC_YASDB_DATAFILE:                                                               j.parseTable,
		define.METRIC_YASDB_SESSION:                                                                j.parseMap,
		define.METRIC_YASDB_WAIT_EVENT:                                                             j.parseTable,
		define.METRIC_YASDB_OBJECT_COUNT:                                                           j.parseMap,
		define.METRIC_YASDB_OBJECT_SUMMARY:                                                         j.parseTable,
		define.METRIC_YASDB_SEGMENTS_COUNT:                                                         j.parseMap,
		define.METRIC_YASDB_SEGMENTS_SUMMARY:                                                       j.parseTable,
		define.METRIC_YASDB_INDEX_BLEVEL:                                                           j.parseTable,
		define.METRIC_YASDB_INDEX_COLUMN:                                                           j.parseTable,
		define.METRIC_YASDB_INDEX_INVISIBLE:                                                        j.parseTable,
		define.METRIC_YASDB_REDO_LOG:                                                               j.parseTable,
		define.METRIC_YASDB_REDO_LOG_COUNT:                                                         j.parseTable,
		define.METRIC_YASDB_RUN_LOG_ERROR:                                                          j.parseText,
		define.METRIC_YASDB_INDEX_TABLE_INDEX_NOT_TOGETHER:                                         j.parseTable,
		define.METRIC_YASDB_INDEX_OVERSIZED:                                                        j.parseTable,
		define.METRIC_YASDB_SEQUENCE_NO_AVAILABLE:                                                  j.parseTable,
		define.METRIC_YASDB_TASK_RUNNING:                                                           j.parseTable,
		define.METRIC_YASDB_PACKAGE_NO_PACKAGE_PACKAGE_BODY:                                        j.parseTable,
		define.METRIC_YASDB_SECURITY_LOGIN_PASSWORD_STRENGTH:                                       j.parseMap,
		define.METRIC_YASDB_AUDITINT_CHECK:                                                         j.parseMap,
		define.METRIC_YASDB_SECURITY_LOGIN_MAXIMUM_LOGIN_ATTEMPTS:                                  j.parseTable,
		define.METRIC_YASDB_SECURITY_USER_NO_OPEN:                                                  j.parseTable,
		define.METRIC_YASDB_SECURITY_USER_WITH_SYSTEM_TABLE_PRIVILEGES:                             j.parseTable,
		define.METRIC_YASDB_SECURITY_USER_WITH_DBA_ROLE:                                            j.parseTable,
		define.METRIC_YASDB_SECURITY_USER_ALL_PRIVILEGE_OR_SYSTEM_PRIVILEGES:                       j.parseTable,
		define.METRIC_YASDB_SECURITY_USER_USE_SYSTEM_TABLESPACE:                                    j.parseTable,
		define.METRIC_YASDB_SECURITY_AUDIT_CLEANUP_TASK:                                            j.parseTable,
		define.METRIC_YASDB_SECURITY_AUDIT_FILE_SIZE:                                               j.parseTable,
		define.METRIC_YASDB_UNDO_LOG_SIZE:                                                          j.parseTable,
		define.METRIC_YASDB_UNDO_LOG_TOTAL_BLOCK:                                                   j.parseTable,
		define.METRIC_YASDB_UNDO_LOG_RUNNING_TRANSACTIONS:                                          j.parseTable,
		define.METRIC_YASDB_RUN_LOG_DATABASE_CHANGES:                                               j.parseText,
		define.METRIC_YASDB_SLOW_LOG_PARAMETER:                                                     j.parseMap,
		define.METRIC_YASDB_SLOW_LOG:                                                               j.parseTable,
		define.METRIC_YASDB_SLOW_LOG_FILE:                                                          j.parseText,
		define.METRIC_YASDB_ALERT_LOG_ERROR:                                                        j.parseText,
		define.METRIC_HOST_DMESG_LOG_ERROR:                                                         j.parseText,
		define.METRIC_HOST_SYSTEM_LOG_ERROR:                                                        j.parseText,
		define.METRIC_YASDB_BACKUP_SET:                                                             j.parseTable,
		define.METRIC_YASDB_FULL_BACKUP_SET_COUNT:                                                  j.parseMap,
		define.METRIC_YASDB_BACKUP_SET_PATH:                                                        j.parseTable,
		define.METRIC_YASDB_SHARE_POOL:                                                             j.parseMap,
		define.METRIC_YASDB_VM_SWAP_RATE:                                                           j.parseMap,
		define.METRIC_YASDB_TOP_SQL_BY_CPU_TIME:                                                    j.parseTable,
		define.METRIC_YASDB_TOP_SQL_BY_BUFFER_GETS:                                                 j.parseTable,
		define.METRIC_YASDB_TOP_SQL_BY_DISK_READS:                                                  j.parseTable,
		define.METRIC_YASDB_TOP_SQL_BY_PARSE_CALLS:                                                 j.parseTable,
		define.METRIC_YASDB_HIGH_FREQUENCY_SQL:                                                     j.parseTable,
		define.METRIC_YASDB_HISTORY_DB_TIME:                                                        j.parseHostWorkload,
		define.METRIC_YASDB_HISTORY_BUFFER_HIT_RATE:                                                j.parseHostWorkload,
		define.METRIC_HOST_HUGE_PAGE:                                                               j.parseMap,
		define.METRIC_HOST_SWAP_MEMORY:                                                             j.parseMap,
		define.METRIC_YASDB_BUFFER_HIT_RATE:                                                        j.parseMap,
		define.METRIC_YASDB_TABLE_LOCK_WAIT:                                                        j.parseMap,
		define.METRIC_YASDB_ROW_LOCK_WAIT:                                                          j.parseMap,
		define.METRIC_YASDB_LONG_RUNNING_TRANSACTION:                                               j.parseTable,
		define.METRIC_YASDB_INVALID_OBJECT:                                                         j.parseTable,
		define.METRIC_YASDB_INVISIBLE_INDEX:                                                        j.parseTable,
		define.METRIC_YASDB_DISABLED_CONSTRAINT:                                                    j.parseTable,
		define.METRIC_YASDB_TABLE_WITH_TOO_MUCH_COLUMNS:                                            j.parseTable,
		define.METRIC_YASDB_TABLE_WITH_TOO_MUCH_INDEXES:                                            j.parseTable,
		define.METRIC_YASDB_PARTITIONED_TABLE_WITHOUT_PARTITIONED_INDEXES:                          j.parseTable,
		define.METRIC_YASDB_PARTITIONED_TABLE_WITH_NUMBER_OF_HASH_PARTITIONS_IS_NOT_A_POWER_OF_TWO: j.parseTable,
		define.METRIC_YASDB_TABLE_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS:     j.parseTable,
		define.METRIC_YASDB_COLUMN_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS:    j.parseTable,
		define.METRIC_YASDB_FOREIGN_KEYS_WITHOUT_INDEXES:                                           j.parseTable,
		define.METRIC_YASDB_FOREIGN_KEYS_WITH_IMPLICIT_DATA_TYPE_CONVERSION:                        j.parseTable,
		define.METRIC_YASDB_TABLE_WITH_ROW_SIZE_EXCEEDS_BLOCK_SIZE:                                 j.parseTable,
	}
	fn, ok := parseFuncMap[define.MetricName(metric.Name)]
	if !ok {
		return nil, fmt.Errorf("failed to find parse func of metric %s", metric.Name)
	}
	return fn, nil
}

func (j *JsonParser) getMetric(name string) (*confdef.YHCMetric, error) {
	for _, metric := range j.metrics {
		if metric.Name == name {
			return metric, nil
		}
	}
	return nil, fmt.Errorf("failed to found metric by %s, may be it does not check", name)
}

func (j *JsonParser) parseTable(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
	if item.Details == nil {
		return fmt.Errorf("failed to parse table of %s because the details is nil", item.Name)
	}
	attributes := define.TableAttributes{
		Title: metric.NameAlias,
	}
	switch detail := item.Details.(type) {
	case map[string]string:
		for _, key := range metric.HiddenColumns {
			delete(detail, key)
		}
		j.dealTableStringRow(&attributes, metric, detail)
	case map[string]interface{}:
		for _, key := range metric.HiddenColumns {
			delete(detail, key)
		}
		j.dealTableAnyRow(&attributes, metric, detail)
	case []map[string]string:
		for _, data := range detail {
			for _, key := range metric.HiddenColumns {
				delete(data, key)
			}
			j.dealTableStringRow(&attributes, metric, data)
		}
	case []map[string]interface{}:
		for _, data := range detail {
			for _, key := range metric.HiddenColumns {
				delete(data, key)
			}
			j.dealTableAnyRow(&attributes, metric, data)
		}
	default:
		return fmt.Errorf("failed to parse table, unsupport data type %T", item.Details)
	}
	attributes.TableColumns = j.sortTableColumns(metric, attributes.TableColumns)
	if _, ok := _fixedTableLayoutMetrics[item.Name]; ok {
		attributes.TableLayout = define.TABLE_LAYOUT_FIXED
	}
	element := &define.PandoraElement{
		MetricName:   metric.Name,
		ElementTitle: j.genElementTitle(metric, item),
		ElementType:  define.ET_TABLE,
		Attributes:   attributes,
	}
	menu.Elements = append(menu.Elements, element)
	return j.parseAlert(menu, item, metric)
}

func (j *JsonParser) sortTableColumns(metric *confdef.YHCMetric, columns []*define.TableColumn) []*define.TableColumn {
	columnMap := map[string]*define.TableColumn{}
	for _, column := range columns {
		columnMap[column.DataIndex] = column
	}
	var order, unorder []*define.TableColumn
	relatedMetric := j.getRelatedMetrics(metric)
	for _, metricName := range relatedMetric {
		m, err := j.getMetric(string(metricName))
		if err != nil {
			j.log.Error(err)
			continue
		}
		for _, o := range m.ColumnOrder {
			if column, ok := columnMap[o]; ok {
				order = append(order, column)
				delete(columnMap, o)
			}
		}
	}
	for _, column := range columnMap {
		unorder = append(unorder, column)
	}
	sort.Slice(unorder, func(i, j int) bool {
		return unorder[i].DataIndex < unorder[j].DataIndex
	})
	return append(order, unorder...)
}

func (j *JsonParser) dealTableStringRow(attributes *define.TableAttributes, metric *confdef.YHCMetric, data map[string]string) {
	if len(attributes.TableColumns) == 0 {
		columnsMap := make(map[string]*define.TableColumn)
		for key := range data {
			title := j.getColumnAlias(metric, key)
			column := &define.TableColumn{
				Title:     title,
				DataIndex: key,
			}
			columnsMap[key] = column
		}
		columns := []*define.TableColumn{}
		for _, column := range columnsMap {
			columns = append(columns, column)
		}
		attributes.TableColumns = columns
	}
	dataSource := make(map[string]interface{})
	for key, value := range data {
		dataSource[key] = value
	}
	attributes.DataSource = append(attributes.DataSource, dataSource)
}

func (j *JsonParser) dealTableAnyRow(attributes *define.TableAttributes, metric *confdef.YHCMetric, data map[string]interface{}) {
	if len(attributes.TableColumns) == 0 {
		columnsMap := make(map[string]*define.TableColumn)
		for key := range data {
			title := j.getColumnAlias(metric, key)
			column := &define.TableColumn{
				Title:     title,
				DataIndex: key,
			}
			columnsMap[key] = column
		}
		columns := []*define.TableColumn{}
		for _, column := range columnsMap {
			columns = append(columns, column)
		}
		attributes.TableColumns = columns
	}
	dataSource := make(map[string]interface{})
	for key, value := range data {
		dataSource[key] = value
	}
	attributes.DataSource = append(attributes.DataSource, dataSource)
}

func (j *JsonParser) parseCode(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
	if item.Details == nil {
		return fmt.Errorf("failed to parse code of %s because the details is nil", item.Name)
	}
	attributes := define.CodeAttributes{
		Title:    confdef.GetModuleAlias(metric.Name),
		Language: "shell",
	}

	switch detail := item.Details.(type) {
	case string:
		attributes.Code = detail
	default:
		return fmt.Errorf("failed to parse code, unsupport type %T", item.Details)
	}
	menu.Elements = append(menu.Elements, &define.PandoraElement{
		MetricName:   metric.Name,
		ElementTitle: j.genElementTitle(metric, item),
		ElementType:  define.ET_CODE,
		Attributes:   attributes,
	})
	return nil
}

func (j *JsonParser) genElementTitle(metric *confdef.YHCMetric, item *define.YHCItem) string {
	var suffix, prefix string
	if len(item.NodeID) != 0 {
		suffix = fmt.Sprintf("(%s)", item.NodeID)
	}
	prefix = metric.Name
	if metric.NameAlias != "" {
		prefix = metric.NameAlias
	}
	return prefix + suffix
}

func (j *JsonParser) parseMap(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
	if item.Details == nil {
		return fmt.Errorf("failed to parse map of %s because the details is nil", item.Name)
	}
	element := &define.PandoraElement{
		MetricName:   metric.Name,
		ElementTitle: j.genElementTitle(metric, item),
		ElementType:  define.ET_DESCRIPTION,
	}
	attributes := define.DescriptionAttributes{}
	switch detail := item.Details.(type) {
	case map[string]string:
		for _, key := range metric.HiddenColumns {
			delete(detail, key)
		}
		for key, value := range detail {
			attributes.Data = append(attributes.Data, &define.DescriptionData{Label: key, Value: value})
		}
	case map[string]interface{}:
		for _, key := range metric.HiddenColumns {
			delete(detail, key)
		}
		for key, value := range detail {
			attributes.Data = append(attributes.Data, &define.DescriptionData{Label: key, Value: value})
		}
	default:
		return fmt.Errorf("failed to parse map, unsupport data type %T", item.Details)
	}
	attributes.Data = j.sortMapData(metric, attributes.Data)
	element.Attributes = attributes
	menu.Elements = append(menu.Elements, element)
	return j.parseAlert(menu, item, metric)
}

func (j *JsonParser) sortMapData(metric *confdef.YHCMetric, datas []*define.DescriptionData) []*define.DescriptionData {
	dataMap := map[string]*define.DescriptionData{}
	for _, data := range datas {
		dataMap[data.Label] = data
	}
	var order, unorder []*define.DescriptionData
	relatedMetric := j.getRelatedMetrics(metric)
	for _, metricName := range relatedMetric {
		m, err := j.getMetric(string(metricName))
		if err != nil {
			j.log.Error(err)
			continue
		}
		for _, o := range m.ColumnOrder {
			if column, ok := dataMap[o]; ok {
				order = append(order, column)
				delete(dataMap, o)
			}
		}
	}
	for _, data := range dataMap {
		unorder = append(unorder, data)
	}
	sort.Slice(unorder, func(i, j int) bool {
		return unorder[i].Label < unorder[j].Label
	})
	order = append(order, unorder...)
	// replace with column alias
	for _, o := range order {
		o.Label = j.getColumnAlias(metric, o.Label)
	}
	return order
}

func (j *JsonParser) parseText(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
	if item.Details == nil {
		return fmt.Errorf("failed to parse text of %s because the details is nil", item.Name)
	}
	element := define.PandoraElement{
		MetricName:   metric.Name,
		ElementTitle: j.genElementTitle(metric, item),
		ElementType:  define.ET_PRE,
	}
	attributes := define.DescriptionAttributes{
		Title: metric.NameAlias,
	}
	switch detail := item.Details.(type) {
	case string:
		element.InnerText = detail
	case []string:
		element.InnerText = strings.Join(detail, stringutil.STR_NEWLINE)
	default:
		return fmt.Errorf("failed to parse code, unsupport type %T", item.Details)
	}
	element.Attributes = attributes
	menu.Elements = append(menu.Elements, &element)
	return nil
}

func (j *JsonParser) parseAlert(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
	if len(item.Alerts) == 0 {
		return nil
	}
	for _, alerts := range item.Alerts {
		for _, alert := range alerts {
			element := define.PandoraElement{
				MetricName:  metric.Name,
				ElementType: define.ET_ALERT,
				Attributes: define.AlertAttributes{
					Message:     alert.Description,
					AlertType:   define.AlertType(alert.Level),
					Description: j.genAlertDescription(metric, alert),
				},
			}
			menu.Elements = append(menu.Elements, &element)
		}
	}
	return nil
}

func (j *JsonParser) genAlertDescription(metric *confdef.YHCMetric, alert *define.YHCAlert) (desc string) {
	if len(alert.Labels) != 0 {
		labelArr := []string{}
		for k, v := range alert.Labels {
			labelAlias := j.getColumnAlias(metric, k)
			labelArr = append(labelArr, fmt.Sprintf("%s：%s", labelAlias, v))
		}
		desc = fmt.Sprintf("告警标签：{%s}\n", strings.Join(labelArr, "；"))
	}
	desc += fmt.Sprintf("检查结果：%v\n", alert.Value)
	desc += fmt.Sprintf("告警建议：%s\n", alert.Suggestion)
	desc += fmt.Sprintf("告警表达式：%s\n", alert.Expression)
	return
}

// 部分指标由于sql限制，分开采集，生成报告的时候需要合并到同一张表格中
func (j *JsonParser) mergeMetrics() {
	for to, from := range _mergeMetricMap {
		j.mergeMetric(to, from)
	}
}

func (j *JsonParser) mergeElements(report *define.PandoraReport) {
	log := log.Module.M("merge element")
	for _, merge := range _mergeOldMenuToNew {
		var parentMenu *define.PandoraMenu
		for _, menu := range report.ReportData {
			parentMenu = j.findMenu(menu, merge.parentModule)
			if parentMenu != nil {
				break
			}
		}
		if parentMenu == nil {
			log.Warningf("report unfound menu: %s", merge.parentModule)
			continue
		}
		childrenMenu := make([]*define.PandoraMenu, 0)
		mergeMenu := &define.PandoraMenu{Title: merge.targetTitle}
		oldChildrenMap := make(map[string]*define.PandoraMenu)
		minMenuIndex := math.MaxInt
		for _, origin := range merge.originMetrics {
			menu := j.findMenu(parentMenu, origin)
			if menu == nil {
				log.Warnf("from %s unfound submenu %s", merge.parentModule, origin)
				continue
			}
			if menu.MenuIndex < minMenuIndex {
				minMenuIndex = menu.MenuIndex
			}
			oldChildrenMap[origin] = menu
		}
		// 将准备合并的孩子元素添加到新菜单中
		for _, childMenu := range parentMenu.Children {
			if _, ok := oldChildrenMap[childMenu.TitleEn]; !ok {
				childrenMenu = append(childrenMenu, childMenu)
				continue
			}
			mergeMenu.Elements = append(mergeMenu.Elements, oldChildrenMap[childMenu.TitleEn].Elements...)
		}
		mergeMenu.MenuIndex = minMenuIndex
		childrenMenu = append(childrenMenu, mergeMenu)
		sort.Slice(childrenMenu, func(i, j int) bool {
			return childrenMenu[i].MenuIndex < childrenMenu[j].MenuIndex
		})
		parentMenu.Children = childrenMenu
	}
}

func (j *JsonParser) findMenu(menu *define.PandoraMenu, menuName string) *define.PandoraMenu {
	if menu == nil {
		return nil
	}
	if menu.TitleEn == menuName {
		return menu
	}
	for _, child := range menu.Children {
		res := j.findMenu(child, menuName)
		if res != nil {
			return res
		}
	}
	return nil
}

func (j *JsonParser) getColumnAlias(metric *confdef.YHCMetric, columnName string) string {
	relatedMetrics := j.getRelatedMetrics(metric)
	for _, metricName := range relatedMetrics {
		metric, err := j.getMetric(string(metricName))
		if err != nil {
			j.log.Errorf("failed to get metric by name %s", metricName)
			continue
		}
		alias, ok := metric.ColumnAlias[columnName]
		if ok {
			return alias
		}
	}
	return columnName
}

// 部分指标在展示的时候需要合并信息，此函数返回当前指标的关联指标
func (j *JsonParser) getRelatedMetrics(metric *confdef.YHCMetric) []define.MetricName {
	for metricName, related := range _mergeMetricMap {
		if metricName == define.MetricName(metric.Name) {
			return related
		}
	}
	return []define.MetricName{define.MetricName(metric.Name)}
}

func (j *JsonParser) mergeMetric(to define.MetricName, froms []define.MetricName) {
	var merge bool
	// 先根据节点ID进行分类
	type nodeRelation struct {
		to    *define.YHCItem
		froms []*define.YHCItem
	}
	nodeMap := make(map[string]*nodeRelation)
	toResult, ok := j.results[to]
	if !ok {
		// 删除所有的from
		for _, m := range froms {
			delete(j.results, m)
		}
		return
	}
	for _, result := range toResult {
		nodeMap[result.NodeID] = &nodeRelation{to: result, froms: make([]*define.YHCItem, 0)}
	}
	// 遍历froms
	for _, from := range froms {
		fromResults, ok := j.results[from]
		if !ok {
			continue
		}
		for _, fromResult := range fromResults {
			node, ok := nodeMap[fromResult.NodeID]
			if !ok {
				continue
			}
			node.froms = append(node.froms, fromResult)
		}
		delete(j.results, from)
		merge = true
	}
	resItems := []*define.YHCItem{}
	for _, node := range nodeMap {
		resAlerts := make(map[string][]*define.YHCAlert)
		resDetail := make(map[string]interface{})
		for _, from := range node.froms {
			switch detail := from.Details.(type) {
			case map[string]interface{}:
				for k, v := range detail {
					resDetail[k] = v
				}
			case map[string]string:
				for k, v := range detail {
					resDetail[k] = v
				}
			default:
				j.log.Errorf("failed to merge metrics, unsupport data type %T", detail)
			}
			for level, alerts := range from.Alerts {
				a, ok := resAlerts[level]
				if !ok {
					a = []*define.YHCAlert{}
				}
				a = append(a, alerts...)
				resAlerts[level] = a
			}
		}
		resItems = append(resItems, &define.YHCItem{
			Name:    node.to.Name,
			Details: resDetail,
			Alerts:  resAlerts,
		})
	}
	if merge {
		j.results[to] = resItems
	}
}

func (j *JsonParser) genCustomBashParseFunc(metric *confdef.YHCMetric) (MetricParseFunc, error) {
	fn := func(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
		if len(item.Error) != 0 {
			return fmt.Errorf("failed to gen parse func because the metric %s check failed", metric.Name)
		}
		if err := j.parseCode(menu, item, metric); err != nil {
			return err
		}
		if err := j.parseAlert(menu, item, metric); err != nil {
			return err
		}
		return nil
	}
	return fn, nil
}

func (j *JsonParser) genCustomSqlParseFunc(metric *confdef.YHCMetric) (MetricParseFunc, error) {
	fn := func(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
		if len(item.Error) != 0 {
			return fmt.Errorf("failed to gen parse func because the metric %s check failed", metric.Name)
		}
		if err := j.parseTable(menu, item, metric); err != nil {
			return err
		}
		if err := j.parseAlert(menu, item, metric); err != nil {
			return err
		}
		return nil
	}
	return fn, nil
}

func (j *JsonParser) parseHostOtherWorkload(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric, includeFields map[string]struct{}) error {
	if len(item.Error) != 0 {
		return fmt.Errorf("failed to gen parse func because the metric %s check failed, err: %v", metric.Name, item.Error)
	}
	data, ok := item.Details.(define.WorkloadOutput)
	if !ok {
		return fmt.Errorf("invalid data type %T", item.Details)
	}
	if len(data) == 0 {
		return nil
	}
	timeArray := []int64{}
	for time := range data {
		timeArray = append(timeArray, time)
	}
	sort.Slice(timeArray, func(i, j int) bool { return timeArray[i] < timeArray[j] })

	// create attributes map to store all attribute
	attributes := make(map[string]define.ChartAttributes)
	for _, value := range data[timeArray[0]] {
		m, err := j.convertObjectToMap(value)
		if err != nil {
			return err
		}
		for field := range m {
			if _, ok := includeFields[field]; !ok {
				continue
			}
			attribute := define.ChartAttributes{
				CustomOptions: define.ChartCustomOptions{
					ChartType: define.CT_LINE,
					Title: define.CustomOptionTitle{
						Text: j.getColumnAlias(metric, field),
					},
					Data: []*define.ChartData{},
				},
			}
			attributes[field] = attribute
		}
	}

	// fill chart data from origin data
	for _, t := range timeArray {
		timeStr := time.Unix(t, 0).Format(timedef.TIME_FORMAT)
		for name, obj := range data[t] {
			// parse data to map
			m, err := j.convertObjectToMap(obj)
			if err != nil {
				j.log.Errorf("failed to parse object %T, err: %v", obj, err)
				continue
			}
			for field, value := range m {
				if _, ok := includeFields[field]; !ok {
					continue
				}
				attribute := attributes[field]

				chartDataMap := make(map[string]*define.ChartData)
				for _, d := range attribute.CustomOptions.Data {
					chartDataMap[d.Name] = d
				}
				chartData, ok := chartDataMap[name]
				if !ok {
					chartData = &define.ChartData{Name: name}
				}
				chartData.Value = append(chartData.Value, &define.ChartCoordinate{X: timeStr, Y: value})
				chartDataMap[name] = chartData
				chartDatas := []*define.ChartData{}
				for _, d := range chartDataMap {
					chartDatas = append(chartDatas, d)
				}
				attribute.CustomOptions.Data = chartDatas
				attributes[field] = attribute
			}
		}
	}
	for _, attribute := range attributes {
		menu.Elements = append(menu.Elements, &define.PandoraElement{
			MetricName:  metric.Name,
			ElementType: define.ET_CHART,
			Attributes:  attribute,
		})
	}
	return nil
}

func (j *JsonParser) parseHostWorkload(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric) error {
	includeFields := map[string]struct{}{}
	for column := range metric.ColumnAlias {
		includeFields[column] = struct{}{}
	}
	switch item.Name {
	case define.METRIC_HOST_CURRENT_CPU_USAGE, define.METRIC_HOST_HISTORY_CPU_USAGE:
		return j.parseHostCPUUsage(menu, item, metric, includeFields)
	default:
		return j.parseHostOtherWorkload(menu, item, metric, includeFields)
	}
}

func (j *JsonParser) parseHostCPUUsage(menu *define.PandoraMenu, item *define.YHCItem, metric *confdef.YHCMetric, includeFields map[string]struct{}) error {
	if len(item.Error) != 0 {
		return fmt.Errorf("failed to gen parse func because the metric %s check failed. err: %v", metric.Name, item.Error)
	}
	data, ok := item.Details.(define.WorkloadOutput)
	if !ok {
		return fmt.Errorf("invalid data type %T", item.Details)
	}
	if len(data) == 0 {
		return nil
	}
	timeArray := []int64{}
	for time := range data {
		timeArray = append(timeArray, time)
	}
	sort.Slice(timeArray, func(i, j int) bool { return timeArray[i] < timeArray[j] })

	// create attributes map to store all attribute
	attributes := make(map[string]define.ChartAttributes)
	for name := range data[timeArray[0]] {
		attribute := define.ChartAttributes{
			CustomOptions: define.ChartCustomOptions{
				ChartType: define.CT_LINE,
				Title: define.CustomOptionTitle{
					Text: metric.NameAlias,
				},
				Data: []*define.ChartData{},
			},
		}
		attributes[name] = attribute
	}

	// fill chart data from origin data
	for _, t := range timeArray {
		timeStr := time.Unix(t, 0).Format(timedef.TIME_FORMAT)
		for name, value := range data[t] {
			// parse data to map
			m, err := j.convertObjectToMap(value)
			if err != nil {
				j.log.Errorf("failed to parse object %T, err: %v", value, err)
				continue
			}
			// use map to record data
			attribute := attributes[name]
			chartDataMap := make(map[string]*define.ChartData)
			for _, d := range attribute.CustomOptions.Data {
				chartDataMap[d.Name] = d
			}
			for lineName, lineValue := range m {
				if _, ok := includeFields[lineName]; !ok {
					continue
				}
				chartData, ok := chartDataMap[lineName]
				if !ok {
					chartData = &define.ChartData{
						Name: lineName,
					}
				}
				chartData.Value = append(chartData.Value, &define.ChartCoordinate{X: timeStr, Y: lineValue})
				chartDataMap[lineName] = chartData
			}
			chartDatas := []*define.ChartData{}
			for _, d := range chartDataMap {
				chartDatas = append(chartDatas, d)
			}
			attribute.CustomOptions.Data = chartDatas
			attributes[name] = attribute
		}
	}
	for _, attribute := range attributes {
		datas := attribute.CustomOptions.Data
		for _, data := range datas {
			data.Name = j.getColumnAlias(metric, data.Name)
		}
		menu.Elements = append(menu.Elements, &define.PandoraElement{
			ElementType: define.ET_CHART,
			Attributes:  attribute,
		})
	}
	return nil
}

func (j *JsonParser) convertObjectToMap(object interface{}) (map[string]interface{}, error) {
	m := map[string]interface{}{}
	bytes, err := json.Marshal(object)
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(bytes, &m); err != nil {
		return m, err
	}
	return m, nil
}
