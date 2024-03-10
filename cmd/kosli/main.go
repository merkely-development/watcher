package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	log "github.com/kosli-dev/cli/internal/logger"
	"github.com/kosli-dev/cli/internal/requests"
	"github.com/spf13/cobra"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	logger      *log.Logger
	kosliClient *requests.Client
)

func init() {
	logger = log.NewStandardLogger()
	kosliClient = requests.NewStandardKosliClient()
}

func resetGlobal() {
	global = &GlobalOpts{}
}

type Getter func() string

func getOrg() string {
	return global.Org
}
func getHost() string {
	return global.Host
}
func getApiToken() string {
	return global.ApiToken
}

func splitGlobal(args []string, g Getter) []string {
	defer resetGlobal()
	_ = nullCmd(args).Execute()
	return strings.Split(g(), ",")
}

func nullCmd(args []string) *cobra.Command {
	var buffer bytes.Buffer
	writer := io.Writer(&buffer)
	cmd, _ := newRootCmd(writer, args)
	return cmd
}

const prodHostURL = "https://app.kosli.com"
const stagingHostURL = "https://staging.app.kosli.com"

func prodAndStagingCyberDojoCallArgs(args []string) ([]string, []string) {
	orgs := splitGlobal(args, getOrg)
	hosts := splitGlobal(args, getHost)
	apiTokens := splitGlobal(args, getApiToken)

	isCyberDojo := len(orgs) == 1 && orgs[0] == "cyber-dojo"
	isDoubledHost := len(hosts) == 2 && hosts[0] == prodHostURL && hosts[1] == stagingHostURL
	isDoubledApiToken := len(apiTokens) == 2

	if isCyberDojo && isDoubledHost && isDoubledApiToken {

		argsAppendHostApiToken := func(n int) []string {
			// No need to strip existing --host/--api-token flags from args
			// as we are appending new flag values which take precedence.
			hostProd := fmt.Sprintf("--host=%s", hosts[n])
			apiTokenProd := fmt.Sprintf("--api-token=%s", apiTokens[n])
			return append(args, hostProd, apiTokenProd)
		}

		argsProd := argsAppendHostApiToken(0)
		argsStaging := argsAppendHostApiToken(1)
		//fmt.Printf("argsProd == <%s>\n", strings.Join(argsProd, " "))
		//fmt.Printf("argsStaging == <%s>\n", strings.Join(argsStaging, " "))
		return argsProd, argsStaging
	} else {
		return nil, nil
	}
}

func runProdAndStagingCyberDojoCalls(prodArgs []string, stagingArgs []string) error {
	// Kosli uses CI pipelines in the cyber-dojo Org repos [*] for two purposes:
	// 1. public facing documentation
	// 2. private development purposes
	//
	// All Kosli CLI calls in [*] are made to _two_ servers
	//   - https://app.kosli.com
	//   - https://staging.app.kolsi.com  (because of 2)
	//
	// Explicitly making each Kosli CLI call in [*] twice is not an option because of 1)
	// The least-worst option is to allow KOLSI_HOST and KOSLI_API_TOKEN to specify two values.

	// If the prod-call and the staging-call succeed:
	//   - do NOT print the staging-call output, so it looks as-if only the prod call occurred.
	// If the staging-call fails:
	//   - print its error message, making it clear it is from staging
	//   - return a non-zero exit-code, so staging errors are not silently ignored

	prodOutput, prodErr := runBufferedInnerMain(prodArgs)
	fmt.Print(prodOutput)
	_, stagingErr := runBufferedInnerMain(stagingArgs)
	// TODO?: print stagingOutput if --debug

	var errorMessage string
	if prodErr != nil {
		errorMessage += prodErr.Error()
	}
	if stagingErr != nil {
		errorMessage += fmt.Sprintf("\n%s\n\t%s", stagingHostURL, stagingErr.Error())
	}

	if errorMessage == "" {
		return nil
	} else {
		return fmt.Errorf("%s", errorMessage)
	}
}

func runBufferedInnerMain(args []string) (string, error) {
	// When errors are logged in main() the non-buffered global logger
	// must be restored so the error messages actually appear.
	globalLogger := &logger
	defer func(logger *log.Logger) { *globalLogger = logger }(logger)
	// Use a buffered Writer so the output of the staging call is NOT printed
	var buffer bytes.Buffer
	writer := io.Writer(&buffer)
	// Set global logger
	logger = log.NewLogger(writer, writer, false)
	// We have to reset os.Args here.
	// Presumably viper is reading os.Args?
	// Note that newRootCmd(args) does not use its args parameter.
	os.Args = args
	// Ensure prod/staging calls do not interfere with each other.
	resetGlobal()
	// Finally!
	err := inner_main(args)
	return fmt.Sprint(&buffer), err
}

func main() {
	var err error
	prodArgs, stagingArgs := prodAndStagingCyberDojoCallArgs(os.Args)
	if prodArgs == nil && stagingArgs == nil {
		err = inner_main(os.Args)
	} else {
		err = runProdAndStagingCyberDojoCalls(prodArgs, stagingArgs)
	}
	if err != nil {
		logger.Error(err.Error())
	}
}

func inner_main(args []string) error {
	cmd, err := newRootCmd(logger.Out, args[1:])
	if err != nil {
		return err
	}

	err = cmd.Execute()
	if err == nil {
		return nil
	}

	// cobra does not capture unknown/missing commands, see https://github.com/spf13/cobra/issues/706
	// so we handle this here until it is fixed in cobra
	if strings.Contains(err.Error(), "unknown flag:") {
		c, flags, err := cmd.Traverse(args[1:])
		if err != nil {
			return err
		}
		if c.HasSubCommands() {
			errMessage := ""
			if strings.HasPrefix(flags[0], "-") {
				errMessage = "missing subcommand"
			} else {
				errMessage = fmt.Sprintf("unknown command: %s", flags[0])
			}
			availableSubcommands := []string{}
			for _, sc := range c.Commands() {
				if !sc.Hidden {
					availableSubcommands = append(availableSubcommands, strings.Split(sc.Use, " ")[0])
				}
			}
			logger.Error("%s\navailable subcommands are: %s", errMessage, strings.Join(availableSubcommands, " | "))
		}
	}
	if global.DryRun {
		logger.Info("Error: %s", err.Error())
		logger.Warning("Encountered an error but --dry-run is enabled. Exiting with 0 exit code.")
		return nil
	}
	return err
}
