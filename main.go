package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"

	"github.com/zmb3/spotify"
)

// redirectURI is the OAuth redirect URI for the application.
// You must register an application at Spotify's developer portal
// and enter this value.
const redirectURI = "http://localhost:8080/callback"

var (
	auth  = spotify.NewAuthenticator(redirectURI, spotify.ScopeUserReadPrivate, spotify.ScopeUserReadCurrentlyPlaying)
	ch    = make(chan *spotify.Client)
	state = "fdjlk1234u9023jif"
)

// thanks to https://gist.github.com/nanmu42/4fbaf26c771da58095fa7a9f14f23d27
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// first start an HTTP server
	http.HandleFunc("/callback", completeAuth)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Got request for:", r.URL.String())
	})
	go http.ListenAndServe(":8080", nil)

	url := auth.AuthURL(state)
	openBrowser(url)
	fmt.Println("Please log in to Spotify by visiting the following page in your browser:", url)

	// wait for auth to complete
	client := <-ch

	// use the client to make calls that require authorization
	user, err := client.CurrentUser()
	if err != nil {
		log.Fatalf("error with getting current user: %r", err)
	}
	fmt.Println("You are logged in as:", user.ID)

	//playing, err := client.PlayerCurrentlyPlaying()
	//if err != nil {
	//  log.Fatalf("error with getting currently playing: %r", err)
	//}
	//fmt.Printf("You are playing item ID: %s\n", playing.Item.ID)

	//analysis, err := client.GetAudioAnalysis(playing.Item.ID)
	//if err != nil {
	//  log.Fatalf("error with getting audio analysis: %r", err)
	//}

	//playing, err = client.PlayerCurrentlyPlaying()
	//if err != nil {
	//  log.Fatalf("error with getting currently playing (the second time): %r", err)
	//}
	//startBeat := findBeatStartIndex(time.Duration(playing.Progress)*time.Millisecond, analysis.Beats)
	//fmt.Printf("progress of song: %dms\n", playing.Progress)
	//fmt.Printf("index of beat we are on: %d\n", startBeat)
	fmt.Printf("Let the beat start...")

	//startTime := time.Now()
	//ls := &LoopStorage{
	//  Beats:            analysis.Beats,
	//  CurrentlyPlaying: playing,
	//  LastBeatIndex:    -1,
	//  LastUpdateTime:   startTime,
	//  StartTime:        startTime,
	//  TotalProgress:    time.Duration(playing.Progress) * time.Millisecond,
	//}

	loopError := make(chan error)
	ls := &LoopStorage {
		Client: client,
	}

	go startLoop(ls, loopError)

	<-loopError
}

func completeAuth(w http.ResponseWriter, r *http.Request) {
	tok, err := auth.Token(state, r)
	if err != nil {
		http.Error(w, "Couldn't get token", http.StatusForbidden)
		log.Fatal(err)
	}
	if st := r.FormValue("state"); st != state {
		http.NotFound(w, r)
		log.Fatalf("State mismatch: %s != %s\n", st, state)
	}
	// use the token to get an authenticated client
	client := auth.NewClient(tok)
	fmt.Fprintf(w, "Login Completed!")
	ch <- &client
}
