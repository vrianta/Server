package server

import (
	"fmt"
	"net/http"
)

type RenderHandeler struct {
	view      []string
	viewCount int
	W         http.ResponseWriter
}

func NewRenderHandlerObj(_w http.ResponseWriter) RenderHandeler {
	return RenderHandeler{
		view:      make([]string, 0),
		viewCount: 0,
		W:         _w,
	}
}

func (rh *RenderHandeler) Render(massages string) {
	rh.view = append(rh.view, massages)
	rh.viewCount++
}

func (rh *RenderHandeler) StartRender() {
	for i := 0; i < rh.viewCount; i++ {
		fmt.Println("Rendering :", rh.view[i])
		fmt.Fprint(rh.W, rh.view[i])
	}
	// fmt.Fprint(W, view)
	rh.view = []string{}
	rh.viewCount = 0 //  Reseting view Index
}
