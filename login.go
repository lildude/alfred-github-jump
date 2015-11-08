package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	ServerAddress = "http://127.0.0.1:7024"
	ServerBind    = ":7024"
	loginComplete sync.Mutex
)

func handleMain(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}

func handleGitHubLogin(w http.ResponseWriter, r *http.Request) {
	url := OAuthConf.AuthCodeURL(OAuthStateString, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	state := r.FormValue("state")
	if state != OAuthStateString {
		fmt.Printf("invalid oauth state, expected '%s', got '%s'\n", OAuthStateString, state)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	code := r.FormValue("code")
	token, err := OAuthConf.Exchange(oauth2.NoContext, code)
	if err != nil {
		fmt.Printf("oauthConf.Exchange() failed with '%s'\n", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	log.Printf("got token %#v", token)
	if err := saveToken(token); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	oauthClient := OAuthConf.Client(oauth2.NoContext, token)
	client := github.NewClient(oauthClient)
	user, _, err := client.Users.Get("")
	if err != nil {
		log.Printf("client.Users.Get() failed with %q", err)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	}

	fmt.Printf("Logged in as GitHub user: %s\n", *user.Login)
	loginComplete.Unlock()
}

func loginCommand() error {
	ln, err := net.Listen("tcp", ServerBind)
	if err != nil {
		return err
	}

	http.HandleFunc("/", handleMain)
	http.HandleFunc("/login", handleGitHubLogin)
	http.HandleFunc("/github_oauth_cb", handleGitHubCallback)

	log.Printf("Started running on %s\n", ServerAddress)

	loginComplete.Lock()
	go http.Serve(ln, nil)

	loginComplete.Lock()
	return ln.Close()
}