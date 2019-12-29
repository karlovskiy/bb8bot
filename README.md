# bb8bot
Slack bot to run SSH commands.

### Build
```
docker build -t bb8bot . 
```

### Run
```
docker run --name bb8bot -v /you_host_dir/config.toml:/etc/bb8bot/config.toml bb8bot
```
