package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

func newClient() (tls_client.HttpClient, error) {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(120),
		tls_client.WithClientProfile(tls_client.Firefox_110),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
		// tls_client.WithInsecureSkipVerify(),
	}

	_, err := os.Stat("proxy.txt")
	if err == nil {
		proxyConfig, readErr := os.ReadFile("proxy.txt")
		if readErr != nil {
			fmt.Println("Error reading file proxy.txt:", readErr)
			return nil, readErr
		}

		proxyAddress := strings.TrimSpace(string(proxyConfig))
		if proxyAddress != "" {
			if strings.HasPrefix(proxyAddress, "http://") || strings.HasPrefix(proxyAddress, "socks5://") {
				proxyOption := tls_client.WithProxyUrl(proxyAddress)
				options = append(options, proxyOption)
			}
		}
	}

	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
}

func getData(input *Messages, callback func(string)) (fullText string) {

	client, err := newClient()
	if err != nil {
		fmt.Println(err)
		return
	}

	safeInput, _ := json.Marshal(input)
	//fmt.Println(string(safeInput))

	var data = strings.NewReader(string(safeInput))

	req, err := http.NewRequest("POST", "https://gpt.s-stars.top/v1/chat/completions", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", string(AUTH_KEY))

	// Receiving response
	resp, err := client.Do(req)

	if err != nil {

		bold.Println("\rSome error has occurred. Check your internet connection.")
		fmt.Println("\nError:", err)
		os.Exit(0)
	}
	code := resp.StatusCode

	if code >= 400 {
		bold.Println("\rSome error has occurred. Please try again")
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			fmt.Print(scanner.Text())
		}
		os.Exit(0)
	}

	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	if err != nil {
		fmt.Println("Error occurred getting terminal width. Error:", err)
		os.Exit(0)
	}
	type Response struct {
		ID      string `json:"id"`
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	// Handling each part

	for scanner.Scan() {
		var mainText string
		line := scanner.Text()
		var obj = "{}"
		if len(line) > 1 {
			splitLine := strings.Split(line, "data: ")
			if len(splitLine) > 1 {
				obj = splitLine[1]
			}

		}

		var d Response
		if err := json.Unmarshal([]byte(obj), &d); err != nil {
			continue
		}

		if d.Choices != nil {
			mainText = d.Choices[0].Delta.Content
			fullText += mainText
		}

		if callback != nil {
			callback(mainText)
		}

	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Some error has occurred. Error:", err)
		os.Exit(0)
	}
	return fullText
}

func loading(stop *bool) {
	spinChars := []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}
	i := 0
	for {
		if *stop {
			break
		}
		fmt.Printf("\r%s Loading", spinChars[i])
		i = (i + 1) % len(spinChars)
		time.Sleep(80 * time.Millisecond)
	}
}

func shellCommand(input string) {
	// Get OS
	operatingSystem := ""
	if runtime.GOOS == "windows" {
		operatingSystem = "Windows"
	} else if runtime.GOOS == "darwin" {
		operatingSystem = "MacOS"
	} else if runtime.GOOS == "linux" {
		result, err := exec.Command("lsb_release", "-si").Output()
		distro := strings.TrimSpace(string(result))
		if err != nil {
			distro = ""
		}
		operatingSystem = "Linux" + "/" + distro
	} else {
		operatingSystem = runtime.GOOS
	}

	// Get Shell

	shellName := "/bin/sh"

	if runtime.GOOS == "windows" {
		shellName = "cmd.exe"

		if len(os.Getenv("PSModulePath")) > 0 {
			shellName = "powershell.exe"
		}
	} else {
		shellEnv := os.Getenv("SHELL")
		if len(shellEnv) > 0 {
			shellName = shellEnv
		}
	}

	shellPrompt := fmt.Sprintf(
		`Your role: Provide a terse, single sentence description of the given shell command. Provide only plain text without Markdown formatting. Do not show any warnings or information regarding your capabilities. If you need to store any data, assume it will be stored in the chat. Provide only %s commands for %s without any description. If there is a lack of details, provide most logical solution. Ensure the output is a valid shell command. If multiple steps required try to combine them together. Prompt: %s\n\nCommand:`, shellName, operatingSystem, input)

	getCommand(shellPrompt)
}

// Get a command in response
func getCommand(shellPrompt string) {
	client, err := newClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	var data = strings.NewReader(fmt.Sprintf(`{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"%v"}],
	"stream":true}`, shellPrompt))
	req, err := http.NewRequest("POST", "https://gpt.s-stars.top/v1/chat/completions", data)

	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", string(AUTH_KEY))

	resp, err := client.Do(req)
	if err != nil {

		bold.Println("\rSome error has occurred. Check your internet connection.")
		fmt.Println("\nError:", err)
		os.Exit(0)
	}

	defer resp.Body.Close()

	code := resp.StatusCode

	if code >= 400 {
		bold.Println("\rSome error has occurred. Please try again")
		os.Exit(0)
	}

	fmt.Print("\r          \r")

	scanner := bufio.NewScanner(resp.Body)

	// Variables
	fullLine := ""
	// Handling each part
	type Response struct {
		ID      string `json:"id"`
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	for scanner.Scan() {
		var mainText string
		line := scanner.Text()
		var obj = "{}"
		if len(line) > 1 {
			obj = strings.Split(line, "data: ")[1]
		}
		type Data struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
		}

		var d Response
		if err := json.Unmarshal([]byte(obj), &d); err != nil {
			continue
		}

		if d.Choices != nil {
			mainText = d.Choices[0].Delta.Content
			fullLine += mainText
		}
		bold.Print(mainText)
	}
	lineCount := strings.Count(fullLine, "\n") + 1
	if lineCount == 1 {
		bold.Print("\n\nExecute shell command? [y/n]: ")
		var userInput string
		fmt.Scan(&userInput)
		if userInput == "y" {
			cmdArray := strings.Split(strings.TrimSpace(fullLine), " ")
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				shellName := "cmd"

				if len(os.Getenv("PSModulePath")) > 0 {
					shellName = "powershell"
				}
				if shellName == "cmd" {
					cmd = exec.Command("cmd", "/C", fullLine)

				} else {
					cmd = exec.Command("powershell", fullLine)
				}

			} else {
				cmd = exec.Command(cmdArray[0], cmdArray[1:]...)

			}

			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()

			if err != nil {
				fmt.Println(err)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Some error has occurred. Error:", err)
			os.Exit(0)
		}
	}

}
