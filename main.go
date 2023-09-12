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
	"regexp"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/olekukonko/ts"
	"net/http"
)

const localVersion = "1.7.6"

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var boldWhite = color.New(color.Bold, color.FgHiWhite)
var boldViolet = color.New(color.Bold, color.FgMagenta)
var codeText = color.New(color.BgBlack, color.FgGreen, color.Bold)
var stopSpin = false

var programLoop = true
var serverID = ""
var configDir = ""
var userInput = ""
var executablePath = ""
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

		memory string

		name     string
		userName string
	)

	flag.BoolVarP(&version, "version", "v", false, "Print version.")
	flag.BoolVarP(&whole, "whole", "w", false, "Gives response back as a whole text.")
	flag.BoolVarP(&quiet, "quiet", "q", false, "Gives response back without loading animation.")
	flag.BoolVarP(&interactive, "interactive", "i", false, "Start normal interactive mode.")
	flag.BoolVarP(&help, "help", "h", false, "Print this message.")
	flag.BoolVarP(&updateKey, "refresh", "r", false, "refresh auth key.")
	flag.StringVar(&systemRole, "system-rule", "", "Customized rule using system role supper text or file path.")
	flag.StringVarP(&memory, "memory", "m", "", "Start with memories with file path.")
	flag.StringVar(&name, "name", "", "set AI name.")
	flag.StringVar(&userName, "user-name", "", "set user name.")

	flag.Parse()

	execPath, err := os.Executable()

	if err == nil {
		executablePath = execPath
	}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		os.Exit(0)
	}()

	configDir, err = os.UserConfigDir()

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
		configData["chat_1"] = []interface{}{}
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
		fmt.Println(strings.TrimSpace(getData([]interface{}{
			map[string]interface{}{"role": "system", "content": getSafeString(systemRole)},
			map[string]interface{}{"role": "user", "content": getSafeString(prompt)},
		}, nil)))
		os.Exit(0)
	}

	if quiet {
		getData([]interface{}{
			map[string]interface{}{"role": "system", "content": getSafeString(systemRole)},
			map[string]interface{}{"role": "user", "content": getSafeString(prompt)},
		}, func(s string) {
			fmt.Print(s)
		})
		os.Exit(0)
	}

	if interactive {
		reader := bufio.NewReader(os.Stdin)
		bold.Print("Interactive mode started. Press Ctrl + C or type exit to quit.\n\n")

		messages := []interface{}{
			map[string]interface{}{"role": "system", "content": getSafeString(systemRole)},
		}

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

					item := map[string]interface{}{"role": "user", "content": getSafeString(input)}
					messages = append(messages, item)
					assistantMessage := getData(messages, func(s string) {
						fmt.Print(s)
					})

					item = map[string]interface{}{"role": "assistant", "content": getSafeString(assistantMessage)}
					fmt.Print("\n\n")
					messages = append(messages, item)
				}

			}

		}
		os.Exit(0)
	}

	getData([]interface{}{
		map[string]interface{}{"role": "system", "content": getSafeString(systemRole)},
		map[string]interface{}{"role": "user", "content": getSafeString(prompt)},
	}, func(s string) {
		fmt.Print(s)
	})

}

type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel() model {
	size, _ := ts.GetSize()
	termWidth := size.Col()
	ti := textarea.New()
	ti.SetWidth(termWidth)
	ti.CharLimit = 200000
	ti.ShowLineNumbers = false
	ti.Placeholder = "Enter your prompt"
	ti.Focus()

	return model{
		textarea: ti,
		err:      nil,
	}
}

func hasDataInStdin() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			programLoop = false
			userInput = ""
			return m, tea.Quit

		case tea.KeyTab:
			userInput = m.textarea.Value()

			if len(userInput) > 1 {
				m.textarea.Blur()
				return m, tea.Quit
			}

		default:
			if !m.textarea.Focused() {
				cmd = m.textarea.Focus()
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.textarea.View()
}

func printProgramDescription() {
	fmt.Println("tgpt [option] <prompt|stdin>\n")
	fmt.Println("DESCRIPTION:")
	fmt.Println("  tgpt is a tool for interacting with the GPT-3.5 language model by OpenAI.\n")
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
}

func getKey() string {
	url := "https://raw.githubusercontent.com/aandrew-me/tgpt/main/main.go"

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("请求失败：", err)
		return ""
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("读取失败：", err)
		return ""
	}
	strBody := string(body)
	pattern := `base64.StdEncoding.DecodeString\("([^"]+)"\)`

	// 编译正则表达式
	regex := regexp.MustCompile(pattern)

	// 查找匹配的字符串
	matches := regex.FindAllStringSubmatch(string(strBody), -1)

	// 提取匹配的字符串
	for _, match := range matches {
		if len(match) >= 2 {
			decodedString := match[1]
			return decodedString
		}
	}
	return ""
}

func getSafeString(value string) string {
	safe, _ := json.Marshal(value)
	return strings.Trim(string(safe), "\"")
}

func tryReadContent(value string) string {
	if value != "" {
		// 检查文件是否可读
		fileInfo, err := os.Stat(value)
		if err == nil && fileInfo.Mode().IsRegular() && fileInfo.Mode().Perm()&0400 != 0 {
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

//////////////////////////////
