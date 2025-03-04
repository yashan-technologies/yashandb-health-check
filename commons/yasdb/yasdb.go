package yasdb

import (
	"os"
	"os/user"

	constdef "yhc/defs/constants"
	"yhc/defs/errdef"
	"yhc/utils/stringutil"
	"yhc/utils/userutil"

	"git.yasdb.com/go/yaslog"
	"git.yasdb.com/pandora/yasqlgo"
)

const (
	_GROUP_YASDBA = "YASDBA"
)

type NodeInfo struct {
	DatabaseName    string
	NodeID          string
	ListenAddr      string
	Role            string
	Connected       bool
	User            string
	Password        string
	Check           bool
	SystemCerticate bool
}

type YashanDB struct {
	IsUdsOpen     bool
	YasdbHome     string
	YasdbData     string
	YasdbUser     string
	YasdbPassword string
	ListenAddr    string
	DatabaseName  string
}

func (y *YashanDB) ValidHome() error {
	if stringutil.IsEmpty(y.YasdbHome) {
		return errdef.NewItemEmpty(constdef.YASDB_HOME)
	}
	if err := y.validatePath(y.YasdbHome); err != nil {
		return err
	}
	return nil
}

func (y *YashanDB) ValidData() error {
	if !stringutil.IsEmpty(y.YasdbData) {
		if err := y.validatePath(y.YasdbData); err != nil {
			return err
		}
	}
	return nil
}

func (y *YashanDB) ValidUser() error {
	if !y.IsUdsOpen && stringutil.IsEmpty(y.YasdbUser) {
		return errdef.NewItemEmpty(constdef.YASDB_USER)
	}
	return nil
}

func (y *YashanDB) ValidPassword() error {
	if !y.IsUdsOpen && stringutil.IsEmpty(y.YasdbPassword) {
		return errdef.NewItemEmpty(constdef.YASDB_PASSWORD)
	}
	return nil
}

func (y *YashanDB) ValidUserAndPwd(log yaslog.YasLog) error {
	y.IsUdsOpen = y.CheckIsUdsOpen()
	if err := y.ValidHome(); err != nil {
		return err
	}
	if err := y.ValidData(); err != nil {
		return err
	}
	if err := y.ValidUser(); err != nil {
		return err
	}
	if err := y.ValidPassword(); err != nil {
		return err
	}
	tx := yasqlgo.NewLocalInstance(y.YasdbUser, y.YasdbPassword, y.YasdbHome, y.YasdbData, log)
	tx.SystemCerticate = y.IsUdsOpen && (y.YasdbUser == "" || y.YasdbPassword == "")
	if err := tx.CheckPassword(); err != nil {
		return err
	}
	return nil
}

func (y *YashanDB) validatePath(path string) error {
	_, err := os.Stat(path)
	return err
}

func (y *YashanDB) CheckIsUdsOpen() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	gs := userutil.GetUserGroups(u)
	if len(gs) == 0 {
		return false
	}
	for _, g := range gs {
		if g == _GROUP_YASDBA {
			return true
		}
	}
	return false
}
