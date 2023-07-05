package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"openai/internal/config"
	"openai/internal/service/bot"
	"openai/internal/service/fiter"

	// "openai/internal/service/openai"
	"openai/internal/service/wechat"
	"sync"
	"time"
)

var (
	success  = []byte("success")
	warn     = "警告，检测到敏感词"
	requests sync.Map // K - 消息ID ， V - chan string
)

func WechatCheck(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	signature := query.Get("signature")
	timestamp := query.Get("timestamp")
	nonce := query.Get("nonce")
	echostr := query.Get("echostr")

	// 校验
	if wechat.CheckSignature(signature, timestamp, nonce, config.Wechat.Token) {
		w.Write([]byte(echostr))
		return
	}

	log.Println("此接口为公众号验证，不应该被手动调用，公众号接入校验失败")
}

// https://developers.weixin.qq.com/doc/offiaccount/Message_Management/Passive_user_reply_message.html
// 微信服务器在五秒内收不到响应会断掉连接，并且重新发起请求，总共重试三次
func ReceiveMsg(w http.ResponseWriter, r *http.Request) {
	bs, _ := io.ReadAll(r.Body)
	msg := wechat.NewMsg(bs)

	if msg == nil {
		echo(w, []byte("xml格式公众号消息接口，请勿手动调用"))
		return
	}

	// 非文本不回复(返回success表示不回复)
	switch msg.MsgType {
	// 未写的类型
	default:
		log.Printf("[%s] 未实现的消息类型 %s\n", msg.FromUserName, msg.MsgType)
		echo(w, success)
	case "event":
		switch msg.Event {
		default:
			log.Printf("[%s] 未实现的事件 %s\n", msg.FromUserName, msg.Event)
			echo(w, success)
		case "subscribe":
			log.Printf("[%s] 新增关注\n", msg.FromUserName)
			b := msg.GenerateEchoData(config.Wechat.SubscribeMsg)
			echo(w, b)
			return
		case "unsubscribe":
			log.Printf("[%s] 取消关注\n", msg.FromUserName)
			echo(w, success)
			return
		}
	// https://developers.weixin.qq.com/doc/offiaccount/Message_Management/Receiving_standard_messages.html
	case "voice":
		msg.Content = msg.Recognition
	case "text":

	}
	log.Printf("[%s] > %s\n", msg.FromUserName, msg.Content)

	// 如开启白名单校验 不在白名单的用户消息 直接跳过不回复
	list := config.Wechat.AllowList
	allowAll := len(list) > 0 && list[0] == "*"
	if !allowAll {
		if !contains(config.Wechat.AllowList, msg.FromUserName) {
			log.Printf("[%s] 用户不在allowList中\n", msg.FromUserName)
			echo(w, success)
			return
		}
	}

	// 敏感词检测
	if !fiter.Check(msg.Content) {
		warnWx := msg.GenerateEchoData(warn)
		echo(w, warnWx)
		return
	}

	var ch chan string
	v, ok := requests.Load(msg.MsgId)
	if !ok {
		ch = make(chan string)
		requests.Store(msg.MsgId, ch)
		// ch <- openai.Query(msg.FromUserName, msg.Content, time.Second*time.Duration(config.Wechat.Timeout))
		// ch <- "收到：" + msg.FromUserName + "：" + msg.Content
		ch <- bot.Query(msg.FromUserName, msg.Content)
	} else {
		ch = v.(chan string)
	}

	select {
	case result := <-ch:
		if !fiter.Check(result) {
			result = warn
		}
		bs := msg.GenerateEchoData(result)
		echo(w, bs)
		log.Printf("[%s] <<< %s\n", msg.FromUserName, result)
		requests.Delete(msg.MsgId)
	// 超时不要回答，会重试的
	case <-time.After(time.Second * 5):
		// log.Printf("[%s] timeout\n", msg.FromUserName)
	}
}

func Test(w http.ResponseWriter, r *http.Request) {
	msg := r.URL.Query().Get("msg")
	if !fiter.Check(msg) {
		echoJson(w, "", warn)
		return
	}
	log.Printf("[Test] > %s\n", msg)

	// s := openai.Query("0", msg, time.Second*5)
	// todo: 注意 这里test账号 需要做限制逻辑 避免被恶意滥用
	// s := bot.Query("test", msg)
	// 不走bot 直接避免被恶意滥用
	s := "收到：" + msg

	echoJson(w, s, "")
	log.Printf("[Test] <<< %s\n", s)
}

func echoJson(w http.ResponseWriter, replyMsg string, errMsg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	var code int
	var message = replyMsg
	if errMsg != "" {
		code = -1
		message = errMsg
	}
	data, _ := json.Marshal(map[string]interface{}{
		"code":    code,
		"message": message,
	})
	w.Write(data)
}

func echo(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// How to check if a slice contains an element in Go
// https://freshman.tech/snippets/go/check-if-slice-contains-element/
// https://play.golang.org/p/Qg_uv_inCek
// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
