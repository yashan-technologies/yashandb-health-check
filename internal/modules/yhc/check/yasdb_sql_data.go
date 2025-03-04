package check

import (
	"sort"

	"yhc/commons/yasdb"
	"yhc/defs/confdef"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/yasdbutil"

	"git.yasdb.com/go/yaserr"
	"git.yasdb.com/go/yaslog"
)

type CheckNodeInfo struct {
	IsLocal bool
	NodeID  string
	*yasdbutil.YashanDB
}

func (c *YHCChecker) GetCheckNodes(log yaslog.YasLog) []*CheckNodeInfo {
	res := []*CheckNodeInfo{}
	if !CheckMutipleNodes {
		res = append(res,
			&CheckNodeInfo{
				YashanDB: yasdbutil.NewYashanDB(log, c.base.DBInfo),
				IsLocal:  true,
			})
		return res
	}
	for _, node := range c.base.NodeInfos {
		yasdb := yasdbutil.NewYashanDB(log, &yasdb.YashanDB{
			YasdbHome:     c.base.DBInfo.YasdbHome,
			YasdbUser:     node.User,
			YasdbPassword: node.Password,
			ListenAddr:    node.ListenAddr})
		res = append(res,
			&CheckNodeInfo{NodeID: node.NodeID, YashanDB: yasdb, IsLocal: node.ListenAddr == c.base.DBInfo.ListenAddr})
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].NodeID < res[j].NodeID
	})
	return res
}

// 获取当前节点的单行数据
func (c *YHCChecker) GetPrimarySingleRowData(name string) (err error) {
	if !CheckMutipleNodes {
		return c.getCurrentNodeRowData(name, false)
	}
	return c.getPrimaryNodeRowData(name, false)
}

// 获取当前节点的多行数据
func (c *YHCChecker) GetPrimaryMultiRowData(name string) (err error) {
	if !CheckMutipleNodes {
		return c.getCurrentNodeRowData(name, true)
	}
	return c.getPrimaryNodeRowData(name, true)
}

// 获取多节点的单行数据
func (c *YHCChecker) GetNodesSingleRowData(name string) (err error) {
	if !CheckMutipleNodes {
		return c.getCurrentNodeRowData(name, false)
	}
	return c.getNodesRowData(name, false)
}

// 获取多节点的多行数据
func (c *YHCChecker) GetNodesMultiRowData(name string) (err error) {
	if !CheckMutipleNodes {
		return c.getCurrentNodeRowData(name, true)
	}
	return c.getNodesRowData(name, true)
}

func (c *YHCChecker) getCurrentNodeRowData(name string, isMulti bool) (err error) {
	data := &define.YHCItem{
		Name: define.MetricName(name),
	}
	defer c.fillResults(data)

	log := log.Module.M(name)
	sql, err := c.getSQL(define.MetricName(name))
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	metric, err := c.getMetric(define.MetricName(name))
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	yasdb := yasdbutil.NewYashanDB(log, c.base.DBInfo)
	res, err := yasdb.QueryMultiRows(sql, confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	if isMulti {
		data.Details = c.convertMultiSqlData(metric, res)
	} else {
		if len(res) == 0 {
			err = yaserr.Wrap(err)
			log.Error(err)
			data.Error = err.Error()
			return
		}
		data.Details = c.convertSqlData(metric, res[0])
	}
	return
}

func (c *YHCChecker) getPrimaryNodeRowData(name string, isMulti bool) (err error) {
	data := &define.YHCItem{
		Name: define.MetricName(name),
	}
	defer c.fillResults(data)

	log := log.Module.M(name)
	sql, err := c.getSQL(define.MetricName(name))
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	metric, err := c.getMetric(define.MetricName(name))
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	// 已经排过序了，如果有主节点，那么主节点在第一个
	node := c.base.NodeInfos[0]
	y := &yasdb.YashanDB{
		YasdbHome: c.base.DBInfo.YasdbHome,
	}
	if !node.SystemCerticate {
		y.YasdbUser = node.User
		y.YasdbPassword = node.Password
		y.ListenAddr = node.ListenAddr
	} else {
		y.YasdbData = c.base.DBInfo.YasdbData
		y.IsUdsOpen = true
	}
	yasdb := yasdbutil.NewYashanDB(log, y)
	res, err := yasdb.QueryMultiRows(sql, confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	if isMulti {
		data.Details = c.convertMultiSqlData(metric, res)
	} else {
		if len(res) == 0 {
			err = yaserr.Wrap(err)
			log.Error(err)
			data.Error = err.Error()
			return
		}
		data.Details = c.convertSqlData(metric, res[0])
	}
	return
}

func (c *YHCChecker) getNodesRowData(name string, isMulti bool) (err error) {
	var datas []*define.YHCItem
	log := log.Module.M(name)
	sql, err := c.getSQL(define.MetricName(name))
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		return
	}
	metric, err := c.getMetric(define.MetricName(name))
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		return
	}
	for _, node := range c.base.NodeInfos {
		data := &define.YHCItem{
			Name:   define.MetricName(name),
			NodeID: node.NodeID,
		}
		datas = append(datas, data)

		y := &yasdb.YashanDB{
			YasdbHome: c.base.DBInfo.YasdbHome,
		}
		if !node.SystemCerticate {
			y.YasdbUser = node.User
			y.YasdbPassword = node.Password
			y.ListenAddr = node.ListenAddr
		} else {
			y.YasdbData = c.base.DBInfo.YasdbData
			y.IsUdsOpen = true
		}

		yasdb := yasdbutil.NewYashanDB(log, y)
		res, err := yasdb.QueryMultiRows(sql, confdef.GetYHCConf().SqlTimeout)
		if err != nil {
			err = yaserr.Wrap(err)
			log.Error(err)
			data.Error = err.Error()
			continue
		}
		if isMulti {
			data.Details = c.convertMultiSqlData(metric, res)
		} else {
			if len(res) == 0 {
				err = yaserr.Wrap(err)
				log.Error(err)
				data.Error = err.Error()
				continue
			}
			data.Details = c.convertSqlData(metric, res[0])
		}
	}
	c.fillResults(datas...)
	return
}
