package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"strings"
	"time"
)

// ParseFile reads the file named by filename and returns the parsed config.
// A successful call returns err == nil.
func ParseFile(configPath string) (*Config, error) {
	c, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	return Parse(string(c))
}

// Parse reads the config text and returns the parsed config.
// A successful call returns err == nil.
func Parse(configData string) (*Config, error) {
	var internal config
	if _, err := toml.Decode(configData, &internal); err != nil {
		return nil, err
	}
	var help strings.Builder

	defaultTimeout, err := time.ParseDuration("30s")
	if err != nil {
		return nil, err
	}
	if internal.Settings.Timeout != "" {
		defaultTimeout, err = time.ParseDuration(internal.Settings.Timeout)
		if err != nil {
			return nil, err
		}
	}

	defaultMaxSymbolsPerMessage := internal.Settings.MaxSymbolsPerMessage
	defaultMaxMessages := internal.Settings.MaxMessages

	var external Config
	external.Settings = &Settings{
		Token:               internal.Settings.Token,
		ArgumentsTrimCutSet: internal.Settings.ArgumentsTrimCutSet,
	}

	external.Settings.Channels = make(map[string]struct{})
	for _, channel := range internal.Settings.Channels {
		external.Settings.Channels[channel] = struct{}{}
	}

	external.Settings.Users = make(map[string]struct{})
	for _, user := range internal.Settings.Users {
		external.Settings.Users[user] = struct{}{}
	}

	external.Settings.Admins = make(map[string]struct{})
	for _, admin := range internal.Settings.Admins {
		external.Settings.Admins[admin] = struct{}{}
	}

	external.Hosts = make(map[string]*Host)
	for _, h := range internal.Hosts {
		auth := h.Auth
		external.Hosts[h.Id] = &Host{
			Id:      h.Id,
			Address: h.Address,
			Port:    h.Port,
			Auth: &Auth{
				Type:           auth.Type,
				Username:       auth.Username,
				Password:       auth.Password,
				PrivateKeyPath: auth.PrivateKeyPath,
				Passphrase:     auth.Passphrase,
			},
		}
	}

	help.WriteString(fmt.Sprintf("%s_*Groups:*_\n", internal.Settings.Description))
	external.Groups = make(map[string]*Group)
	for _, g := range internal.Groups {
		help.WriteString(fmt.Sprintf("\n`%s`   _%s_", g.Id, g.Description))
		group := &Group{
			Id: g.Id,
		}
		external.Groups[group.Id] = group
		var groupHelp strings.Builder

		groupHelp.WriteString(fmt.Sprintf("%s\n_*Hosts:*_", internal.Settings.Description))
		group.Hosts = make(map[string]*Host)
		for _, gh := range g.Hosts {
			group.Hosts[gh] = external.Hosts[gh]
			groupHelp.WriteString(fmt.Sprintf(" `%s`", gh))
		}

		groupArgs := make(map[string]*Argument)
		for _, a := range g.Arguments {
			items := make([]*Item, len(a.Items))
			var itemsHelp strings.Builder
			itemsHelp.WriteString(fmt.Sprintf("`%s`   _%s:_", a.Id, a.Description))
			for i, item := range a.Items {
				items[i] = &Item{
					Name:  item.Name,
					Value: item.Value,
				}
				itemsHelp.WriteString(fmt.Sprintf(" `%s`", item.Name))
			}
			groupArgs[a.Id] = &Argument{
				Id:    a.Id,
				Help:  itemsHelp.String(),
				Items: items,
			}
		}

		groupHelp.WriteString("\n_*Commands:*_")
		group.Commands = make(map[string]*Command)
		for _, c := range g.Commands {
			groupHelp.WriteString(fmt.Sprintf("\n`%s`   _%s_", c.Id, c.Description))

			var commandHelp strings.Builder
			var argsHelp strings.Builder
			commandHelp.WriteString(fmt.Sprintf("_%s_\n_*Format:*_\n```%s [host] %s", c.Description, group.Id, c.Id))
			args := make([]*Argument, len(c.Arguments))
			for i, a := range c.Arguments {
				args[i] = groupArgs[a]
				commandHelp.WriteString(fmt.Sprintf(" <%s>", args[i].Id))
				argsHelp.WriteString(fmt.Sprintf("\n%s", args[i].Help))
			}
			commandHelp.WriteString(fmt.Sprintf("```%s", argsHelp.String()))

			timeout := defaultTimeout
			if c.Timeout != "" {
				timeout, err = time.ParseDuration(c.Timeout)
				if err != nil {
					return nil, err
				}
			}

			maxSymbolsPerMessage := defaultMaxSymbolsPerMessage
			if c.MaxSymbolsPerMessage != 0 {
				maxSymbolsPerMessage = c.MaxSymbolsPerMessage
			}
			maxMessages := defaultMaxMessages
			if c.MaxMessages != 0 {
				maxMessages = c.MaxMessages
			}

			group.Commands[c.Id] = &Command{
				Id:                   c.Id,
				Help:                 commandHelp.String(),
				Format:               c.Format,
				Arguments:            args,
				MaxSymbolsPerMessage: maxSymbolsPerMessage,
				MaxMessages:          maxMessages,
				Timeout:              timeout,
			}
		}
		group.Help = groupHelp.String()
	}
	external.Help = help.String()
	return &external, nil
}

// Config is the main config type
type Config struct {
	Settings *Settings
	Hosts    map[string]*Host
	Groups   map[string]*Group
	Help     string
}

// Settings is the config's part with slack token, users, channels and etc.
type Settings struct {
	Token               string
	Channels            map[string]struct{}
	Users               map[string]struct{}
	Admins              map[string]struct{}
	ArgumentsTrimCutSet string
}

// Group is the group with commands
type Group struct {
	Id       string
	Help     string
	Hosts    map[string]*Host
	Commands map[string]*Command
}

// Host is the host address, port, authentication and etc.
type Host struct {
	Id      string
	Address string
	Port    int
	Auth    *Auth
}

// Auth is the authentication information
type Auth struct {
	Type           string
	Username       string
	Password       string
	PrivateKeyPath string
	Passphrase     string
}

// Command is the command attributes and arguments
type Command struct {
	Id                   string
	Help                 string
	Format               string
	Arguments            []*Argument
	Timeout              time.Duration
	MaxSymbolsPerMessage int
	MaxMessages          int
}

// Argument is the command argument
type Argument struct {
	Id    string
	Help  string
	Items []*Item
}

// Item is the argument item information
type Item struct {
	Name  string
	Value string
}

type config struct {
	Settings settings `toml:"settings"`
	Hosts    []host   `toml:"host"`
	Groups   []group  `toml:"group"`
}

type settings struct {
	Token                string   `toml:"token"`
	Description          string   `toml:"description"`
	MaxSymbolsPerMessage int      `toml:"maxSymbolsPerMessage"`
	MaxMessages          int      `toml:"maxMessages"`
	Timeout              string   `toml:"timeout"`
	Channels             []string `toml:"channels"`
	Users                []string `toml:"users"`
	Admins               []string `toml:"admins"`
	ArgumentsTrimCutSet  string   `toml:"argumentsTrimCutSet"`
}

type group struct {
	Id          string     `toml:"id"`
	Description string     `toml:"description"`
	Hosts       []string   `toml:"hosts"`
	Commands    []command  `toml:"command"`
	Arguments   []argument `toml:"argument"`
}

type command struct {
	Id                   string   `toml:"id"`
	Description          string   `toml:"description"`
	Format               string   `toml:"cmdFmt"`
	Arguments            []string `toml:"arguments"`
	MaxSymbolsPerMessage int      `toml:"maxSymbolsPerMessage"`
	MaxMessages          int      `toml:"maxMessages"`
	Timeout              string   `toml:"timeout"`
}

type host struct {
	Id      string `toml:"id"`
	Address string `toml:"address"`
	Port    int    `toml:"port"`
	Auth    auth   `toml:"auth"`
}

type auth struct {
	Type           string `toml:"type"`
	Username       string `toml:"username"`
	Password       string `toml:"password"`
	PrivateKeyPath string `toml:"privateKeyPath"`
	Passphrase     string `toml:"passphrase"`
}

type argument struct {
	Id          string `toml:"id"`
	Description string `toml:"description"`
	Items       []item `toml:"item"`
}

type item struct {
	Name  string `toml:"name"`
	Value string `toml:"value"`
}
