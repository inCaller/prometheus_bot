# prometheus_bot

This bot is designed to alert messages from [alertmanager](https://github.com/prometheus/alertmanager).


## Compile

[GOPATH related doc](https://golang.org/doc/code.html#GOPATH).
```bash
export GOPATH="your go path"
make clean
make
```

## Usage

1. Specify telegram token in ```config.yaml```:

    ```yml
    telegram_token: "token goes here"
    ```

2. Run ```telegram_bot```. See ```prometheus_bot --help``` for command line options
3. Add your bot to a group. It should report group id now

### Configuring alertmanager

Here's the receivers part of the alertmanager configuration file:

```yml
- name: 'admins'
  webhook_configs:
  - send_resolved: True
    url: http://127.0.0.1:9087/alert/-chat_id
```

Replace ```-chat_id``` with the number you got from your bot, with ```-```. To use multiple chats just add more receivers.


## Test your instance
For test your instance, you must only export TELEGRAM_CHATID environment variable
```bash
export TELEGRAM_CHATID="-YOUR TELEGRAM CHAT ID"
make test
```
## For build docker image
```bash
make docker
```
