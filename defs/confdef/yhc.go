package confdef

import (
	"regexp"
	"strings"
	"time"

	"yhc/utils/stringutil"
	"yhc/utils/timeutil"
)

var _yhcConf YHC

type YHC struct {
	LogLevel               string   `toml:"log_level"`
	Language               string   `toml:"language"`
	Range                  string   `toml:"range"`
	MaxDuration            string   `toml:"max_duration"`
	MinDuration            string   `toml:"min_duration"`
	SqlTimeout             int      `toml:"sql_timeout"`
	SarDir                 string   `toml:"sar_dir"`
	ScrapeInterval         int      `toml:"scrape_interval"`
	ScrapeTimes            int      `toml:"scrape_times"`
	Output                 string   `toml:"output"`
	MetricPaths            []string `toml:"metric_paths"`
	DefaultModulePath      string   `toml:"default_module_path"`
	EvaluateModelPath      string   `toml:"evaluate_model_path"`
	AfterInstallMetricPath []string `toml:"after_install_metric_path"`
	AfterInstallModulePath string   `toml:"after_install_module_path"`
	NodesConfigPath        string   `toml:"nodes_config_path"`
	NetworkIODiscard       string   `toml:"network_io_discard"`
	SkipGenWordReport      bool     `toml:"skip_gen_word_report"`
	SkipGenHtmlReport      bool     `toml:"skip_gen_html_report"`
}

func GetYHCConf() YHC {
	return _yhcConf
}

func (c YHC) GetMaxDuration() (time.Duration, error) {
	if len(c.MaxDuration) == 0 {
		return time.Hour * 24, nil
	}
	maxDuration, err := timeutil.GetDuration(c.MaxDuration)
	if err != nil {
		return 0, err
	}
	return maxDuration, err
}

func (c YHC) GetMinDuration() (time.Duration, error) {
	if len(c.MinDuration) == 0 {
		return time.Minute * 1, nil
	}
	minDuration, err := timeutil.GetDuration(c.MinDuration)
	if err != nil {
		return 0, err
	}
	return minDuration, err
}

func (c YHC) GetMinAndMaxDuration() (min time.Duration, max time.Duration, err error) {
	min, err = c.GetMinDuration()
	if err != nil {
		return
	}
	max, err = c.GetMaxDuration()
	if err != nil {
		return
	}
	return
}

func (c YHC) GetRange() (r time.Duration) {
	r, err := timeutil.GetDuration(c.Range)
	if err != nil {
		return time.Hour * 24
	}
	return
}

func (c YHC) GetSqlTimeout() (t int) {
	return c.SqlTimeout
}

func (c YHC) GetSarDir() (dir string) {
	return c.SarDir
}

func (c YHC) GetScrapeInterval() (interval int) {
	return c.ScrapeInterval
}

func (c YHC) GetScrapeTimes() (times int) {
	return c.ScrapeTimes
}

func (c YHC) GetNetworkIODiscard() []string {
	return strings.Split(c.NetworkIODiscard, stringutil.STR_COMMA)
}

func IsDiscardNetwork(name string) bool {
	discards := GetYHCConf().GetNetworkIODiscard()
	for _, discard := range discards {
		re, err := regexp.Compile(discard)
		if err != nil {
			continue
		}
		if re.MatchString(name) {
			return true
		}
	}
	return false
}
