package check

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"yhc/defs/confdef"
	"yhc/defs/timedef"
	"yhc/internal/modules/yhc/check/alertgenner"
	"yhc/internal/modules/yhc/check/define"
	"yhc/internal/modules/yhc/check/evaluator"
	"yhc/internal/modules/yhc/check/gopsutil"
	"yhc/internal/modules/yhc/check/jsonparser"
	"yhc/internal/modules/yhc/check/sar"
	"yhc/log"
	"yhc/utils/stringutil"
	"yhc/utils/yasdbutil"

	"git.yasdb.com/go/yaserr"
	"git.yasdb.com/go/yaslog"
	"git.yasdb.com/go/yasutil/size"
)

const (
	SQL_QUERY_SINGLE_PARAMETER_FORMATER = "select value from v$parameter where name='%s'"
)

const (
	KEY_CURRENT = "current"
	KEY_HISTORY = "history"
)

var CheckMutipleNodes = false

var _envs = []string{"LANG=en_US.UTF-8", "LC_TIME=en_US.UTF-8"}

var MetricNameToWorkloadTypeMap = map[define.MetricName]define.WorkloadType{
	define.METRIC_HOST_HISTORY_CPU_USAGE:    define.WT_CPU,
	define.METRIC_HOST_CURRENT_CPU_USAGE:    define.WT_CPU,
	define.METRIC_HOST_CURRENT_DISK_IO:      define.WT_DISK,
	define.METRIC_HOST_HISTORY_DISK_IO:      define.WT_DISK,
	define.METRIC_HOST_CURRENT_MEMORY_USAGE: define.WT_MEMORY,
	define.METRIC_HOST_HISTORY_MEMORY_USAGE: define.WT_MEMORY,
	define.METRIC_HOST_CURRENT_NETWORK_IO:   define.WT_NETWORK,
	define.METRIC_HOST_HISTORY_NETWORK_IO:   define.WT_NETWORK,
}

var SQLMap = map[define.MetricName]string{
	define.METRIC_YASDB_DATAFILE:                                                               define.SQL_QUERY_DATAFILE,
	define.METRIC_YASDB_CONTROLFILE_COUNT:                                                      define.SQL_QUERY_CONTROLFILE_COUNT,
	define.METRIC_YASDB_DEPLOYMENT_ARCHITECTURE:                                                define.SQL_QUERY_DEPLYMENT_ARCHITECTURE,
	define.METRIC_YASDB_DATABASE:                                                               define.SQL_QUERY_DATABASE,
	define.METRIC_YASDB_INSTANCE:                                                               define.SQL_QUERY_INSTANCE,
	define.METRIC_YASDB_INDEX_BLEVEL:                                                           define.SQL_QUERY_INDEX_BLEVEL,
	define.METRIC_YASDB_INDEX_COLUMN:                                                           define.SQL_QUERY_INDEX_COLUMN,
	define.METRIC_YASDB_INDEX_INVISIBLE:                                                        define.SQL_QUERY_INDEX_INVISIBLE,
	define.METRIC_YASDB_INDEX_OVERSIZED:                                                        define.SQL_QUERY_OVERSIZED_INDEX,
	define.METRIC_YASDB_INDEX_TABLE_INDEX_NOT_TOGETHER:                                         define.SQL_QUERY_TABLE_INDEX_NOT_TOGETHER,
	define.METRIC_YASDB_SEQUENCE_NO_AVAILABLE:                                                  define.SQL_QUERY_NO_AVAILABLE_VALUE,
	define.METRIC_YASDB_TASK_RUNNING:                                                           define.SQL_QUERY_RUNNING_JOB,
	define.METRIC_YASDB_PACKAGE_NO_PACKAGE_PACKAGE_BODY:                                        define.SQL_NO_PACKAGE_PACKAGE_BODY,
	define.METRIC_YASDB_SECURITY_LOGIN_PASSWORD_STRENGTH:                                       define.SQL_QUERY_PASSWORD_STRENGTH,
	define.METRIC_YASDB_SECURITY_LOGIN_MAXIMUM_LOGIN_ATTEMPTS:                                  define.SQL_QUERY_MAXIMUM_LOGIN_ATTEMPTS,
	define.METRIC_YASDB_SECURITY_USER_NO_OPEN:                                                  define.SQL_QUERY_USER_NO_OPEN,
	define.METRIC_YASDB_SECURITY_USER_WITH_SYSTEM_TABLE_PRIVILEGES:                             define.SQL_QUERY_USER_WITH_SYSTEM_TABLE_PRIVILEGES,
	define.METRIC_YASDB_SECURITY_USER_WITH_DBA_ROLE:                                            define.SQL_QUERY_ALL_USERS_WITH_DBA_ROLE,
	define.METRIC_YASDB_SECURITY_USER_ALL_PRIVILEGE_OR_SYSTEM_PRIVILEGES:                       define.SQL_QUERY_ALL_USERS_ALL_PRIVILEGE_OR_SYSTEM_PRIVILEGES,
	define.METRIC_YASDB_SECURITY_USER_USE_SYSTEM_TABLESPACE:                                    define.SQL_QUERY_USERS_USE_SYSTEM_TABLESPACE,
	define.METRIC_YASDB_SECURITY_AUDIT_CLEANUP_TASK:                                            define.SQL_QUERY_AUDIT_CLEANUP_TASK,
	define.METRIC_YASDB_SECURITY_AUDIT_FILE_SIZE:                                               define.SQL_QUERY_AUDIT_FILE_SIZE,
	define.METRIC_YASDB_LISTEN_ADDR:                                                            define.SQL_QUERY_LISTEN_ADDR,
	define.METRIC_YASDB_TABLESPACE:                                                             define.SQL_QUERY_TABLESPACE,
	define.METRIC_YASDB_WAIT_EVENT:                                                             define.SQL_QUERY_WAIT_EVENT,
	define.METRIC_YASDB_ARCHIVE_DEST_STATUS:                                                    define.SQL_QUERY_ARCHIVE_DEST_STATUS,
	define.METRIC_YASDB_ARCHIVE_LOG_SPACE:                                                      define.SQL_QUERY_ARCHIVE_LOG_SPACE,
	define.METRIC_YASDB_ARCHIVE_LOG:                                                            define.SQL_QUERY_ARCHIVE_LOG,
	define.METRIC_YASDB_PARAMETER:                                                              define.SQL_QUERY_PARAMETER,
	define.METRIC_YASDB_OBJECT_COUNT:                                                           define.SQL_QUERY_TOTAL_OBJECT,
	define.METRIC_YASDB_OBJECT_SUMMARY:                                                         define.SQL_QUERY_OBJECT_SUMMARY,
	define.METRIC_YASDB_SEGMENTS_COUNT:                                                         define.SQL_QUERY_YASDB_SEGMENTS_COUNT,
	define.METRIC_YASDB_SEGMENTS_SUMMARY:                                                       define.SQL_QUERY_METRIC_YASDB_SEGMENTS_SUMMARY,
	define.METRIC_YASDB_REDO_LOG:                                                               define.SQL_QUERY_LOGFILE,
	define.METRIC_YASDB_REDO_LOG_COUNT:                                                         define.SQL_QUERY_LOGFILE_COUNT,
	define.METRIC_YASDB_CONTROLFILE:                                                            define.SQL_QUERY_CONTROLFILE,
	define.METRIC_YASDB_BACKUP_SET:                                                             define.SQL_QUERY_BACKUP_SET,
	define.METRIC_YASDB_FULL_BACKUP_SET_COUNT:                                                  define.SQL_QUERY_FULL_BACKUP_SET_COUNT,
	define.METRIC_YASDB_BACKUP_SET_PATH:                                                        define.SQL_QUERY_BACKUP_SET_PATH,
	define.METRIC_YASDB_VM_SWAP_RATE:                                                           define.SQL_QUERY_VM_SWAP_RATE,
	define.METRIC_YASDB_TOP_SQL_BY_CPU_TIME:                                                    define.SQL_QUERY_YASDB_TOP_SQL_BY_CPU_TIME,
	define.METRIC_YASDB_TOP_SQL_BY_BUFFER_GETS:                                                 define.SQL_QUERY_YASDB_TOP_SQL_BY_BUFFER_GETS,
	define.METRIC_YASDB_TOP_SQL_BY_DISK_READS:                                                  define.SQL_QUERY_YASDB_TOP_SQL_BY_DISK_READS,
	define.METRIC_YASDB_TOP_SQL_BY_PARSE_CALLS:                                                 define.SQL_QUERY_YASDB_TOP_SQL_BY_PARSE_CALLS,
	define.METRIC_YASDB_HIGH_FREQUENCY_SQL:                                                     define.SQL_QUERY_HIGH_FREQUENCY_SQL,
	define.METRIC_YASDB_BUFFER_HIT_RATE:                                                        define.SQL_QUERY_BUFFER_HIT_RATE,
	define.METRIC_YASDB_TABLE_LOCK_WAIT:                                                        define.SQL_QUERY_TABLE_LOCK_WAIT,
	define.METRIC_YASDB_ROW_LOCK_WAIT:                                                          define.SQL_QUERY_ROW_LOCK_WAIT,
	define.METRIC_YASDB_LONG_RUNNING_TRANSACTION:                                               define.SQL_QUERY_LONG_RUNNING_TRANSACTION,
	define.METRIC_YASDB_INVALID_OBJECT:                                                         define.SQL_QUERY_INVALID_OBJECT,
	define.METRIC_YASDB_INVISIBLE_INDEX:                                                        define.SQL_QUERY_INVISIBLE_INDEX,
	define.METRIC_YASDB_DISABLED_CONSTRAINT:                                                    define.SQL_QUERY_DISABLED_CONSTRAINT,
	define.METRIC_YASDB_TABLE_WITH_TOO_MUCH_COLUMNS:                                            define.SQL_QUERY_TABLE_WITH_TOO_MUCH_COLUMNS,
	define.METRIC_YASDB_TABLE_WITH_TOO_MUCH_INDEXES:                                            define.SQL_QUERY_TABLE_WITH_TOO_MUCH_INDEXES,
	define.METRIC_YASDB_PARTITIONED_TABLE_WITHOUT_PARTITIONED_INDEXES:                          define.SQL_QUERY_PARTITIONED_TABLE_WITHOUT_PARTITIONED_INDEXES,
	define.METRIC_YASDB_PARTITIONED_TABLE_WITH_NUMBER_OF_HASH_PARTITIONS_IS_NOT_A_POWER_OF_TWO: define.SQL_QUERY_YASDB_PARTITIONED_TABLE_WITH_NUMBER_OF_HASH_PARTITIONS_IS_NOT_A_POWER_OF_TWO,
	define.METRIC_YASDB_TABLE_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS:     define.SQL_QUERY_YASDB_TABLE_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS,
	define.METRIC_YASDB_COLUMN_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS:    define.SQL_QUERY_YASDB_COLUMN_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS,
	define.METRIC_YASDB_FOREIGN_KEYS_WITHOUT_INDEXES:                                           define.SQL_QUERY_YASDB_FOREIGN_KEYS_WITHOUT_INDEXES,
	define.METRIC_YASDB_FOREIGN_KEYS_WITH_IMPLICIT_DATA_TYPE_CONVERSION:                        define.SQL_QUERY_YASDB_FOREIGN_KEYS_WITH_IMPLICIT_DATA_TYPE_CONVERSION,
}

type logTimeParseFunc func(date time.Time, line string) (time.Time, error)

type logPredicate func(line string) bool

type Checker interface {
	CheckFuncs(metrics []*confdef.YHCMetric) map[string]func(string) error
	GetResult(startCheck, endCheck time.Time) (map[define.MetricName][]*define.YHCItem, *define.PandoraReport, map[define.MetricName][]*define.YHCItem)
}

type YHCChecker struct {
	mtx            sync.RWMutex
	base           *define.CheckerBase
	metrics        []*confdef.YHCMetric
	Result         map[define.MetricName][]*define.YHCItem
	evaluateResult *define.EvaluateResult
	FailedItem     map[define.MetricName][]*define.YHCItem
}

func NewYHCChecker(base *define.CheckerBase, metrics []*confdef.YHCMetric) *YHCChecker {
	return &YHCChecker{
		base:       base,
		metrics:    metrics,
		Result:     map[define.MetricName][]*define.YHCItem{},
		FailedItem: map[define.MetricName][]*define.YHCItem{},
	}
}

// [Interface Func]
func (c *YHCChecker) GetResult(startCheck, endCheck time.Time) (map[define.MetricName][]*define.YHCItem, *define.PandoraReport, map[define.MetricName][]*define.YHCItem) {
	c.filterFailed()
	c.genAlerts()
	c.evaluate()
	return c.Result, c.genReportJson(startCheck, endCheck), c.FailedItem
}

func (c *YHCChecker) genReportJson(startCheck, endCheck time.Time) *define.PandoraReport {
	log := log.Module.M("gen-report-json")
	parser := jsonparser.NewJsonParser(log, *c.base, startCheck, endCheck, c.metrics, c.Result, c.evaluateResult)
	return parser.Parse()
}

func (c *YHCChecker) genAlerts() {
	log := log.Module.M("gen-alert")
	alertGenner := alertgenner.NewAlterGenner(log, c.metrics, c.Result)
	c.Result = alertGenner.GenAlerts()
}

func (c *YHCChecker) evaluate() {
	log := log.Module.M("evaluate")
	evaluator := evaluator.NewEvaluator(log, c.Result, c.FailedItem)
	c.evaluateResult = evaluator.Evaluate()
}

func (c *YHCChecker) filterFailed() {
	for _, metric := range c.metrics {
		name := define.MetricName(metric.Name)
		items, ok := c.Result[name]
		if !ok {
			continue
		}
		result := []*define.YHCItem{}
		for _, item := range items {
			if !stringutil.IsEmpty(item.Error) {
				c.FailedItem[name] = append(c.FailedItem[name], item)
			} else {
				result = append(result, item)
			}
		}
		c.Result[name] = result
	}
}

// [Interface Func]
func (c *YHCChecker) CheckFuncs(metrics []*confdef.YHCMetric) (res map[string]func(string) error) {
	res = make(map[string]func(string) error)
	defaultFuncMap := c.funcMap()
	for _, metric := range metrics {
		if metric.Default {
			fn, ok := defaultFuncMap[define.MetricName(metric.Name)]
			if !ok {
				log.Module.Errorf("failed to find function of default metric %s", metric.Name)
				continue
			}
			res[metric.Name] = fn
			continue
		}
		fn, err := c.GenCustomCheckFunc(metric)
		if err != nil {
			log.Module.Errorf("failed to gen function of custom metric %s", metric.Name)
			continue
		}
		res[metric.Name] = fn
	}
	return
}

func (c *YHCChecker) funcMap() (res map[define.MetricName]func(string) error) {
	res = map[define.MetricName]func(string) error{
		define.METRIC_HOST_INFO:                                                                    c.GetHostInfo,
		define.METRIC_HOST_FIREWALLD:                                                               c.GetHostFirewalldStatus,
		define.METRIC_HOST_IPTABLES:                                                                c.GetHostIPTables,
		define.METRIC_HOST_CPU_INFO:                                                                c.GetHostCPUInfo,
		define.METRIC_HOST_DISK_INFO:                                                               c.GetHostDiskInfo,
		define.METRIC_HOST_DISK_BLOCK_INFO:                                                         c.GetHostDiskBlockInfo,
		define.METRIC_HOST_BIOS_INFO:                                                               c.GetHostBIOSInfo,
		define.METRIC_HOST_MEMORY_INFO:                                                             c.GetHostMemoryInfo,
		define.METRIC_HOST_NETWORK_INFO:                                                            c.GetHostNetworkInfo,
		define.METRIC_HOST_HISTORY_CPU_USAGE:                                                       c.GetHostHistoryCPUUsage,
		define.METRIC_HOST_CURRENT_CPU_USAGE:                                                       c.GetHostCurrentCPUUsage,
		define.METRIC_HOST_CURRENT_DISK_IO:                                                         c.GetHostCurrentDiskIO,
		define.METRIC_HOST_HISTORY_DISK_IO:                                                         c.GetHostHistoryDiskIO,
		define.METRIC_HOST_CURRENT_MEMORY_USAGE:                                                    c.GetHostCurrentMemoryUsage,
		define.METRIC_HOST_HISTORY_MEMORY_USAGE:                                                    c.GetHostHistoryMemoryUsage,
		define.METRIC_HOST_CURRENT_NETWORK_IO:                                                      c.GetHostCurrentNetworkIO,
		define.METRIC_HOST_HISTORY_NETWORK_IO:                                                      c.GetHostHistoryNetworkIO,
		define.METRIC_YASDB_CONTROLFILE:                                                            c.GetNodesMultiRowData,
		define.METRIC_YASDB_CONTROLFILE_COUNT:                                                      c.GetPrimarySingleRowData,
		define.METRIC_YASDB_DATAFILE:                                                               c.GetYasdbDataFile,
		define.METRIC_YASDB_DATABASE:                                                               c.GetPrimarySingleRowData,
		define.METRIC_YASDB_DEPLOYMENT_ARCHITECTURE:                                                c.GetYasdbDeploymentArchitecture,
		define.METRIC_YASDB_ARCHIVE_THRESHOLD:                                                      c.GetNodesMultiRowData,
		define.METRIC_YASDB_FILE_PERMISSION:                                                        c.GetYasdbFilePermission,
		define.METRIC_YASDB_INDEX_BLEVEL:                                                           c.GetNodesMultiRowData,
		define.METRIC_YASDB_INDEX_COLUMN:                                                           c.GetNodesMultiRowData,
		define.METRIC_YASDB_INDEX_INVISIBLE:                                                        c.GetNodesMultiRowData,
		define.METRIC_YASDB_INSTANCE:                                                               c.GetPrimarySingleRowData,
		define.METRIC_YASDB_LISTEN_ADDR:                                                            c.GetPrimarySingleRowData,
		define.METRIC_YASDB_OS_AUTH:                                                                c.GetYasdbOSAuth,
		define.METRIC_YASDB_RUN_LOG_ERROR:                                                          c.GetYasdbRunLogError,
		define.METRIC_YASDB_REDO_LOG:                                                               c.GetNodesMultiRowData,
		define.METRIC_YASDB_REDO_LOG_COUNT:                                                         c.GetPrimarySingleRowData,
		define.METRIC_YASDB_OBJECT_COUNT:                                                           c.GetNodesSingleRowData,
		define.METRIC_YASDB_OBJECT_SUMMARY:                                                         c.GetNodesMultiRowData,
		define.METRIC_YASDB_SEGMENTS_COUNT:                                                         c.GetNodesSingleRowData,
		define.METRIC_YASDB_SEGMENTS_SUMMARY:                                                       c.GetNodesMultiRowData,
		define.METRIC_YASDB_ARCHIVE_DEST_STATUS:                                                    c.GetYasdbArchiveDestStatus,
		define.METRIC_YASDB_ARCHIVE_LOG:                                                            c.GetNodesMultiRowData,
		define.METRIC_YASDB_ARCHIVE_LOG_SPACE:                                                      c.GetNodesSingleRowData,
		define.METRIC_YASDB_PARAMETER:                                                              c.GetYasdbParameter,
		define.METRIC_YASDB_SESSION:                                                                c.GetNodesSingleRowData,
		define.METRIC_YASDB_TABLESPACE:                                                             c.GetNodesMultiRowData,
		define.METRIC_YASDB_WAIT_EVENT:                                                             c.GetYasdbWaitEvent,
		define.METRIC_YASDB_INDEX_TABLE_INDEX_NOT_TOGETHER:                                         c.GetNodesMultiRowData,
		define.METRIC_YASDB_INDEX_OVERSIZED:                                                        c.GetNodesMultiRowData,
		define.METRIC_YASDB_SEQUENCE_NO_AVAILABLE:                                                  c.GetNodesMultiRowData,
		define.METRIC_YASDB_TASK_RUNNING:                                                           c.GetNodesMultiRowData,
		define.METRIC_YASDB_PACKAGE_NO_PACKAGE_PACKAGE_BODY:                                        c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_LOGIN_PASSWORD_STRENGTH:                                       c.GetNodesSingleRowData,
		define.METRIC_YASDB_AUDITINT_CHECK:                                                         c.GetNodesSingleRowData,
		define.METRIC_YASDB_SECURITY_LOGIN_MAXIMUM_LOGIN_ATTEMPTS:                                  c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_USER_NO_OPEN:                                                  c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_USER_WITH_SYSTEM_TABLE_PRIVILEGES:                             c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_USER_WITH_DBA_ROLE:                                            c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_USER_ALL_PRIVILEGE_OR_SYSTEM_PRIVILEGES:                       c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_USER_USE_SYSTEM_TABLESPACE:                                    c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_AUDIT_CLEANUP_TASK:                                            c.GetNodesMultiRowData,
		define.METRIC_YASDB_SECURITY_AUDIT_FILE_SIZE:                                               c.GetNodesMultiRowData,
		define.METRIC_YASDB_UNDO_LOG_SIZE:                                                          c.GetNodesMultiRowData,
		define.METRIC_YASDB_UNDO_LOG_TOTAL_BLOCK:                                                   c.GetNodesMultiRowData,
		define.METRIC_YASDB_UNDO_LOG_RUNNING_TRANSACTIONS:                                          c.GetNodesMultiRowData,
		define.METRIC_YASDB_RUN_LOG_DATABASE_CHANGES:                                               c.GetDatabaseChangeLog,
		define.METRIC_YASDB_SLOW_LOG_PARAMETER:                                                     c.GetYasdbSlowLogParameter,
		define.METRIC_YASDB_SLOW_LOG:                                                               c.GetYasdbSlowLog,
		define.METRIC_YASDB_SLOW_LOG_FILE:                                                          c.GetYasdbSlowLogFile,
		define.METRIC_YASDB_ALERT_LOG_ERROR:                                                        c.GetRisingAlertLog,
		define.METRIC_HOST_DMESG_LOG_ERROR:                                                         c.GetDmesgLog,
		define.METRIC_HOST_SYSTEM_LOG_ERROR:                                                        c.GetSystemLog,
		define.METRIC_YASDB_BACKUP_SET:                                                             c.GetNodesMultiRowData,
		define.METRIC_YASDB_FULL_BACKUP_SET_COUNT:                                                  c.GetNodesSingleRowData,
		define.METRIC_YASDB_BACKUP_SET_PATH:                                                        c.GetYasdbBackupSetPath,
		define.METRIC_YASDB_INVALID_OBJECT:                                                         c.GetNodesMultiRowData,
		define.METRIC_YASDB_INVISIBLE_INDEX:                                                        c.GetNodesMultiRowData,
		define.METRIC_YASDB_DISABLED_CONSTRAINT:                                                    c.GetNodesMultiRowData,
		define.METRIC_YASDB_TABLE_WITH_TOO_MUCH_COLUMNS:                                            c.GetNodesMultiRowData,
		define.METRIC_YASDB_TABLE_WITH_TOO_MUCH_INDEXES:                                            c.GetNodesMultiRowData,
		define.METRIC_YASDB_PARTITIONED_TABLE_WITHOUT_PARTITIONED_INDEXES:                          c.GetNodesMultiRowData,
		define.METRIC_YASDB_PARTITIONED_TABLE_WITH_NUMBER_OF_HASH_PARTITIONS_IS_NOT_A_POWER_OF_TWO: c.GetNodesMultiRowData,
		define.METRIC_YASDB_TABLE_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS:     c.GetNodesMultiRowData,
		define.METRIC_YASDB_COLUMN_NAME_CASE_SENSITIVE_OR_INCLUDE_KEYWORD_OR_SPECIAL_CHARACTERS:    c.GetNodesMultiRowData,
		define.METRIC_YASDB_FOREIGN_KEYS_WITHOUT_INDEXES:                                           c.GetNodesMultiRowData,
		define.METRIC_YASDB_FOREIGN_KEYS_WITH_IMPLICIT_DATA_TYPE_CONVERSION:                        c.GetNodesMultiRowData,
		define.METRIC_YASDB_TABLE_WITH_ROW_SIZE_EXCEEDS_BLOCK_SIZE:                                 c.GetNodesMultiRowData,
		define.METRIC_YASDB_SHARE_POOL:                                                             c.GetYasdbSharePool,
		define.METRIC_YASDB_VM_SWAP_RATE:                                                           c.GetNodesSingleRowData,
		define.METRIC_YASDB_TOP_SQL_BY_CPU_TIME:                                                    c.GetNodesMultiRowData,
		define.METRIC_YASDB_TOP_SQL_BY_BUFFER_GETS:                                                 c.GetNodesMultiRowData,
		define.METRIC_YASDB_TOP_SQL_BY_DISK_READS:                                                  c.GetNodesMultiRowData,
		define.METRIC_YASDB_TOP_SQL_BY_PARSE_CALLS:                                                 c.GetNodesMultiRowData,
		define.METRIC_YASDB_HIGH_FREQUENCY_SQL:                                                     c.GetNodesMultiRowData,
		define.METRIC_YASDB_HISTORY_DB_TIME:                                                        c.GetYasdbHistoryDBTime,
		define.METRIC_YASDB_HISTORY_BUFFER_HIT_RATE:                                                c.GetYasdbHistoryBufferHitRate,
		define.METRIC_HOST_HUGE_PAGE:                                                               c.GetHugePageEnabled,
		define.METRIC_HOST_SWAP_MEMORY:                                                             c.GetSwapMemoryEnabled,
		define.METRIC_YASDB_BUFFER_HIT_RATE:                                                        c.GetNodesSingleRowData,
		define.METRIC_YASDB_TABLE_LOCK_WAIT:                                                        c.GetNodesSingleRowData,
		define.METRIC_YASDB_ROW_LOCK_WAIT:                                                          c.GetNodesSingleRowData,
		define.METRIC_YASDB_LONG_RUNNING_TRANSACTION:                                               c.GetNodesMultiRowData,
	}
	return
}

func (c *YHCChecker) fillResults(datas ...*define.YHCItem) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	for _, data := range datas {
		c.Result[data.Name] = append(c.Result[data.Name], data)
	}
}

func (c *YHCChecker) querySingleParameter(log yaslog.YasLog, name string) (string, error) {
	sql := fmt.Sprintf(SQL_QUERY_SINGLE_PARAMETER_FORMATER, name)
	yasdb := yasdbutil.NewYashanDB(log, c.base.DBInfo)
	res, err := yasdb.QueryMultiRows(sql, confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		return "", err
	}
	if len(res) <= 0 {
		err = yaserr.Wrap(err)
		log.Error(err)
		return "", err
	}
	return res[0]["VALUE"], nil
}

func (c *YHCChecker) getLogFiles(log yaslog.YasLog, logPath string, prefix string) (logFiles []string, err error) {
	entrys, err := os.ReadDir(logPath)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		return
	}
	for _, entry := range entrys {
		if !entry.Type().IsRegular() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		logFiles = append(logFiles, path.Join(logPath, entry.Name()))
	}
	// sort with file name
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i] < logFiles[j]
	})
	return
}

func (c *YHCChecker) collectLog(log yaslog.YasLog, src string, date time.Time, predicate logPredicate, timeParseFunc logTimeParseFunc) ([]string, error) {
	res := []string{}
	srcFile, err := os.Open(src)
	if err != nil {
		return res, err
	}
	defer srcFile.Close()

	var t time.Time
	scanner := bufio.NewScanner(srcFile)
	for scanner.Scan() {
		txt := scanner.Text()
		line := stringutil.RemoveExtraSpaces(strings.TrimSpace(txt))
		if stringutil.IsEmpty(line) || !predicate(line) {
			continue
		}
		if t, err = timeParseFunc(date, line); err != nil {
			log.Errorf("skip line: %s, err: %s", txt, err.Error())
			continue
		}
		if t.Before(c.base.Start) {
			continue
		}
		if t.After(c.base.End) {
			break
		}
		res = append(res, txt)
	}
	return res, nil
}

func (c *YHCChecker) hostHistoryWorkload(log yaslog.YasLog, name define.MetricName) (resp define.WorkloadOutput, err error) {
	// get sar args
	workloadType, ok := MetricNameToWorkloadTypeMap[name]
	if !ok {
		err = fmt.Errorf("failed to get workload type from metric name: %s", name)
		log.Error(err)
		return
	}
	sarArg, ok := sar.WorkloadTypeToSarArgMap[workloadType]
	if !ok {
		err = fmt.Errorf("failed to get SAR arg from workload type: %s", workloadType)
		log.Error(err)
		return
	}
	// collect
	sarCollector := sar.NewSar(log)
	sarDir := confdef.GetYHCConf().GetSarDir()
	if stringutil.IsEmpty(sarDir) {
		sarDir = sarCollector.GetSarDir()
	}
	sarOutput := make(define.WorkloadOutput)
	args := c.genHistoryWorkloadArgs(c.base.Start, c.base.End, sarDir)
	for _, arg := range args {
		output, e := sarCollector.Collect(workloadType, sarArg, arg)
		if e != nil {
			log.Error(e)
			continue
		}
		for timestamp, output := range output {
			sarOutput[timestamp] = output
		}
	}
	resp = sarOutput
	return
}

func (c *YHCChecker) genHistoryWorkloadArgs(start, end time.Time, sarDir string) (args []string) {
	// get data between start and end
	var dates []time.Time
	begin := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	for date := begin; !date.After(end); date = date.AddDate(0, 0, 1) {
		dates = append(dates, date)
	}
	for i, date := range dates {
		var startArg, endArg, fileArg string
		// the frist
		if i == 0 && !date.Equal(start) {
			startArg = fmt.Sprintf("-s %s", start.Format(timedef.TIME_FORMAT_TIME))
		}
		// the last one
		if i == len(dates)-1 {
			if date.Equal(end) {
				// skip
				continue
			}
			endArg = fmt.Sprintf("-e %s", end.Format(timedef.TIME_FORMAT_TIME))
		}
		fileArg = fmt.Sprintf("-f %s", path.Join(sarDir, fmt.Sprintf("sa%s", date.Format(timedef.TIME_FORMAT_DAY))))
		args = append(args, fmt.Sprintf("%s %s %s", fileArg, startArg, endArg))
	}
	return
}

func (c *YHCChecker) hostCurrentWorkload(log yaslog.YasLog, name define.MetricName, hasSar bool) (resp define.WorkloadOutput, err error) {
	// global conf
	scrapeInterval, scrapeTimes := confdef.GetYHCConf().GetScrapeInterval(), confdef.GetYHCConf().GetScrapeTimes()
	// get sar args
	workloadType, ok := MetricNameToWorkloadTypeMap[name]
	if !ok {
		err = fmt.Errorf("failed to get workload type from metric name: %s", name)
		log.Error(err)
		return
	}
	if !hasSar {
		// use gopsutil to calculate by ourself
		return gopsutil.Collect(workloadType, scrapeInterval, scrapeTimes)
	}
	sarArg, ok := sar.WorkloadTypeToSarArgMap[workloadType]
	if !ok {
		err = fmt.Errorf("failed to get SAR arg from workload type: %s", workloadType)
		log.Error(err)
		return
	}
	sarCollector := sar.NewSar(log)
	return sarCollector.Collect(workloadType, sarArg, strconv.Itoa(scrapeInterval), strconv.Itoa(scrapeTimes))

}

func (c *YHCChecker) convertObjectData(object interface{}) (res map[string]any, err error) {
	res = make(map[string]any)
	bytes, err := json.Marshal(object)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &res)
	return
}

func (c *YHCChecker) convertMultiSqlData(metric *confdef.YHCMetric, datas []map[string]string) []map[string]interface{} {
	res := []map[string]interface{}{}
	for _, data := range datas {
		res = append(res, c.convertSqlData(metric, data))
	}
	return res
}

func (c *YHCChecker) convertSqlData(metric *confdef.YHCMetric, data map[string]string) map[string]interface{} {
	log := log.Module.M("convert-sql-data")
	res := make(map[string]interface{})
	for _, col := range metric.NumberColumns {
		value, ok := data[col]
		if !ok {
			log.Debugf("column %s not found, skip", col)
			continue
		}
		if len(value) == 0 {
			res[col] = 0
			continue
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Errorf("failed to parse column %s to float64, value: %s, metric: %s, err: %v", col, value, metric.Name, err)
			continue
		}
		res[col] = f
	}
	for _, col := range metric.ByteColumns {
		value, ok := data[col]
		if !ok {
			log.Debugf("column %s not found, skip", col)
			continue
		}
		if len(value) == 0 {
			res[col] = 0
			continue
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Errorf("failed to parse column %s to float64, value: %s, metric: %s, err: %v", col, value, metric.Name, err)
			continue
		}
		res[col] = size.GenHumanReadableSize(f, decimal)
	}
	for _, col := range metric.PercentColumns {
		value, ok := data[col]
		if !ok {
			log.Debugf("column %s not found, skip", col)
			continue
		}
		if len(value) == 0 {
			continue
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Errorf("failed to parse column %s to float64, value: %s, metric: %s, err: %v", col, value, metric.Name, err)
			continue
		}
		res[col] = fmt.Sprintf("%.2f%%", f)
	}
	for k, v := range data {
		if _, ok := res[k]; !ok {
			res[k] = v
		}
	}
	return res
}

func (c *YHCChecker) getMetric(name define.MetricName) (*confdef.YHCMetric, error) {
	for _, metric := range c.metrics {
		if metric.Name == string(name) {
			return metric, nil
		}
	}
	return nil, fmt.Errorf("failed to found metric by name %s", name)
}

func (c *YHCChecker) getSQL(name define.MetricName) (string, error) {
	metric, err := c.getMetric(name)
	if err != nil {
		return "", err
	}
	if stringutil.IsEmpty(metric.SQL) {
		return SQLMap[name], nil
	}
	return metric.SQL, nil
}
