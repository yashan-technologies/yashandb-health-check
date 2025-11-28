package checkhandler

import (
	"fmt"
	"time"

	"yhc/defs/bashdef"
	"yhc/defs/confdef"
	constdef "yhc/defs/constants"
	"yhc/defs/runtimedef"
	"yhc/i18n"
	yhccheck "yhc/internal/modules/yhc/check"
	"yhc/internal/modules/yhc/check/define"
	"yhc/internal/modules/yhc/check/reporter"
	"yhc/log"
	"yhc/utils/terminalutil/barutil"

	"git.yasdb.com/go/yasutil/fs"
)

type CheckHandler struct {
	checker  yhccheck.Checker
	metrics  map[string][]*confdef.YHCMetric
	base     *define.CheckerBase
	reporter *reporter.YHCReport
}

func NewCheckHandler(modules []*constdef.ModuleMetrics, base *define.CheckerBase) *CheckHandler {
	handler := &CheckHandler{
		metrics:  make(map[string][]*confdef.YHCMetric),
		base:     base,
		reporter: reporter.NewYHCReport(runtimedef.GetYHCHome(), base),
	}
	metrics := []*confdef.YHCMetric{}
	for _, module := range modules {
		if !module.Enabled {
			continue
		}
		if _, ok := handler.metrics[module.Name]; !ok {
			handler.metrics[module.Name] = make([]*confdef.YHCMetric, 0)
		}
		for _, metric := range module.Metrics {
			if !metric.Enabled {
				continue
			}
			metrics = append(metrics, metric)
			handler.metrics[module.Name] = append(handler.metrics[module.Name], metric)
		}
	}
	handler.checker = yhccheck.NewYHCChecker(base, metrics)
	return handler
}

func (c *CheckHandler) Check() error {
	if err := c.preCheck(); err != nil {
		return err
	}
	c.check()
	if err := c.afterCheck(); err != nil {
		return err
	}
	return nil
}

func (c *CheckHandler) preCheck() error {
	paths := []string{c.getOutputDir()}
	for _, p := range paths {
		if err := fs.Mkdir(p); err != nil {
			log.Handler.Errorf("mkdir: %s err: %s", p, err.Error())
			return err
		}
	}
	c.reporter.BeginTime = time.Now()
	yhccheck.CheckMutipleNodes = c.base.MultipleNodes
	return nil
}

func (c *CheckHandler) check() {
	moduleCheckFunc := c.moduleMetricsFunc()
	progress := c.newProgress(moduleCheckFunc)
	fmt.Print(i18n.T("check.starting"))
	progress.Start()
}

func (c *CheckHandler) afterCheck() error {
	c.reporter.EndTime = time.Now()
	fmt.Print(i18n.T("check.packing_results"))
	c.reporter.Items, c.reporter.Report, c.reporter.FailedItem = c.getResults(c.reporter.BeginTime, c.reporter.EndTime)
	path, err := c.reporter.GenResult()
	if err != nil {
		return err
	}
	fmt.Printf(i18n.T("check.result_saved"), bashdef.WithColor(path, bashdef.COLOR_BLUE))
	return nil
}

func (c *CheckHandler) moduleMetricsFunc() (moduleCheckFunc map[string]map[string]func(string) error) {
	moduleCheckFunc = make(map[string]map[string]func(string) error)
	for module, metrics := range c.metrics {
		funcMap := c.checker.CheckFuncs(metrics)
		if len(funcMap) == 0 {
			continue
		}
		moduleCheckFunc[module] = funcMap
	}
	return
}

func (c *CheckHandler) newProgress(moduleCheckFunc map[string]map[string]func(string) error) *barutil.Progress {
	progress := barutil.NewProgress(
		barutil.WithWidth(100),
	)
	for _, module := range define.Level1ModuleOrder {
		moduleStr := string(module)
		if _, ok := moduleCheckFunc[moduleStr]; !ok {
			continue
		}
		if len(moduleCheckFunc[moduleStr]) == 0 {
			log.Handler.Warnf("module %s no metric item executor available skip add bar", moduleStr)
			continue
		}
		progress.AddBar(confdef.GetModuleAlias(moduleStr), moduleCheckFunc[moduleStr])

	}
	return progress
}

func (c *CheckHandler) getResults(startCheck, endCheck time.Time) (map[define.MetricName][]*define.YHCItem, *define.PandoraReport, map[define.MetricName][]*define.YHCItem) {
	return c.checker.GetResult(startCheck, endCheck)
}

func (c *CheckHandler) getOutputDir() string {
	return c.base.Output
}
