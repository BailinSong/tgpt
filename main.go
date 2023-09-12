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

const localVersion = "1.7.6"

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var configDir = ""
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
	)

	flag.BoolVarP(&version, "version", "v", false, "Print version.")
	flag.BoolVarP(&whole, "whole", "w", false, "Gives response back as a whole text.")
	flag.BoolVarP(&quiet, "quiet", "q", false, "Gives response back without loading animation.")
	flag.BoolVarP(&interactive, "interactive", "i", false, "Start normal interactive mode.")
	flag.BoolVarP(&help, "help", "h", false, "Print this message.")
	flag.BoolVarP(&updateKey, "refresh", "r", false, "Refresh auth key.")
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
		}
	} else {
		messages.AddSystemMessage(systemRole)
	}

	prompt := ""
	if hasDataInStdin() {
		if interactive {
			printProgramDescription()
			os.Exit(0)
		}
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		prompt = scanner.Text()
		prompt = strings.TrimSpace(flag.Args()[0])
	} else {
		switch len(flag.Args()) {
		case 1:
			prompt = flag.Args()[0]
			prompt = strings.TrimSpace(flag.Args()[0])
			break
		case 0:
			if interactive {
				break
			} else {
				printProgramDescription()
				os.Exit(0)
			}
		default:
			printProgramDescription()
			os.Exit(0)

		}
	}

	if whole {
		messages.AddUserMessage(getSafeString(prompt))
		assistantMessage := getData(messages, nil)
		fmt.Println(strings.TrimSpace(assistantMessage))

		if memory != "" {
			messages.AddAssistantMessage(getSafeString(assistantMessage))
			messages.save(memory)
		}
		os.Exit(0)
	}

	if quiet {
		messages.AddUserMessage(getSafeString(prompt))
		assistantMessage := getData(messages, func(s string) {
			fmt.Print(s)
		})
		if memory != "" {
			messages.AddAssistantMessage(getSafeString(assistantMessage))
			messages.save(memory)
		}
		os.Exit(0)
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
		os.Exit(0)
	}

	loadingFlag := false
	go loading(&loadingFlag)
	messages.AddUserMessage(getSafeString(prompt))
	assistantMessage := getData(messages, func(s string) {

		if !loadingFlag {
			loadingFlag = true
			fmt.Printf("\r                     \r")
		}
		fmt.Print(s)
	})
	if memory != "" {
		messages.AddAssistantMessage(getSafeString(assistantMessage))
		messages.save(memory)
	}
	os.Exit(0)

}

func hasDataInStdin() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func printProgramDescription() {
	fmt.Println("Usage:")
	fmt.Println("  tgpt [option] <prompt|stdin>\n")
	fmt.Println("DESCRIPTION:")
	fmt.Println("  tgpt is a tool for interacting with the GPT-3.5 language model by OpenAI.\n")
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
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

//////////////////////////////
