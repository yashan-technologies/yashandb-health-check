package check

import (
	"fmt"
	"time"

	"yhc/defs/bashdef"
	"yhc/defs/runtimedef"
	"yhc/defs/timedef"
	"yhc/i18n"
	"yhc/internal/modules/yhc/check/define"
	"yhc/log"
	"yhc/utils/execerutil"
	"yhc/utils/osutil"

	"git.yasdb.com/go/yaserr"
	"git.yasdb.com/go/yaslog"
	"github.com/shirou/gopsutil/host"
)

const (
	KEY_BOOT_TIME             = "bootTime"
	KEY_UP_TIME               = "uptime"
	KEY_HOST_ID               = "hostid"
	KEY_VIRTUALIZATION_SYSTEM = "virtualizationSystem"
	KEY_VIRTUALIZATION_ROLE   = "virtualizationRole"
	KEY_PLATFORM_FAMILY       = "platformFamily"
	KEY_PLATFORM_VERSION      = "platformVersion"
)

func (c *YHCChecker) GetHostInfo(name string) (err error) {
	data := &define.YHCItem{
		Name: define.METRIC_HOST_INFO,
	}
	defer c.fillResults(data)

	log := log.Module.M(string(define.METRIC_HOST_INFO))
	host, err := host.Info()
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	detail, err := c.convertObjectData(host)
	if err != nil {
		err = yaserr.Wrap(err)
		log.Error(err)
		data.Error = err.Error()
		return
	}
	data.Details = c.dealHostInfo(log, detail)
	return
}

func (c *YHCChecker) dealHostInfo(log yaslog.YasLog, res map[string]interface{}) map[string]interface{} {
	delete(res, KEY_VIRTUALIZATION_ROLE)
	delete(res, KEY_VIRTUALIZATION_SYSTEM)
	delete(res, KEY_HOST_ID)
	bootTime := res[KEY_BOOT_TIME].(float64)
	res[KEY_BOOT_TIME] = time.Unix(int64(bootTime), 0).Format(timedef.TIME_FORMAT)
	upTime := res[KEY_UP_TIME].(float64)
	formatDuration := func(duration time.Duration) string {
		hours := int(duration.Hours()) % 24
		minutes := int(duration.Minutes()) % 60
		seconds := int(duration.Seconds()) % 60
		days := int(duration.Hours()) / 24
		if days > 0 {
			return fmt.Sprintf(i18n.T("time.format_with_days"), days, hours, minutes, seconds)
		}
		return fmt.Sprintf(i18n.T("time.format_without_days"), hours, minutes, seconds)
	}
	res[KEY_UP_TIME] = formatDuration(time.Second * time.Duration(upTime))
	if runtimedef.GetOSRelease().Id == osutil.KYLIN_ID {
		delete(res, KEY_PLATFORM_FAMILY)
		platformVersion, err := c.getKyPlatformVersion(log)
		if err != nil {
			log.Error(err)
			delete(res, KEY_PLATFORM_VERSION)
			return res
		}
		res[KEY_PLATFORM_VERSION] = platformVersion
	}
	return res
}

func (c *YHCChecker) getKyPlatformVersion(log yaslog.YasLog) (string, error) {
	execer := execerutil.NewExecer(log)
	cmd := fmt.Sprintf("%s %s", bashdef.CMD_CAT, KY_PRODUCT_INFO)
	ret, stdout, stderr := execer.Exec(bashdef.CMD_BASH, "-c", cmd)
	if ret != 0 {
		err := fmt.Errorf("failed to get kylin platform info, err: %s", stderr)
		return "", err
	}
	return stdout, nil
}
