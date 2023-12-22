package check

import (
	"fmt"
	"time"

	"yhc/defs/confdef"
	"yhc/defs/timedef"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/yasdbutil"

	"git.yasdb.com/go/yaserr"
)

const (
	KEY_HIT_RATE = "HIT_RATE"
)

func (c *YHCChecker) GetYasdbHistoryBufferHitRate(name string) (err error) {
	data := &define.YHCItem{Name: define.METRIC_YASDB_HISTORY_BUFFER_HIT_RATE}
	defer c.fillResult(data)

	logger := log.Module.M(string(define.METRIC_YASDB_HISTORY_BUFFER_HIT_RATE))
	yasdb := yasdbutil.NewYashanDB(logger, c.base.DBInfo)
	dbTimes, err := yasdb.QueryMultiRows(
		fmt.Sprintf(define.SQL_QUERY_HISTORY_BUFFER_HIT_RATE,
			c.formatFunc(c.base.Start),
			c.formatFunc(c.base.End)),
		confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		err = yaserr.Wrap(err)
		logger.Error(err)
		data.Error = err.Error()
		return
	}
	content := make(define.WorkloadOutput)
	for _, row := range dbTimes {
		t, err := time.ParseInLocation(timedef.TIME_FORMAT, row[KEY_SNAP_TIME], time.Local)
		if err != nil {
			logger.Errorf("parse time %s failed: %s", row[KEY_SNAP_TIME], err)
			continue
		}
		content[t.Unix()] = define.WorkloadItem{KEY_HIT_RATE: row}
	}
	data.Details = content
	return
}
