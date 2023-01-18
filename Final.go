package main

import (
	"fmt"
	"github.com/mitchellh/go-ps"
	"log"
	"os"
	"os/exec"
)

const processEntrySize = 568

var apps []string = []string{"explorer.exe", "gofmt.exe", "conhost.exe", "svchost.exe", "SpeechRuntime.exe", "git.exe"}

func difference(a, b []ps.Process) []string {
	mb := make(map[int]struct{}, len(b))
	for _, x := range b {
		mb[x.Pid()] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x.Pid()]; !found {
			diff = append(diff, x.Executable())
		}
	}
	return diff
}

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
		diff := difference(processList, processList2)
		for x := range diff {
			if !isInArray(apps, diff[x]) {
				fmt.Println(diff[x])
				kill := exec.Command("TASKKILL", "/T", "/F", "/IM", diff[x])
				fmt.Println("TASKKILL", "/T", "/F", "/IM", diff[x])
				kill.Stderr = os.Stderr
				kill.Stdout = os.Stdout
				err := kill.Run()
				if err != nil {
				}
			}
		}
		//for x := range processList {
		//	var process ps.Process
		//	process = processList[x]
		//	if process.Executable() == "cmd.exe" {
		//		fmt.Println("open?")
		//	}
		//	isIn := false
		//	for y := range processList2 {
		//		var process2 ps.Process
		//		process2 = processList2[y]
		//		if process2.Executable() == "cmd.exe" || process.Executable() == "cmd.exe" {
		//			fmt.Println("open2?")
		//			fmt.Println(process.Executable())
		//			fmt.Println(process)
		//			fmt.Println(process2.Executable())
		//			fmt.Println(process2)
		//		}
		//		//fmt.Println(process.Executable())
		//		if process.Pid() == process2.Pid() {
		//			isIn = true
		//		}
		//	}
		//	if isIn == false {
		//		fmt.Println(process.Pid(), " ", process.Executable())
		//
		//	}
		//
		//	// do os.* stuff on the pid
		//}
		processList, err = ps.Processes()
		if err != nil {
			log.Println("ps.Processes() Failed")
			return
		}
	}
}

func main() {
	//whatever()
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
