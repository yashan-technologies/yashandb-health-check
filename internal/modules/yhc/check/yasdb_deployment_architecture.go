package check

import (
	"fmt"

	"yhc/defs/confdef"
	"yhc/i18n"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/yasdbutil"
)

const (
	KEY_NODE_NUM = "NODE_NUM"
)

func (c *YHCChecker) GetYasdbDeploymentArchitecture(name string) (err error) {
	data := &define.YHCItem{
		Name: define.METRIC_YASDB_DEPLOYMENT_ARCHITECTURE,
	}
	defer c.fillResults(data)

	log := log.Module.M(string(define.METRIC_YASDB_DEPLOYMENT_ARCHITECTURE))
	yasdb := yasdbutil.NewYashanDB(log, c.base.DBInfo)
	res, err := yasdb.QueryMultiRows(define.SQL_QUERY_DEPLYMENT_ARCHITECTURE, confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		log.Errorf("failed to get data with sql %s, err: %v", define.SQL_QUERY_DEPLYMENT_ARCHITECTURE, err)
		data.Error = err.Error()
		return
	}
	if len(res) == 0 {
		err = fmt.Errorf("failed to get data with sql %s", define.SQL_QUERY_DEPLYMENT_ARCHITECTURE)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	detail := map[string]interface{}{
		KEY_NODE_NUM: fmt.Sprintf(i18n.T("deployment.primary_standby"), res[0][KEY_NODE_NUM]),
	}
	data.Details = detail
	return
}
