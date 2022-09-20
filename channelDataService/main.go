package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("/api/channelData", getChannelData)
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func getUserChannelId(w http.ResponseWriter, r *http.Request, token string) string {
	req, err := http.NewRequest("GET", "https://www.googleapis.com/youtube/v3/channels", nil)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	q := req.URL.Query()
	q.Add("part", "id")
	q.Add("mine", "true")
	q.Add("access_token", token)
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	defer resp.Body.Close()

	fmt.Println("Response Status: " + resp.Status)

	if strings.Fields(resp.Status)[0] == "401" {
		if sessionExists(token) {
			w.WriteHeader(500)
			return ""
		} else {
			w.WriteHeader(401)
			return ""
			// return getUserChannelId(w, r, getRefreshedToken(token))
		}
	}

	var response struct {
		Kind  string `json:"kind"`
		Items []struct {
			Kind string `json:"kind"`
			Id   string `json:"id"`
		} `json:"items"`
	}

	body, _ := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &response)

	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("404 Not Found"))
	}

	return response.Items[0].Id
}

func getRefreshedToken(expiredToken string) string {
	req, err := http.NewRequest("GET", "http://localhost:8081/api/refresh", nil)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	req.Header.Add("Authorization", expiredToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	defer resp.Body.Close()
	return expiredToken
}

func sessionExists(token string) bool {
	req, err := http.NewRequest("GET", "http://localhost:8081/api/sessions", nil)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	req.Header.Add("Authorization", token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	defer resp.Body.Close()

	if strings.Fields(resp.Status)[0] == "404" {
		return false
	} else {
		return true
	}
}

func getChannelData(w http.ResponseWriter, r *http.Request) {
	channelId := getUserChannelId(w, r, strings.Fields(r.Header.Get("Authorization"))[1])
	w.Write([]byte(channelId))
}
