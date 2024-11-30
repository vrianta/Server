package Server

import "fmt"

func Render(massages string) {
	view = append(view, massages)
	viewCount++
}

func startRender() {
	for i := 0; i < viewCount; i++ {
		fmt.Fprint(W, view[i])
	}
	// fmt.Fprint(W, view)
	view = []string{}
	viewCount = 0 //  Reseting view Index
}
