package elasticshell

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/marclop/elasticshell/cli"
	"github.com/marclop/elasticshell/client"
)

// Application contains the full application and its dependencies
type Application struct {
	config    *Config
	client    client.ClientInterface
	formatter cli.FormatterInterface
	parser    cli.ParserInterface
	repl      *readline.Instance
}

// Init ties all the application pieces together and returns a conveninent *Application struct
// that allows easy interaction with all the pieces of the application
func Init(config *Config, client client.ClientInterface, parser cli.ParserInterface) *Application {
	return &Application{
		config: config,
		client: client,
		parser: parser,
	}
}

// HandleCli handles the the interaction between the validated input and
// remote HTTP calls to the specified host including the call to the JSON formatter
func (app *Application) HandleCli(method string, url string, body string) error {
	res, err := app.client.HandleCall(method, url, body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if app.config.interactive {
		app.formatter = cli.NewIteractiveJSONFormatter(res)
	} else {
		app.formatter = cli.NewJSONFormatter(res)
	}

	app.formatter.FormatJSON(app.config.verbose)
	return nil
}

func (app *Application) initInteractive() {
	app.repl, _ = readline.NewEx(
		&readline.Config{
			Prompt:          "\x1b[34mElasticShell> \x1b[0m",
			InterruptPrompt: "^C",
			EOFPrompt:       "exit",
			AutoComplete:    cli.Completer,
			HistoryFile:     "/tmp/elasticshell.history",
		},
	)
	app.config.interactive = true
}

// Interactive runs the application like a readline / REPL
func (app *Application) Interactive() {
	app.initInteractive()
	defer app.repl.Close()
	for {
		line, err := app.repl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			return
		}

		if len(line) == 0 {
			continue
		}

		cleanLine := strings.TrimSpace(line)
		if cleanLine == "exit" || cleanLine == "quit" {
			return
		}
		lineSliced := strings.Fields(cleanLine)

		// TODO: Split conditional hell
		if lineSliced[0] == "set" {
			if len(lineSliced) == 3 {
				switch lineSliced[1] {
				case "host":
					if strings.HasSuffix(lineSliced[2], "http") {
						app.client.SetHost(lineSliced[2])
					} else {
						fmt.Println(lineSliced[2], "does not contain a valid protocol scheme")
					}
				case "port":
					port, err := strconv.Atoi(lineSliced[2])
					if err != nil {
						fmt.Println(lineSliced[2], "is not a valid port")
						continue
					}
					app.client.SetPort(port)
				case "user":
					app.client.SetUser(lineSliced[2])
				case "pass":
					app.client.SetPass(lineSliced[2])
				}
				continue
			} else if (len(lineSliced) == 2) && (lineSliced[1] == "verbose") {
				app.config.verbose = true
				continue
			} else {
				continue
			}
		} else {
			app.parser, err = cli.NewIteractiveParser(cleanLine)
			if err != nil {
				fmt.Println(err)
				continue
			}
			app.parser.Validate()
		}

		err = app.HandleCli(app.parser.Method(), app.parser.URL(), app.parser.Body())
		if err != nil {
			fmt.Println(err)
		}
	}
}
