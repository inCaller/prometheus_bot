# Prometheus Bot

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
    telegram_token: "token here"
    ```

2. Run ```telegram_bot```. See ```prometheus_bot --help``` for command line options
3. Add your bot to a group. It should report group id now

### Configuring alert manager

Here's the receivers part of the alert manager configuration file:

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
### Create your own test
When alert manager send alert to bot, only debug ```-d``` it dump json that generate alert. You can copy paste this json for your test.
Test will send ```*.json``` file into ```testdata``` folder

## Customizing messages with template

This bot support [go templating language](https://golang.org/pkg/text/template/).
Use it for customizing your message.

For enable template you must set ```template_path``` in your ```config.yaml```.

```yml
telegram_token: "token here"
template_path: "template.tmpl" # your template file name
```

Best way for build your custom template is:
-    Enable bot with ```-d``` flag
-    Catch some of your alerts in json, then copy it from bot STDOUT
-    Save json in testdata / yourname.json
-    Launch ```make test```

```-d``` options will enable ```debug``` mode and template file will reload every message, else template is load once on startup.

Is provided as default template file with all possibile variable. Remeber that telegram bot support HTML check [here](https://core.telegram.org/bots/api#html-style) list of aviable tags.

### Template extra functions
Template language support many different functions for impoving text, number and data formatting.

#### Support this functions list

-   ```str_UpperCase```: Convert string to uppercase
-   ```str_LowerCase```: Convert string to lowercase
-   ```str_Title```: Convert string in Title, "title" --> "Title" fist letter become Uppercase
-   ```str_Format_byte```: Convert number expressed in ```Byte``` to number in related measure unit. It use ```strconv.ParseFloat(..., 64)``` take look at go related doc for possible input format, usually evry think '35.95e+06' is correct converted.
Example:
    -    35'000'000 [Kb] will converter to '35 Gb'
    -    89'000 [Kb] will converter to '89 Mb'
-   ```HasKey```: Param:dict map, key_search string Search in map if there requeted key

-    ```str_FormatDate```: Convert prometheus string date in your preferred date time format, config file param ```time_outdata``` could be used for setup your favourite format
Require more setting in your cofig.yaml
```yaml
template_time_zone: "Europe/Rome"
template_time_outdata: "02/01/2006 15:04:05"
```
[WIKI List of tz database time zones](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)
