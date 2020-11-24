package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func MakeNoobaaRequest(api string, method string, params string) (string, error){

	auth_token_content, err := ioutil.ReadFile("/etc/noobaa/auth/auth_token")
	if err != nil {
		log.Fatal(err)
	}

	auth_token := string(auth_token_content)

	url := "https://noobaa-mgmt.noobaa.svc:443/rpc"
	requestBody := `
    {
  	"api":"`+api+`",
    "method":"`+method+`",
	"auth_token":"`+auth_token+`",
    "params": `+params+`
    }`
	var jsonStr = []byte(requestBody)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	if err!=nil {
		return "", err
	}
	fmt.Println("response Body:", string(body))
	return string(body),nil
}