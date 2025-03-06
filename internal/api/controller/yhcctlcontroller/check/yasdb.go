package checkcontroller

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"yhc/commons/yasdb"
	"yhc/defs/bashdef"
	"yhc/defs/confdef"
	constdef "yhc/defs/constants"
	yhcyasdb "yhc/internal/modules/yhc/yasdb"
	"yhc/log"
	"yhc/utils/execerutil"
	"yhc/utils/fileutil"
	"yhc/utils/processutil"
	"yhc/utils/stringutil"
	"yhc/utils/userutil"
	"yhc/utils/yasdbutil"

	"git.yasdb.com/go/yaslog"
	"git.yasdb.com/pandora/yasqlgo"
)

var globalYasdb *YashanDB

const (
	HEADER_LISTEN_ADDRESS = "LISTEN_ADDRESS"
	HEADER_NODEID         = "NODEID"
	HEADER_DATABASE_ROLE  = "DATABASE_ROLE"

	SQL_QUERY_DATABASE_INFO = "SELECT DATABASE_NAME, DATABASE_ROLE FROM V$DATABASE"

	KEY_DATABASE_NAME = "DATABASE_NAME"
	KEY_DATABASE_ROLE = "DATABASE_ROLE"

	STATUS_NOT_CHECK = "NOT CHECK"
	STATUS_CHECKING  = "CHECKING"
	STATUS_CHECKED   = "CHECKED"
)

type YashanDB struct {
	*yasdb.YashanDB
	Nodes       []*yasdb.NodeInfo
	checkStatus string
	sync.Mutex
}

func (y *YashanDB) GetCheckStatus() string {
	y.Lock()
	defer y.Unlock()
	if y.checkStatus == "" {
		return STATUS_NOT_CHECK
	}
	return y.checkStatus
}

func (y *YashanDB) SetCheckStatus(status string) {
	y.Lock()
	defer y.Unlock()
	y.checkStatus = status
}

func fillListenAddrAndDBName(db *yasdb.YashanDB) error {
	log := log.Controller.M("fill listen addr and DB name")
	tx := yasqlgo.NewLocalInstance(db.YasdbUser, db.YasdbPassword, db.YasdbHome, db.YasdbData, log)
	db.IsUdsOpen = db.CheckIsUdsOpen()
	// 属于dba用户并且没有填写用户名和密码，使用操作系统认证
	tx.SystemCerticate = db.IsUdsOpen && (db.YasdbUser == "" || db.YasdbPassword == "")

	listenAddr, err := yhcyasdb.QueryParameter(tx, yhcyasdb.LISTEN_ADDR)
	if err != nil {
		return err
	}
	db.ListenAddr = trimSpace(listenAddr)
	driver := yasdbutil.NewYashanDB(log, db)
	res, err := driver.QueryMultiRows(SQL_QUERY_DATABASE_INFO, confdef.GetYHCConf().SqlTimeout)
	if err != nil {
		return err
	}
	if len(res) == 0 {
		return fmt.Errorf("failed to get database_name")
	}
	db.DatabaseName = trimSpace(res[0][KEY_DATABASE_NAME])
	return nil
}

func fillNodeInfos(db *YashanDB) {
	if db.GetCheckStatus() == STATUS_CHECKED {
		return
	}
	db.SetCheckStatus(STATUS_CHECKING)
	var nodeInfos []*yasdb.NodeInfo
	nodeInfos = append(nodeInfos, getNodeInfosFromYasboot(db.YashanDB)...)
	nodeInfos = append(nodeInfos, getNodeInfosFromConfig(db.YashanDB)...)
	db.Nodes = removeDuplicateNodes(nodeInfos)
	db.SetCheckStatus(STATUS_CHECKED)
}

func removeDuplicateNodes(nodes []*yasdb.NodeInfo) []*yasdb.NodeInfo {
	res := []*yasdb.NodeInfo{}
	nodeMap := make(map[string]struct{})
	for _, node := range nodes {
		listenAddr := strings.TrimSpace(node.ListenAddr)
		if _, ok := nodeMap[listenAddr]; ok {
			continue
		}
		nodeMap[listenAddr] = struct{}{}
		res = append(res, node)
	}
	return res
}

func getClusterStatusFromYasboot(db *yasdb.YashanDB) (string, error) {
	log := log.Controller.M("get node infos")
	if len(db.DatabaseName) == 0 {
		return "", errors.New("empty database name")
	}
	yasbootBin := path.Join(db.YasdbHome, "bin", bashdef.CMD_YASBOOT)
	cmd := yasbootBin
	args := []string{
		"cluster",
		"status",
		"-d",
		"-c",
		db.DatabaseName,
	}
	yasbootUser, err := fileutil.GetOwner(cmd)
	if err != nil {
		return "", err
	}
	if userutil.IsCurrentUserRoot() && yasbootUser.Uid != 0 {
		cmd = bashdef.CMD_SU
		yasbootArgs := strings.Join(args, " ")
		args = []string{
			"-c",
			fmt.Sprintf("%s %s", yasbootBin, yasbootArgs),
			yasbootUser.Username,
		}
	}
	execer := execerutil.NewExecer(log)
	ret, stdout, stderr := execer.Exec(cmd, args...)
	if ret != 0 {
		return "", errors.New(stderr)
	}
	return stdout, nil
}

func getNodeInfosFromYasboot(db *yasdb.YashanDB) []*yasdb.NodeInfo {
	log := log.Controller.M("get node infos")
	stdout, err := getClusterStatusFromYasboot(db)
	if err != nil {
		log.Errorf("failed to get cluster status from yasboot, err: %v", err)
		return nil
	}
	var nodeInfos []*yasdb.NodeInfo
	// 从yasboot的输出中解析出节点信息
	stdout = strings.ReplaceAll(stdout, "\r\n", "\n")
	stdout = strings.ReplaceAll(stdout, "\n\n", "\n")
	lines := strings.Split(stdout, "\n")
	dataLines := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 去除分隔线
		if len(line) == 0 || strings.HasPrefix(line, "-") || strings.HasPrefix(line, "+") {
			continue
		}
		dataLines = append(dataLines, line)
	}
	if len(dataLines) <= 1 {
		// 说明最多有一个标题，没有实际的集群各节点信息
		return nil
	}

	var wg sync.WaitGroup
	var lock sync.Mutex
	fn := func(wg *sync.WaitGroup, node *yasdb.NodeInfo) {
		defer wg.Done()
		// 检查非操作系统认证的节点的联通性
		_, _, node.Connected = getConnectedInfo(log, db.YasdbHome, node.User, node.Password, node.ListenAddr)
		node.Check = node.Connected
		lock.Lock()
		defer lock.Unlock()
		nodeInfos = append(nodeInfos, node)
	}

	headers := strings.Split(dataLines[0], "|")
	for _, line := range dataLines[1:] {
		node := &yasdb.NodeInfo{
			DatabaseName: db.DatabaseName,
			User:         db.YasdbUser,
			Password:     db.YasdbPassword,
		}
		datas := strings.Split(line, "|")
		for index, data := range datas {
			switch strings.ToUpper(strings.TrimSpace(headers[index])) {
			case HEADER_NODEID:
				node.NodeID = strings.ToUpper(strings.TrimSpace(data))
			case HEADER_LISTEN_ADDRESS:
				node.ListenAddr = strings.ToUpper(strings.TrimSpace(data))
			case HEADER_DATABASE_ROLE:
				node.Role = strings.ToUpper(strings.TrimSpace(data))
			}
		}
		if node.ListenAddr == db.ListenAddr {
			// 说明是通过此数据库节点的相关yasdb_home来访问其他节点
			// 此节点如果配置了操作系统认证，那么可以免密登录
			if db.CheckIsUdsOpen() && (node.User == "" && node.Password == "") {
				// 未输入密码并且允许操作系统免密
				node.Check = true
				node.Connected = true
				node.SystemCerticate = true
				nodeInfos = append(nodeInfos, node)
				continue
			}
		}
		if node.User != "" && node.Password != "" {
			// 如果填写了用户密码，则尝试用用户密码连接
			wg.Add(1)
			go fn(&wg, node)
		}
	}
	wg.Wait()
	return nodeInfos
}

// 从配置文件中
func getNodeInfosFromConfig(db *yasdb.YashanDB) []*yasdb.NodeInfo {
	log := log.Controller.M("get node infos")
	var nodeInfos []*yasdb.NodeInfo
	config := confdef.GetNodesConfig()
	var wg sync.WaitGroup
	var lock sync.Mutex
	fn := func(wg *sync.WaitGroup, node *yasdb.NodeInfo) {
		defer wg.Done()
		node.DatabaseName, node.Role, node.Connected = getConnectedInfo(log, db.YasdbHome, node.User, node.Password, node.ListenAddr)
		node.Check = node.Connected
		lock.Lock()
		defer lock.Unlock()
		nodeInfos = append(nodeInfos, node)
	}

	for _, configNode := range config.Nodes {
		if len(configNode.ListenAddr) == 0 {
			continue
		}
		user, password := configNode.User, configNode.Password
		if len(user) == 0 || len(password) == 0 {
			user, password = db.YasdbUser, db.YasdbPassword
		}
		node := &yasdb.NodeInfo{
			ListenAddr: configNode.ListenAddr,
			User:       user,
			Password:   password,
		}
		wg.Add(1)
		go fn(&wg, node)
	}
	wg.Wait()
	return nodeInfos
}

func getConnectedInfo(log yaslog.YasLog, yasdbHome, user, password, listenAddr string) (string, string, bool) {
	otherNode := yasdb.YashanDB{
		YasdbUser:     user,
		YasdbPassword: password,
		ListenAddr:    listenAddr,
		YasdbHome:     yasdbHome,
	}
	var databaseName, role string
	var connected bool
	otherDirver := yasdbutil.NewYashanDB(log, &otherNode)
	res, err := otherDirver.QueryMultiRows(SQL_QUERY_DATABASE_INFO, confdef.GetYHCConf().SqlTimeout)
	if err != nil || len(res) == 0 {
		connected = false
	} else {
		databaseName = res[0][KEY_DATABASE_NAME]
		role = res[0][KEY_DATABASE_ROLE]
		connected = true
	}
	return databaseName, role, connected
}

func getYasdbPath() (yasdbHome, yasdbData string) {
	yasdbData = os.Getenv(constdef.YASDB_DATA)
	yasdbHome = os.Getenv(constdef.YASDB_HOME)
	processYasdbHome, processYasdbData := getYasdbPathFromProcess()
	if stringutil.IsEmpty(yasdbHome) {
		yasdbHome = processYasdbHome
	}
	if stringutil.IsEmpty(yasdbData) {
		yasdbData = processYasdbData
	}
	return
}

func getYasdbPathFromProcess() (yasdbHome, yasdbData string) {
	log := log.Controller.M("get yasdb process from cmdline")
	processes, err := processutil.ListAnyUserProcessByCmdline(_base_yasdb_process_format, true)
	if err != nil {
		log.Errorf("get process err: %s", err.Error())
		return
	}
	if len(processes) == 0 {
		log.Infof("process result is empty")
		return
	}
	for _, p := range processes {
		fields := strings.Split(p.ReadableCmdline, "-D")
		if len(fields) < 2 {
			log.Infof("process cmdline: %s format err, skip", p.ReadableCmdline)
			continue
		}
		yasdbData = trimSpace(fields[1])
		full := trimSpace(p.FullCommand)
		if !path.IsAbs(full) {
			return
		}
		yasdbHome = path.Dir(path.Dir(full))
		return
	}
	return
}

func newYasdb() *yasdb.YashanDB {
	home, data := getYasdbPath()
	yasdb := &yasdb.YashanDB{
		YasdbData: data,
		YasdbHome: home,
	}
	return yasdb
}
