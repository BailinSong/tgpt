package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	flag "github.com/spf13/pflag"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"net/http"
)

const localVersion = "1.8.0"

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var AUTH_KEY []byte

func main() {

	//fmt.Println(os.Args)

	var (
		version     bool
		whole       bool
		quiet       bool
		interactive bool
		help        bool
		updateKey   bool
		systemRole  string
		memory      string
		name        string
		userName    string
		block       bool
	)

	flag.BoolVarP(&version, "version", "v", false, "Print version.")
	flag.BoolVarP(&whole, "whole", "w", false, "Gives response back as a whole text.")
	flag.BoolVarP(&quiet, "quiet", "q", false, "Gives response back without loading animation.")
	flag.BoolVarP(&interactive, "interactive", "i", false, "Start normal interactive mode.")
	flag.BoolVarP(&help, "help", "h", false, "Print this message.")
	flag.BoolVarP(&updateKey, "refresh", "r", false, "Refresh auth key.")
	flag.BoolVarP(&block, "block", "b", false, "Block content by stdin.")

	flag.StringVar(&systemRole, "system-rule", "", "Customized rule using system role support text or file path.")
	flag.StringVarP(&memory, "memory", "m", "", "Start with a memory file or start with a new memory file.")
	flag.StringVar(&name, "ai-name", "", "Set AI name.")
	flag.StringVar(&userName, "user-name", "", "Set user name.")

	flag.Parse()

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		os.Exit(0)
	}()

	configDir, err := os.UserConfigDir()

	configFile := configDir + "/gpt/.config.json"
	configManager := NewConfigManager(configFile)
	defaultConfig := map[string]interface{}{
		"AUTH_KEY": "",
		"MEMORY":   map[string]interface{}{},
		"SYSTEM":   map[string]interface{}{},
	}
	configData, err := configManager.ReadConfig(defaultConfig)
	if err != nil {
		fmt.Println("Unable to read configuration file:", err)
		return
	}

	if configData["AUTH_KEY"].(string) == "" {
		configData["AUTH_KEY"] = getKey()
		configManager.WriteConfig(configData)
	}

	AUTH_KEY, _ = base64.StdEncoding.DecodeString(configData["AUTH_KEY"].(string))

	if updateKey {
		configData["AUTH_KEY"] = getKey()
		fmt.Println("Updating configuration")
		configManager.WriteConfig(configData)
		os.Exit(0)
	}

	if help {
		printProgramDescription()
		os.Exit(0)
	}

	if version {
		fmt.Println("gpt", localVersion)
		fmt.Println("Source Code:")
		fmt.Println("  https://github.com/BailinSong/tgpt.git")
		os.Exit(0)
	}

	systemRole = tryReadContent(systemRole)

	if userName != "" {
		systemRole = "User name is " + userName + "\n" + systemRole
	}

	if name != "" {
		systemRole = "You name is " + name + "\n" + systemRole
	}

	messages := NewMessages()

	if memory != "" {
		fileInfo, err := os.Stat(memory)
		if canRead(err, fileInfo) {
			messages.load(memory)
		} else {
			messages.AddSystemMessage(systemRole)
		}
	} else {
		messages.AddSystemMessage(systemRole)
	}

	prompt := ""

	switch len(flag.Args()) {
	case 1:
		prompt = strings.TrimSpace(flag.Args()[0])
		break
	case 0:
		if interactive {
			break
		} else {
			fmt.Printf("parameter len error:%v\n", len(flag.Args()))
			os.Exit(-1)
		}
	default:
		fmt.Printf("parameter len error:%v\n", len(flag.Args()))
		os.Exit(-1)

	}

	if hasDataInStdin() {
		if block {
			loadingFlag := false
			if !(quiet && whole) {
				go loading(&loadingFlag)
			}

			if interactive {
				fmt.Println("interactive stdin is occupied!")
				os.Exit(-1)
			}
			message := ""

			bytes, _ := io.ReadAll(os.Stdin)
			strMessage := string(pretreatment(bytes))

			if len(bytes) > 3072 {

				for _, i := range SplitString(strMessage, 3072) {

					cMessages := NewMessages()
					cMessages.Temperature = 0.1
					cMessages.AddSystemMessage("Your Role: only output summarized., no description is provided.\nIMPORTANT: Ignore short lines.\nIMPORTANT: Provide only plain text without Markdown formatting.\nIMPORTANT: Do not include markdown formatting.\nIf there is a lack of details, provide most logical solution. You are not allowed to ask for more details.\nIgnore any potential risk of errors or confusion.")
					cMessages.AddUserMessage("Focus on" + prompt + ":\n\n" + i)
					message += getData(cMessages, nil)

				}
			} else {

				message = strMessage
			}

			messages.AddUserMessage(message)
			messages.AddAssistantMessage("I will answer based on the data you provide")
			loadingFlag = true
			process(whole, messages, prompt, block, memory, quiet, interactive, userName, name)

		} else {

			scanner := bufio.NewScanner(os.Stdin)

			for scanner.Scan() {
				message := scanner.Text()
				clonedMessages := messages.CloneMessages()
				clonedMessages.AddUserMessage(message)
				clonedMessages.AddAssistantMessage("I will answer based on the data you provide")
				process(whole, clonedMessages, prompt, block, memory, quiet, interactive, userName, name)
			}

		}
	}

}

func process(whole bool, messages *Messages, prompt string, block bool, memory string, quiet bool, interactive bool, userName string, name string) {
	if whole {
		messages.AddUserMessage(getSafeString(prompt))
		assistantMessage := getData(messages, nil)
		fmt.Println(strings.TrimSpace(assistantMessage))

		if !block {

		}
		if memory != "" {
			messages.AddAssistantMessage(getSafeString(assistantMessage))
			messages.save(memory)
		}

		return
	}

	if quiet {
		messages.AddUserMessage(getSafeString(prompt))
		assistantMessage := getData(messages, func(s string) {
			fmt.Print(s)
		})
		fmt.Print("\n")
		if memory != "" {
			messages.AddAssistantMessage(getSafeString(assistantMessage))
			messages.save(memory)
		}
		return
	}

	if interactive {

		reader := bufio.NewReader(os.Stdin)
		bold.Print("Interactive mode started. Press Ctrl + C or type exit to quit.\n\n")

		for {

			if userName != "" {
				boldBlue.Print(userName + ":")
			} else {
				boldBlue.Print("YOU:")
			}

			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading input:", err)
				break
			}

			if len(input) > 1 {
				input = strings.TrimSpace(input)
				if len(input) > 1 {
					if input == "exit" {
						bold.Println("Exiting...")
						return
					}

					if name != "" {
						bold.Print(name + ":")
					} else {
						bold.Print("AI:")
					}

					messages.AddUserMessage(getSafeString(input))
					assistantMessage := getData(messages, func(s string) {
						fmt.Print(s)
					})
					messages.AddAssistantMessage(getSafeString(assistantMessage))

					fmt.Print("\n\n")

					if memory != "" {
						messages.save(memory)
					}

				}

			}

		}
		return
	}

	loadingFlag := false
	go loading(&loadingFlag)
	messages.AddUserMessage(getSafeString(prompt))
	assistantMessage := getData(messages, func(s string) {

		if !loadingFlag {
			loadingFlag = true
			fmt.Print("\r                     \r")
		}
		fmt.Print(s)
	})
	fmt.Print("\n")
	if memory != "" {
		messages.AddAssistantMessage(getSafeString(assistantMessage))
		messages.save(memory)
	}

	return
}

func hasDataInStdin() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func printProgramDescription() {
	fmt.Println("USAGE:")
	fmt.Println("  tgpt [option] <prompt|stdin>\n")
	fmt.Println("DESCRIPTION:")
	fmt.Println("  tgpt is a tool for interacting with the GPT-3.5 language model by OpenAI.\n")
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("EXAMPLES:\n  tgpt -r\n  tgpt \"What is internet?\"\n  echo \"What is internet?\" | tgpt \n  tgpt -w \"What is internet?\"\n  echo \"What is internet?\" | tgpt -w\n  tgpt --system-rule code.rule \"golang Hello, World!\"\n  tgpt --system-rule \"Add ‘~~~’ at the end of the reply\" \"hello\"\n  tgpt --memory \"chat01\" --system-rule \"Add ‘~~~’ at the end of the reply\" \"your name is Cindy\"\n  tgpt --memory \"chat01\" \"what is your name\"\n  tgpt --ai-name \"Cindy\" \"what is your name\"\n  tgpt --user-name \"Tom\" \"who am i\"\n  tgpt -i --user-name \"Tom\" --ai-name \"Cindy\" --memory \"chat02\" --system-rule \"Add ‘~~~’ at the end of the reply\"")
}

func getKey() string {
	url := "https://raw.githubusercontent.com/aandrew-me/tgpt/main/imp.txt"

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Request failed：", err)
		return ""
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Failed to read：", err)
		return ""
	}
	return string(body)
}

func getSafeString(value string) string {
	safe, _ := json.Marshal(value)
	return strings.Trim(string(safe), "\"")
}

func tryReadContent(value string) string {
	if value != "" {
		// 检查文件是否可读
		fileInfo, err := os.Stat(value)
		if canRead(err, fileInfo) {
			// 打开文件
			file, err := os.Open(value)
			if err == nil {
				// 读取文件内容
				content, err := io.ReadAll(file)
				file.Close() // 关闭文件
				if err == nil {
					// 将文件内容赋值给 value 变量
					value = string(content)
				}
			}
		} else if err != nil {
			//不是文件保持value 不变
		} else {
			// 文件不可读，将 value 设置为空字符串
			value = ""
		}
	}

	return value
}

func canRead(err error, fileInfo os.FileInfo) bool {
	return err == nil && fileInfo.Mode().IsRegular() && fileInfo.Mode().Perm()&0400 != 0
}

func SplitString(str string, chunkSize int) []string {
	var result []string

	for i := 0; i < len(str); i += chunkSize {
		end := i + chunkSize
		if end > len(str) {
			end = len(str)
		}
		result = append(result, str[i:end])
	}

	return result
}

//////////////////////////////
