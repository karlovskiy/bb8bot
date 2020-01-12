package config

import (
	"reflect"
	"testing"
	"time"
)

var testConfig = `
# bb8bot configuration file

# Bot settings
[settings]
    description = "test bot description"
    token = "xoxb-36484"
    maxSymbols = 3000
    timeout = "30s"

# Commands
[[group]]
    id = "group1"
    description = "Group1 commands"
    hosts = ["onehost", "anotherhost"]

    [[group.command]]
        id="no-args-cmd"
        description = "No args cmd"
        cmdFmt="cmd-no-args"
	
    [[group.command]]
        id="command-with-custom-timeout"
        description = "Command with custom timeout"
        cmdFmt="command-with-custom-timeout"
        timeout = "2m"

    [[group.command]]
        id = "with-args-cmd"
        description = "With args cmd"
        cmdFmt = "cmd-with-args %s"
        arguments = ["argument"]

    [[group.argument]]
        id = "argument"
        description = "Argument for command"

        [[group.argument.item]]
            name = "name"
            value = "value"

[[group]]
    id = "group2"
    description = "Group 2 commands"
    hosts = ["onehost"]

	[[group.command]]
		id="test-cmd"
		description = "Test command"
		cmdFmt="test"	

# Hosts
[[host]]
    id = "onehost"
    address = "onehost"
    port = 22

    [host.auth]
        type = "password"
        username = "shmee"
        password = "gayjke"


[[host]]
    id = "anotherhost"
    address = "anotherhost"
    port = 22

    [host.auth]
        type = "publickey"
		username = "root"
        privateKeyPath = "~/.ssh/your_private_key"
        passphrase = "your_passphrase"
`

func TestParse(t *testing.T) {
	actual, err := Parse(testConfig)
	if err != nil {
		t.Fatal(err)
	}
	if actual == nil {
		t.Fatal(err)
	}

	defaultTimeout, err := time.ParseDuration("30s")
	if err != nil {
		panic(err)
	}

	customTimeout, err := time.ParseDuration("2m")
	if err != nil {
		panic(err)
	}

	hosts := make(map[string]*Host)
	hosts["onehost"] = &Host{
		Id:      "onehost",
		Address: "onehost",
		Port:    22,
		Auth: &Auth{
			Type:           "password",
			Username:       "shmee",
			Password:       "gayjke",
			PrivateKeyPath: "",
			Passphrase:     "",
		},
	}
	hosts["anotherhost"] = &Host{
		Id:      "anotherhost",
		Address: "anotherhost",
		Port:    22,
		Auth: &Auth{
			Type:           "publickey",
			Username:       "root",
			Password:       "",
			PrivateKeyPath: "~/.ssh/your_private_key",
			Passphrase:     "your_passphrase",
		},
	}

	var items []*Item
	items = append(items, &Item{
		Name:  "name",
		Value: "value",
	})

	var arguments []*Argument
	arguments = append(arguments, &Argument{
		Id:    "argument",
		Help:  "`argument`   _Argument for command:_ `name`",
		Items: items,
	})

	group1Commands := make(map[string]*Command)
	group1Commands["no-args-cmd"] = &Command{
		Id:        "no-args-cmd",
		Help:      "_No args cmd_\n_*Format:*_\n```group1 [host] no-args-cmd```",
		Format:    "cmd-no-args",
		Arguments: []*Argument{},
		Timeout:   defaultTimeout,
	}
	group1Commands["command-with-custom-timeout"] = &Command{
		Id:        "command-with-custom-timeout",
		Help:      "_Command with custom timeout_\n_*Format:*_\n```group1 [host] command-with-custom-timeout```",
		Format:    "command-with-custom-timeout",
		Arguments: []*Argument{},
		Timeout:   customTimeout,
	}
	group1Commands["with-args-cmd"] = &Command{
		Id:        "with-args-cmd",
		Help:      "_With args cmd_\n_*Format:*_\n```group1 [host] with-args-cmd <argument>```\n`argument`   _Argument for command:_ `name`",
		Format:    "cmd-with-args %s",
		Arguments: arguments,
		Timeout:   defaultTimeout,
	}

	group2Commands := make(map[string]*Command)
	group2Commands["test-cmd"] = &Command{
		Id:        "test-cmd",
		Help:      "_Test command_\n_*Format:*_\n```group2 [host] test-cmd```",
		Format:    "test",
		Arguments: []*Argument{},
		Timeout:   defaultTimeout,
	}

	group1Hosts := make(map[string]*Host)
	group1Hosts["onehost"] = hosts["onehost"]
	group1Hosts["anotherhost"] = hosts["anotherhost"]

	group2Hosts := make(map[string]*Host)
	group2Hosts["onehost"] = hosts["onehost"]

	groups := make(map[string]*Group)
	groups["group1"] = &Group{
		Id:       "group1",
		Help:     "test bot description\n_*Hosts:*_ `onehost` `anotherhost`\n_*Commands:*_\n`no-args-cmd`   _No args cmd_\n`command-with-custom-timeout`   _Command with custom timeout_\n`with-args-cmd`   _With args cmd_",
		Hosts:    group1Hosts,
		Commands: group1Commands,
	}
	groups["group2"] = &Group{
		Id:       "group2",
		Help:     "test bot description\n_*Hosts:*_ `onehost`\n_*Commands:*_\n`test-cmd`   _Test command_",
		Hosts:    group2Hosts,
		Commands: group2Commands,
	}

	expected := &Config{
		Settings: &Settings{
			Token:      "xoxb-36484",
			MaxSymbols: 3000,
		},
		Hosts:  hosts,
		Groups: groups,
		Help:   "test bot description_*Groups:*_\n\n`group1`   _Group1 commands_\n`group2`   _Group 2 commands_",
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("Expected\n-----\n%#v\n-----\nbut got\n-----\n%#v\n",
			expected, actual)
	}
}
