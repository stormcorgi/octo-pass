package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

var execFolderPath string
var s3sPath string

type S3SConf struct {
	Api_key       string `json:"api_key"`
	Acc_loc       string `json:"acc_loc"`
	Gtoken        string `json:"gtoken"`
	Bullettoken   string `json:"bullettoken"`
	Session_token string `json:"session_token"`
	F_gen         string `json:"f_gen"`
}

func openBrowser(url string) {
	log.Println("[info] opening browser...")
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}
}

func prints3sOutput(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Printf("%s\n", scanner.Text())
	}
}

func monitor(bindStr binding.String) {
	log.Println("[info] monitor start!")
	// TODO: change s3s binary by GOOS
	// TODO: need filepath.Join(execFilePath,<s3s-binary>)
	log.Println("[info] calling s3s -M")
	s3s := exec.Command(s3sPath, "-M")
	mon, _ := s3s.StdoutPipe()
	s3s.Start()
	prints3sOutput(mon)
	log.Println("[info] monitor done")
}

func history() {
	log.Println("[info] history upload start!")
	log.Println("[info] calling s3s -r")
	s3s := exec.Command(s3sPath, "-r")
	mon, _ := s3s.StdoutPipe()
	s3s.Start()
	prints3sOutput(mon)
	log.Println("[info] history upload done")
}

func setStatinkAPI(apiKey string) {
	log.Println("[info] setting stat.ink API_key")
	// TODO: change s3s binary by GOOS
	// TODO: need filepath.Join(execFilePath,<s3s-binary>)
	s3s := exec.Command(s3sPath)
	stdin, _ := s3s.StdinPipe()
	io.WriteString(stdin, apiKey)
	stdin.Close()
	out, _ := s3s.Output()
	log.Println("[info] stat.ink API_Key set!")
	log.Println(string(out))
}

func obtainTokens(ch chan string) {
	log.Println("[info] start obtainTokens...")
	// stage.2.1 login url generation / open browser
	// -> authLink
	// TODO: change s3s binary by GOOS
	// TODO: need filepath.Join(execFilePath,<s3s-binary>)
	s3sRec := exec.Command(s3sPath, "-r")
	stdin, _ := s3sRec.StdinPipe()
	stdout, _ := s3sRec.StdoutPipe()
	s3sRec.Start()

	scanner := bufio.NewScanner(stdout)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "https") {
				authLink := line
				log.Printf("URL Found! : %s\n", authLink)
				openBrowser(authLink)
			}
		}
	}()

	// stage.2.2 login url paste
	uri := <-ch
	log.Printf("[info] token string input : %s", uri)
	io.WriteString(stdin, uri)
	stdin.Close()

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("%s\n", line)
		}
	}()
	s3sRec.Wait()
	log.Println("[info] token set done!")
}

func validateConfig() S3SConf {
	log.Println("[info] validate ./bin/config.txt")
	var myConf S3SConf
	configPath := filepath.Join(execFolderPath, "bin/config.txt")
	_, err := os.Stat(configPath)
	if err != nil {
		log.Println(err)
		return myConf
	}

	jsonString, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
		return myConf
	}
	if err := json.Unmarshal(jsonString, &myConf); err != nil {
		fmt.Println(err)
		return myConf
	}
	log.Println("[info] ./bin/config.txt is valid.")
	return myConf
}

func main() {
	execFilePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execFolderPath = filepath.Dir(execFilePath)
	log.Printf("[info] %s", execFolderPath)

	// TODO: validate s3s binary exist?
	switch runtime.GOOS {
	case "windows":
		s3sPath = filepath.Join(execFolderPath, "bin/s3s.exe")
	case "darwin":
		s3sPath = filepath.Join(execFolderPath, "bin/s3s")
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		log.Fatal(err)
	}

	// parse config.txt into Conf structure if exist
	Conf := validateConfig()
	if Conf.Api_key == "" {
		log.Println("[info] API_Key is blank")
	}
	if Conf.Gtoken == "" {
		log.Println("[info] Gtoken is blank")
	}

	// for input Nintendo Switch Online URI
	NSOch := make(chan string)
	// for display s3s's stdout in GUI
	bindStdout := binding.NewString()

	// GUI with fyne-io
	a := app.New()
	w := a.NewWindow("octo-pass")
	w.Resize(fyne.NewSize(500, 240))

	apiInput := widget.NewEntry()
	apiInput.SetPlaceHolder("Input stat.ink API")
	apiButton := widget.NewButton("stat.ink API", func() {
		// TODO: validate text nil?
		setStatinkAPI(apiInput.Text)
		go obtainTokens(NSOch)
	})

	NSOInput := widget.NewEntry()
	NSOInput.SetPlaceHolder("Input Nintendo Online Service URI")
	NSOInputButton := widget.NewButton("Nintendo Online URI", func() {
		// TODO: validate text nil?
		NSOch <- NSOInput.Text
	})
	monStdout := widget.NewEntryWithData(bindStdout)
	monStdout.MultiLine = true

	// already configured -> disable button
	if Conf.Api_key != "" {
		log.Println("[info] API_Key is alredy exist, disable button")
		apiButton.Disable()
	}
	if Conf.Gtoken != "" {
		log.Println("[info] Nintendo Switch Online Token is alredy exist, disable button")
		NSOInputButton.Disable()
	}

	content := container.NewVBox(
		apiInput,
		apiButton,
		NSOInput,
		NSOInputButton,
		widget.NewButton("Upload History", func() {
			go history()
		}),
		widget.NewButton("Monitoring", func() {
			go monitor(bindStdout)
		}),
		// FIXME: poor display now. and if exec time, another window shows log...
		// monStdout,
	)

	w.SetContent(content)
	w.ShowAndRun()
}
