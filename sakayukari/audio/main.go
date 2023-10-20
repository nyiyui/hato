package audio

import (
	"os"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
)

var buffer *beep.Buffer

func init() {
	f, err := os.Open("/home/nyiyui/inaba/hato-private/melodie.mp3")
	if err != nil {
		panic(err)
	}
	func() {
		streamer, format, err := mp3.Decode(f)
		if err != nil {
			panic(err)
		}
		defer streamer.Close()
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		buffer = beep.NewBuffer(format)
		buffer.Append(streamer)
	}()
}

func Play() {
	oneshot := buffer.Streamer(0, buffer.Len())
	speaker.Play(oneshot)
}
