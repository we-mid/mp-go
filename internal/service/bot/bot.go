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
		return err.Error()
	}
	defer resp.Body.Close()

	var result Result
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &result)
	return result.Text
}
