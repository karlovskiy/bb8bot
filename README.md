# bb8bot
Slack bot to run SSH commands.

## Build
```
docker build -t bb8bot . 
```

## Run
```
docker run --name bb8bot -v /you_host_dir/config.toml:/etc/bb8bot/config.toml bb8bot
```

## Configuration

### Bot settings
```toml
[settings]
    # Bot description
    description = "My name is *bb8bot* and i can run ssh commands on remote hosts."
    # slack token for your bot user
    token = "YOUR SLACK TOKEN"
    # long command output will be splitted by maxSymbolsPerMessage
    maxSymbolsPerMessage = 3000
    # splitted long command output messages will be truncated after maxMessages
    maxMessages = 5
    # ssh command timeout for all commands (can be overridden on command config section)
    timeout = "30s"
    # users that can use bot (if this parameter not set - all users can)
    users = ["URG2CGE2D", "URG3DGE7M"]
    # channels in what bot can be used (if this parameter not set - bot can used in any channel)
    channels = ["CRKR3KRN3"]
    # for admins users and channels restrictions will be skipped
    admins = ["URG2EGE1K"]
```

### Host configuration
```toml
[[host]]
    # host id that could be used in command host parameter
    id = "localhost"
    # host address
    address = "localhost"
    # host ssh port
    port = 22
    
    # public key auth
    [host.auth]    
        type = "publickey"
        username = "your_user"
        privateKeyPath = "~/.ssh/your_private_key"
        passphrase = "your_passphrase"
```
You can also use `password` authentication instead of `publickey`:
```toml
    # password auth
    [host.auth]
        type = "password"
        username = "your_user"
        password = "your_pass"
```

### Group commands configuration
```toml
[[group]]
    # group id (should be unique for all bot groups)
    id = "unix"
    # group description will be used in the group help
    description = "Unix useful commands"
    # hosts ids that this group can be used with
    hosts = ["localhost", "somehost"]

    [[group.command]]
        # command id (should be unique for bot this group)
        id="memory"
        # command description will be used in the command help
        description = "Display amount of free and used memory in the system"
        # real ssh command template
        cmdFmt="free -th"

    [[group.command]]
        id = "lsof"
        description = "List open connections"
        # command template with arguments
        cmdFmt = "lsof -i %s -P"
        # override bot settings
        maxSymbolsPerMessage = 2000
        # override bot settings
        maxMessages = 10
        # arguments ids
        arguments = ["protocol"]

    [[group.argument]]
        # argument id (should be unique for bot this group)
        id = "protocol"
        # argument description will be used in the command help
        description = "Supported protocols"

        [[group.argument.item]]
            # value that will be used by user
            name = "ssh"
            # value that will be used by command template
            value = ":22"

```