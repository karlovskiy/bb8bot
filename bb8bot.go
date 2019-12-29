package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/karlovskiy/bb8bot/config"
	"github.com/nlopes/slack"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	configPath = flag.String("c", "", "config path")
)

func main() {

	flag.Parse()

	conf, err := config.ParseFile(*configPath)
	if err != nil {
		log.Fatalf("Error parsing '%s': %v", *configPath, err)
	}

	api := slack.New(
		conf.Settings.Token,
		slack.OptionDebug(true),
		slack.OptionLog(log.New(os.Stdout, "", log.Lshortfile|log.LstdFlags)),
	)

	rtm := api.NewRTM()
	go rtm.ManageConnection()
	handleIncomingEvents(rtm, conf)

}

// handle all incoming RTM events
func handleIncomingEvents(rtm *slack.RTM, conf *config.Config) {
	for msg := range rtm.IncomingEvents {

		switch ev := msg.Data.(type) {

		case *slack.MessageEvent:
			fmt.Printf("Message: %v\n", ev)
			info := rtm.GetInfo()
			prefix := "<@" + info.User.ID + ">"
			text := ev.Msg.Text
			action := strings.TrimPrefix(text, prefix)
			if action != text {
				action = strings.TrimSpace(action)
				cmd, host, err := parseAction(action, conf)
				if err != nil {
					rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("%v", err), ev.Msg.Channel))
				} else {
					output, err := execute(conf, cmd, host)
					if err != nil {
						rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("error execution action: %v", err), ev.Msg.Channel))
					}
					rtm.SendMessage(rtm.NewOutgoingMessage(output, ev.Msg.Channel))
				}
			}
		default:
			// Ignore other events..
		}
	}
}

// parse action from chat message and convert it to ssh command for execution
func parseAction(action string, conf *config.Config) (cmd *string, host *config.Host, err error) {
	if action == "" || action == "help" {
		return nil, nil, errors.New(conf.Help)
	}

	actionParts := strings.Fields(action)
	searchGroup := actionParts[0]
	group, exist := conf.Groups[searchGroup]
	if !exist {
		return nil, nil, errors.New(fmt.Sprintf("group *%s* not found\n%s", searchGroup, conf.Help))
	}
	if len(actionParts) < 2 {
		return nil, nil, errors.New(group.Help)
	}

	searchHostOrCommand := actionParts[1]
	if len(actionParts) == 3 && actionParts[2] == "help" {
		helpCommand, exist := group.Commands[searchHostOrCommand]
		if exist {
			return nil, nil, errors.New(helpCommand.Help)
		}
	}

	if len(group.Hosts) == 0 {
		return nil, nil, errors.New(fmt.Sprintf("hosts for group *%s* not found\n%s", group.Id, group.Help))
	} else if len(group.Hosts) == 1 {
		for _, h := range group.Hosts {
			host = h
		}
	} else {
		host, exist = conf.Hosts[searchHostOrCommand]
		if !exist {
			return nil, nil, errors.New(fmt.Sprintf("host *%s* not found,\n%s", searchHostOrCommand, group.Help))
		}
	}
	if host == nil {
		return nil, nil, errors.New(fmt.Sprintf("host or command *%s* not found\n%s", searchHostOrCommand, conf.Help))
	}
	searchCommand := searchHostOrCommand
	cmdIndex := 1
	if host.Id == searchHostOrCommand {
		if len(actionParts) < 3 {
			return nil, nil, errors.New(fmt.Sprintf("command *%s* not found\n%s", searchHostOrCommand, group.Help))
		}
		searchCommand = actionParts[2]
		cmdIndex = 2
	}
	var rawCommand string
	command, exist := group.Commands[searchCommand]
	if exist {
		helpIndex := cmdIndex + 1
		if len(actionParts) > helpIndex && actionParts[helpIndex] == "help" {
			return nil, nil, errors.New(command.Help)
		}
		if len(command.Arguments) == 0 {
			rawCommand = command.Format
		} else {
			args := make([]interface{}, len(command.Arguments))
			for i, arg := range command.Arguments {
				argIndex := i + cmdIndex + 1
				if len(actionParts) <= argIndex {
					return nil, nil, errors.New(
						fmt.Sprintf("*%d* argument not found\n%s", i+1, command.Help))
				}
				argValue := actionParts[argIndex]
				found := false
				for _, item := range arg.Items {
					if argValue == item.Name {
						args[i] = item.Value
						found = true
						break
					}
				}
				if !found {
					return nil, nil, errors.New(
						fmt.Sprintf("argument value *%s* not found\n%s", argValue, command.Help))
				}
			}
			rawCommand = fmt.Sprintf(command.Format, args...)
		}
	}
	if rawCommand == "" {
		return nil, nil, errors.New(fmt.Sprintf("command *%s* not found\n%s", searchCommand, group.Help))
	}

	return &rawCommand, host, nil
}

// execute ssh command on specified host
func execute(conf *config.Config, cmd *string, host *config.Host) (string, error) {
	addr := fmt.Sprintf("%s:%d", host.Address, host.Port)
	auth := host.Auth
	sshConf := &ssh.ClientConfig{
		User:            auth.Username,
		Timeout:         conf.Settings.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if auth.Type == "password" {
		sshConf.Auth = []ssh.AuthMethod{
			ssh.Password(auth.Password),
		}
	} else if auth.Type == "" {
		key, err := ioutil.ReadFile(auth.PrivateKeyPath)
		if err != nil {
			return "", errors.New(fmt.Sprintf("error loading private key %q: %v", auth.PrivateKeyPath, err))
		}
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(auth.Passphrase))
		if err != nil {
			return "", errors.New(fmt.Sprintf("error parsing private key %q: %v", auth.PrivateKeyPath, err))
		}
		sshConf.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else {
		return "", errors.New(fmt.Sprintf("bad host %q auth type: %q", addr, auth.Type))
	}

	client, err := ssh.Dial("tcp", addr, sshConf)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error opening ssh connection: %v", err))
	}
	session, err := client.NewSession()
	if err != nil {
		return "", errors.New(fmt.Sprintf("error creating ssh session: %v", err))
	}
	defer session.Close()

	data, err := session.CombinedOutput(*cmd)
	if err != nil {
		return "", errors.New(fmt.Sprintf("error calling ssh command: %v", err))
	}
	output := string(data)
	outputSymbols := []rune(output)
	maxSymbols := conf.Settings.MaxSymbols

	if len(outputSymbols) <= maxSymbols {
		return fmt.Sprintf("```%s```", string(outputSymbols)), nil
	}

	outputSymbols = outputSymbols[:maxSymbols]
	return fmt.Sprintf("```%s\ntrimmed...```", string(outputSymbols)), nil
}