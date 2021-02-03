package main

import (
	"time"

	"github.com/alexmaze/clink/lib/spinner"
)

func main() {
	sp := spinner.New().Start()

	stop1 := time.NewTimer(time.Second * 1)
	stop2 := time.NewTimer(time.Second * 2)
	stop3 := time.NewTimer(time.Second * 3)

OUT:
	for {
		select {
		case <-stop1.C:
			sp.SetMsg("Working on 1, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
		case <-stop2.C:
			sp.CheckPoint(spinner.IconCheck, spinner.ColorBlue, "what", spinner.ColorPurple)
			// sp.CheckPoint("O", spinner.ColorBlue, "what", spinner.ColorPurple)
			sp.SetMsg("Working on 2, 12345678901234567890123456789012345678901234567890123456789012345678901234567890")
		case <-stop3.C:
			sp.Success("Everything good!")
			sp.Success("Bye")
			break OUT
		}
	}

}
