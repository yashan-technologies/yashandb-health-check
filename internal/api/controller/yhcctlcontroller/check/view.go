package checkcontroller

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"yhc/commons/std"
	"yhc/commons/yasdb"
	"yhc/defs/confdef"
	constdef "yhc/defs/constants"
	"yhc/defs/errdef"
	"yhc/internal/modules/yhc/check"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/jsonutil"

	"git.yasdb.com/go/yasutil/tabler"
	"git.yasdb.com/pandora/tview"
	"github.com/gdamore/tcell/v2"
)

const (
	EXIT_CONTINUE = iota
	EXIT_NOT_CONTINUE
	EXIT_CONTROL_C
)

const (
	// yashan health check
	_header = "Yashan Health Check"

	// terminal view item chinese name
	_module               = "模块"
	_check_item           = "检查项"
	_database             = "数据库"
	_node                 = "节点"
	_disconnected_node    = "失联节点"
	_detail               = "详情"
	_health_check_summary = "健康检查概览"
	_node_info            = "节点信息"
	_wait_info            = "等待"
	_tips_header          = "以下检查项,不会进行检查,详细如下"
	_previous_button_name = "< 上一步"
	_next_button_name     = "下一步 >"
	_exit_button_name     = "退出 X"
	_no_alert_rule        = "该检查项未配置告警"

	// yashan health check page name
	_yasdb   = "yasdb"
	_nodes   = "nodes"
	_tips    = "tips"
	_summary = "summary"
	_wait    = "wait"
	// summary flex index
	_summary_metric_flex_index = 1
	_summary_table_flex_index  = 2

	_check_list_width          = 40
	_table_cell_max_width      = 50
	_alert_level_max_width     = 12
	_validate_dba_sql          = define.SQL_QUERY_TOTAL_OBJECT
	_base_yasdb_process_format = `.*yasdb (?i:(nomount|mount|open))`
)

var (
	// terminal view exit code
	globalExitCode int

	// Filled in after yasdb page validation
	moduleNoNeedCheckMetrics = map[string]map[string]*define.NoNeedCheckMetric{}

	// Filled in after yasdb page validation
	globalFilterModule = []*constdef.ModuleMetrics{}

	alertRuleOrder = []string{
		confdef.AL_CRITICAL,
		confdef.AL_WARNING,
		confdef.AL_INFO,
		confdef.AL_INVALID,
	}

	exitCodeMap = map[int]string{
		EXIT_CONTINUE:     "continue health check",
		EXIT_NOT_CONTINUE: "stop health check",
		EXIT_CONTROL_C:    "exit with control c",
	}

	tipsTableColumns = []string{"模块名称", "检查项名称", "原因"}

	alarmTableColumns = []string{"告警等级", "告警表达式", "描述", "建议"}
)

type PagePrimitive struct {
	Name      string          // page name
	Primitive tview.Primitive // page view
	Show      bool            // 是否展示
}

type databaseInfo struct {
	databaseName string
	Check        bool
}

func StartTerminalView(modules []*constdef.ModuleMetrics, yasdb *YashanDB, multipleNodes bool) {
	app := tview.NewApplication()
	app.SetInputCapture(captureCtrlCFunc(app))
	if err := app.SetRoot(index(app, yasdb, modules, multipleNodes), true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func captureCtrlCFunc(app *tview.Application) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyESC {
			exitFunc(app, EXIT_CONTROL_C)
		}
		return event
	}
}

func index(app *tview.Application, yasdb *YashanDB, modules []*constdef.ModuleMetrics, multipleNodes bool) *tview.Flex {
	f := newFlex(_header, true, tview.FlexRow)
	yasdbPage := newYasdbPage(yasdb.YashanDB)
	pages := newPages(yasdbPage)
	f.AddItem(pages, 0, 1, true)
	f.AddItem(indexFooter(app, pages, f, modules, multipleNodes), 3, 1, false)
	return f
}

func indexFooter(app *tview.Application, page *tview.Pages, index *tview.Flex, modules []*constdef.ModuleMetrics, multipleNodes bool) *tview.Flex {
	f := newFlex("", false, tview.FlexColumn)
	previous := newButton(_previous_button_name, true)
	previous.SetDisabled(true)
	next := newButton(_next_button_name, true)
	exit := newButton(_exit_button_name, true)
	exit.SetStyle(tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite))
	exit.SetActivatedStyle(tcell.StyleDefault.Background(tcell.ColorRed).Foreground(tcell.ColorWhite).Bold(true))

	previous.SetSelectedFunc(previousClickFunc(app, page, previous, index, modules, multipleNodes))
	next.SetSelectedFunc(nextClickFunc(app, page, previous, next, index, modules, multipleNodes))
	exit.SetSelectedFunc(func() { exitFunc(app, EXIT_NOT_CONTINUE) })

	f.AddItem(previous, 0, 1, false)
	f.AddItem(next, 0, 1, false)
	f.AddItem(exit, 0, 1, false)
	return f
}

func exitFunc(app *tview.Application, code int) {
	globalExitCode = code
	app.Stop()
}

// yashandb info form page
func dbInfoPage(yasdb *yasdb.YashanDB) *tview.Form {
	form := tview.NewForm()
	form.SetTitleAlign(tview.AlignCenter)
	form.AddInputField(constdef.YASDB_HOME, yasdb.YasdbHome, 100, nil, func(text string) { yasdb.YasdbHome = trimSpace(text) })
	form.AddInputField(constdef.YASDB_DATA, yasdb.YasdbData, 100, nil, func(text string) { yasdb.YasdbData = trimSpace(text) })
	form.AddInputField(constdef.YASDB_USER, yasdb.YasdbUser, 100, nil, func(text string) { yasdb.YasdbUser = trimSpace(text) })
	form.AddPasswordField(constdef.YASDB_PASSWORD, yasdb.YasdbPassword, 100, '*', func(text string) { yasdb.YasdbPassword = trimSpace(text) })
	if yasdb.CheckIsUdsOpen() {
		form.AddTextView("提示信息", "当前操作系统用户属于YASDBA用户组，无需输入数据库用户密码即可进行健康检查", 100, 0, false, true)
	}
	return form
}

func nodesPage() *tview.Flex {
	flex := newFlex("", true, tview.FlexRow)
	header := newTextView(_wait_info, true, tview.AlignCenter, tcell.ColorYellow)
	body := genNodesPageBody()
	flex.AddItem(header, 3, 1, false)
	flex.AddItem(body, 0, 1, false)
	return flex
}

func waitCheckNodePage(app *tview.Application, page *tview.Pages, index *tview.Flex) *tview.Modal {
	modal := tview.NewModal()
	modal.SetBackgroundColor(tcell.ColorRed)
	modal.SetText("正在检查备节点连接状态, 请稍后点击 -> 键")
	modal.AddButtons([]string{_previous, _next})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		switch buttonLabel {
		case _previous:
			app.SetRoot(index, true)
		case _next:
			if globalYasdb.GetCheckStatus() == STATUS_CHECKED {
				app.SetRoot(index, true)
				if page.HasPage(_nodes) {
					page.SwitchToPage(_nodes)
				} else {
					page.AddAndSwitchToPage(_nodes, nodesPage(), true)
				}
			}
		}
	})
	return modal
}

func genNodesPageBody() *tview.Flex {
	flex := tview.NewFlex()
	// 数据处理
	databases := []*databaseInfo{}
	nodes := map[string][]*yasdb.NodeInfo{}
	for _, node := range globalYasdb.Nodes {
		if !node.Connected {
			continue
		}
		nodes[node.DatabaseName] = append(nodes[node.DatabaseName], node)
	}
	for database := range nodes {
		databases = append(databases, &databaseInfo{
			databaseName: database,
			Check:        true,
		})
	}
	if len(databases) == 0 {
		return flex
	}
	// 集群名称排序
	sort.Slice(databases, func(i, j int) bool {
		return databases[i].databaseName < databases[j].databaseName
	})

	databaseList := newCheckedList(_database, true)
	nodeList := newCheckedList(_node, true)
	table := genDisconnectedNodeTable()

	for _, database := range databases {
		databaseList.AddItem(database.databaseName, "", 0, nil, database.Check)
	}
	currentNodes := nodes[databases[0].databaseName]
	sort.Slice(currentNodes, func(i, j int) bool {
		return currentNodes[i].Role < currentNodes[j].Role
	})
	for _, node := range currentNodes {
		nodeList.AddItem(fmt.Sprintf("%s(%s)", node.ListenAddr, node.Role), "", 0, nil, node.Check)
	}

	flex.AddItem(databaseList, _check_list_width, 1, false)
	flex.AddItem(nodeList, _check_list_width, 1, false)
	flex.AddItem(table, 0, 1, false)

	databaseList.SetChangedFunc(func(index int, mainText string, secondaryText string, shortcut rune) {
		databaseListChangedFunc(flex, databases, nodes, index)
	})
	databaseList.SetCheckedFunc(func(index int, checked bool) {
		databases[index].Check = checked
		for _, node := range nodes[databases[index].databaseName] {
			node.Check = checked
		}
		databaseListChangedFunc(flex, databases, nodes, index)
	})
	nodeList.SetCheckedFunc(func(index int, checked bool) { currentNodes[index].Check = checked })

	return flex
}

func databaseListChangedFunc(flex *tview.Flex, databases []*databaseInfo, nodes map[string][]*yasdb.NodeInfo, index int) {

	currentNodes := nodes[databases[index].databaseName]
	sort.Slice(currentNodes, func(i, j int) bool {
		return currentNodes[i].Role < currentNodes[j].Role
	})
	newNodeList := newCheckedList(_node, true)
	for _, node := range currentNodes {
		newNodeList.AddItem(fmt.Sprintf("%s(%s)", node.Role, node.ListenAddr), "", 0, nil, node.Check)
	}
	newNodeList.SetCheckedFunc(func(index int, checked bool) { currentNodes[index].Check = checked })

	databaseList := flex.GetItem(0)
	table := flex.GetItem(2)

	flex.Clear()
	flex.AddItem(databaseList, _check_list_width, 1, false)
	flex.AddItem(newNodeList, _check_list_width, 1, false)
	flex.AddItem(table, 0, 1, false)
}

func genDisconnectedNodeTable() *tview.Table {
	table := newTable(_disconnected_node, true, true)
	type failedNode struct {
		DatabaseName string
		ListenAddr   string
		User         string
		Password     string
	}
	var data []interface{}
	for _, node := range globalYasdb.Nodes {
		if !node.Connected {
			database := node.DatabaseName
			if len(database) == 0 {
				database = "unknow"
			}
			data = append(data, failedNode{
				DatabaseName: database,
				ListenAddr:   node.ListenAddr,
				User:         node.User,
				Password:     node.Password,
			})
		}
	}
	if len(data) == 0 {
		return table
	}
	header := []string{
		"数据库",
		"监听地址",
		"连接用户",
		"连接密码",
	}
	fillTableCell(table, header, data)
	return table
}

// yashan health check metric summary page
func summaryFlexPage(modules []*constdef.ModuleMetrics) *tview.Flex {
	flex := newFlex("", true, tview.FlexRow)
	header := newTextView(_health_check_summary, true, tview.AlignCenter, tcell.ColorYellow)
	body := summaryBody(modules)
	flex.AddItem(header, 3, 1, false)
	flex.AddItem(body, 0, 1, false)
	return flex
}

// body of summary page
func summaryBody(modules []*constdef.ModuleMetrics) *tview.Flex {
	flex := tview.NewFlex()
	fillSummaryBody(modules, flex)
	return flex
}

func fillSummaryBody(modules []*constdef.ModuleMetrics, flex *tview.Flex) {
	moduleList := newCheckedList(_module, true)
	itemList := newCheckedList(_check_item, true)
	table := summaryTable(modules)
	flex.AddItem(moduleList, _check_list_width, 1, false)
	flex.AddItem(itemList, _check_list_width, 1, false)
	flex.AddItem(table, 0, 1, false)
	moduleList.SetCheckedFunc(func(index int, checked bool) { modules[index].Enabled = checked })
	moduleList.SetChangedFunc(func(i int, s1, s2 string, r rune) { moduleListChangedFunc(modules[i], flex) })
	addModuleList(moduleList, modules)
	if len(modules) == 0 {
		return
	}
	addItemList(itemList, modules[0].Name, modules[0].Metrics)
	if len(modules[0].Metrics) == 0 {
		return
	}
	drawAlertRuleTable(table, modules[0].Metrics[0].AlertRules)
}

func addItemList(itemList *tview.CheckList, moduleName string, metrics []*confdef.YHCMetric) {
	for _, item := range metrics {
		itemList.AddItem(item.NameAlias, "", 0, nil, item.Enabled)
	}
}

func addModuleList(moduleList *tview.CheckList, modules []*constdef.ModuleMetrics) {
	for _, module := range modules {
		alias := confdef.GetModuleAlias(module.Name)
		moduleList.AddItem(alias, "", 0, nil, module.Enabled)
	}
}

func summaryTable(modules []*constdef.ModuleMetrics) *tview.Table {
	if len((modules)) == 0 {
		return nil
	}
	if len(modules[0].Metrics) == 0 {
		return nil
	}
	table := newTable(_detail, true, true)
	drawAlertRuleTable(table, modules[0].Metrics[0].AlertRules)
	return table
}

func moduleListChangedFunc(module *constdef.ModuleMetrics, flex *tview.Flex) {
	itemList := flex.GetItem(_summary_metric_flex_index)
	table := flex.GetItem(_summary_table_flex_index)
	newItemList := newCheckedList(_check_item, true)
	newTable := newTable(_detail, true, true)
	newItemList.SetCheckedFunc(func(index int, checked bool) { module.Metrics[index].Enabled = checked })
	newItemList.SetChangedFunc(func(i int, m, s string, sc rune) { itemListChangedFunc(module.Metrics[i], flex) })
	flex.RemoveItem(table)
	flex.RemoveItem(itemList)
	flex.AddItem(newItemList, _check_list_width, 1, false)
	flex.AddItem(table, 0, 1, false)
	addItemList(newItemList, module.Name, module.Metrics)
	if len(module.Metrics) == 0 {
		return
	}
	drawAlertRuleTable(newTable, module.Metrics[0].AlertRules)
}

func itemListChangedFunc(item *confdef.YHCMetric, flex *tview.Flex) {
	table := flex.GetItem(_summary_table_flex_index)
	newTable := newTable(_detail, true, true)
	drawAlertRuleTable(newTable, item.AlertRules)
	flex.RemoveItem(table)
	flex.AddItem(newTable, 0, 1, false)
}

func previousClickFunc(app *tview.Application, page *tview.Pages, button *tview.Button, index *tview.Flex, modules []*constdef.ModuleMetrics, multipleNodes bool) func() {
	return func() {
		pageName, _ := page.GetFrontPage()
		log.Controller.Infof("current page: %s", pageName)
		switch pageName {
		case _summary:
			page.SwitchToPage(_tips)
			log.Controller.Infof("switch to previous page: %s", pageName)
		case _tips:
			page.SwitchToPage(_nodes)
			log.Controller.Infof("switch to previous page: %s", pageName)
		case _nodes:
			page.SwitchToPage(_yasdb)
			log.Controller.Infof("switch to previous page: %s", pageName)
		case _yasdb:
			button.SetDisabled(true)
		}
	}
}

func nextClickFunc(app *tview.Application, page *tview.Pages, previous, next *tview.Button, index *tview.Flex, modules []*constdef.ModuleMetrics, multipleNodes bool) func() {
	return func() {
		pageName, pageView := page.GetFrontPage()
		log.Controller.Infof("click next current page: %s", pageName)
		switch pageName {
		case _yasdb:
			previous.SetDisabled(false)
			// validate yasdb
			form, ok := pageView.(*tview.Form)
			if ok {
				yasdbEnv, err := yasdbValidate(form)
				if err != nil {
					modal := newModal(app, index, err)
					app.SetRoot(modal, true)
					return
				}
				previous.SetDisabled(false)
				hasChanged := inputChanged(yasdbEnv)
				if hasChanged {
					globalYasdb = &YashanDB{
						YashanDB: yasdbEnv,
					}
				}
				validateMetrics(yasdbEnv, modules)
				if len(moduleNoNeedCheckMetrics) != 0 {
					// write no need check metrics to console.log
					std.WriteToFile("the following metric will not be checked \n")
					noNeedStr := genNoNeedCheckMetricsStr()
					std.WriteToFile(noNeedStr)
					if multipleNodes {
						if globalYasdb.GetCheckStatus() == STATUS_NOT_CHECK {
							// 使用协程后台检查备机连接情况
							go fillNodeInfos(globalYasdb)
						}
						if globalYasdb.GetCheckStatus() != STATUS_CHECKED {
							time.Sleep(200 * time.Millisecond) // 等待0.2s，如果在这段时间内检查完毕，则无需弹窗提示
						}
						// 重新获取检查状态，如果检查完毕则进入下一页面，否则弹窗提示
						switch globalYasdb.GetCheckStatus() {
						case STATUS_CHECKED:
							if page.HasPage(_nodes) {
								page.SwitchToPage(_nodes)
							} else {
								page.AddAndSwitchToPage(_nodes, nodesPage(), true)
							}
							return
						case STATUS_CHECKING, STATUS_NOT_CHECK:
							modal := waitCheckNodePage(app, page, index)
							app.SetRoot(modal, true)
							return
						}
					} else {
						if page.HasPage(_tips) {
							page.SwitchToPage(_tips)
							return
						}
						page.AddAndSwitchToPage(_tips, tipsPage(), true)
					}
					return
				}
			}
		case _nodes:
			err := validateCheckedNodes()
			if err != nil {
				modal := newModal(app, index, err)
				app.SetRoot(modal, true)
				return
			}
			if page.HasPage(_tips) {
				page.SwitchToPage(_tips)
			} else {
				page.AddAndSwitchToPage(_tips, tipsPage(), true)
			}
		case _tips:
			if page.HasPage(_summary) {
				page.SwitchToPage(_summary)
			} else {
				page.AddAndSwitchToPage(_summary, summaryFlexPage(globalFilterModule), true)
			}
		case _summary:
			exitFunc(app, EXIT_CONTINUE)
			return
		}
	}
}

func tipsPage() *tview.Flex {
	f := tview.NewFlex()
	f.SetDirection(tview.FlexRow)
	header := newTextView(_tips_header, true, tview.AlignCenter, tcell.ColorYellow)
	table := newTable(_detail, true, true)
	type moduleMetric struct {
		ModuleName  string
		MetricName  string
		Description string
	}
	tips := make([]interface{}, 0)
	for _, moduleName := range define.Level1ModuleOrder {
		moduleStr := string(moduleName)
		if _, ok := moduleNoNeedCheckMetrics[moduleStr]; !ok {
			continue
		}
		moduleAlias := confdef.GetModuleAlias(moduleStr)
		for _, notCheck := range moduleNoNeedCheckMetrics[moduleStr] {
			tips = append(tips, &moduleMetric{
				ModuleName:  moduleAlias,
				MetricName:  notCheck.Name,
				Description: notCheck.Description,
			})
		}
	}
	fillTableCell(table, tipsTableColumns, tips)
	f.AddItem(header, 3, 1, false)
	f.AddItem(table, 0, 1, false)
	return f
}

func drawAlertRuleTable(table *tview.Table, alertRules map[string][]confdef.AlertDetails) {
	if len(alertRules) == 0 {
		cell := tview.NewTableCell(_no_alert_rule)
		cell.SetTextColor(tcell.ColorYellow)
		table.SetCell(0, 0, cell)
		return
	}
	type rule struct {
		Level       string
		Expression  string
		Description string
		Suggestion  string
	}
	rules := make([]interface{}, 0)
	for _, level := range alertRuleOrder {
		if _, ok := alertRules[level]; !ok {
			continue
		}
		for _, alert := range alertRules[level] {
			rules = append(rules, rule{
				Level:       level,
				Expression:  alert.Expression,
				Description: alert.Description,
				Suggestion:  alert.Suggestion,
			})
		}
	}
	fillTableCell(table, alarmTableColumns, rules)
}

func fillTableCell(table *tview.Table, columns []string, data []interface{}) {
	cols := len(columns)
	rows := len(data) + 1
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			color, text := tcell.ColorYellow, columns[c]
			maxWidth := _table_cell_max_width
			if r > 0 {
				color = tcell.ColorWhite
				value := reflect.ValueOf(data[r-1])
				if value.Kind() == reflect.Ptr {
					value = value.Elem()
				}
				fieldValue := value.Field(c)
				text = fmt.Sprintf("%v", fieldValue.Interface())
			}
			if c == 0 {
				maxWidth = _alert_level_max_width
			}
			cell := tview.NewTableCell("")
			cell.SetMaxWidth(maxWidth)
			cell.SetAlign(tview.AlignLeft)
			cell.SetText(text)
			cell.SetTextColor(color)
			table.SetCell(r, c, cell)
		}
	}
}

func yasdbValidate(form *tview.Form) (*yasdb.YashanDB, error) {
	res, err := getFormDataByLabels(form, constdef.YASDB_HOME, constdef.YASDB_DATA, constdef.YASDB_USER, constdef.YASDB_PASSWORD)
	if err != nil {
		log.Controller.Errorf("get form data err: %s", err.Error())
		return nil, err
	}
	yasdb := &yasdb.YashanDB{
		YasdbHome: res[constdef.YASDB_HOME],
		YasdbData: res[constdef.YASDB_DATA],
		YasdbUser: res[constdef.YASDB_USER],
	}
	std.WriteToFile("get yasdb info : \n")
	std.WriteToFile(jsonutil.ToJSONString(yasdb) + "\n")
	// after log fill password
	yasdb.YasdbPassword = res[constdef.YASDB_PASSWORD]
	if err := yasdb.ValidUserAndPwd(log.Controller); err != nil {
		return nil, err
	}
	if err := fillListenAddrAndDBName(yasdb); err != nil {
		log.Controller.Errorf("fill listen addr err: %s", err.Error())
		return nil, err
	}
	return yasdb, nil

}

func inputChanged(yasdb *yasdb.YashanDB) bool {
	if globalYasdb.DatabaseName == "" || globalYasdb.ListenAddr == "" {
		return true
	}
	if yasdb != nil && globalYasdb != nil {
		return !(yasdb.YasdbUser == globalYasdb.YasdbUser && yasdb.YasdbPassword == globalYasdb.YasdbPassword && yasdb.YasdbHome == globalYasdb.YasdbHome && yasdb.YasdbData == globalYasdb.YasdbData)
	}
	return true
}

func validateMetrics(yasdb *yasdb.YashanDB, modules []*constdef.ModuleMetrics) {
	log := log.Controller.M("metric validate")
	for _, module := range modules {
		for _, metric := range module.Metrics {
			metricDefine := define.MetricName(metric.Name)
			if _, ok := check.NeedCheckMetricMap[metricDefine]; !ok {
				continue
			}
			if _, ok := check.NeedCheckMetricFuncMap[metricDefine]; !ok {
				log.Warnf("metric %s is defined in NeedCheckMetricMap, but NeedCheckMetricFuncMap is not defined", metric.Name)
				continue
			}
			if noNeedCheck := check.NeedCheckMetricFuncMap[metricDefine](log, yasdb, metric); noNeedCheck != nil {
				if _, ok := moduleNoNeedCheckMetrics[module.Name]; !ok {
					moduleNoNeedCheckMetrics[module.Name] = make(map[string]*define.NoNeedCheckMetric)
				}
				moduleNoNeedCheckMetrics[module.Name][metric.Name] = noNeedCheck
			}
		}
	}
	globalFilterModule = filterNeedCheckMetric(modules)
}

func validateCheckedNodes() error {
	num := 0
	for _, node := range globalYasdb.Nodes {
		if node.Connected && node.Check {
			num++
		}
	}
	if num <= 0 {
		return errors.New("please choose nodes you want to check")
	}
	return nil
}

func getFormData(form *tview.Form, label string) (string, error) {
	item := form.GetFormItemByLabel(label)
	if item == nil {
		return "", errdef.NewFormItemUnFound(label)
	}
	return item.(*tview.InputField).GetText(), nil
}

func getFormDataByLabels(form *tview.Form, labels ...string) (res map[string]string, err error) {
	res = make(map[string]string)
	for _, label := range labels {
		value, valueErr := getFormData(form, label)
		if valueErr != nil {
			err = valueErr
			return
		}
		res[label] = trimSpace(value)
	}
	return
}

func filterNeedCheckMetric(modules []*constdef.ModuleMetrics) (result []*constdef.ModuleMetrics) {
	result = make([]*constdef.ModuleMetrics, 0)
	for _, module := range modules {
		if _, ok := moduleNoNeedCheckMetrics[module.Name]; !ok {
			result = append(result, module)
			continue
		}
		metrics := make([]*confdef.YHCMetric, 0)
		for _, metric := range module.Metrics {
			if _, ok := moduleNoNeedCheckMetrics[module.Name][metric.Name]; ok {
				continue
			}
			metrics = append(metrics, metric)
			module.Metrics = metrics
		}
		if len(metrics) != 0 {
			result = append(result, &constdef.ModuleMetrics{
				Name:    module.Name,
				Metrics: metrics,
				Enabled: module.Enabled,
			})
		}
	}
	return result
}

func genNoNeedCheckMetricsStr() string {
	t := tabler.NewTable("",
		tabler.NewRowTitle("module", 25),
		tabler.NewRowTitle("metric", 25),
		tabler.NewRowTitle("description", 10),
		tabler.NewRowTitle("error", 10),
	)
	for _, module := range define.Level1ModuleOrder {
		moduleStr := string(module)
		if _, ok := moduleNoNeedCheckMetrics[moduleStr]; !ok {
			continue
		}
		moduleAlias := confdef.GetModuleAlias(moduleStr)
		for _, metric := range moduleNoNeedCheckMetrics[moduleStr] {
			if err := t.AddColumn(moduleAlias, metric.Name, metric.Description, metric.Error.Error()); err != nil {
				log.Controller.Errorf("add columns err: %s", err.Error())
				continue
			}
		}
	}
	return t.String()
}
