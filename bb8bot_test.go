package main

import (
	"errors"
	"github.com/karlovskiy/bb8bot/config"
	"reflect"
	"testing"
	"time"
)

func TestParseAction(t *testing.T) {

	tests := []struct {
		action string
		cmd    string
		host   string
		err    error
	}{
		{
			"",
			"",
			"",
			errors.New("bot help"),
		},
		{
			"help",
			"",
			"",
			errors.New("bot help"),
		},
		{
			"group",
			"",
			"",
			errors.New("group *group* not found\nbot help"),
		},
		{
			"group",
			"",
			"",
			errors.New("group *group* not found\nbot help"),
		},
		{
			"group1",
			"",
			"",
			errors.New("group help"),
		},
		{
			"group1 command1 help",
			"",
			"",
			errors.New("command1 help"),
		},
		{
			"group1 dfdfdf",
			"",
			"",
			errors.New("command *dfdfdf* not found\ngroup help"),
		},
		{
			"group1 command1 help",
			"",
			"",
			errors.New("command1 help"),
		},
		{
			"group1 onehost command1 help",
			"",
			"",
			errors.New("command1 help"),
		},
		{
			"group1 command1",
			"raw command1",
			"onehost",
			nil,
		},
		{
			"group1 command1",
			"raw command1",
			"onehost",
			nil,
		},
		{
			"group1 command2",
			"",
			"",
			errors.New("*1* argument not found\ncommand2 help"),
		},
		{
			"group1 command2 sdsdsd",
			"",
			"",
			errors.New("argument value *sdsdsd* not found\ncommand2 help"),
		},
		{
			"group1 command2 arg-name",
			"raw command2 arg-value",
			"onehost",
			nil,
		},
	}

	conf := makeTestConfig()

	for i, test := range tests {
		cmd, host, err := parseAction(test.action, conf)
		if !reflect.DeepEqual(err, test.err) {
			t.Errorf("%d: Got err: %v, want: %v", i, err, test.err)
		}
		var c string
		if cmd != nil {
			c = *cmd
		}
		if !reflect.DeepEqual(c, test.cmd) {
			t.Errorf("%d: Got cmd: %q, want: %q", i, c, test.cmd)
		}
		var h string
		if host != nil {
			h = host.Id
		}
		if !reflect.DeepEqual(h, test.host) {
			t.Errorf("%d: Got host: %q, want: %q", i, h, test.host)
		}

	}

}

func makeTestConfig() *config.Config {

	hosts := make(map[string]*config.Host)
	hosts["onehost"] = &config.Host{
		Id:      "onehost",
		Address: "onehost",
		Port:    22,
		Auth: &config.Auth{
			Type:           "password",
			Username:       "shmee",
			Password:       "gayjke",
			PrivateKeyPath: "",
			Passphrase:     "",
		},
	}

	var items []*config.Item
	items = append(items, &config.Item{
		Name:  "arg-name",
		Value: "arg-value",
	})

	var arguments []*config.Argument
	arguments = append(arguments, &config.Argument{
		Id:    "argument",
		Help:  "argument help",
		Items: items,
	})

	commands := make(map[string]*config.Command)
	commands["command1"] = &config.Command{
		Id:        "command1",
		Help:      "command1 help",
		Format:    "raw command1",
		Arguments: []*config.Argument{},
	}
	commands["command2"] = &config.Command{
		Id:        "command2",
		Help:      "command2 help",
		Format:    "raw command2 %s",
		Arguments: arguments,
	}

	groups := make(map[string]*config.Group)
	groups["group1"] = &config.Group{
		Id:       "group1",
		Help:     "group help",
		Hosts:    hosts,
		Commands: commands,
	}

	timeout, err := time.ParseDuration("30s")
	if err != nil {
		panic(err)
	}

	conf := &config.Config{
		Settings: &config.Settings{
			Token:      "xoxb-36484",
			MaxSymbols: 3000,
			Timeout:    timeout,
		},
		Hosts:  hosts,
		Groups: groups,
		Help:   "bot help",
	}

	return conf
}
