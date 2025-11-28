// The flags package defines some common flag structures and methods.
package flags

import (
	"github.com/alecthomas/kong"
)

const (
	_VERSION_KEY = "version"
)

type Globals struct {
	// normal flags
	Version kong.VersionFlag `short:"v" help:"Show version."`
	Config  string           `name:"config" short:"c" default:"./config/yhc.toml" help:"Configuration file."`
	Lang    string           `name:"lang" short:"l" help:"Report language (zh/en, zh-CN/en-US)."`
	// hidden flags
	Show showFlag `name:"show" hidden:"true" help:"[Hidden] Show software compilation information."`
}

// NewVersionOption returns kong.Option for version.
func NewVersionOption(version string) kong.Option {
	return kong.Vars{
		_VERSION_KEY: version,
	}
}

// NewAppOptions returns common app kong options.
func NewAppOptions(name, description, version string) []kong.Option {
	return []kong.Option{
		kong.Name(name),
		kong.Description(description),
		NewVersionOption(version),
		kong.ShortUsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Tree: true,
		}),
	}
}
