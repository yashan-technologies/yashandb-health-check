package check

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"yhc/commons/constants"
	"yhc/defs/bashdef"
	"yhc/defs/regexpdef"
	"yhc/defs/timedef"
	"yhc/i18n"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/execerutil"
	"yhc/utils/fileutil"
	"yhc/utils/stringutil"
	"yhc/utils/timeutil"

	"git.yasdb.com/go/yaserr"
	"git.yasdb.com/go/yaslog"
	"github.com/google/uuid"
	"github.com/shirou/gopsutil/host"
)

const (
	PARAMETER_RUN_LOG_FILE_PATH = "RUN_LOG_FILE_PATH"

	KEY_YASDB_RUN_LOG     = "run"
	KEY_YASDB_RUN_LOG_ERR = "errno"
	KEY_YASDB_ALERT_LOG   = "alert"
	KEY_YASDB_LOG         = "log"
	KEY_YASDB_CONFIG      = "config"

	NAME_YASDB_RUN_LOG   = "run.log"
	NAME_YASDB_ALERT_LOG = "alert.log"
	NAME_YASDB_SLOW_LOG  = "slow.log"

	ALERT_LOG_RISE_ACTION  = "0"
	ALERT_LOG_ACTION_INDEX = 4

	LOG_ROTATE_CONFIG = "/etc/logrotate.conf"

	SYSTEM_LOG_MESSAGES = "/var/log/messages"
	SYSTEM_LOG_SYSLOG   = "/var/log/syslog"
)

func (c *YHCChecker) GetYasdbRunLogError(name string) (err error) {
	data := &define.YHCItem{
		Name: define.METRIC_YASDB_RUN_LOG_ERROR,
	}
	defer c.fillResults(data)

	log := log.Module.M(string(define.METRIC_YASDB_RUN_LOG_ERROR))
	var res []string
	runLogPath, err := c.getRunLogPath(log)
	if err != nil {
		log.Errorf("failed to get run log path, err: %v", err)
		data.Error = err.Error()
		return
	}
	runLogFiles, err := c.getLogFiles(log, runLogPath, KEY_YASDB_RUN_LOG)
	if err != nil {
		log.Error(err)
		data.Error = err.Error()
		return
	}
	if res, err = c.getYasdbRunLogError(log, runLogFiles); err != nil {
		log.Error(err)
		data.Error = err.Error()
		return
	}
	if len(res) == 0 {
		res = append(res, i18n.T("log.no_obvious_error"))
	}
	data.Details = res
	return
}

func (c *YHCChecker) getRunLogPath(log yaslog.YasLog) (path string, err error) {
	path, err = c.querySingleParameter(log, PARAMETER_RUN_LOG_FILE_PATH)
	if err != nil {
		return
	}
	return strings.ReplaceAll(path, stringutil.STR_QUESTION_MARK, c.base.DBInfo.YasdbData), nil
}

func (c *YHCChecker) getYasdbRunLogError(log yaslog.YasLog, srcs []string) (res []string, err error) {
	logPredicateFunc := func(line string) bool {
		return strings.Contains(line, KEY_YASDB_RUN_LOG_ERR)
	}
	for _, f := range srcs {
		logEndTime := time.Now()
		if path.Base(f) != NAME_YASDB_RUN_LOG {
			fileds := strings.Split(strings.TrimSuffix(path.Base(f), ".log"), stringutil.STR_HYPHEN)
			// run.log归档的文件名是run-yyyymmdd.log, 用'-'切分后第二个字符串是最后一行日志的日期
			if len(fileds) < 2 {
				log.Errorf("failed to get log end time from %s, skip", f)
				continue
			}
			if logEndTime, err = time.ParseInLocation(timedef.TIME_FORMAT_IN_FILE, fileds[1], time.Local); err != nil {
				log.Errorf("failed to parse log end time from %s", fileds[1])
				continue
			}
		}
		if logEndTime.Before(c.base.Start) {
			// no need to write into dest
			log.Debugf("skip run log file: %s", f)
			continue
		}
		if res, err = c.collectLog(log, f, time.Now(), logPredicateFunc, c.yasdbLogTimeParse); err != nil {
			return
		}
	}
	return
}

func (c *YHCChecker) GetRisingAlertLog(name string) (err error) {
	data := &define.YHCItem{
		Name: define.METRIC_YASDB_ALERT_LOG_ERROR,
	}
	defer c.fillResults(data)
	alertLogPredicateFunc := func(line string) bool {
		fields := strings.Split(line, stringutil.STR_BAR)
		// Action 在第五列
		if len(fields) < ALERT_LOG_ACTION_INDEX+1 {
			return false
		}
		return strings.TrimSpace(fields[ALERT_LOG_ACTION_INDEX]) == ALERT_LOG_RISE_ACTION
	}
	alertLog := path.Join(c.base.DBInfo.YasdbData, KEY_YASDB_LOG, KEY_YASDB_ALERT_LOG, NAME_YASDB_ALERT_LOG)
	log := log.Module.M("get-alert-log")
	res, err := c.collectLog(log, alertLog, time.Now(), alertLogPredicateFunc, c.yasdbLogTimeParse)
	if err != nil {
		log.Errorf("get alert log detail err: %s", err.Error())
		data.Error = err.Error()
		return err
	}
	if len(res) == 0 {
		res = append(res, i18n.T("log.no_obvious_error"))
	}
	data.Details = res
	return
}

func (c *YHCChecker) yasdbLogTimeParse(date time.Time, line string) (t time.Time, err error) {
	match := regexpdef.YasdbLogTimeRegex.FindStringSubmatch(line)
	if len(match) < 2 {
		err = fmt.Errorf("line %s no match time, skip", line)
		return
	}
	return time.ParseInLocation(timedef.TIME_FORMAT_WITH_MICROSECOND, match[1], time.Local)
}

func (c *YHCChecker) GetDmesgLog(name string) (err error) {
	data := &define.YHCItem{
		Name: define.METRIC_HOST_DMESG_LOG_ERROR,
	}
	defer c.fillResults(data)
	log := log.Module.M("get-dmesg-log")
	exec := execerutil.NewExecer(log)
	tmpFileName := path.Join("/tmp", uuid.NewString()[0:6]+".log")
	ret, _, stderr := exec.Exec(bashdef.CMD_BASH, "-c", fmt.Sprintf("%s > %s", bashdef.CMD_DMESG, tmpFileName))
	if ret != 0 {
		err := fmt.Errorf("exec dmesg err: %s", stderr)
		log.Error(err)
		data.Error = stderr
		return err
	}
	defer os.Remove(tmpFileName)
	dmesgTimeParse := c.genDmesgLogTimeParseFunc()
	dmesgPredicate := c.genDmesgLogPredicateFunc()
	res, err := c.collectLog(log, tmpFileName, time.Now(), dmesgPredicate, dmesgTimeParse)
	if err != nil {
		log.Errorf("get alert log detail err: %s", err.Error())
		data.Error = err.Error()
		return err
	}
	if len(res) == 0 {
		res = append(res, i18n.T("log.no_obvious_error"))
	}
	data.Details = res
	return nil
}

func (c *YHCChecker) genDmesgLogTimeParseFunc() logTimeParseFunc {
	info, err := host.Info()
	if err != nil {
		return func(date time.Time, line string) (time.Time, error) {
			return time.Time{}, err
		}
	}
	return func(date time.Time, line string) (t time.Time, err error) {
		matches := regexpdef.DmesgTimeRegex.FindStringSubmatch(line)
		if len(matches) < 2 {
			err = fmt.Errorf("dmesg log: %s format err, skip", line)
			return
		}
		secondFromBoot, err := strconv.ParseFloat(matches[1], constants.BIT_SIZE_64)
		if err != nil {
			err = fmt.Errorf("dmesg log time: %s format err: %s, skip", matches[1], err.Error())
			return
		}
		t = time.Unix(int64(info.BootTime+uint64(secondFromBoot)), 0)
		return
	}
}

func (c *YHCChecker) genDmesgLogPredicateFunc() logPredicate {
	dmesgErrKeys := []string{
		"error",
		"warning",
		"warn",
		"failed",
		"invalid",
		"fault",
		"faulty",
		"timeout",
		"unable",
		"cannot",
		"corrupt",
		"corruption",
	}
	return func(line string) bool {
		for _, key := range dmesgErrKeys {
			if strings.Contains(line, key) {
				return true
			}
		}
		return false
	}
}

func (c *YHCChecker) GetSystemLog(name string) (err error) {
	data := &define.YHCItem{
		Name: define.METRIC_HOST_SYSTEM_LOG_ERROR,
	}
	defer c.fillResults(data)
	log := log.Module.M("get-system-log")
	logName, err := getSystemLogName()
	if err != nil {
		log.Error(err)
		data.Error = err.Error()
		return
	}
	prefix := path.Base(logName)
	res, err := c.collectHostLog(log, logName, prefix)
	if err != nil {
		log.Error(err)
		data.Error = err.Error()
		return
	}
	if len(res) == 0 {
		res = append(res, i18n.T("log.no_obvious_error"))
	}
	data.Details = res
	return
}

func getSystemLogName() (string, error) {
	_, err := os.Stat(SYSTEM_LOG_MESSAGES)
	if err == nil || !os.IsNotExist(err) {
		return SYSTEM_LOG_MESSAGES, nil
	}
	_, err = os.Stat(SYSTEM_LOG_SYSLOG)
	if err == nil || !os.IsNotExist(err) {
		return SYSTEM_LOG_SYSLOG, nil
	}
	return "", fmt.Errorf("both of %s and %s not exist", SYSTEM_LOG_MESSAGES, SYSTEM_LOG_SYSLOG)
}

func (c *YHCChecker) collectHostLog(log yaslog.YasLog, src, prefix string) (res []string, err error) {
	hasSetDateext, err := c.hasSetDateext()
	if err != nil {
		return
	}
	if hasSetDateext {
		return c.collectHostLogWithSetDateext(log, src, prefix)
	}
	return c.collectHostLogWithoutSetDateext(log, src)
}

func (c *YHCChecker) collectHostLogWithoutSetDateext(log yaslog.YasLog, src string) (res []string, err error) {
	// get log file last modify time
	srcInfo, err := os.Stat(src)
	if err != nil {
		return
	}
	srcModTime := srcInfo.ModTime()
	if srcModTime.Before(c.base.Start) {
		log.Infof("log %s last modify time is %s, skip", src, srcModTime)
		return
	}
	return c.reverseCollectLog(src, srcModTime, c.hostLogTimeParse)
}

func (c *YHCChecker) collectHostLogWithSetDateext(log yaslog.YasLog, src, prefix string) (res []string, err error) {
	var srcs []string
	srcs, err = c.getLogFiles(log, path.Dir(src), prefix)
	if err != nil {
		return
	}
	var logFiles []string // resort logFile so that the current log file is the last one, other file sorted by time is in the first
	for _, v := range srcs {
		if v == src {
			continue
		}
		logFiles = append(logFiles, v)
	}
	if len(srcs) != len(logFiles) {
		logFiles = append(logFiles, src)
	}
	for _, logFile := range logFiles {
		log.Debugf("try to collect %s", logFile)
		date := time.Now()
		if logFile != src {
			fileds := strings.Split(path.Base(logFile), stringutil.STR_HYPHEN)
			if len(fileds) < 2 {
				log.Errorf("failed to get log end time from %s, skip", logFile)
				continue
			}
			// get date from log file name
			date, err = time.ParseInLocation(timedef.TIME_FORMAT_DATE_IN_FILE, fileds[1], time.Local)
			if err != nil {
				log.Error("failed to get date from: %s, err: %s", logFile, err.Error())
				continue
			}
			// try to get log end time from last 3 line in log
			k := 3
			lastKLines, err := fileutil.Tail(logFile, k)
			if err != nil {
				log.Errorf("failed to read file %s last %d line, err: %s", logFile, k, err.Error())
			} else {
				for i := 0; i < len(lastKLines); i++ {
					if stringutil.IsEmpty(lastKLines[i]) {
						continue
					}
					var tmpData time.Time
					tmpData, err = c.hostLogTimeParse(date, stringutil.RemoveExtraSpaces(strings.TrimSpace(lastKLines[i])))
					if err != nil {
						log.Errorf("failed to parse time from line: %s, err: %s", lastKLines[i], err.Error())
						continue
					}
					date = tmpData
				}
			}
			log.Debugf("log file %s end date is %s", logFile, date)
			if date.Before(c.base.Start) {
				log.Infof("skip to collect log %s, log file end date: %s , collect start date %s", logFile, date.AddDate(0, 0, -1), c.base.Start)
				continue
			}
		}
		res, err = c.collectLog(log, logFile, date, c.systemLogPredicate, c.hostLogTimeParse)
		if err != nil {
			log.Errorf("failed to collect from: %s, err: %s", logFile, err.Error())
			continue
		}
		log.Debugf("succeed to collect %s", logFile)
	}
	return
}

func (c *YHCChecker) reverseCollectLog(src string, date time.Time, timeParseFunc logTimeParseFunc) (res []string, err error) {
	reverseSrcFile, err := fileutil.NewReverseFile(src)
	if err != nil {
		return
	}
	defer reverseSrcFile.Close()
	for {
		line, e := reverseSrcFile.ReadLine()
		if e != nil {
			if e == io.EOF {
				// read to end
				break
			}
			err = e
			return
		}
		var t time.Time
		t, err = timeParseFunc(date, stringutil.RemoveExtraSpaces(strings.TrimSpace(line)))
		if err != nil {
			return
		}
		if t.After(c.base.End) {
			continue
		}
		if t.Before(c.base.Start) {
			break
		}
		// write to tmp file
		if c.systemLogPredicate(line) {
			res = append(res, fmt.Sprintf("%s\n", line))
		}
	}
	for i := 0; i < len(res)/2; i++ {
		j := len(res) - i - 1
		res[i], res[j] = res[j], res[i]
	}
	return
}

func (c *YHCChecker) hasSetDateext() (res bool, err error) {
	config, err := os.Open(LOG_ROTATE_CONFIG)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(config)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "dateext") {
			res = true
			return
		}
	}
	return
}

func (c *YHCChecker) hostLogTimeParse(date time.Time, line string) (t time.Time, err error) {
	fields := strings.Split(line, stringutil.STR_BLANK_SPACE)
	if len(fields) < 3 {
		err = fmt.Errorf("invalid line: %s, skip", line)
		return
	}
	tmpTime, err := time.ParseInLocation(timedef.TIME_FORMAT_TIME, fields[2], time.Local)
	if err != nil {
		return
	}
	hour, min, sec := tmpTime.Hour(), tmpTime.Minute(), tmpTime.Second()
	day, err := strconv.Atoi(fields[1])
	if err != nil {
		return
	}
	mon, err := timeutil.GetMonth(fields[0])
	year := date.Year()
	if date.Month() < mon {
		year = year - 1
	}
	t = time.Date(year, mon, day, hour, min, sec, 0, time.Local)
	return
}

func (c *YHCChecker) systemLogPredicate(line string) bool {
	// oom 内核错误 存储错误 网络错误之类的异常
	filterKeys := []string{
		"BIOS Error",
		"Error",
		"FAILED Result",
		"Hardware Error",
		"I/O error",
		"iptables denied",
		"refused connect",
		"Possible SYN flooding",
		"drop open reques",
		"connection reset",
		"error",
		"Out of memory",
		"Killed process",
	}
	for _, key := range filterKeys {
		if strings.Contains(line, key) {
			return true
		}
	}
	return false
}

func (c *YHCChecker) GetDatabaseChangeLog(name string) (err error) {
	data := &define.YHCItem{
		Name: define.METRIC_YASDB_RUN_LOG_DATABASE_CHANGES,
	}
	defer c.fillResults(data)
	log := log.Module.M("run-log-database-change")
	runLogPath, err := c.getRunLogPath(log)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	res, err := c.collectLog(log, path.Join(runLogPath, NAME_YASDB_RUN_LOG), time.Now(), c.filterDatabaseChange, c.yasdbLogTimeParse)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	if len(res) == 0 {
		res = append(res, i18n.T("log.no_obvious_error"))
	}
	data.Details = res
	return
}

func (c *YHCChecker) filterDatabaseChange(line string) bool {
	/*INFO级别记录数据库正常运行中发生的关键事件，主要包括：
	  数据库状态变更：例如启动、关闭、升主降备。
	  数据库关键资源的变更：例如表空间、用户增加删除等。
	  数据库资源变更：例如线程的启动和停止等。
	  数据库关键活动：例如重启恢复、残留事务等。
	  数据库参数变化：修改系统配置、隐藏参数等。*/
	return strings.Contains(line, "INFO")
}
