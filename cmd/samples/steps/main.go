package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/samarabbas/cadence-samples/cmd/samples/common"
	"go.uber.org/cadence/client"
	"go.uber.org/cadence/worker"
)

// This needs to be done as part of a bootstrap step when the process starts.
// The workers are supposed to be long running.
func startWorkers(h *common.SampleHelper) {
	// Configure worker options.
	workerOptions := worker.Options{
		MetricsScope: h.Scope,
		Logger:       h.Logger,
	}
	h.StartWorkers(h.Config.DomainName, ApplicationName, workerOptions)
}

func startWorkflow(h *common.SampleHelper, template string, objectId string) {
	workflowOptions := client.StartWorkflowOptions{
		ID:                              "test_" + objectId,
		TaskList:                        ApplicationName,
		ExecutionStartToCloseTimeout:    time.Minute*50,
		DecisionTaskStartToCloseTimeout: time.Minute*5,
	}
	h.StartWorkflow(workflowOptions, RootWorkflow, template, objectId)
}

func main() {
	var mode, templateName, objectId, token string
	flag.StringVar(&mode, "m", "trigger", "Mode is worker or trigger.")
	flag.StringVar(&templateName, "t", "template", "the workflow template name")
	flag.StringVar(&objectId, "o", "objectId", "the object id to process")
	flag.StringVar(&token, "token", "token", "the activity token")
	flag.Parse()

	var h common.SampleHelper
	h.SetupServiceConfig()

	switch mode {
	case "worker":
		startWorkers(&h)

		// The workers are supposed to be long running process that should not exit.
		// Use select{} to block indefinitely for samples, you can quit by CMD+C.
		select {}
	case "trigger":
		startWorkflow(&h, templateName, objectId)
	case "complete":
		cadenceClient, err := h.Builder.BuildCadenceClient()
		if err != nil {
			log.Fatalf("failed to create cadence client %v", err)
		}

		taskToken, err := base64.StdEncoding.DecodeString(token)

		if err != nil {
			log.Fatalf("failed to decode the token %v " , err)
		}

		fmt.Println("token => \n", string(taskToken))

		execRes := ExecutionResult{Status: "Jooon"}
		err = cadenceClient.CompleteActivity(context.Background(), taskToken, &execRes, nil)

		if err != nil{
			log.Fatalf("failed to complete the activity %v", err)
		}

		fmt.Println("Done.")
	}
}
