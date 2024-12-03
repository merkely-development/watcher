package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type CreateAttestationTypeTestSuite struct {
	suite.Suite
	defaultKosliArguments string
}

func (suite *CreateAttestationTypeTestSuite) SetupTest() {
	global = &GlobalOpts{
		ApiToken: "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpZCI6ImNkNzg4OTg5In0.e8i_lA_QrEhFncb05Xw6E_tkCHU9QfcY4OLTVUCHffY",
		Org:      "docs-cmd-test-user",
		Host:     "http://localhost:8001",
	}
	suite.defaultKosliArguments = fmt.Sprintf(" --host %s --org %s --api-token %s", global.Host, global.Org, global.ApiToken)
}

func (suite *CreateAttestationTypeTestSuite) TestCustomAttestationTypeCmd() {
	tests := []cmdTestCase{
		{
			wantError: true,
			name:      "fails when more arguments are provided",
			cmd:       "create attestation-type" + suite.defaultKosliArguments,
			golden:    "Error: accepts 1 arg(s), received 0\n",
		},
	}

	runTestCmd(suite.T(), tests)
}u

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestCreateAttestationTypeTestSuite(t *testing.T) {
	suite.Run(t, new(CreateAttestationTypeTestSuite))
}
