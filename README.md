<p align="center"><img src="tgpt.svg"></p>

# Terminal GPT (tgpt) 

tgpt is a cross-platform command-line interface (CLI) tool that allows you to use ChatGPT 3.5 in your Terminal without requiring API keys. Modification based on [aandrew-me/tgpt](https://github.com/aandrew-me/tgpt).Thanks to the original author.

## Usage 

```bash
Usage:
  tgpt [option] <prompt|stdin>

DESCRIPTION:
  tgpt is a tool for interacting with the GPT-3.5 language model by OpenAI.

OPTIONS:
      --ai-name string       Set AI name.
  -h, --help                 Print this message.
  -i, --interactive          Start normal interactive mode.
  -m, --memory string        Start with a memory file or start with a new memory file.
  -q, --quiet                Gives response back without loading animation.
  -r, --refresh              Refresh auth key.
      --system-rule string   Customized rule using system role support text or file path.
      --user-name string     Set user name.
  -v, --version              Print version.
  -w, --whole                Gives response back as a whole text.


Examples:
tgpt "What is internet?"
echo "What is internet?" | tgpt 
tgpt -w "What is internet?"
echo "What is internet?" | tgpt -w
tgpt --system-rule code.rule "golang Hello, World!"
tgpt --system-rule "Add ‘~~~’ at the end of the reply" "hello"
tgpt --memory "chat01" --system-rule "Add ‘~~~’ at the end of the reply" "your name is Cindy"
tgpt --memory "chat01" "what is your name"
tgpt --ai-name "Cindy" "what is your name"
tgpt --user-name "Tom" "who am i"
tgpt -i --user-name "Tom" --ai-name "Cindy" --memory "chat02" --system-rule "Add ‘~~~’ at the end of the reply"



```

You can download the executable for your operating system, rename it to `tgpt` (or any other desired name), and then execute it by typing `./tgpt` while in that directory. Alternatively, you can add it to your PATH environmental variable and then execute it by simply typing `tgpt`.

