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

	goog := google.New(CLIENT_ID, CLIENT_SECRET, CALLBACK_URL, SCOPES...)
	goog.SetAccessType("offline")

	goth.UseProviders(
		goog,
	)

	// google.SetAccessType("offline")

	p := pat.New()
	p.Get("/auth/{provider}/callback", authCallback)

	p.Get("/auth/{provider}", func(w http.ResponseWriter, r *http.Request) {
		gothic.BeginAuthHandler(w, r)
	})

	p.Get("/logout", func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/revoke", nil)
		if err != nil {
			fmt.Fprintf(w, "%+v\n", err)
		}

		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		q := req.URL.Query()
		q.Add("token", strings.Fields(r.Header.Get("Authorization"))[1])
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

	// insertToDB(user.AccessToken, user.RefreshToken)

	fmt.Println(user.AccessToken + " | " + user.RefreshToken)

	w.Header().Add("Authorization", "Bearer "+user.AccessToken)
	w.WriteHeader(200)
}

// func insertToDB(access_token, refresh_token string) bool {
// 	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI("mongodb://localhost:27017"))
// 	if err != nil {
// 		return false
// 	}

// 	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
// 		return false
// 	}

// 	usersCollection := client.Database("youtube-stats-app").Collection("users")

// 	// insert a single document into a collection
// 	// create a bson.D object
// 	user := bson.D{{"access_token", access_token}, {"refresh_token", refresh_token}}
// 	// insert the bson object using InsertOne()
// 	result, err := usersCollection.InsertOne(context.TODO(), user)
// 	// check for errors in the insertion
// 	if err != nil {
// 		return false
// 	}
// 	// display the id of the newly inserted object
// 	fmt.Println(result.InsertedID)
// 	return true
// }
