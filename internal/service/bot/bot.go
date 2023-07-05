package bot

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"openai/internal/config"
	"regexp"
	"time"
)

func Query(userId string, prompt string) string {
	p := newPayload(userId, prompt)
	bs, _ := json.Marshal(&p)

	client := &http.Client{Timeout: time.Second * 200}
	req, _ := http.NewRequest("POST", config.Bot.Api, bytes.NewReader(bs))
	s := base64.StdEncoding.EncodeToString([]byte(config.Bot.User + ":" + config.Bot.Pass))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Basic "+s)

	// 设置代理
	if config.Http.Proxy != "" {
		proxyURL, _ := url.Parse(config.Http.Proxy)
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	resp, err := client.Do(req)
	if err == nil && resp.StatusCode != http.StatusOK {
		err = errors.New(resp.Status)
	}
	if err != nil {
		// 避免暴露 bot.api url信息
		m := regexp.MustCompile(`".+?"(:.*tcp)`)
		return m.ReplaceAllString(err.Error(), "\"******\"$1")
	}
	defer resp.Body.Close()

	var result Result
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &result)
	text := result.Text

	// 优化bot返回的markdown 替换如`1\.`=>`1.`
	// m := regexp.MustCompile(`(\d)\\\.`)
	// text = m.ReplaceAllString(text, "$1.")
	return text
}
