package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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
	StartsAt     string                 `json:"startsAt"`
	EndsAt       string                 `json:"sendsAt"`
	Status       string                 `json:"status"`
	GeneratorURL string                 `json:"generatorURL"`
	Labels       map[string]interface{} `json:"labels"`
}

var config_path = flag.String("c", "config/config.yaml", "Path to a config file")

type Config struct {
	TelegramToken string `yaml:"telegram_token"`
	ListenAddr    string `yaml:"listen_addr"`
	DebugBot      bool   `yaml:"debug_bot"`
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

	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":9087"
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	bot.Debug = cfg.DebugBot
	if err != nil {
		log.Fatalf("Error create Bot %v", err)
	}

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

		contains := func(s []string, e string) bool {
			for _, a := range s {
				if a == e {
					return true
				}
			}
			return false
		}

		if err != nil {
			log.Printf("Cat't parse chat id: %q", c.Param("chatid"))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"err": fmt.Sprint(err),
			})
			return
		}

		log.Printf("Bot alert post: %d", chatid)

		var alerts Alerts
		//		c.BindJSON(&alerts)
		binding.JSON.Bind(c.Request, &alerts)

		s, err := json.Marshal(alerts)
		if err != nil {
			log.Printf("Error Marchal json. %v", err)
			// XXX c.JSON() isn't needed here?
			return
		}
		log.Printf("Received Alerts JSON: %s", s)

		GroupLabels_keys := make([]string, 0, len(alerts.GroupLabels))
		for k := range alerts.GroupLabels {
			GroupLabels_keys = append(GroupLabels_keys, k)
		}
		sort.Strings(GroupLabels_keys)

		groupLabels := make([]string, 0, len(alerts.GroupLabels))
		for _, k := range GroupLabels_keys {
			groupLabels = append(groupLabels, fmt.Sprintf("    %s: <code>%s</code>", k, alerts.GroupLabels[k]))
		}

		CommonLabels_keys := make([]string, 0, len(alerts.CommonLabels))
		for k := range alerts.CommonLabels {
			CommonLabels_keys = append(CommonLabels_keys, k)
		}
		sort.Strings(CommonLabels_keys)
		commonLabels := make([]string, 0, len(alerts.CommonLabels))
		for _, k := range CommonLabels_keys {
			if _, ok := alerts.GroupLabels[k]; !ok {
				commonLabels = append(commonLabels, fmt.Sprintf("    %s: <code>%s</code>", k, alerts.CommonLabels[k]))
			}
		}

		keys := make([]string, 0, len(alerts.CommonAnnotations))
		for k := range alerts.CommonAnnotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		commonAnnotations := make([]string, 0, len(alerts.CommonAnnotations))
		for _, k := range keys {
			commonAnnotations = append(commonAnnotations, fmt.Sprintf("    %s: <code>%s</code>", k, alerts.CommonAnnotations[k]))
		}

		alertDetails := make([]string, 0, len(alerts.Alerts))
		var i int
		for _, a := range alerts.Alerts {
			var alert string
			keys := make([]string, 0)
			i += 1
			alert = fmt.Sprintf("%d.\n", i)

			for k, _ := range a.Labels {
				keys = append(keys, k)
			}

			alert += fmt.Sprintf("    status: <b>%s</b>\n", strings.ToUpper(a.Status))
			sort.Strings(keys)
			for _, val := range keys {
				if !contains(CommonLabels_keys, val) {
					alert += fmt.Sprintf("    %s: <code>%s</code>\n", val, a.Labels[val].(string))
				}
			}

			StartAtTime, _ := time.Parse(time.RFC3339, a.StartsAt)
			StartAtStr := fmt.Sprintf("%02d.%02d.%04d %02d:%02d:%02d",
				StartAtTime.Day(), StartAtTime.Month(), StartAtTime.Year(),
				StartAtTime.Hour(), StartAtTime.Minute(), StartAtTime.Second())
			alert += fmt.Sprintf("    startAt: <code>%s</code>\n", StartAtStr)
			alert += fmt.Sprintf("    <b>Annotation:</b>\n")

			keys = make([]string, 0)
			for k, _ := range a.Annotations {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				alert += fmt.Sprintf("        %s: <code>%s</code>\n", k, a.Annotations[k])
			}

			alertDetails = append(alertDetails, alert)
		}

		msgtext := fmt.Sprintf("[%s:<b>%d</b>]\n<b>Grouped by:</b>\n%s\n<b>Common labels:</b>\n%s\n<b>Common annotations:</b>\n%s\n<b>Alerts:</b>\n%s",
			strings.ToUpper(alerts.Status),
			len(alerts.Alerts),
			strings.Join(groupLabels, "\n"),
			strings.Join(commonLabels, "\n"),
			strings.Join(commonAnnotations, "\n"),
			strings.Join(alertDetails, "\n"),
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
	router.Run(cfg.ListenAddr)
}

func telegramBot(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		log.Fatalf("Error GetUpdatesChan. %v", err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}
		log.Println("Received message: [%s] %s", update.Message.From.UserName, update.Message.Text)
	}
}
