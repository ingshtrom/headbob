package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/zmb3/spotify"
)

// data for dealing with a processing loop
type LoopStorage struct {
	Client *spotify.Client

	CurrentlyPlaying *spotify.CurrentlyPlaying

	// all beats in the song
	Beats []spotify.Marker

	LastCheckForPlayingState time.Time

	// last beat to "hit" or reach in our progress
	LastBeatIndex int

	LastUpdateTime time.Time

	Mutex sync.Mutex

	StartTime time.Time

	TotalProgress time.Duration
}

func (ls *LoopStorage) findBeatStartIndex() int {
	// get the progress in seconds to compare with in the beats struct
	prog := ls.TotalProgress.Seconds()
	for i, m := range ls.Beats {
		if m.Start < prog && prog < m.Start+m.Duration {
			return i
		}
	}

	return -1
}

func (ls *LoopStorage) newSong(playing *spotify.CurrentlyPlaying, analysis *spotify.AudioAnalysis) {
	ls.Mutex.Lock()
	ls.CurrentlyPlaying = playing
	ls.LastBeatIndex = -1
	ls.TotalProgress = time.Duration(playing.Progress) * time.Millisecond
	ls.Beats = analysis.Beats
	ts := time.Unix(playing.Timestamp/1000, playing.Timestamp%1000)
	ls.LastUpdateTime = ts
	ls.StartTime = ts
	ls.Mutex.Unlock()
}

func (ls *LoopStorage) updateCurrentlyPlaying() {
	playing, err := ls.Client.PlayerCurrentlyPlaying()
	if err != nil {
		log.Fatalf("error with getting currently playing: %r", err)
	}

	analysis, err := ls.Client.GetAudioAnalysis(playing.Item.ID)
	if err != nil {
		log.Fatalf("error with getting audio analysis: %r", err)
	}

	ls.newSong(playing, analysis)
}

func startLoop(ls *LoopStorage, errorChannel chan error) {
	if ls.CurrentlyPlaying == nil {
		ls.updateCurrentlyPlaying()
	}
	if ls.LastBeatIndex == -1 {
		ls.LastBeatIndex = ls.findBeatStartIndex()
	}

	d, _ := json.Marshal(*ls)
	fmt.Printf("%s\n", d)

	for {
		ls.Mutex.Lock()
		newBeatIndex := ls.findBeatStartIndex()
		if newBeatIndex > ls.LastBeatIndex {
			fmt.Printf("beat: %d, %f => %s\n", newBeatIndex, ls.Beats[newBeatIndex].Duration, ls.CurrentlyPlaying.Item.Name)
			ls.LastBeatIndex = newBeatIndex
		}
		ls.TotalProgress = ls.TotalProgress + time.Since(ls.LastUpdateTime)
		ls.LastUpdateTime = time.Now()
		// a very naive 60FPS
		time.Sleep(16 * time.Millisecond)

		//if time.Now().Sub(ls.LastCheckForPlayingState) > 5*time.Second {
		//  go ls.updateCurrentlyPlaying()
		//  ls.LastCheckForPlayingState = time.Now()
		//}
		ls.Mutex.Unlock()
	}

	errorChannel <- fmt.Errorf("Something bad happened and we failed to keep the beat alive..")
}
