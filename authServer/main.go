package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"crypto/rand"
	"log"

	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

var SCOPES = []string{"profile", "https://www.googleapis.com/auth/youtube.readonly"}

const (
	CLIENT_ID     = "1056709788044-9k78m5n55u6mbrp6fp004hq0fi5r0ltj.apps.googleusercontent.com"
	CLIENT_SECRET = "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"
	CALLBACK_URL  = "http://localhost:8080/auth/google/callback"
)

func generateSessionKey() []byte {
	key := make([]byte, 64)

	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	return key
}

func main() {

	maxAge := 86400 * 30 // 30 days
	isProd := false      // Set to true when serving over https

	store := sessions.NewCookieStore(generateSessionKey())
	store.MaxAge(maxAge)
	store.Options.Path = "/"
	store.Options.HttpOnly = true // HttpOnly should always be enabled
	store.Options.Secure = isProd

	gothic.Store = store

	goth.UseProviders(
		google.New(CLIENT_ID, CLIENT_SECRET, CALLBACK_URL, SCOPES...),
	)

	p := pat.New()
	p.Get("/auth/{provider}/callback", authCallback)

	p.Get("/auth/{provider}", func(w http.ResponseWriter, r *http.Request) {
		gothic.BeginAuthHandler(w, r)
	})

	p.Get("/channelData", getChannelData)

	p.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		gothic.Logout(w, r)
	})

	p.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello world")
	})

	log.Println("listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", p))
}

func getAccessTokenFromCookies(r *http.Request) string {
	accessToken, err := gothic.GetFromSession("access_token", r)

	if err != nil {
		panic(err)
	}

	return accessToken
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
		fmt.Fprintf(w, "404 Not Found")
	}

	return response.Items[0].Id
}

func getChannelData(w http.ResponseWriter, r *http.Request) {

	channelId := getUserChannelId(w, r, r.Header.Get("Authorization"))
	// fmt.Fprintf(w, "%+v\n", channelId)
	w.Write([]byte(channelId))
}

func authCallback(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	w.Header().Add("Authorization", user.AccessToken)
	w.WriteHeader(200)
}
