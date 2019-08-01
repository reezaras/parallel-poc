package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"
	"io/ioutil"
	"os"
	"path/filepath"
)



func LoadWorkflowTemplateActivity(ctx context.Context, templateName string, templatePath string) (*WFStep, error) {
	templateAbsPath := filepath.Join(templatePath, templateName)
	_, err := os.Stat(templateAbsPath)
	if os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "template file not found")
	}

	bytes, err := ioutil.ReadFile(templateAbsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read from file")
	}

	var root WFStep
	err = json.Unmarshal(bytes, &root)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse template json file")
	}

	return &root, nil
}

func SendTaskRequestActivity(ctx context.Context, req ExecutionRequest) (ExecutionResult, error) {

	activityInfo := activity.GetInfo(ctx)


	logger := activity.GetLogger(ctx).With(
		zap.String("LEAF_STEP_REQ_UUID", req.UUID),
		zap.String("LEAF_STEP_TEMPLATE_ID", req.StepId),
		zap.String("LEAF_STEP_CADENCE_WF_ID", req.TaskWFId),
		zap.String("LEAF_STEP_ROOT_WF_ID", req.TaskRootWFId),
		zap.String("LEAF_STEP_ASSIGNEE", req.Assignee),
	)

	logger.Info("### => Task Token " + base64.StdEncoding.EncodeToString(activityInfo.TaskToken))
	logger.Info("####### sent the request.... waiting for the async completion ############")

	return ExecutionResult{Error:"From Activity!"}, activity.ErrResultPending
}
