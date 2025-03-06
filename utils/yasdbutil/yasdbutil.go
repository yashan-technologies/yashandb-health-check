package yasdbutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"yhc/commons/constants"
	"yhc/commons/yasdb"
	"yhc/defs/runtimedef"
	"yhc/utils/execerutil"

	"git.yasdb.com/go/yaslog"
	"git.yasdb.com/go/yasutil/execer"
)

const (
	LD_LIBRARY_KEY = "LD_LIBRARY_PATH"
)

const (
	_YASDB_GO_BIN = "yasdb-go"
)

const (
	CMD_QUERY = "query"
	CMD_EXEC  = "exec"
)

type YashanDB struct {
	*yasdb.YashanDB
	logger yaslog.YasLog `json:"-"`
}

func NewYashanDB(log yaslog.YasLog, yasdb *yasdb.YashanDB) *YashanDB {
	return &YashanDB{
		YashanDB: yasdb,
		logger:   log,
	}
}

func (y *YashanDB) getYasdbGoBin() string {
	return path.Join(runtimedef.GetScriptsPath(), _YASDB_GO_BIN)
}

func (y *YashanDB) procEnv() []string {
	env := os.Environ()
	return append(env, fmt.Sprintf("%s=%s", LD_LIBRARY_KEY, filepath.Join(y.YasdbHome, "lib")))
}

func (y *YashanDB) exec(args ...string) (int, string, string) {
	execer := execerutil.NewExecer(y.logger, execer.WithHidden(y.YasdbPassword))
	return execer.EnvExec(y.procEnv(), y.getYasdbGoBin(), args...)
}

func (y *YashanDB) genArgs(cmdType, sql string, timeout int) []string {
	args := []string{
		"-t", cmdType,
		"-s", sql,
		"-a", y.ListenAddr,
		"--timeout=" + strconv.FormatInt(int64(timeout), constants.BASE_DECIMAL),
	}
	if y.IsUdsOpen && (y.YasdbUser == "" || y.YasdbPassword == "") && y.YasdbData != "" {
		// 操作系统认证登录
		args = append(args, "-d", y.YasdbData)
	} else {
		args = append(args, "-u", y.YasdbUser)
		args = append(args, "-p", y.YasdbPassword)
	}
	return args
}

func (y *YashanDB) ExecSQL(sql string, timeout int) error {
	args := y.genArgs(CMD_EXEC, sql, timeout)
	ret, _, stderr := y.exec(args...)
	if ret != 0 {
		err := fmt.Errorf("failed to exec sql: %s, err: %s", sql, stderr)
		y.logger.Error(err)
		return err
	}
	return nil
}

func (y *YashanDB) QueryMultiRows(sql string, timeout int) ([]map[string]string, error) {
	res := []map[string]string{}
	args := y.genArgs(CMD_QUERY, sql, timeout)
	ret, stdout, stderr := y.exec(args...)
	if ret != 0 {
		err := fmt.Errorf("failed to exec sql: %s, err: %s", sql, stderr)
		y.logger.Error(err)
		return res, err
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil {
		y.logger.Errorf("failed to unmarshal result from %s, err: %s", stdout, err.Error())
		return res, err
	}
	return res, nil
}
