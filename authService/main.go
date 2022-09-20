package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/pat"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

type AppUser struct {
	Id           string
	AccessToken  string
	RefreshToken string
}

type Token struct {
	accessToken string   `json: "access_token"`
	expiresIn   int      `json: "expires_in"`
	scope       []string `json: "scope"`
	tokenType   string   `json: "token_type"`
	id_token    string   `json: "id_token"`
}

var SCOPES = []string{"profile", "https://www.googleapis.com/auth/youtube.readonly"}

const (
	USER_ID_LENGTH = 15
	CLIENT_ID      = "1056709788044-9k78m5n55u6mbrp6fp004hq0fi5r0ltj.apps.googleusercontent.com"
	CLIENT_SECRET  = "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"
	CALLBACK_URL   = "http://localhost:8080/auth/google/callback"
)

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

	p := pat.New()
	p.Get("/auth/{provider}/callback", authCallback)

	p.Get("/auth/{provider}", func(w http.ResponseWriter, r *http.Request) {
		gothic.BeginAuthHandler(w, r)
	})

	p.Get("/api/sessions", checkIfSessionExists)

	p.Get("/api/refresh", refreshedToken)

	p.Get("/logout", logout)

	log.Println("listening on localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", p))
}

func refreshedToken(w http.ResponseWriter, r *http.Request) {
	access_token := strings.Fields(r.Header.Get("Authorization"))[1]
	var dataJson = []byte(`{
		"client_id": "` + CLIENT_ID + `",
		"client_secret": "` + CLIENT_SECRET + `",
		"refresh_token": "` + getRefreshToken(access_token) + `",
		"grant_type": "refresh_token"
	}`)
	req, err := http.NewRequest("POST", "https://www.googleapis.com/oauth2/v4/token", bytes.NewBuffer(dataJson))
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	if err != nil {
		fmt.Printf("%+v\n", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	newToken := strings.Fields(string(body))[2]
	newToken = newToken[:len(newToken)-2]
	newToken = newToken[1:]

	if newToken != "invalid_request" {
		changeStoredUserData(access_token, newToken)
		w.Header().Add("Authorization", "Bearer "+newToken)
		w.WriteHeader(200)
		return
	}
	w.WriteHeader(401)
}

func changeStoredUserData(oldToken, newToken string) {
	oldFile, err := os.Open("./user_files/" + oldToken + ".txt")
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
	defer oldFile.Close()

	scanner := bufio.NewScanner(oldFile)
	i := 0
	var neededLines []string
	for scanner.Scan() {
		if i != 1 {
			neededLines = append(neededLines, scanner.Text())
		}
		i++
	}

	newFile, err := os.Create("./user_files/" + newToken + ".txt")
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
	defer newFile.Close()

	_, err = newFile.WriteString(neededLines[0] + "\naccess_token: " + newToken + "\nrefresh_token" + neededLines[1])
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
	removeUsersFile(oldToken)
}

func getRefreshToken(access_token string) string {
	file, err := os.Open("./user_files/" + access_token + ".txt")
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		if i == 2 {
			return strings.Fields(scanner.Text())[1]
		}
		i++
	}
	return ""
}

func logout(w http.ResponseWriter, r *http.Request) {
	access_token := strings.Fields(r.Header.Get("Authorization"))[1]
	req, err := http.NewRequest("POST", "https://oauth2.googleapis.com/revoke", nil)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	q := req.URL.Query()
	q.Add("token", access_token)
	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
	}

	defer resp.Body.Close()

	err = removeUsersFile(access_token)
	if err != nil {
		fmt.Fprint(w, "Can't delete user data")
	}

	fmt.Println("Response Status: " + resp.Status)
	gothic.Logout(w, r)
}

func authCallback(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	currentUser := AppUser{
		Id:           generateId(),
		AccessToken:  user.AccessToken,
		RefreshToken: user.RefreshToken,
	}
	err = save(currentUser)
	if err != nil {
		fmt.Fprintf(w, "%+v", err)
		return
	}

	w.Header().Add("Authorization", "Bearer "+user.AccessToken)
	w.WriteHeader(200)
}

func generateSessionKey() []byte {
	key := make([]byte, 64)

	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}

	return key
}

func generateId() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, USER_ID_LENGTH)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:USER_ID_LENGTH]
}

func save(user AppUser) error {
	f, err := os.Create("./user_files/" + user.AccessToken + ".txt")
	if err != nil {
		fmt.Printf("Error while creating: %v\n", err)
		return err
	}
	defer f.Close()

	_, err = f.WriteString("id: " + user.Id + "\naccess_token: " + user.AccessToken + "\nrefresh_token: " + user.RefreshToken)

	if err != nil {
		fmt.Printf("Error while writing: %v\n", err)
		return err
	}

	return nil
}

func removeUsersFile(access_token string) error {
	err := os.Remove("./user_files/" + access_token + ".txt")
	if err != nil {
		return err
	}
	return nil
}

func checkIfSessionExists(w http.ResponseWriter, r *http.Request) {
	access_token := strings.Fields(r.Header.Get("Authorization"))[1]
	if _, err := os.Stat("./user_files/" + access_token + ".txt"); err == nil {
		w.WriteHeader(200)
		return
	}
	w.WriteHeader(404)
	return
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
