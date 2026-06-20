package ginp

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Data struct {
	Key   string `respJson:"key"`
	Value string `respJson:"value"`
}

func GetJsonUrlBytes(jsonUrl string) ([]byte, error) {
	// 发送HTTP请求到自己的API
	resp, err := http.Get(jsonUrl)
	if err != nil {
		// 处理请求错误
		fmt.Println("请求失败：", err)

		return nil, err
	}

	defer resp.Body.Close()

	// 读取响应内容
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// 处理读取响应错误
		fmt.Println("读取响应失败：", err)
		return nil, err
	}

	return body, nil
}

func JsonToGinH(jsonUrl string) (gin.H, error) {
	// 读取响应内容
	body, err := GetJsonUrlBytes(jsonUrl)

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func formatSuccessMsg(messages ...string) (msg string) {
	msg = successMsgDefault
	if len(messages) > 0 {
		for _, message := range messages {
			msg += message
		}
	}
	return msg
}
func formatFailMsg(messages ...string) (msg string) {
	msg = failMsgDefault
	if len(messages) > 0 {
		for _, message := range messages {
			msg += message
		}
	}
	return msg
}
