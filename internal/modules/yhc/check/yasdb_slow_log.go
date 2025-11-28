package check

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"yhc/defs/confdef"
	"yhc/defs/timedef"
	"yhc/i18n"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/stringutil"

	"git.yasdb.com/go/yaserr"
	"git.yasdb.com/go/yaslog"
)

const (
	ENABLE_SLOW_LOG         = `'ENABLE_SLOW_LOG'`
	SLOW_LOG_OUTPUT         = `'SLOW_LOG_OUTPUT'`
	SLOW_LOG_FILE_PATH      = `'SLOW_LOG_FILE_PATH'`
	SLOW_LOG_TIME_THRESHOLD = `'SLOW_LOG_TIME_THRESHOLD'`
	SLOW_LOG_SQL_MAX_LEN    = `'SLOW_LOG_SQL_MAX_LEN'`
	TIME_PREFIX             = "# TIME: "
)

var _slowParameter = []string{
	ENABLE_SLOW_LOG,
	SLOW_LOG_OUTPUT,
	SLOW_LOG_FILE_PATH,
	SLOW_LOG_TIME_THRESHOLD,
	SLOW_LOG_SQL_MAX_LEN,
}

func (c *YHCChecker) GetYasdbSlowLogParameter(name string) (err error) {
	var datas []*define.YHCItem
	defer c.fillResults(datas...)

	logger := log.Module.M(string(define.METRIC_YASDB_SLOW_LOG_PARAMETER))
	for _, yasdb := range c.GetCheckNodes(logger) {
		data := &define.YHCItem{Name: define.METRIC_YASDB_SLOW_LOG_PARAMETER, NodeID: yasdb.NodeID}
		datas = append(datas, data)

		var parameters []map[string]string
		sql := fmt.Sprintf(define.SQL_QUERY_SLOW_LOG_PARAMETER, strings.Join(_slowParameter, stringutil.STR_COMMA))
		parameters, err = yasdb.QueryMultiRows(sql, confdef.GetYHCConf().SqlTimeout)
		if err != nil {
			err = yaserr.Wrap(err)
			logger.Error(err)
			data.Error = err.Error()
			continue
		}
		pmap := make(map[string]string)
		for _, p := range parameters {
			pmap[p[KEY_PARAMETER_NAME]] = p[KEY_PARAMETER_VALUE]
		}
		data.Details = pmap
	}
	return
}

func (c *YHCChecker) GetYasdbSlowLog(name string) (err error) {
	var datas []*define.YHCItem
	defer c.fillResults(datas...)

	logger := log.Module.M(string(define.METRIC_YASDB_SLOW_LOG))
	for _, yasdb := range c.GetCheckNodes(logger) {
		data := &define.YHCItem{Name: define.METRIC_YASDB_SLOW_LOG, NodeID: yasdb.NodeID}
		datas = append(datas, data)

		var slowSQLs []map[string]string
		sql := fmt.Sprintf(define.SQL_QUERY_SLOW_SQL, c.base.Start.Format(timedef.TIME_FORMAT), c.base.End.Format(timedef.TIME_FORMAT))
		slowSQLs, err = yasdb.QueryMultiRows(sql, confdef.GetYHCConf().SqlTimeout)
		if err != nil {
			err = yaserr.Wrap(err)
			logger.Error(err)
			data.Error = err.Error()
			continue
		}
		data.Details = slowSQLs
	}
	return
}

func (c *YHCChecker) GetYasdbSlowLogFile(name string) (err error) {
	data := &define.YHCItem{Name: define.METRIC_YASDB_SLOW_LOG_FILE}
	defer c.fillResults(data)

	logger := log.Module.M(string(define.METRIC_YASDB_SLOW_LOG_FILE))
	slowLogPath, err := getSlowLogPath(logger, c.base.DBInfo)
	if err != nil {
		err = yaserr.Wrapf(err, "query slow log path")
		logger.Error(err)
		data.Error = err.Error()
		return
	}
	lines, err := c.filterSlowLog(slowLogPath, logger)
	if err != nil {
		err = yaserr.Wrapf(err, "get slow log from file")
		logger.Error(err)
		data.Error = err.Error()
		return
	}
	if len(lines) == 0 {
		lines = append(lines, i18n.T("log.no_content"))
	}
	data.Details = lines
	return
}

func (c *YHCChecker) filterSlowLog(slowLog string, log yaslog.YasLog) ([]string, error) {
	slowLogFn, err := os.Open(slowLog)
	if err != nil {
		log.Errorf("open slow log err :%s", err.Error())
		return nil, err
	}
	scanner := bufio.NewScanner(slowLogFn)
	var lines []string
	var toBeCollected bool
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, TIME_PREFIX) {
			timeStr := strings.TrimPrefix(line, TIME_PREFIX)
			currentSqlTime, err := time.ParseInLocation(timedef.TIME_FORMAT, timeStr, time.Local)
			if err != nil {
				log.Errorf("parse time err: %s", err.Error())
				continue
			}
			toBeCollected = currentSqlTime.After(c.base.Start) && currentSqlTime.Before(c.base.End)
		}
		if !toBeCollected {
			continue
		}
		lines = append(lines, line)
	}
	return lines, nil
}
