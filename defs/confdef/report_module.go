package confdef

import (
	"path"
	"strings"

	"yhc/defs/errdef"
	"yhc/defs/runtimedef"
	"yhc/i18n"
	"yhc/utils/stringutil"

	"git.yasdb.com/go/yasutil/fs"
	"github.com/BurntSushi/toml"
)

var _moduleConfig *YHCModuleConfig

type YHCModuleConfig struct {
	Modules            []*YHCModuleNode `toml:"modules"`
	moduleAliasMap     map[string]string
	moduleAliasEnMap   map[string]string
	metricModulesMap   map[string][]string
	metricOrder        []string
	moduleMetricsMap   map[string][]string
}

type YHCModuleNode struct {
	Name        string           `toml:"name"`
	NameAlias   string           `toml:"name_alias"`
	NameAliasEn string           `toml:"name_alias_en"`
	Children    []*YHCModuleNode `toml:"children"`
	MetricNames []string         `toml:"metric_names"`
}

func InitModuleConf(p string) error {
	if !path.IsAbs(p) {
		p = path.Join(runtimedef.GetYHCHome(), p)
	}
	conf, err := loadModuleConf(p)
	if err != nil {
		return err
	}
	conf.moduleAliasMap, conf.moduleAliasEnMap = genModuleAliasMaps(conf.Modules)
	conf.metricModulesMap = genMetricModulesMap(conf.Modules)
	conf.metricOrder = genMetricOrder(conf.Modules)
	conf.moduleMetricsMap = genModuleMetricsMap(conf.Modules)
	_moduleConfig = conf
	return nil
}

func genModuleAliasMaps(modules []*YHCModuleNode) (map[string]string, map[string]string) {
	resZh := make(map[string]string)
	resEn := make(map[string]string)

	var fn func(resZh, resEn map[string]string, node *YHCModuleNode)
	fn = func(resZh, resEn map[string]string, node *YHCModuleNode) {
		if node == nil {
			return
		}
		if stringutil.IsEmpty(node.NameAlias) {
			node.NameAlias = node.Name
		}
		if stringutil.IsEmpty(node.NameAliasEn) {
			node.NameAliasEn = node.Name
		}
		resZh[node.Name] = node.NameAlias
		resEn[node.Name] = node.NameAliasEn
		for _, child := range node.Children {
			fn(resZh, resEn, child)
		}
	}

	for _, module := range modules {
		fn(resZh, resEn, module)
	}
	return resZh, resEn
}

func genMetricModulesMap(nodes []*YHCModuleNode) map[string][]string {
	var fn func(node *YHCModuleNode, path []string, index map[string][]string)
	fn = func(node *YHCModuleNode, path []string, index map[string][]string) {
		path = append(path, node.Name)
		for _, metricName := range node.MetricNames {
			index[metricName] = append([]string{}, path...)
		}
		for _, child := range node.Children {
			fn(child, path, index)
		}
	}

	index := make(map[string][]string)
	for _, node := range nodes {
		fn(node, []string{}, index)
	}
	return index
}

func genModuleMetricsMap(modules []*YHCModuleNode) map[string][]string {
	moduleMetricMap := make(map[string][]string)
	var fn func(module *YHCModuleNode, moduleMetricMap map[string][]string)
	fn = func(module *YHCModuleNode, moduleMetricMap map[string][]string) {
		moduleMetricMap[module.Name] = module.MetricNames
		for _, child := range module.Children {
			fn(child, moduleMetricMap)
		}
	}
	for _, module := range modules {
		fn(module, moduleMetricMap)
	}
	return moduleMetricMap
}

func genMetricOrder(modules []*YHCModuleNode) []string {
	var result []string

	var fn func(node *YHCModuleNode, result *[]string)
	fn = func(node *YHCModuleNode, result *[]string) {
		*result = append(*result, node.MetricNames...)
		for _, child := range node.Children {
			fn(child, result)
		}
	}

	for _, module := range modules {
		fn(module, &result)
	}
	return result
}

func loadModuleConf(p string) (*YHCModuleConfig, error) {
	conf := &YHCModuleConfig{}
	if !fs.IsFileExist(p) {
		return conf, &errdef.ErrFileNotFound{FName: p}
	}
	if _, err := toml.DecodeFile(p, conf); err != nil {
		return conf, &errdef.ErrFileParseFailed{FName: p, Err: err}
	}
	return conf, nil
}

func GetModuleConf() *YHCModuleConfig {
	return _moduleConfig
}

func GetModuleAliasMap() map[string]string {
	return _moduleConfig.moduleAliasMap
}

func GetModuleAlias(name string) string {
	name = strings.TrimSpace(name)
	// 优先使用i18n翻译
	i18nKey := "module." + name
	translated := i18n.T(i18nKey)
	if translated != i18nKey {
		return translated
	}
	// 回退到配置文件，根据当前语言选择
	if i18n.GetLanguage() == i18n.EnUS {
		return _moduleConfig.moduleAliasEnMap[name]
	}
	return _moduleConfig.moduleAliasMap[name]
}

func GetMetricModules(metricName string) []string {
	return _moduleConfig.metricModulesMap[metricName]
}

func GetMetricOrder() []string {
	return _moduleConfig.metricOrder
}

func GetModuleMetrics() map[string][]string {
	return _moduleConfig.moduleMetricsMap
}
