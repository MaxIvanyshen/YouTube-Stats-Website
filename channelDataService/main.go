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
	q.Add("part", "snippet")
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

func getChannelData(w http.ResponseWriter, r *http.Request) {
	channelId := getUserChannelId(w, r, strings.Fields(r.Header.Get("Authorization"))[1])
	w.Write([]byte(channelId))
}
