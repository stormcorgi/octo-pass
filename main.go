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

var execFilePath string

type S3SConf struct {
	Api_key       string `json:"api_key"`
	Acc_loc       string `json:"acc_loc"`
	Gtoken        string `json:"gtoken"`
	Bullettoken   string `json:"bullettoken"`
	Session_token string `json:"session_token"`
	F_gen         string `json:"f_gen"`
}

func openBrowser(url string) {
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
	// TODO: change s3s binary by GOOS
	// TODO: need filepath.Join(execFilePath,<s3s-binary>)
	s3s := exec.Command("./bin/s3s", "-M")
	mon, _ := s3s.StdoutPipe()
	s3s.Start()
	monScanner := bufio.NewScanner(mon)
	for monScanner.Scan() {
		line := monScanner.Text()
		fmt.Printf("%s\n", line)
		bindStr.Set(line)
	}
}

func setStatinkAPI(apiKey string) {
	// TODO: change s3s binary by GOOS
	// TODO: need filepath.Join(execFilePath,<s3s-binary>)
	s3s := exec.Command("./bin/s3s")
	stdin, _ := s3s.StdinPipe()
	io.WriteString(stdin, apiKey)
	stdin.Close()
	out, _ := s3s.Output()
	log.Println(string(out))
}

func obtainTokens(ch chan string) {
	// stage.2.1 login url generation / open browser
	// -> authLink
	// TODO: change s3s binary by GOOS
	// TODO: need filepath.Join(execFilePath,<s3s-binary>)
	s3sRec := exec.Command("./bin/s3s", "-r")
	stdin, _ := s3sRec.StdinPipe()
	stdout, _ := s3sRec.StdoutPipe()
	s3sRec.Start()

	scanner := bufio.NewScanner(stdout)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "https") {
				authLink := line
				fmt.Printf("URL Found! : %s\n", authLink)
				openBrowser(authLink)
			}
		}
	}()

	// stage.2.2 login url paste
	uri := <-ch
	io.WriteString(stdin, uri)
	stdin.Close()

	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("%s\n", line)
		}
	}()
	s3sRec.Wait()
	log.Println("done!")
}

func validateConfig() S3SConf {
	var myConf S3SConf
	configPath := filepath.Join(execFilePath, "bin/config.txt")
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
	return myConf
}

func main() {
	execFilePath, _ = os.Executable()
	// TODO: validate s3s binary exist?

	Conf := validateConfig()
	if Conf.Api_key == "" {
		log.Println("API_Key is blank")
	}
	if Conf.Gtoken == "" {
		log.Println("Gtoken is blank")
	}

	// for input Nintendo Switch Online URI
	NSOch := make(chan string)
	// for display s3s's stdout in GUI
	bindStdout := binding.NewString()

	// GUI with fyne-io
	a := app.New()
	w := a.NewWindow("octo-pass")
	w.Resize(fyne.NewSize(640, 480))

	apiInput := widget.NewEntry()
	apiInput.SetPlaceHolder("Input stat.ink API")
	apiButton := widget.NewButton("stat.ink API", func() {
		setStatinkAPI(apiInput.Text)
		go obtainTokens(NSOch)
	})

	NSOInput := widget.NewEntry()
	NSOInput.SetPlaceHolder("Input Nintendo Online Service URI")
	NSOInputButton := widget.NewButton("Nintendo Online URI", func() {
		NSOch <- NSOInput.Text
	})
	monStdout := widget.NewEntryWithData(bindStdout)
	monStdout.MultiLine = true

	// already configured -> disable button
	if Conf.Api_key != "" {
		apiButton.Disable()
	}
	if Conf.Gtoken != "" {
		NSOInputButton.Disable()
	}

	content := container.NewVBox(
		apiInput,
		apiButton,
		NSOInput,
		NSOInputButton,
		widget.NewButton("Monitoring", func() {
			go monitor(bindStdout)
		}),
		// FIXME: poor display now. and if exec time, another window shows log...
		// monStdout,
	)

	w.SetContent(content)
	w.ShowAndRun()
}
