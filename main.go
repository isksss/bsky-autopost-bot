package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const name = "bsky-autopost-bot"
const version = "0.0.1"

var (
	// flag
	versionFlag = flag.Bool("v", false, "show version")
	postFlag    = flag.String("p", "", "post text")

	// url
	url     = "https://bsky.social/xrpc/com.atproto.server.createSession"
	postUrl = "https://bsky.social/xrpc/com.atproto.repo.createRecord"

	// user
	user = User{}
)

type User struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type Token struct {
	AccessJwt  string `json:"accessJwt"`
	RefreshJwt string `json:"refreshJwt"`
}

type Record struct {
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

type PostData struct {
	Repo       string `json:"repo"`
	Collection string `json:"collection"`
	Record     Record `json:"record"`
}

func init() {
	flag.Parse()

	// env
	user.Identifier = os.Getenv("BSKY_USERNAME")
	user.Password = os.Getenv("BSKY_PASSWORD")

	// check env
	if user.Identifier == "" || user.Password == "" {
		log.Fatal("BSKY_USERNAME and BSKY_PASSWORD are required")
	}
}

func getToken() (Token, error) {
	var token Token
	// create body
	payload, err := json.Marshal(user)
	if err != nil {
		return token, err
	}

	// request
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()

	// check status code
	if resp.StatusCode != http.StatusOK {
		return token, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	// decode
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return token, err
	}

	return token, nil
}

func postRecord(token Token, text string) error {
	// create body
	payload, err := json.Marshal(PostData{
		Repo:       user.Identifier,
		Collection: "app.bsky.feed.post",
		Record:     Record{Text: text, CreatedAt: time.Now()},
	})
	if err != nil {
		return err
	}

	// create header
	header := http.Header{}
	// bearer token
	header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessJwt))
	// content type
	header.Set("Content-Type", "application/json")

	// request

	req, err := http.NewRequest(http.MethodPost, postUrl, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header = header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	type Response struct {
		Uri string `json:"uri"`
		Cid string `json:"cid"`
	}

	// decode
	var res Response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	fmt.Printf("Post: %s\n", res.Uri)

	return nil
}

func run() error {
	token, err := getToken()
	if err != nil {
		return err
	}

	// post
	if *postFlag != "" {
		if err := postRecord(token, *postFlag); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if *versionFlag {
		fmt.Printf("Version: %s\n", version)
		return
	}

	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
