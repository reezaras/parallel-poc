package main

import (
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/testsuite"
	"testing"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func(s *UnitTestSuite) TestRootWorkflow(t *testing.T) {
	s.NewTestWorkflowEnvironment()
}