package goex

//http request 工具函数
import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func NewHttpRequest(client *http.Client, reqType string, reqUrl string, postData string, requstHeaders map[string]string) ([]byte, error) {
	req, _ := http.NewRequest(reqType, reqUrl, strings.NewReader(postData))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 5.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36")

	if requstHeaders != nil {
		for k, v := range requstHeaders {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("HttpStatusCode:%d ,Desc:%s", resp.StatusCode, string(bodyData)))
	}

	return bodyData, nil
}

func NewHttpRequestEx(client *http.Client, reqType string, reqUrl string, postData string, requstHeaders map[string]string) ([]byte, http.Header, error) {
	req, _ := http.NewRequest(reqType, reqUrl, strings.NewReader(postData))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 5.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.63 Safari/537.36")

	if requstHeaders != nil {
		for k, v := range requstHeaders {
			req.Header.Add(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.Header, err
	}

	if resp.StatusCode != 200 {
		return nil, resp.Header, errors.New(fmt.Sprintf("HttpStatusCode:%d ,Desc:%s", resp.StatusCode, string(bodyData)))
	}

	return bodyData, resp.Header, nil
}

func HttpGet(client *http.Client, reqUrl string) (map[string]interface{}, error) {
	respData, err := NewHttpRequest(client, "GET", reqUrl, "", nil)
	if err != nil {
		return nil, err
	}

	var bodyDataMap map[string]interface{}
	err = json.Unmarshal(respData, &bodyDataMap)
	if err != nil {
		log.Println(string(respData))
		return nil, err
	}
	return bodyDataMap, nil
}

func HttpGet2(client *http.Client, reqUrl string, headers map[string]string) (map[string]interface{}, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	respData, err := NewHttpRequest(client, "GET", reqUrl, "", headers)
	if err != nil {
		return nil, err
	}

	var bodyDataMap map[string]interface{}
	err = json.Unmarshal(respData, &bodyDataMap)
	if err != nil {
		log.Println("respData", string(respData))
		return nil, err
	}
	return bodyDataMap, nil
}

func HttpGet3(client *http.Client, reqUrl string, headers map[string]string) ([]interface{}, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	respData, err := NewHttpRequest(client, "GET", reqUrl, "", headers)
	if err != nil {
		return nil, err
	}

	var bodyDataMap []interface{}
	err = json.Unmarshal(respData, &bodyDataMap)
	if err != nil {
		log.Println("respData", string(respData))
		return nil, err
	}
	return bodyDataMap, nil
}

func HttpGet4(client *http.Client, reqUrl string, headers map[string]string, result interface{}) error {
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	respData, err := NewHttpRequest(client, "GET", reqUrl, "", headers)
	if err != nil {
		println(reqUrl + err.Error())
		return err
	}
	err = json.Unmarshal(respData, result)
	if err != nil {
		log.Printf("HttpGet4 - json.Unmarshal failed : %v, resp %s", err, string(respData))
		return err
	}

	return nil
}

func HttpGet5(client *http.Client, reqUrl string, headers map[string]string, result interface{}) (error, http.Header) {
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	respData, respHeader, err := NewHttpRequestEx(client, "GET", reqUrl, "", headers)
	if err != nil {
		print(err.Error())
		return err, respHeader
	}
	err = json.Unmarshal(respData, result)
	if err != nil {
		log.Printf("HttpGet4 - json.Unmarshal failed : %v, resp %s", err, string(respData))
		return err, respHeader
	}

	return nil, respHeader
}

func HttpGet6(client *http.Client, reqUrl string, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Content-Type"]; !ok {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	respData, err := NewHttpRequest(client, "GET", reqUrl, "", headers)
	if err != nil {
		println(reqUrl + err.Error())
		return nil, err
	}
	return respData, nil
}

func HttpPostForm(client *http.Client, reqUrl string, postData url.Values) ([]byte, error) {
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded"}
	return NewHttpRequest(client, "POST", reqUrl, postData.Encode(), headers)
}

func HttpPostForm2(client *http.Client, reqUrl string, postData url.Values, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	return NewHttpRequest(client, "POST", reqUrl, postData.Encode(), headers)
}

func HttpPostForm3(client *http.Client, reqUrl string, postData string, headers map[string]string) ([]byte, error) {
	return NewHttpRequest(client, "POST", reqUrl, postData, headers)
}

func HttpPostForm4(client *http.Client, reqUrl string, postData interface{}, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/json"
	data, _ := json.Marshal(postData)
	return NewHttpRequest(client, "POST", reqUrl, string(data), headers)
}

func HttpPostJson(client *http.Client, reqUrl string, body string, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/json"
	return NewHttpRequest(client, "POST", reqUrl, body, headers)
}

func HttpDeleteForm(client *http.Client, reqUrl string, postData url.Values, headers map[string]string) ([]byte, error) {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	return NewHttpRequest(client, "DELETE", reqUrl, postData.Encode(), headers)
}

func HttpDeleteForm3(client *http.Client, reqUrl string, postData string, headers map[string]string) ([]byte, error) {
	return NewHttpRequest(client, "DELETE", reqUrl, postData, headers)
}

func HttpPutForm3(client *http.Client, reqUrl string, postData string, headers map[string]string) ([]byte, error) {
	return NewHttpRequest(client, "PUT", reqUrl, postData, headers)
}
