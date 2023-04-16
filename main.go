package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	m "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	strftime "github.com/itchyny/timefmt-go"
	"github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/cache"
	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"
	"github.com/silenceper/wechat/v2/officialaccount/message"
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
	"wxChatGPT/vo"
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

	r.Get("/test", test2)

	r.Get("/sendFood", sendFood)

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
	err := menu.SetMenuByJSON(data)
	if err != nil {
		fmt.Println(err)
	}

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

func someFunc(ctx context.Context) {
	for {

		select {
		case <-ctx.Done():
			fmt.Println("bbbb")
			return
		default:
			fmt.Println("sss")

		}
	}

}
func sendFood(w http.ResponseWriter, r *http.Request) {
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

	data := message.MediaText{
		Content: "哦吼重新问吧",
	}
	customerMessage := message.CustomerMessage{
		Msgtype: message.MsgTypeText,
		Text:    &data,
		ToUser:  "oKPCA1d6cAjqDDqHkoAO3YHRWVgg",
	}
	error := officialAccount.GetCustomerMessageManager().Send(&customerMessage)
	if error != nil {
		fmt.Println(error)
	}
}

func test(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("监控退出，停止了...")
				return
			default:
				fmt.Println("goroutine监控中...")
				time.Sleep(2 * time.Second)
			}
		}
	}(ctx)

	time.Sleep(10 * time.Second)
	fmt.Println("可以了，通知监控停止")
	cancel()
	//为了检测监控过是否停止，如果没有监控输出，就表示停止了
	time.Sleep(5 * time.Second)
}

func test2(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	num := 1
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("监控退出，停止了...")
				return
			case <-time.After(3*time.Second + 500*time.Millisecond):
				fmt.Println("goroutine监控中...")
				time.Sleep(2 * time.Second)
				if num >= 3 {
					cancel()
				}
				num++
			}
		}
	}(ctx)
	fmt.Println("我来了")

}

//func SendMsgChan(msg string, ctx context.Context) <-chan Result {
//	ch := make(chan Result, 1)
//	go func() {
//		defer func() {
//			if err := recover(); err != nil {
//				err = err.(error)
//				if err != context.Canceled {
//					ch <- Result{Err: err.(error)}
//				}
//			}
//		}()
//		a, e := Completions(msg)
//		ch <- Result{Val: a, Err: e}
//	}()
//	return ch
//}

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
		replyMsg = "简历 " + "https://rvjrijs0ha.feishu.cn/docx/JmJYda0KwoHsLGx0oZOctIP6nxc" + "\n"
		replyMsg = replyMsg + "招聘要求 " + "https://rvjrijs0ha.feishu.cn/docx/YTMxdDMccoVh6ZxKdf7cQsa2nKf"
		//// 替换掉@文本，然后向GPT发起请求
		if xmlMsg.Content == "cheng" {

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
			return
		}

		t := time.Now()
		title := strftime.Format(t, "%Y-%m-%d")

		if strings.HasPrefix(xmlMsg.Content, "写日记") {
			writeDinary(xmlMsg, w, "写日记", "日记", title)
			return
		}

		if strings.HasPrefix(xmlMsg.Content, "看日记") {
			seeDinary(xmlMsg, w, "日记", title)
			return
		}

		if strings.HasPrefix(xmlMsg.Content, "写行程") {
			writeDinary(xmlMsg, w, "写行程", "行程", title)
			return
		}

		if strings.HasPrefix(xmlMsg.Content, "看行程") {
			seeDinary(xmlMsg, w, "行程", title)
			return
		}

		if strings.HasPrefix(xmlMsg.Content, "写") {
			action := strings.Split(xmlMsg.Content, " ")[0]
			category := strings.ReplaceAll(action, "写", "")
			writeDinary(xmlMsg, w, action, category, category)
			return
		}

		if strings.HasPrefix(xmlMsg.Content, "看") {
			action := strings.Split(xmlMsg.Content, " ")[0]
			category := strings.ReplaceAll(action, "看", "")
			seeDinary(xmlMsg, w, category, category)
			return
		}

		requestText := strings.TrimSpace(strings.ReplaceAll(xmlMsg.Content, "@cheng", ""))
		//ss, err := gtp.Completions(requestText)
		//if err != nil {
		//	log.Printf("gtp request error: %v \n", err)
		//	ss = "机器人神了，我一会发现了就去修。"
		//}
		//replyMsg = strings.TrimSpace(ss)

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

		ctx, cancel := context.WithCancel(context.Background())
		num := 1
		fmt.Println("我来了")
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					fmt.Println("完成...")
					return

				case <-time.After(4*time.Second + 500*time.Millisecond):
					if num <= 10 {
						fmt.Println("稍等")
						//data := message.MediaText{
						//	Content: "稍等",
						//}
						//customerMessage := message.CustomerMessage{
						//	Msgtype: message.MsgTypeText,
						//	Text:    &data,
						//	ToUser:  xmlMsg.FromUserName,
						//}
						//officialAccount.GetCustomerMessageManager().Send(&customerMessage)
						num++
					} else {
						fmt.Println("哦吼重新问吧")

						data := message.MediaText{
							Content: "哦吼重新问吧",
						}
						customerMessage := message.CustomerMessage{
							Msgtype: message.MsgTypeText,
							Text:    &data,
							ToUser:  xmlMsg.FromUserName,
						}
						officialAccount.GetCustomerMessageManager().Send(&customerMessage)
						cancel()
					}

				}
			}
		}(ctx)

		go func() {

			a, err := gtp.Completions(requestText)
			if err != nil {
				num = 100
			}
			data := message.MediaText{
				Content: strings.TrimSpace(a),
			}
			customerMessage := message.CustomerMessage{
				Msgtype: message.MsgTypeText,
				Text:    &data,
				ToUser:  xmlMsg.FromUserName,
			}
			if num <= 5 {
				error := officialAccount.GetCustomerMessageManager().Send(&customerMessage)
				if error != nil {
					fmt.Println(error)
				}
			}

			cancel()
		}()

		replyMsg = strings.TrimSpace("收到，稍等")
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

func writeDinary(xmlMsg *convert.TextMsg, w http.ResponseWriter, action string, category string, title string) {

	requestText := strings.TrimSpace(strings.ReplaceAll(xmlMsg.Content, action, ""))

	//posturl := "https://api.punengshuo.com/api/addDinary"
	posturl := "https://chengapi.yufu.pub/openapi/articles/"
	jsonStr := []byte(`{ "chapter": 1,
		"category": "` + category + `", "title": "` + title + `", "content": "` + requestText + `" }`)

	content := util.Post(posturl, jsonStr, "application/json")
	fmt.Printf("data: s%", content)
	textRes := &convert.TextRes{
		ToUserName:   xmlMsg.FromUserName,
		FromUserName: xmlMsg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      "ok",
	}
	_, err := w.Write(textRes.ToXml())
	if err != nil {
		log.Errorln(err)
		if config.GetIsDebug() {
			m.PrintPrettyStack(err)
		}
	}
}

func seeDinary(xmlMsg *convert.TextMsg, w http.ResponseWriter, category string, title string) {

	//geturl := "https://api.punengshuo.com/api/seeDinary?"
	geturl := "https://chengapi.yufu.pub/openapi/articles/see?"
	geturl = geturl + "title=" + title
	geturl = geturl + "&category=" + category

	content := util.Get(geturl)
	fmt.Printf("data: s%", content)

	var result vo.ArticleResultVO

	err2 := json.Unmarshal([]byte(content), &result)
	if err2 != nil {
		fmt.Println("error:", err2)
	}

	textRes := &convert.TextRes{
		ToUserName:   xmlMsg.FromUserName,
		FromUserName: xmlMsg.ToUserName,
		CreateTime:   time.Now().Unix(),
		MsgType:      "text",
		Content:      result.Data.Content,
	}
	_, err := w.Write(textRes.ToXml())
	if err != nil {
		log.Errorln(err)
		if config.GetIsDebug() {
			m.PrintPrettyStack(err)
		}
	}
}
