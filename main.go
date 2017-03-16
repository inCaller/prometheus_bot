package main // import "github.com/inCaller/prometheus_bot"

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/yaml.v2"

	"text/template"
)

type Alerts struct {
	Alerts            []Alert                `json:"alerts"`
	CommonAnnotations map[string]interface{} `json:"commonAnnotations"`
	CommonLabels      map[string]interface{} `json:"commonLabels"`
	ExternalURL       string                 `json:"externalURL"`
	GroupKey          int                    `json:"groupKey"`
	GroupLabels       map[string]interface{} `json:"groupLabels"`
	Receiver          string                 `json:"receiver"`
	Status            string                 `json:"status"`
	Version           int                    `json:"version"`
}

type Alert struct {
	Annotations  map[string]interface{} `json:"annotations"`
	EndsAt       string                 `json:"sendsAt"`
	GeneratorURL string                 `json:"generatorURL"`
	Labels       map[string]interface{} `json:"labels"`
	StartsAt     string                 `json:"startsAt"`
}

type Config struct {
	TelegramToken string `yaml:"telegram_token"`
	TemplatePath  string `yaml:"template_path"`
	TimeZone      string `yaml:"time_zone"`
}

// Global
var config_path = flag.String("c", "config.yaml", "Path to a config file")
var listen_addr = flag.String("l", ":9087", "Listen address")
var template_path = flag.String("t", "", "Path to a template file")
var debug = flag.Bool("d", false, "Debug template")

var cfg = Config{}
var bot *tgbotapi.BotAPI
var tmpH *template.Template

// Template addictional functions map
var funcMap = template.FuncMap{
	"FormatDate": func(toformat string) string {
		IN_layout := "2006-01-02T15:04:05.000-07:00"
		OUT_layout := "02/01/2006 15:04:05"

		t, err := time.Parse(IN_layout, toformat)

		if err != nil {
			fmt.Println(err)
		}

		loc, _ := time.LoadLocation(cfg.TimeZone)

		return t.In(loc).Format(OUT_layout)
	},
}

func telegramBot(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatal(err)
	}

	for update := range updates {
		if update.Message.NewChatMember != nil {
			if update.Message.NewChatMember.UserName == bot.Self.UserName && update.Message.Chat.Type == "group" {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Chat id is '%d'", update.Message.Chat.ID))
				bot.Send(msg)
			}
		}
	}
}

func loadTemplate(tmplPath string) *template.Template {
	// let's read template
	tmpH, err := template.New(path.Base(tmplPath)).Funcs(funcMap).ParseFiles(cfg.TemplatePath)

	if err != nil {
		log.Fatalf("Problem reading parsing template file: %v", err)
	} else {
		log.Printf("Load template file:%s", tmplPath)
	}

	return tmpH
}

func main() {
	flag.Parse()

	content, err := ioutil.ReadFile(*config_path)
	if err != nil {
		log.Fatalf("Problem reading configuration file: %v", err)
	}
	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		log.Fatalf("Error parsing configuration file: %v", err)
	}

	if *template_path != "" {
		cfg.TemplatePath = *template_path
	}

	bot_tmp, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}

	bot = bot_tmp
	if cfg.TemplatePath != "" {

		tmpH = loadTemplate(cfg.TemplatePath)

		if cfg.TimeZone == "" {
			log.Fatalf("You must define time_zone of your bot")
			panic(-1)
		}

	} else {
		*debug = false
		tmpH = nil
	}
	//bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	go telegramBot(bot)

	router := gin.Default()

	router.GET("/ping/:chatid", GET_Handling)
	router.POST("/alert/:chatid", POST_Handling)
	router.Run(*listen_addr)
}

func GET_Handling(c *gin.Context) {
	log.Printf("Recived GET")
	chatid, err := strconv.ParseInt(c.Param("chatid"), 10, 64)
	if err != nil {
		log.Printf("Cat't parse chat id: %q", c.Param("chatid"))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"err": fmt.Sprint(err),
		})
		return
	}

	log.Printf("Bot test: %d", chatid)
	msgtext := fmt.Sprintf("Some HTTP triggered notification by prometheus bot... %d", chatid)
	msg := tgbotapi.NewMessage(chatid, msgtext)
	sendmsg, err := bot.Send(msg)
	if err == nil {
		c.String(http.StatusOK, msgtext)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"err":     fmt.Sprint(err),
			"message": sendmsg,
		})
	}
}

func AlertFormatStandard(alerts Alerts) string {
	keys := make([]string, 0, len(alerts.GroupLabels))
	for k := range alerts.GroupLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	groupLabels := make([]string, 0, len(alerts.GroupLabels))
	for _, k := range keys {
		groupLabels = append(groupLabels, fmt.Sprintf("%s=<code>%s</code>", k, alerts.GroupLabels[k]))
	}

	keys = make([]string, 0, len(alerts.CommonLabels))
	for k := range alerts.CommonLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	commonLabels := make([]string, 0, len(alerts.CommonLabels))
	for _, k := range keys {
		if _, ok := alerts.GroupLabels[k]; !ok {
			commonLabels = append(commonLabels, fmt.Sprintf("%s=<code>%s</code>", k, alerts.CommonLabels[k]))
		}
	}

	keys = make([]string, 0, len(alerts.CommonAnnotations))
	for k := range alerts.CommonAnnotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	commonAnnotations := make([]string, 0, len(alerts.CommonAnnotations))
	for _, k := range keys {
		commonAnnotations = append(commonAnnotations, fmt.Sprintf("\n%s: <code>%s</code>", k, alerts.CommonAnnotations[k]))
	}

	alertDetails := make([]string, len(alerts.Alerts))
	for i, a := range alerts.Alerts {
		if instance, ok := a.Labels["instance"]; ok {
			instanceString, _ := instance.(string)
			alertDetails[i] += strings.Split(instanceString, ":")[0]
		}
		if job, ok := a.Labels["job"]; ok {
			alertDetails[i] += fmt.Sprintf("[%s]", job)
		}
		if a.GeneratorURL != "" {
			alertDetails[i] = fmt.Sprintf("<a href='%s'>%s</a>", a.GeneratorURL, alertDetails[i])
		}
	}
	return fmt.Sprintf(
		"<a href='%s/#/alerts?receiver=%s'>[%s:%d]</a>\ngrouped by: %s\nlabels: %s%s\n%s",
		alerts.ExternalURL,
		alerts.Receiver,
		strings.ToUpper(alerts.Status),
		len(alerts.Alerts),
		strings.Join(groupLabels, ", "),
		strings.Join(commonLabels, ", "),
		strings.Join(commonAnnotations, ""),
		strings.Join(alertDetails, ", "),
	)
}

func AlertFormatTemplate(alerts Alerts) string {
	var bytesBuff bytes.Buffer
	var err error

	writer := io.Writer(&bytesBuff)

	if *debug {
		log.Printf("Reloading Template\n")
		// reload template bacause we in debug mode
		tmpH = loadTemplate(cfg.TemplatePath)
	}

	tmpH.Funcs(funcMap)
	err = tmpH.Execute(writer, alerts)

	if err != nil {
		log.Fatalf("Problem with template execution: %v", err)
		panic(err)
	}

	return bytesBuff.String()
}

func POST_Handling(c *gin.Context) {
	var msgtext string
	var alerts Alerts

	chatid, err := strconv.ParseInt(c.Param("chatid"), 10, 64)

	log.Printf("Bot alert post: %d", chatid)

	if err != nil {
		log.Printf("Cat't parse chat id: %q", c.Param("chatid"))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"err": fmt.Sprint(err),
		})
		return
	}

	binding.JSON.Bind(c.Request, &alerts)

	s, err := json.Marshal(alerts)
	if err != nil {
		log.Print(err)
		return
	}

	log.Println("+------------------  A L E R T  J S O N  -------------------+")
	log.Printf("%s", s)
	log.Println("+-----------------------------------------------------------+\n\n")

	// Decide how format Text
	if cfg.TemplatePath == "" {
		msgtext = AlertFormatStandard(alerts)
	} else {
		msgtext = AlertFormatTemplate(alerts)
	}
	// Print in Log result message
	log.Println("+---------------  F I N A L   M E S S A G E  ---------------+")
	log.Println(msgtext)
	log.Println("+-----------------------------------------------------------+")

	msg := tgbotapi.NewMessage(chatid, msgtext)
	msg.ParseMode = tgbotapi.ModeHTML

	msg.DisableWebPagePreview = true

	sendmsg, err := bot.Send(msg)
	if err == nil {
		c.String(http.StatusOK, "telegram msg sent.")
	} else {
		log.Printf("Error sending message: %s", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"err":     fmt.Sprint(err),
			"message": sendmsg,
			"srcmsg":  fmt.Sprint(msgtext),
		})
		msg := tgbotapi.NewMessage(chatid, "Error sending message, checkout logs")
		bot.Send(msg)
	}
}
