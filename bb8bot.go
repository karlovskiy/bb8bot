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
	"unicode/utf8"
)

var (
	configPath = flag.String("c", "", "config path")
)

func main() {

	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)

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

// handleIncomingEvents handles all incoming RTM events
func handleIncomingEvents(rtm *slack.RTM, conf *config.Config) {
	for msg := range rtm.IncomingEvents {

		switch ev := msg.Data.(type) {

		case *slack.MessageEvent:
			log.Printf("Message: %+v", ev)

			info := rtm.GetInfo()
			prefix := "<@" + info.User.ID + ">"
			text := ev.Msg.Text
			action := strings.TrimPrefix(text, prefix)
			if action != text {

				user := ev.User
				channel := ev.Channel
				log.Printf("User : %q Channel: %q", user, channel)

				users := conf.Settings.Users
				channels := conf.Settings.Channels

				_, isAdmin := conf.Settings.Admins[user]
				if _, isUserPermitted := users[user]; len(users) > 0 && !isUserPermitted && !isAdmin {
					log.Printf("User %q doesn't have enough permissions", user)
					rtm.SendMessage(rtm.NewOutgoingMessage("You don't have enough permissions", channel))
				} else {
					if _, isChannelPermitted := channels[channel]; len(channels) > 0 && !isChannelPermitted && !isAdmin {
						log.Printf("Channel %q doesn't have enough permissions", channel)
						rtm.SendMessage(rtm.NewOutgoingMessage("This channel doesn't have enough permissions", channel))
					} else {

						action = strings.TrimSpace(action)
						rawCmd, command, host, err := parseAction(action, conf)
						if err != nil {
							rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("%v", err), channel))
						} else {
							msgs, err := execute(rawCmd, command, host)
							if err != nil {
								rtm.SendMessage(rtm.NewOutgoingMessage(fmt.Sprintf("error execution action: %v", err), channel))
							}
							for _, msg := range msgs {
								rtm.SendMessage(rtm.NewOutgoingMessage(msg, channel))
							}
						}
					}
				}
			}
		default:
			// Ignore other events..
		}
	}
}

// parseAction parses action from chat message and convert it to ssh command for execution
func parseAction(action string, conf *config.Config) (rawCmd *string, command *config.Command, host *config.Host, err error) {
	log.Printf("Parse action: %q", action)

	if action == "" || action == "help" {
		return nil, nil, nil, errors.New(conf.Help)
	}

	actionParts := strings.Fields(action)
	searchGroup := actionParts[0]
	group, exist := conf.Groups[searchGroup]
	if !exist {
		return nil, nil, nil, errors.New(fmt.Sprintf("group *%s* not found\n%s", searchGroup, conf.Help))
	}
	if len(actionParts) < 2 {
		return nil, nil, nil, errors.New(group.Help)
	}

	searchHostOrCommand := actionParts[1]
	if len(actionParts) == 3 && actionParts[2] == "help" {
		helpCommand, exist := group.Commands[searchHostOrCommand]
		if exist {
			return nil, nil, nil, errors.New(helpCommand.Help)
		}
	}

	if len(group.Hosts) == 0 {
		return nil, nil, nil, errors.New(fmt.Sprintf("hosts for group *%s* not found\n%s", group.Id, group.Help))
	} else if len(group.Hosts) == 1 {
		for _, h := range group.Hosts {
			host = h
		}
	} else {
		host, exist = conf.Hosts[searchHostOrCommand]
		if !exist {
			return nil, nil, nil, errors.New(fmt.Sprintf("host *%s* not found,\n%s", searchHostOrCommand, group.Help))
		}
	}
	if host == nil {
		return nil, nil, nil, errors.New(fmt.Sprintf("host or command *%s* not found\n%s", searchHostOrCommand, conf.Help))
	}
	searchCommand := searchHostOrCommand
	cmdIndex := 1
	if host.Id == searchHostOrCommand {
		if len(actionParts) < 3 {
			return nil, nil, nil, errors.New(fmt.Sprintf("command *%s* not found\n%s", searchHostOrCommand, group.Help))
		}
		searchCommand = actionParts[2]
		cmdIndex = 2
	}
	var rawCommand string
	command, exist = group.Commands[searchCommand]
	if exist {
		helpIndex := cmdIndex + 1
		if len(actionParts) > helpIndex && actionParts[helpIndex] == "help" {
			return nil, nil, nil, errors.New(command.Help)
		}
		if len(command.Arguments) == 0 {
			rawCommand = command.Format
		} else {
			args := make([]interface{}, len(command.Arguments))
			for i, arg := range command.Arguments {
				argIndex := i + cmdIndex + 1
				if len(actionParts) <= argIndex {
					return nil, nil, nil, errors.New(
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
					return nil, nil, nil, errors.New(
						fmt.Sprintf("argument value *%s* not found\n%s", argValue, command.Help))
				}
			}
			rawCommand = fmt.Sprintf(command.Format, args...)
		}
	}
	if rawCommand == "" {
		return nil, nil, nil, errors.New(fmt.Sprintf("command *%s* not found\n%s", searchCommand, group.Help))
	}

	return &rawCommand, command, host, nil
}

// execute executes ssh command on specified host
func execute(rawCmd *string, command *config.Command, host *config.Host) ([]string, error) {
	addr := fmt.Sprintf("%s:%d", host.Address, host.Port)
	log.Printf("Execute cmd: %q on host: %q", *rawCmd, addr)

	auth := host.Auth
	sshConf := &ssh.ClientConfig{
		User:            auth.Username,
		Timeout:         command.Timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if auth.Type == "password" {
		sshConf.Auth = []ssh.AuthMethod{
			ssh.Password(auth.Password),
		}
	} else if auth.Type == "publickey" {
		key, err := ioutil.ReadFile(auth.PrivateKeyPath)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error loading private key %q: %v", auth.PrivateKeyPath, err))
		}
		signer, err := ssh.ParsePrivateKeyWithPassphrase(key, []byte(auth.Passphrase))
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error parsing private key %q: %v", auth.PrivateKeyPath, err))
		}
		sshConf.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	} else {
		return nil, errors.New(fmt.Sprintf("bad host %q auth type: %q", addr, auth.Type))
	}

	client, err := ssh.Dial("tcp", addr, sshConf)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error opening ssh connection: %v", err))
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error creating ssh session: %v", err))
	}
	defer session.Close()

	data, err := session.CombinedOutput(*rawCmd)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error calling ssh command: %v, out: %s", err, data))
	}

	return createMessages(string(data), command.MaxSymbolsPerMessage, command.MaxMessages), nil
}

// createMessages creates messages to send after execution
func createMessages(output string, maxSymbolsPerMessage int, maxMessages int) (msgs []string) {
	var b strings.Builder
	size := utf8.RuneCountInString(output)
	symbolsPerMessage := 0
	for i, r := range []rune(output) {
		b.WriteRune(r)
		symbolsPerMessage++
		if (maxSymbolsPerMessage > 0 && symbolsPerMessage == maxSymbolsPerMessage) || i == size-1 {
			msg := b.String()
			b.Reset()
			lastLFIndex := strings.LastIndex(msg, "\n")
			if lastLFIndex == -1 || lastLFIndex == len(msg) {
				msgs = append(msgs, msg)
			} else {
				beforeLF := msg[:lastLFIndex]
				if beforeLF != "" {
					msgs = append(msgs, beforeLF)
				}
				b.WriteString(msg[lastLFIndex+1:])
			}
			symbolsPerMessage = 0
			if maxMessages > 0 && maxMessages == len(msgs) {
				return
			}
		}
	}
	return
}
