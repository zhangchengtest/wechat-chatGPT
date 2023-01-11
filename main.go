package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	m "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
	"wxChatGPT/chatGPT"
	"wxChatGPT/config"
	"wxChatGPT/convert"
	"wxChatGPT/util"
	"wxChatGPT/util/middleware"
	"wxChatGPT/util/signature"
)

const wxToken = "cheng12345678" // 这里填微信开发平台里设置的 Token

var reqGroup singleflight.Group

func init() {
	log.SetLevel(config.GetLogLevel())
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   runtime.GOOS == "windows",
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	config.AddConfigChangeCallback(func() {
		log.SetLevel(config.GetLogLevel())
	})
}

func main() {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recover)

	// ChatGPT 可用性检查
	r.Get("/healthCheck", healthCheck)
	// 微信接入校验
	r.Get("/weChatGPT", wechatCheck)
	// 微信消息处理
	r.Post("/weChatGPT", wechatMsgReceive)

	r.Get("/createMenu", createMenu)

	l, err := net.Listen("tcp", ":"+config.ReadConfig().Port)
	if err != nil {
		log.Fatalln(err)
	}
	log.Infof("Server listening at %s", l.Addr())
	if err = http.Serve(l, r); err != nil {
		log.Fatalln(err)
	}
}

// ChatGPT 可用性检查
func healthCheck(w http.ResponseWriter, r *http.Request) {
	msg, err := gtp.Completions("宇宙的终极答案是什么?")
	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		msg = "机器人神了，我一会发现了就去修。"
	}

	log.Infof("测试返回：%s", msg)
	render.PlainText(w, r, "ok")
}

// 微信接入校验
func wechatCheck(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	sign := query.Get("signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	echostr := query.Get("echostr")

	// 校验
	if signature.CheckSignature(sign, timestamp, nonce, wxToken) {
		render.PlainText(w, r, echostr)
		return
	}

	log.Warnln("微信接入校验失败")
}

// 微信接入校验
func createMenu(w http.ResponseWriter, r *http.Request) {

	wc := wechat.NewWechat()
	//这里本地内存保存access_token，也可选择redis，memcache或者自定cache
	memory := cache.NewMemory()
	cfg := &offConfig.Config{
		AppID:     "wx70711c9b88f9c12f",
		AppSecret: "20993710aa48342888d3a0b1755af9d6",
		Token:     wxToken,
		//EncodingAESKey: "xxxx",
		Cache: memory,
	}
	officialAccount := wc.GetOfficialAccount(cfg)
	menu := officialAccount.GetMenu()
	data := readJson("menu.json")
	menu.SetMenuByJSON(data)
}

func readJson(name string) string {
	b, err := ioutil.ReadFile(name) // just pass the file name
	if err != nil {
		fmt.Print(err)
	}
	str := string(b) // convert content to a 'string'
	fmt.Println(str) // print the content as a 'string'
	return str
}

// 微信消息处理
func wechatMsgReceive(w http.ResponseWriter, r *http.Request) {
	// 解析消息
	body, _ := io.ReadAll(r.Body)
	xmlMsg := convert.ToTextMsg(body)

	log.Infof("[消息接收] Type: %s, From: %s, MsgId: %d, Content: %s", xmlMsg.MsgType, xmlMsg.FromUserName, xmlMsg.MsgId, xmlMsg.Content)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// 回复消息
	replyMsg := ""

	// 关注公众号事件
	if xmlMsg.MsgType == "event" {
		if xmlMsg.Event == "unsubscribe" {
			//chatGPT.DefaultGPT().DeleteUser(xmlMsg.FromUserName)
		}
		if xmlMsg.Event != "subscribe" {
			util.TodoEvent(w)
			return
		}
		replyMsg = ":) 感谢你发现了这里"
	} else if xmlMsg.MsgType == "text" {
		// 【收到不支持的消息类型，暂无法显示】
		if strings.Contains(xmlMsg.Content, "【收到不支持的消息类型，暂无法显示】") {
			util.TodoEvent(w)
			return
		}
		// 替换掉@文本，然后向GPT发起请求
		replaceText := "@cheng"
		requestText := strings.TrimSpace(strings.ReplaceAll(xmlMsg.Content, replaceText, ""))
		ss, err := gtp.Completions(requestText)
		if err != nil {
			log.Printf("gtp request error: %v \n", err)
			ss = "机器人神了，我一会发现了就去修。"
		}
		replyMsg = strings.TrimSpace(ss)
	} else {
		util.TodoEvent(w)
		return
	}

	textRes := &convert.TextRes{
		ToUserName:   xmlMsg.FromUserName,
		FromUserName: xmlMsg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      replyMsg,
	}
	_, err := w.Write(textRes.ToXml())
	if err != nil {
		log.Errorln(err)
		if config.GetIsDebug() {
			m.PrintPrettyStack(err)
		}
	}
}
