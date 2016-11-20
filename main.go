package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"gopkg.in/yaml.v2"
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

var config_path = flag.String("c", "config.yaml", "Path to a config file")
var listen_addr = flag.String("l", ":9087", "Listen address")

type Config struct {
	TelegramToken string `yaml:"telegram_token"`
}

var cfg = Config{}

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

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatal(err)
	}

	// bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	go telegramBot(bot)

	router := gin.Default()

	router.GET("/ping/:chatid", func(c *gin.Context) {
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
	})

	router.POST("/alert/:chatid", func(c *gin.Context) {
		chatid, err := strconv.ParseInt(c.Param("chatid"), 10, 64)

		log.Printf("Bot alert post: %d", chatid)

		if err != nil {
			log.Printf("Cat't parse chat id: %q", c.Param("chatid"))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"err": fmt.Sprint(err),
			})
			return
		}

		var alerts Alerts
		//		c.BindJSON(&alerts)
		binding.JSON.Bind(c.Request, &alerts)

		s, err := json.Marshal(alerts)
		if err != nil {
			log.Print(err)
			// XXX c.JSON() isn't needed here?
			return
		}
		log.Printf("Alert: %s", s)

		groupLabels := make([]string, 0, len(alerts.GroupLabels))
		for k := range alerts.GroupLabels {
			groupLabels = append(groupLabels, fmt.Sprintf("%s=<pre>%s</pre>", k, alerts.GroupLabels[k]))
		}

		commonLabels := make([]string, 0, len(alerts.CommonLabels))
		for k := range alerts.CommonLabels {
			if _, ok := alerts.GroupLabels[k]; !ok {
				commonLabels = append(commonLabels, fmt.Sprintf("%s=<pre>%s</pre>", k, alerts.CommonLabels[k]))
			}
		}

		commonAnnotations := make([]string, 0, len(alerts.CommonAnnotations))
		for k := range alerts.CommonAnnotations {
			commonAnnotations = append(commonAnnotations, fmt.Sprintf("\n%s: <pre>%s</pre>", k, alerts.CommonAnnotations[k]))
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

		msgtext := fmt.Sprintf(
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

		log.Printf("message: ", msgtext)

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
	})
	router.Run(*listen_addr)
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
