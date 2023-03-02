package main

import (
	"fmt"
	"testing"

	"github.com/kosli-dev/cli/internal/requests"
	"github.com/kosli-dev/cli/internal/testHelpers"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ArtifactEvidencePRBitbucketCommandTestSuite struct {
	suite.Suite
	defaultKosliArguments string
	artifactFingerprint   string
	pipelineName          string
}

func (suite *ArtifactEvidencePRBitbucketCommandTestSuite) SetupTest() {
	testHelpers.SkipIfEnvVarUnset(suite.T(), []string{"KOSLI_BITBUCKET_PASSWORD"})

	suite.pipelineName = "bitbucket-pr"
	suite.artifactFingerprint = "847411c6124e719a4e8da2550ac5c116b7ff930493ce8a061486b48db8a5aaa0"
	global = &GlobalOpts{
		ApiToken: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpZCI6ImNkNzg4OTg5In0.e8i_lA_QrEhFncb05Xw6E_tkCHU9QfcY4OLTVUCHffY",
		Owner:    "docs-cmd-test-user",
		Host:     "http://localhost:8001",
	}
	suite.defaultKosliArguments = fmt.Sprintf(" --host %s --owner %s --api-token %s", global.Host, global.Owner, global.ApiToken)
	kosliClient = requests.NewKosliClient(1, false, logger)

	CreatePipeline(suite.pipelineName, suite.T())
	CreateArtifact(suite.pipelineName, suite.artifactFingerprint, "foobar", suite.T())
}

func (suite *ArtifactEvidencePRBitbucketCommandTestSuite) TestArtifactEvidencePRBitbucketCmd() {
	tests := []cmdTestCase{
		{
			name: "report Bitbucket PR evidence works with new flags (fingerprint, name ...)",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "bitbucket pull request evidence is reported to artifact: " + suite.artifactFingerprint + "\n",
		},
		{
			name: "report Bitbucket PR evidence works with evidence url and fingerprint flags (fingerprint, name ...)",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint +
				` --name bb-pr --pipeline ` + suite.pipelineName +
				`--build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz 
				--repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69
				--evidence-url yr.no --evidence-fingerprint deadbeef ` + suite.defaultKosliArguments,
			golden: "bitbucket pull request evidence is reported to artifact: " + suite.artifactFingerprint + "\n",
		},
		{
			name: "report Bitbucket PR evidence works with deprecated flags",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --sha256 ` + suite.artifactFingerprint + ` --evidence-type bb-pr --pipeline ` + suite.pipelineName + `
			          --description text --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Flag --sha256 has been deprecated, use --fingerprint instead\n" +
				"Flag --evidence-type has been deprecated, use --name instead\n" +
				"Flag --description has been deprecated, description is no longer used\n" +
				"bitbucket pull request evidence is reported to artifact: " + suite.artifactFingerprint + "\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when --owner is missing",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69 --api-token foo --host bar`,
			golden: "Error: --owner is not set\n" +
				"Usage: kosli pipeline artifact report evidence bitbucket-pullrequest [IMAGE-NAME | FILE-PATH | DIR-PATH] [flags]\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when both --name and --evidence-type are missing",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --pipeline ` + suite.pipelineName + `
			          --build-url example.com --repository cli --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Error: at least one of --name, --evidence-type is required\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when --bitbucket-username is missing",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Error: required flag(s) \"bitbucket-username\" not set\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when --repository is missing",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Error: required flag(s) \"repository\" not set\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when --commit is missing",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test` + suite.defaultKosliArguments,
			golden: "Error: required flag(s) \"commit\" not set\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when neither --fingerprint nor --artifact-type are set",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest artifactNameArg --name bb-pr --pipeline ` + suite.pipelineName + `
					  --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Error: either --artifact-type or --sha256 must be specified\n" +
				"Usage: kosli pipeline artifact report evidence bitbucket-pullrequest [IMAGE-NAME | FILE-PATH | DIR-PATH] [flags]\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when commit does not exist",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 73d7fee2f31ade8e1a9c456c324255212c3123ab` + suite.defaultKosliArguments,
			golden: "Error: map[error:map[message:Resource not found] type:error]\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when --assert is used and commit has no PRs",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
					  --assert
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit cb6ec5fcbb25b1ebe4859d35ab7995ab973f894c` + suite.defaultKosliArguments,
			golden: "Error: no pull requests found for the given commit: cb6ec5fcbb25b1ebe4859d35ab7995ab973f894c\n",
		},
		{
			name: "report Bitbucket PR evidence does not fail when commit has no PRs",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit cb6ec5fcbb25b1ebe4859d35ab7995ab973f894c` + suite.defaultKosliArguments,
			golden: "no pull requests found for given commit: cb6ec5fcbb25b1ebe4859d35ab7995ab973f894c\n" +
				"bitbucket pull request evidence is reported to artifact: " + suite.artifactFingerprint + "\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when the artifact does not exist in the server",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest testdata/file1 --artifact-type file --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Error: Artifact with fingerprint '7509e5bda0c762d2bac7f90d758b5b2263fa01ccbc542ab5e3df163be08e6ca9' does not exist in pipeline 'bitbucket-pr' belonging to 'docs-cmd-test-user'. \n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when --artifact-type is unsupported",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest testdata/file1 --artifact-type unsupported --name bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Error: unsupported is not a supported artifact type\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when --user-data is not found",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --pipeline ` + suite.pipelineName + `
					  --user-data non-existing.json
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Error: open non-existing.json: no such file or directory\n",
		},
		{
			wantError: true,
			name:      "report Bitbucket PR evidence fails when both --name and --evidence-type are set",
			cmd: `pipeline artifact report evidence bitbucket-pullrequest --fingerprint ` + suite.artifactFingerprint + ` --name bb-pr --evidence-type bb-pr --pipeline ` + suite.pipelineName + `
			          --build-url example.com --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test --commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "Flag --evidence-type has been deprecated, use --name instead\n" +
				"Error: only one of --name, --evidence-type is allowed\n",
		},
	}

	runTestCmd(suite.T(), tests)
}

func (suite *ArtifactEvidencePRBitbucketCommandTestSuite) TestAssertPRBitbucketCmd() {
	tests := []cmdTestCase{
		{
			name: "assert Bitbucket PR evidence passes when commit has a PR in bitbucket",
			cmd: `assert bitbucket-pullrequest --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test 
			--commit 2492011ef04a9da09d35be706cf6a4c5bc6f1e69` + suite.defaultKosliArguments,
			golden: "found [1] pull request(s) in Bitbucket for commit: 2492011ef04a9da09d35be706cf6a4c5bc6f1e69\n",
		},
		{
			wantError: true,
			name:      "assert Bitbucket PR evidence fails when commit has no PRs in bitbucket",
			cmd: `assert bitbucket-pullrequest --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test 
			--commit cb6ec5fcbb25b1ebe4859d35ab7995ab973f894c` + suite.defaultKosliArguments,
			golden: "Error: no pull requests found for the given commit: cb6ec5fcbb25b1ebe4859d35ab7995ab973f894c\n",
		},
		{
			wantError: true,
			name:      "assert Bitbucket PR evidence fails when commit does not exist",
			cmd: `assert bitbucket-pullrequest --bitbucket-username ewelinawilkosz --bitbucket-workspace ewelinawilkosz --repository cli-test 
			--commit 19aab7f063147614451c88969602a10afba123ab` + suite.defaultKosliArguments,
			golden: "Error: map[error:map[message:Resource not found] type:error]\n",
		},
	}

	runTestCmd(suite.T(), tests)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestArtifactEvidencePRBitbucketCommandTestSuite(t *testing.T) {
	suite.Run(t, new(ArtifactEvidencePRBitbucketCommandTestSuite))
}
