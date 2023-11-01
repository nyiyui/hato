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

type OnePlay struct {
	ctrl *beep.Ctrl
}

func Play() *OnePlay {
	op := OnePlay{&beep.Ctrl{Streamer: buffer.Streamer(0, buffer.Len())}}
	speaker.Play(op.ctrl)
	return &op
}

func (op *OnePlay) Stop() {
	speaker.Lock()
	op.ctrl.Streamer = nil
	speaker.Unlock()
}
