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
    template_path: "template.tmpl" # ONLY IF YOU USING TEMPLATE
    time_zone: "Europe/Rome" # ONLY IF YOU USING TEMPLATE
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


## Test

To run tests with `make test` you have to:

- Create `config.yml` with a valid telegram API key and timezone in the project directory
- Create `prometheus_bot` executable binary in the project directory
- Define chat ID with `TELEGRAM_CHATID` environment variable
- Ensure port `9087` on localhost is available to bind to

```bash
export TELEGRAM_CHATID="-YOUR TELEGRAM CHAT ID"
make test
```

or

```sh
TELEGRAM_CHATID="-YOUR TELEGRAM CHAT ID" make test
```

## Customizing messages with template

This bot support [go templating language](https://golang.org/pkg/text/template/).
Use it for customizing your message.

For enable template you must set this two settings in your ```config.yaml``` or template will skip.
```yml
template_path: "template.tmpl" # your template file name
time_zone: "Europe/Rome" # yor time zone check it out from WIKI
```

You can also pass template path with `-t` command line argument, it has higher priority than the config option.

[WIKI List of tz database time zones](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)

Best way for build your custom template is:
-    Enable bot with ```-d``` flag
-    Catch some of your alerts in json, then copy it from bot STDOUT
-    Save json in testdata/yourname.json
-    Launch ```make test```

```-d``` options will enable ```debug``` mode and template file will reload every message, else template is load once on startup.

Is provided as [default template file](testdata/default.tmpl) with all possibile variable. Remember that telegram bot support HTML check [here](https://core.telegram.org/bots/api#html-style) list of aviable tags.
