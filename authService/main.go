package main

import (
	"fmt"
	"net/http"
	"strings"

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

	p.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/revoke?token="+strings.Fields(r.Header.Get("Authorization"))[1], nil)
		if err != nil {
			fmt.Fprintf(w, "%+v\n", err)
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		q := req.URL.Query()
		req.URL.RawQuery = q.Encode()
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(w, "%+v\n", err)
		}

		defer resp.Body.Close()

		fmt.Println("Response Status: " + resp.Status)
		gothic.Logout(w, r)
	})

	log.Println("listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", p))
}

func authCallback(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	w.Header().Add("Authorization", "Bearer "+user.AccessToken)
	w.WriteHeader(200)
}
