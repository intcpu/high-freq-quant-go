package notice

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
)

func NoticeLog(text string) error {
	client := http.DefaultClient
	cont := Content{
		Text: text,
	}
	msg := Message{
		MsgType: "text",
		Content: cont,
	}
	bytebody, _ := json.Marshal(&msg)
	data := bytes.NewReader(bytebody)
	resp, err := client.Post(LogUrl, "application/json", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		return errors.New(response.Msg)
	}
	if response.StatusMessage != "success" {
		return errors.New(response.StatusMessage)
	}
	return err
}

func ErrorLog(text string) error {
	client := http.DefaultClient
	cont := Content{
		Text: text,
	}
	msg := Message{
		MsgType: "text",
		Content: cont,
	}
	bytebody, _ := json.Marshal(&msg)
	data := bytes.NewReader(bytebody)
	resp, err := client.Post(ErrUrl, "application/json", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		return errors.New(response.Msg)
	}
	if response.StatusMessage != "success" {
		return errors.New(response.StatusMessage)
	}
	return err
}

func AlignLog(text string) error {
	client := http.DefaultClient
	cont := Content{
		Text: text,
	}
	msg := Message{
		MsgType: "text",
		Content: cont,
	}
	bytebody, _ := json.Marshal(&msg)
	data := bytes.NewReader(bytebody)
	//todo errurl
	resp, err := client.Post(LogUrl, "application/json", data)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}
	if response.Code != 0 {
		return errors.New(response.Msg)
	}
	if response.StatusMessage != "success" {
		return errors.New(response.StatusMessage)
	}
	return err
}
