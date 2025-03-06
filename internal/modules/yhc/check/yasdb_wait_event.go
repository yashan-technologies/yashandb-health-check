package check

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"yhc/defs/bashdef"
	"yhc/defs/confdef"
	"yhc/defs/runtimedef"
	"yhc/defs/timedef"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/execerutil"
	"yhc/utils/fileutil"
	"yhc/utils/stringutil"
	"yhc/utils/yasdbutil"

	"git.yasdb.com/go/yaserr"
	"git.yasdb.com/go/yaslog"
	"git.yasdb.com/pandora/yasqlgo"
	"github.com/google/uuid"
)

type WaitEventOutput struct {
	Title string         `json:"title"`
	Table WaitEventTable `json:"table"`
}

type WaitEventTable struct {
	Header []string   `json:"header"`
	Body   [][]string `json:"body"`
}

const (
	KEY_DB_ID           = "DBID"
	KEY_INSTANCE_NUMBER = "INSTANCE_NUMBER"
	KEY_STARTUP_TIME    = "STARTUP_TIME"
	KEY_SNAP_ID         = "SNAP_ID"
	KEY_PLSQL_SUCCEED   = "PL/SQL Succeed."
)

const (
	_set_output      = "set serveroutput on;\n"
	_exec_wait_event = "exec sys.dbms_awr.create_top_forground_waitEvent(%d,%s,%d,%d);\n"
)

var waitEventHeaderAlias = map[string]string{
	"Event":                "EVENT",
	"Total Wait Time(sec)": "TOTAL_WAIT",
	"Avg Wait(ms)":         "AVG_WAIT",
	"% DB Time":            "DB_TIME",
	"Wait Class":           "WAIT_CLASS",
	"Waits":                "WAITS",
}

func (c *YHCChecker) GetYasdbWaitEvent(name string) (err error) {
	data := &define.YHCItem{Name: define.METRIC_YASDB_WAIT_EVENT}
	defer c.fillResults(data)

	log := log.Module.M(string(define.METRIC_YASDB_WAIT_EVENT))
	path, err := c.createYasdbEventSqlFile(log)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return err
	}
	defer os.Remove(path)

	ret, stdout, stderr := c.execSqlFile(log, path)
	if ret != 0 {
		err = fmt.Errorf("failed to get yasdb wait event, err: %s", stderr)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	detail, err := c.parseWaitEventStdout(stdout)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return err
	}
	data.Details = detail
	return
}

func (c *YHCChecker) parseWaitEventStdout(stdout string) ([]map[string]any, error) {
	stdout = strings.TrimSpace(stdout)
	stdout = strings.TrimSuffix(stdout, KEY_PLSQL_SUCCEED)
	waitEvents := &WaitEventOutput{}
	if err := json.Unmarshal([]byte(stdout), waitEvents); err != nil {
		return nil, yaserr.Wrapf(err, "json unmarshal")
	}
	table := map[int]map[string]any{}
	isBase64 := c.isWaitEventBodyBase64(waitEvents)
	for col, h := range waitEvents.Table.Header {
		if headerAlias, ok := waitEventHeaderAlias[h]; ok {
			h = headerAlias
		}
		for row, b := range waitEvents.Table.Body {
			m, ok := table[row]
			if !ok {
				m = map[string]any{}
			}
			content := b[col]
			if isBase64 {
				b, err := base64.StdEncoding.DecodeString(content)
				if err != nil {
					return nil, yaserr.Wrapf(err, "base64 decode")
				}
				content = string(b)
			}
			m[h] = content
			table[row] = m
		}
	}
	var res []map[string]any
	for i := 0; i < len(waitEvents.Table.Body); i++ {
		res = append(res, table[i])
	}
	return res, nil
}

func (c *YHCChecker) isWaitEventBodyBase64(waitEvents *WaitEventOutput) bool {
	if len(waitEvents.Table.Body) == 0 {
		return false
	}
	if len(waitEvents.Table.Body[0]) == 0 {
		return false
	}
	return stringutil.IsBase64(waitEvents.Table.Body[0][0])
}

func (c *YHCChecker) createYasdbEventSqlFile(log yaslog.YasLog) (filePath string, err error) {
	yasdb := yasdbutil.NewYashanDB(log, c.base.DBInfo)
	res, err := yasdb.QueryMultiRows(define.SQL_QUERY_DB_ID, confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		return
	}
	if len(res) == 0 {
		err = fmt.Errorf("failed to get instance info from sql '%s'", define.SQL_QUERY_DB_ID)
		return
	}
	dbId, err := strconv.ParseFloat(res[0][KEY_DB_ID], 64)
	if err != nil {
		return
	}
	instanceNumber := res[0][KEY_INSTANCE_NUMBER]
	startID, endID, err := c.genStartAndEndSnapId(yasdb, c.base.Start.Format((timedef.TIME_FORMAT)), c.base.End.Format((timedef.TIME_FORMAT)), res[0][KEY_STARTUP_TIME])
	if err != nil {
		return
	}
	var buffer bytes.Buffer
	buffer.WriteString(_set_output)
	buffer.WriteString(fmt.Sprintf(_exec_wait_event, int64(dbId), instanceNumber, startID, endID))
	filename := fmt.Sprintf("%s.sql", uuid.New())
	filePath = path.Join(runtimedef.GetYHCHome(), filename)
	err = fileutil.WriteFile(filePath, buffer.Bytes())
	return
}

func (c *YHCChecker) genStartAndEndSnapId(yasdb *yasdbutil.YashanDB, start, end, startup string) (int, int, error) {
	sql := fmt.Sprintf(define.SQL_QUERY_SNAPSHOT_FORMATER, start, end, startup)
	res, err := yasdb.QueryMultiRows(sql, confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		return 0, 0, err
	}
	if len(res) == 0 {
		return 0, 0, fmt.Errorf("failed to get info from sql '%s'", sql)
	}
	max, min := math.MinInt32, math.MaxInt32
	for _, snap := range res {
		snapId, err := strconv.ParseFloat(snap[KEY_SNAP_ID], 64)
		if err != nil {
			return 0, 0, err
		}
		if int(snapId) > max {
			max = int(snapId)
		}
		if int(snapId) < min {
			min = int(snapId)
		}
	}
	return min, max, nil
}

func (c *YHCChecker) execSqlFile(log yaslog.YasLog, sqlFile string) (int, string, string) {
	yasqlBin := path.Join(c.base.DBInfo.YasdbHome, yasqlgo.BIN_PATH, yasqlgo.YASQL_BIN)
	env := []string{
		fmt.Sprintf("%s=%s", yasqlgo.LIB_KEY, path.Join(c.base.DBInfo.YasdbHome, yasqlgo.LIB_PATH)),
		fmt.Sprintf("%s=%s", yasqlgo.YASDB_DATA, c.base.DBInfo.YasdbData),
	}
	systemCerticate := c.base.DBInfo.IsUdsOpen && (c.base.DBInfo.YasdbUser == "" || c.base.DBInfo.YasdbPassword == "")
	var connectStr string
	if systemCerticate {
		connectStr = " / as sysdba"
	} else {
		connectStr = fmt.Sprintf("%s/%s", c.base.DBInfo.YasdbUser, c.base.DBInfo.YasdbPassword)
	}
	cmd := fmt.Sprintf("%s %s -f %s", yasqlBin, connectStr, sqlFile)
	exec := execerutil.NewExecer(log)
	return exec.EnvExec(env, bashdef.CMD_BASH, "-c", cmd)
}
