package config

import (
	"flag"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

var (
	Conf Setting
)

type Setting struct {
	User           string   `json:"user" yaml:"user"`
	Pass           string   `json:"pass" yaml:"pass"`
	Https          bool     `json:"https" yaml:"https"`
	Insecure       bool     `json:"insecure" yaml:"insecure"`
	OpTimeout      string   `json:"op_timeout" yaml:"op_timeout"`
	MaxOpsPerShell int      `json:"max_ops_per_shell" yaml:"max_ops_per_shell"`
	Before         []string `json:"before" yaml:"before"`
	After          []string `json:"after" yaml:"after"`
	Tasks          []Copy   `json:"tasks" yaml:"tasks"`
	Servers        []Server `json:"servers" yaml:"servers"`
}

type Copy struct {
	Src string `json:"src" yaml:"src"`
	Dst string `json:"dst" yaml:"dst"`
}

type Server struct {
	Addr           string   `json:"addr" yaml:"addr"`
	User           string   `json:"user" yaml:"user"`
	Pass           string   `json:"pass" yaml:"pass"`
	Https          bool     `json:"https" yaml:"https"`
	Insecure       bool     `json:"insecure" yaml:"insecure"`
	OpTimeout      string   `json:"op_timeout" yaml:"op_timeout"`
	MaxOpsPerShell int      `json:"max_ops_per_shell" yaml:"max_ops_per_shell"`
	Before         []string `json:"before" yaml:"before"`
	After          []string `json:"after" yaml:"after"`
	Tasks          []Copy   `json:"tasks" yaml:"tasks"`
}

func Parse() {
	var (
		location string
	)
	flag.StringVar(&location, "config", "", "set config location")
	flag.Parse()
	b, err := ioutil.ReadFile(location)
	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(b, &Conf)
	if err != nil {
		panic(err)
	}

	for i := range Conf.Servers {
		server := &Conf.Servers[i]
		if server.User == "" {
			server.User = Conf.User
		}

		if server.Pass == "" {
			server.Pass = Conf.Pass
		}

		if !server.Https {
			server.Https = Conf.Https
		}

		if !server.Insecure {
			server.Insecure = Conf.Insecure
		}

		if server.OpTimeout == "" {
			server.OpTimeout = Conf.OpTimeout
		}

		if server.MaxOpsPerShell == 0 {
			server.MaxOpsPerShell = Conf.MaxOpsPerShell
		}

		if server.Before == nil {
			server.Before = Conf.Before
		}
		if server.Tasks == nil {
			server.Tasks = Conf.Tasks
		}
		if server.After == nil {
			server.After = Conf.After
		}
	}
	log.Printf("Config is %+v\n", Conf)
}
