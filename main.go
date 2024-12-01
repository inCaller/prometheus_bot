package main // import "github.com/inCaller/prometheus_bot"

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"html/template"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/microcosm-cc/bluemonday"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"gopkg.in/yaml.v3"
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
	EndsAt       string                 `json:"endsAt"`
	GeneratorURL string                 `json:"generatorURL"`
	Labels       map[string]interface{} `json:"labels"`
	StartsAt     string                 `json:"startsAt"`
	Status       string                 `json:"status"`
}

type Config struct {
	TelegramToken       string `yaml:"telegram_token"`
	TemplatePath        string `yaml:"template_path"`
	TimeZone            string `yaml:"time_zone"`
	TimeOutFormat       string `yaml:"time_outdata"`
	SplitChart          string `yaml:"split_token"`
	SplitMessageBytes   int    `yaml:"split_msg_byte"`
	SendOnly            bool   `yaml:"send_only"`
	DisableNotification bool   `yaml:"disable_notification"`
}

/**
 * Subdivided by 1024
 */
const (
	Kb = iota
	Mb
	Gb
	Tb
	Pb
	Eb
	Zb
	Yb
	Information_Size_MAX
)

/**
 * Subdivided by 10000
 */
const (
	K = iota
	M
	G
	T
	P
	E
	Z
	Y
	Scale_Size_MAX
)

func RoundPrec(x float64, prec int) float64 {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return x
	}

	sign := 1.0
	if x < 0 {
		sign = -1
		x *= -1
	}

	var rounder float64
	pow := math.Pow(10, float64(prec))
	intermed := x * pow
	_, frac := math.Modf(intermed)

	if frac >= 0.5 {
		rounder = math.Ceil(intermed)
	} else {
		rounder = math.Floor(intermed)
	}

	return rounder / pow * sign
}

/******************************************************************************
 *
 *          Function for formatting template
 *
 ******************************************************************************/
func str_Format_MeasureUnit(MeasureUnit string, value string) string {
	var RetStr string
	cfg.SplitChart = "|"
	MeasureUnit = strings.TrimSpace(MeasureUnit) // Remove space
	SplittedMUnit := strings.SplitN(MeasureUnit, cfg.SplitChart, 3)

	Initial := 0
	// If is declared third part of array, then Measure unit start from just scaled measure unit.
	// Example Kg is Kilo g, but all people use Kg not g, then you will put here 3 Kilo. Bot strart convert from here.
	if len(SplittedMUnit) > 2 {
		tmp, err := strconv.ParseInt(SplittedMUnit[2], 10, 8)
		if err != nil {
			log.Println("Could not convert value to int")
			if !*debug {
				// If is running in production leave daemon live. else here will die with log error.
				return "" // Break execution and return void string, bot will log somethink
			}
		}
		Initial = int(tmp)
	}

	switch SplittedMUnit[0] {
	case "kb":
		RetStr = str_Format_Byte(value, Initial)
	case "s":
		RetStr = str_Format_Scale(value, Initial)
	case "f":
		RetStr = str_FormatFloat(value)
	case "i":
		RetStr = str_FormatInt(value)
	default:
		RetStr = str_FormatInt(value)
	}

	if len(SplittedMUnit) > 1 {
		RetStr += SplittedMUnit[1]
	}

	return RetStr
}

// Scale number for It measure unit
func str_Format_Byte(in string, initial int) string {
	var str_Size string

	f, err := strconv.ParseFloat(in, 64)

	if err != nil {
		panic(err)
	}

	for j1 := initial; j1 < (Information_Size_MAX + 1); j1++ {

		if j1 >= Information_Size_MAX {
			str_Size = "Yb"
			break
		} else if f > 1024 {
			f /= 1024.0
		} else {

			switch j1 {
			case Kb:
				str_Size = "Kb"
			case Mb:
				str_Size = "Mb"
			case Gb:
				str_Size = "Gb"
			case Tb:
				str_Size = "Tb"
			case Pb:
				str_Size = "Pb"
			case Eb:
				str_Size = "Eb"
			case Zb:
				str_Size = "Zb"
			case Yb:
				str_Size = "Yb"
			}
			break
		}
	}

	str_fl := strconv.FormatFloat(f, 'f', 2, 64)
	return fmt.Sprintf("%s %s", str_fl, str_Size)
}

// Format number for fisics measure unit
func str_Format_Scale(in string, initial int) string {
	var str_Size string

	f, err := strconv.ParseFloat(in, 64)

	if err != nil {
		panic(err)
	}

	for j1 := initial; j1 < (Scale_Size_MAX + 1); j1++ {

		if j1 >= Scale_Size_MAX {
			str_Size = "Y"
			break
		} else if f > 1000 {
			f /= 1000.0
		} else {
			switch j1 {
			case K:
				str_Size = "K"
			case M:
				str_Size = "M"
			case G:
				str_Size = "G"
			case T:
				str_Size = "T"
			case P:
				str_Size = "P"
			case E:
				str_Size = "E"
			case Z:
				str_Size = "Z"
			case Y:
				str_Size = "Y"
			default:
				str_Size = "Y"
			}
			break
		}
	}

	str_fl := strconv.FormatFloat(f, 'f', 2, 64)
	return fmt.Sprintf("%s %s", str_fl, str_Size)
}

func str_FormatInt(i string) string {
	v, _ := strconv.ParseInt(i, 10, 64)
	val := strconv.FormatInt(v, 10)
	return val
}

func str_FormatFloat(f string) string {
	v, _ := strconv.ParseFloat(f, 64)
	v = RoundPrec(v, 2)
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func str_FormatDate(toformat string) string {

	// Error handling
	if cfg.TimeZone == "" {
		log.Fatal("template_time_zone is not set, if you use template and `str_FormatDate` func is required")
	}

	if cfg.TimeOutFormat == "" {
		log.Fatal("template_time_outdata param is not set, if you use template and `str_FormatDate` func is required")
	}

	t, err := time.Parse(time.RFC3339Nano, toformat)

	if err != nil {
		fmt.Println(err)
	}

	loc, _ := time.LoadLocation(cfg.TimeZone)

	return t.In(loc).Format(cfg.TimeOutFormat)
}

func HasKey(dict map[string]interface{}, key_search string) bool {
	if _, ok := dict[key_search]; ok {
		return true
	}
	return false
}

// Global
var config_path = flag.String("c", "config.yaml", "Path to a config file")
var token_path = flag.String("token-from", "", "Path to a file containing telegram_token")
var listen_addr = flag.String("l", ":9087", "Listen address")
var template_path = flag.String("t", "", "Path to a template file")
var debug = flag.Bool("d", false, "Debug template")

var cfg = Config{}
var bot *tgbotapi.BotAPI
var tmpH *template.Template

// Template additional functions map
var funcMap = template.FuncMap{
	"str_FormatDate":         str_FormatDate,
	"str_UpperCase":          strings.ToUpper,
	"str_LowerCase":          strings.ToLower,
	"str_Title":              strings.Title,
	"str_FormatFloat":        str_FormatFloat,
	"str_Format_Byte":        str_Format_Byte,
	"str_Format_MeasureUnit": str_Format_MeasureUnit,
	"HasKey":                 HasKey,
	"contains":               strings.Contains,
}

func telegramBot(bot *tgbotapi.BotAPI) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	introduce := func(update tgbotapi.Update) {
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Chat id is '%d'", update.Message.Chat.ID))
		if cfg.DisableNotification {
			msg.DisableNotification = true
		}
		bot.Send(msg)
	}

	for update := range updates {
		if update.Message == nil {
			if *debug {
				log.Printf("[UNKNOWN_MESSAGE] [%v]", update)
			}
			continue
		}

		if len(update.Message.NewChatMembers) > 0 {
			for _, member := range update.Message.NewChatMembers {
				if member.UserName == bot.Self.UserName && update.Message.Chat.Type == "group" {
					introduce(update)
				}
			}
		} else if update.Message != nil && update.Message.Text != "" {
			introduce(update)
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

func SplitString(s string, n int) []string {
	sub := ""
	subs := []string{}

	runes := bytes.Runes([]byte(s))
	l := len(runes)
	for i, r := range runes {
		sub = sub + string(r)
		if (i+1)%n == 0 {
			subs = append(subs, sub)
			sub = ""
		} else if (i + 1) == l {
			subs = append(subs, sub)
		}
	}

	return subs
}

func main() {
	flag.Parse()

	content, err := os.ReadFile(*config_path)
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

	if *token_path != "" {
		content, err := os.ReadFile(*token_path)
		if err != nil {
			log.Fatalf("Problem reading token file: %v", err)
		}
		cfg.TelegramToken = strings.TrimSpace(string(content))
	}

	if cfg.SplitMessageBytes == 0 {
		cfg.SplitMessageBytes = 4000
	}

	if cfg.TemplatePath != "" {

		tmpH = loadTemplate(cfg.TemplatePath)

		if cfg.TimeZone == "" {
			log.Fatalf("You must define time_zone of your bot")
		}

	} else {
		*debug = false
		tmpH = nil
	}
	if !(*debug) {
		gin.SetMode(gin.ReleaseMode)
	}

	for {
		bot_tmp, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
		if err == nil {
			bot = bot_tmp
			break
		} else {
			log.Printf("Error initializing telegram connection: %s", err)
			time.Sleep(time.Second)
		}
	}

	if *debug {
		bot.Debug = true
	}

	log.Printf("Authorised on account %s", bot.Self.UserName)

	if cfg.SendOnly {
		log.Printf("Works in send_only mode")
	} else {
		go telegramBot(bot)
	}

	router := gin.Default()

	router.GET("/ping/:chatid", GET_Handling)
	router.GET("/ping/:chatid/:topicid", GET_Handling)
	router.POST("/alert/:chatid", POST_Handling)
	router.POST("/alert/:chatid/:topicid", POST_Handling)

	err = router.Run(*listen_addr)
	if err != nil {
		log.Fatal(err)
	}
}

func GET_Handling(c *gin.Context) {
	log.Printf("Received GET")
	topicid := getID(c, "topicid")
	chatid := getID(c, "chatid")
	log.Printf("Bot test: %d:%d", chatid, topicid)

	msgtext := fmt.Sprintf("Some HTTP triggered notification by prometheus bot... %d:%d", chatid, topicid)
	msg := tgbotapi.NewMessage(chatid, msgtext)
	msg.ReplyToMessageID = int(topicid)
	if cfg.DisableNotification {
		msg.DisableNotification = true
	}
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
	}

	return bytesBuff.String()
}

// SanitizeMsg check string for HTML validity and
// strips all HTML tags if it not valid
func SanitizeMsg(str string) string {
	r := strings.NewReader(str)
	d := xml.NewDecoder(r)

	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity

	for {
		_, err := d.Token()
		if err == io.EOF {
			log.Println("HTML is valid, sending it...")
			break
		} else if err != nil {
			log.Println("HTML is not valid, strip all tags to prevent error")
			p := bluemonday.StrictPolicy()
			str = p.Sanitize(str)
			break
		}
	}

	return str
}

// get id from relative path
func getID(c *gin.Context, param string) int64 {
	// default topicid for messageConfig 0
	if c.Param(param) == "" && param == "topicid" {
		return 0
	}
	id, err := strconv.ParseInt(c.Param(param), 10, 64)
	if err != nil {
		log.Printf("Can't parse %s id: %q", param, c.Param(param))
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"err": fmt.Sprint(err),
		})
	}
	return id
}

func POST_Handling(c *gin.Context) {
	var msgtext string
	var alerts Alerts

	topicid := getID(c, "topicid")
	chatid := getID(c, "chatid")
	log.Printf("Bot alert post: chat %d, topic %d", chatid, topicid)

	binding.JSON.Bind(c.Request, &alerts)

	s, err := json.Marshal(alerts)
	if err != nil {
		log.Print(err)
		return
	}

	log.Println("+------------------  A L E R T  J S O N  -------------------+")
	log.Printf("%s", s)
	log.Println("+-----------------------------------------------------------+")

	// Decide how format Text
	if cfg.TemplatePath == "" {
		msgtext = AlertFormatStandard(alerts)
	} else {
		msgtext = AlertFormatTemplate(alerts)
	}
	for _, subString := range SplitString(msgtext, cfg.SplitMessageBytes) {

		sanitizedString := SanitizeMsg(subString)

		msg := tgbotapi.NewMessage(chatid, sanitizedString)
		msg.ParseMode = tgbotapi.ModeHTML
		msg.ReplyToMessageID = int(topicid)

		// Print in Log result message
		log.Println("+---------------  F I N A L   M E S S A G E  ---------------+")
		log.Println(subString)
		log.Println("+-----------------------------------------------------------+")

		msg.DisableWebPagePreview = true
		if cfg.DisableNotification {
			msg.DisableNotification = true
		}

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
			if cfg.DisableNotification {
				msg.DisableNotification = true
			}
			bot.Send(msg)
		}
	}

}
