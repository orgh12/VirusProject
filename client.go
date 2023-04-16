package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/aarzilli/nucular"
	"github.com/aarzilli/nucular/rect"
	_ "github.com/aarzilli/nucular/style"
	nstyle "github.com/aarzilli/nucular/style"
	"github.com/disintegration/imaging"
	"golang.org/x/image/draw"
	"image"
	"image/jpeg"
	"log"
	"net"
	_ "runtime/pprof"
	_ "runtime/trace"
	"time"
)

var img image.Image

func displayimg(conn net.Conn) {

	for {
		buf := make([]byte, 1024*1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println(err)
			return
		}
		encoded := buf[:n]
		decoded, err := base64.StdEncoding.DecodeString(string(encoded))
		if err != nil {
			log.Println(err)
			continue
		}
		img, err = jpeg.Decode(bytes.NewReader(decoded))
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

type overview struct {
	HeaderAlign nstyle.HeaderAlign
	Theme       nstyle.Theme
}

func (od *overview) overviewfunc(w *nucular.Window) {
	mw := w.Master()

	style := mw.Style()
	style.NormalWindow.Header.Align = od.HeaderAlign
	if w.TreePush(nucular.TreeTab, "Image & Custom", false) {

		if img != nil {
			resized := imaging.Resize(img, w.LayoutAvailableWidth(), w.LayoutAvailableHeight(), imaging.Lanczos)
			bounds := resized.Bounds()
			img2 := image.NewRGBA(bounds)
			draw.Draw(img2, bounds, resized, image.Point{}, draw.Src)
			w.RowScaled(img2.Bounds().Dy()).StaticScaled(img2.Bounds().Dx())
			w.Image(img2)
		} else {
			w.Row(25).Dynamic(1)
			w.Label("could not load example image", "LC")
		}

		w.RowScaled(335).StaticScaled(500)
		w.TreePop()
	}
}

type menu struct {
	Name     string
	Title    string
	Flags    nucular.WindowFlags
	UpdateFn func() func(*nucular.Window)
}

var theme nstyle.Theme = nstyle.DarkTheme

var menu1 = menu{"menu", "menu", 0, func() func(*nucular.Window) {
	od := &overview{}
	od.Theme = theme
	return od.overviewfunc
}}

func multiDemo(w *nucular.Window) {
	//mw := w.Master()

	//style := mw.Style()
	//style.NormalWindow.Header.Align = od.HeaderAlign
	if w.TreePush(nucular.TreeTab, "Image & Custom", false) {
		w.Master().PopupOpen("screen", nucular.WindowDefaultFlags|nucular.WindowNonmodal|0, rect.Rect{0, 0, 400, 300}, true, menu1.UpdateFn())
		//if img != nil {
		//	resized := imaging.Resize(img, w.LayoutAvailableWidth(), w.LayoutAvailableHeight(), imaging.Lanczos)
		//	bounds := resized.Bounds()
		//	img2 := image.NewRGBA(bounds)
		//	draw.Draw(img2, bounds, resized, image.Point{}, draw.Src)
		//	w.RowScaled(img2.Bounds().Dy()).StaticScaled(img2.Bounds().Dx())
		//	w.Image(img2)
		//} else {
		//	w.Row(25).Dynamic(1)
		//	w.Label("could not load example image", "LC")
		//}

		w.RowScaled(335).StaticScaled(500)
		w.TreePop()
	}
	if w.TreePush(nucular.TreeTab, "closing", false) {
		w.RowScaled(25).Dynamic(1)
		w.TreePop()
	}
}

const scaling = 1.8

var Wnd nucular.MasterWindow

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:9090")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()
	go displayimg(conn)

	Wnd = nucular.NewMasterWindow(0, "menu", func(w *nucular.Window) {})
	Wnd.PopupOpen("menu", nucular.WindowTitle|nucular.WindowBorder|nucular.WindowMovable|nucular.WindowScalable|nucular.WindowNonmodal, rect.Rect{0, 0, 400, 300}, true, multiDemo)
	Wnd.SetStyle(nstyle.FromTheme(theme, scaling))
	go func() {
		for {
			time.Sleep(time.Millisecond)
			Wnd.Changed()
		}
	}()
	Wnd.Main()

	//wnd.SetStyle(style.FromTheme(style.DarkTheme, 2.0))
	//wnd.Main()

}

//func updatefn(w *nucular.Window) {
//	if w.TreePush(nucular.TreeTab, "Image & Custom", false) {
//
//		if img3 != nil {
//			w.RowScaled(img3.Bounds().Dy()).StaticScaled(img3.Bounds().Dx())
//			w.Image(img3)
//		} else {
//			w.Row(25).Dynamic(1)
//			w.Label("could not load example image", "LC")
//		}
//
//		w.RowScaled(335).StaticScaled(500)
//		w.TreePop()
//	}
//}
