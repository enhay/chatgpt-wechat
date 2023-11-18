package openai

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/fishtailstudio/imgo"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	copenai "github.com/sashabaranov/go-openai"
	"github.com/zeromicro/go-zero/core/logx"

	"golang.org/x/net/proxy"
)

type Draw struct {
	Host   string
	APIKey string
	Proxy  string
}

func NewOpenaiDraw(host, key, proxy string) *Draw {
	return &Draw{
		Host:   host,
		APIKey: key,
		Proxy:  proxy,
	}
}

// WithProxy 设置代理
func (od *Draw) WithProxy(proxy string) *Draw {
	od.Proxy = proxy
	return od
}

func (od *Draw) Txt2Img(prompt string, ch chan string) error {
	cli := copenai.NewClientWithConfig(od.buildConfig())

	ch <- "start"

	imgReq := copenai.ImageRequest{
		Prompt:         prompt,
		N:              1,
		Model:          copenai.CreateImageModelDallE3,
		Size:           copenai.CreateImageSize1024x1024,
		Style:          copenai.CreateImageStyleVivid,
		ResponseFormat: copenai.CreateImageResponseFormatB64JSON,
	}

	imageRes, err := cli.CreateImage(context.Background(), imgReq)
	if err != nil {
		return err
	}

	// 读取
	if len(imageRes.Data) > 0 {

		imageBase64 := "data:image/png;base64," + imageRes.Data[0].B64JSON

		// 判断目录是否存在
		_, err = os.Stat("/tmp/image")
		if err != nil {
			err := os.MkdirAll("/tmp/image", os.ModePerm)
			if err != nil {
				fmt.Println("mkdir err:", err)
				return errors.New("绘画请求响应保存至目录失败，请重新尝试~")
			}
		}

		// 创建一个新的文件
		path := fmt.Sprintf("/tmp/image/%s.png", uuid.New().String())

		imgo.Load(imageBase64).Save(path)

		// 再将 image 信息发送到用户
		ch <- path
	}

	ch <- "stop"

	return nil
}

func (od *Draw) buildConfig() copenai.ClientConfig {
	config := copenai.DefaultConfig(od.APIKey)
	if od.Proxy != "" {
		if strings.HasPrefix(od.Proxy, "http") {
			proxyUrl, err := url.Parse(od.Proxy)
			if err != nil {
				logx.Error("proxy parse error", err)
			}
			transport := &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			}
			config.HTTPClient = &http.Client{
				Transport: transport,
			}
		} else {
			socks5Transport := &http.Transport{}
			dialer, _ := proxy.SOCKS5("tcp", od.Proxy, nil, proxy.Direct)
			socks5Transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
			config.HTTPClient = &http.Client{
				Transport: socks5Transport,
			}
		}
	}
	// trim last slash
	config.BaseURL = strings.TrimRight(od.Host, "/") + "/v1"

	return config
}
