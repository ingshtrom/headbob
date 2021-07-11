// originated from https://github.com/zmb3/spotify/blob/master/examples/authenticate/authcode/authenticate.go
//
// This example demonstrates how to authenticate with Spotify using the authorization code flow.
// In order to run this example yourself, you'll need to:
//
//  1. Register an application at: https://developer.spotify.com/my-applications/
//       - Use "http://localhost:8080/callback" as the redirect URI
//  2. Set the SPOTIFY_ID environment variable to the client ID you got in step 1.
//  3. Set the SPOTIFY_SECRET environment variable to the client secret from step 1.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"time"

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

func findBeatStartIndex(progress time.Duration, beats []spotify.Marker) int {
	// get the progress in seconds to compare with in the beats struct
	prog := progress.Seconds()
	for i, m := range beats {
		if m.Start < prog && prog < m.Start + m.Duration {
			return i
		}
	}

	return -1
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

	playing, err := client.PlayerCurrentlyPlaying()
	if err != nil {
		log.Fatalf("error with getting currently playing: %r", err)
	}
	fmt.Printf("You are playing item ID: %s\n", playing.Item.ID)

	//features, err := client.GetAudioFeatures(playing.Item.ID)
	//if err != nil {
	//  log.Fatalf("error with getting audio features: %r", err)
	//}
	//for _, feature := range features {
	//  d, e := json.Marshal(feature)
	//  fmt.Printf("data: %s, err: %r\n", string(d), e)
	//}

	analysis, err := client.GetAudioAnalysis(playing.Item.ID)
	if err != nil {
		log.Fatalf("error with getting audio analysis: %r", err)
	}
	//d, _ := json.Marshal(analysis.Beats)
	//fmt.Printf("analysis of beats: %s\n", string(d))

	playing, err = client.PlayerCurrentlyPlaying()
	if err != nil {
		log.Fatalf("error with getting currently playing (the second time): %r", err)
	}
	startTime := time.Now()
	startBeat := findBeatStartIndex(time.Duration(playing.Progress) * time.Millisecond, analysis.Beats)
	fmt.Printf("progress of song: %dms\n", playing.Progress)
	fmt.Printf("index of beat we are on: %d\n", startBeat)
	fmt.Printf("Let the beat start...")

	//nextBeat := startBeat
	lastTime := startTime
	totalProgress := time.Duration(playing.Progress) * time.Millisecond
	// we need to operate on the following data for each loop
	// TODO: note down what variables are needed for each iteration
	for {
		beatIndex := findBeatStartIndex(totalProgress, analysis.Beats)
		fmt.Printf("beat: %d\n", beatIndex)
		durationOfBeat := analysis.Beats[beatIndex].Duration
		fmt.Printf("--> %fs\n", durationOfBeat)
		time.Sleep(time.Duration(durationOfBeat * 1000000000))
		totalProgress = totalProgress + time.Since(lastTime)
		lastTime = time.Now()

		// TODO: every N beats, or more likely every 10 seconds, we should make
		// sure that we're still setting the beats for the correct song
	}
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
