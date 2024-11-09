/*
@author: sk
@date: 2024/11/9
*/
package main

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

func TestMp3(t *testing.T) {
	err := speaker.Init(SampleRate, SampleRate/10)
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open("/Users/bytedance/Documents/go/my_midi/res/8.mp3")
	if err != nil {
		log.Fatal(err)
	}
	stream, _, err := mp3.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		err = stream.Seek(0)
		if err != nil {
			log.Fatal(err)
		}
		speaker.Play(stream)
		time.Sleep(time.Second)
	}
	select {}
}
