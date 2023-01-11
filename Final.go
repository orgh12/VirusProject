package main

import (
	"fmt"
	"github.com/mitchellh/go-ps"
	"golang.org/x/sys/windows"
	"log"
	"os"
	"os/exec"
)

const processEntrySize = 568

var apps []string = []string{"explorer.exe", "gofmt.exe", "conhost.exe", "svchost.exe", "SpeechRuntime.exe"}

func isInArray(arr []string, str string) bool {

	for i := 0; i < len(arr); i++ {
		if arr[i] == str {
			return true
		}
	}
	return false
}

func whatever() {
	processList, err := ps.Processes()
	if err != nil {
		log.Println("ps.Processes() Failed")
		return
	}
	for {
		processList2, err2 := ps.Processes()
		if err2 != nil {
			log.Println("ps.Processes() Failed")
			return
		}
		for x := range processList {
			var process ps.Process
			process = processList[x]
			//fmt.Println("p1", process.Executable())
			isIn := false
			//fmt.Println(process.Executable())
			for y := range processList2 {
				var process2 ps.Process
				process2 = processList2[y]
				//fmt.Println(process.Executable())
				if process.Pid() == process2.Pid() {
					isIn = true
				}
			}
			if isIn == false {
				fmt.Println(process.Pid(), " ", process.Executable())
				if !isInArray(apps, process.Executable()) {
					kill := exec.Command("TASKKILL", "/T", "/F", "/IM", process.Executable())
					fmt.Println("TASKKILL", "/T", "/F", "/IM", process.Executable())
					kill.Stderr = os.Stderr
					kill.Stdout = os.Stdout
					err := kill.Run()
					if err != nil {
					}
				}
			}

			// do os.* stuff on the pid
		}
		processList = processList2
	}
}

func getnewapp() {
	h, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		panic(e)
	}
	p := windows.ProcessEntry32{Size: processEntrySize}
	for {
		h2, e2 := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
		if e2 != nil {
			panic(e2)
		}
		p2 := windows.ProcessEntry32{Size: processEntrySize}
		e := windows.Process32Next(h, &p)
		e2 = windows.Process32Next(h2, &p2)
		if e != nil || e2 != nil {
			break
		}
		s := windows.UTF16ToString(p.ExeFile[:])
		s2 := windows.UTF16ToString(p2.ExeFile[:])
		if s != s2 {
			println(s2)
		}
		h, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
		if e != nil {
			panic(e)
		}
		p := windows.ProcessEntry32{Size: processEntrySize}
		if h != 4 || p != p2 {
			//do nothing
		}
	}
}

func main() {
	//getnewapp()
	whatever()
	//processList, err := ps.Processes()
	//if err != nil {
	//	log.Println("ps.Processes() Failed")
	//	return
	//}
	//for x := range processList {
	//	var process ps.Process
	//	process = processList[x]
	//	fmt.Println(process.Executable())
	//}

}
