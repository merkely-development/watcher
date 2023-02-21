package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kosli-dev/cli/internal/output"
	"github.com/kosli-dev/cli/internal/requests"
	"github.com/spf13/cobra"
)

const listArtifactsShortDesc = `List artifacts in a flow. `

const listArtifactsLongDesc = listArtifactsShortDesc + `The results are paginated and ordered from latests to oldest. 
By default, the page limit is 15 artifacts per page.
`
const artifactLsExample = `
# list the last 15 artifacts for a flow:
kosli list artifacts yourFlowName \
	--api-token yourAPIToken \
	--owner yourOrgName

# list the last 30 artifacts for a flow:
kosli list artifacts yourFlowName \
	--page-limit 30 \
	--api-token yourAPIToken \
	--owner yourOrgName

# list the last 30 artifacts for a flow (in JSON):
kosli list artifacts yourFlowName \
	--page-limit 30 \
	--api-token yourAPIToken \
	--owner yourOrgName \
	--output json
`

type listArtifactsOptions struct {
	listOptions
}

func newListArtifactsCmd(out io.Writer) *cobra.Command {
	o := new(listArtifactsOptions)
	cmd := &cobra.Command{
		Use:     "artifacts FLOW-NAME",
		Short:   listArtifactsShortDesc,
		Long:    listArtifactsLongDesc,
		Example: artifactLsExample,
		Args:    cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			err := RequireGlobalFlags(global, []string{"Owner", "ApiToken"})
			if err != nil {
				return ErrorBeforePrintingUsage(cmd, err.Error())
			}
			return o.validate(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(out, args)
		},
	}

	addListFlags(cmd, &o.listOptions)

	return cmd
}

func (o *listArtifactsOptions) run(out io.Writer, args []string) error {
	url := fmt.Sprintf("%s/api/v1/projects/%s/%s/artifacts/?page=%d&per_page=%d",
		global.Host, global.Owner, args[0], o.pageNumber, o.pageLimit)

	reqParams := &requests.RequestParams{
		Method:   http.MethodGet,
		URL:      url,
		Password: global.ApiToken,
	}
	response, err := kosliClient.Do(reqParams)
	if err != nil {
		return err
	}

	return output.FormattedPrint(response.Body, o.output, out, o.pageNumber,
		map[string]output.FormatOutputFunc{
			"table": printArtifactsListAsTable,
			"json":  output.PrintJson,
		})
}

func printArtifactsListAsTable(raw string, out io.Writer, page int) error {
	var artifacts []map[string]interface{}
	err := json.Unmarshal([]byte(raw), &artifacts)
	if err != nil {
		return err
	}

	if len(artifacts) == 0 {
		msg := "No artifacts were found"
		if page != 1 {
			msg = fmt.Sprintf("%s at page number %d", msg, page)
		}
		logger.Info(msg + ".")
		return nil
	}

	header := []string{"COMMIT", "ARTIFACT", "STATE", "CREATED_AT"}
	rows := []string{}
	for _, artifact := range artifacts {
		evidenceMap := artifact["evidence"].(map[string]interface{})
		artifactData := evidenceMap["artifact"].(map[string]interface{})

		gitCommit := artifactData["git_commit"].(string)[:7]
		artifactName := artifactData["filename"].(string)

		artifactDigest := artifactData["sha256"].(string)
		artifactState := artifact["state"].(string)
		createdAt, err := formattedTimestamp(artifact["created_at"], true)
		if err != nil {
			return err
		}

		row := fmt.Sprintf("%s\tName: %s\t%s\t%s", gitCommit, artifactName, artifactState, createdAt)
		rows = append(rows, row)
		row = fmt.Sprintf("\tFingerprint: %s\t\t", artifactDigest)
		rows = append(rows, row)
		rows = append(rows, "\t\t\t")

	}
	tabFormattedPrint(out, header, rows)

	return nil
}
