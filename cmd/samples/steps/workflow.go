package main

import (
	"fmt"
	"github.com/prometheus/common/log"
	uuid "github.com/satori/go.uuid"
	"go.uber.org/cadence"
	"go.uber.org/cadence/activity"
	"time"

	"go.uber.org/cadence/workflow"
	"go.uber.org/zap"
)

// ApplicationName is the task list for this sample
const ApplicationName = "wf_engine"

// This is registration process where you register all your workflows
// and activity function handlers.
func init() {
	workflow.Register(RootWorkflow)
	workflow.Register(LeafStepWorkflow)
	workflow.Register(BranchingWorkflow)
	activity.Register(LoadWorkflowTemplateActivity)
	activity.Register(SendTaskRequestActivity)
}

func RootWorkflow(ctx workflow.Context, templateName string, objectId string) error {

	logger := workflow.GetLogger(ctx).With(zap.String("templateName", templateName),
		zap.String("objectId", objectId))

	logger.Info("Loading the template file.")

	step, err := parseWorkflowTemplate(ctx, logger, templateName, templatePath)

	if err != nil {
		logger.Info(">>>>>>> failed to parse workflow template !", zap.Error(err))
		return cadence.NewCustomError("root workflow failed because could not parse the workflow template!", zap.Error(err))
	}

	wfInfo := workflow.GetInfo(ctx)
	rootWFId := wfInfo.WorkflowExecution.ID
	rootWFRunId := wfInfo.WorkflowExecution.RunID

	logger.Info("*********** [Starting Root Branch] ************")
	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 20 * time.Minute,
		TaskStartToCloseTimeout:      1 * time.Minute,
	}
	childCtx := workflow.WithChildOptions(ctx, cwo)

	logger.Info(">>>>> Starting branching workflow...",
		zap.String("stepId", step.Name),
		zap.String("RootWFID", rootWFId),
		zap.String("RunWFID", rootWFRunId),
		zap.String("ObjectId", objectId))

	err = workflow.ExecuteChildWorkflow(childCtx, BranchingWorkflow, step, rootWFId, rootWFRunId, objectId).Get(childCtx, nil)

	if err != nil {
		logger.Info("branching wf at root failed", zap.Error(err))
		return cadence.NewCustomError("branching wf at root failed", zap.Error(err))
	}

	logger.Info(">>>>>> root workflow finished !")

	return nil
}

func BranchingWorkflow(ctx workflow.Context, step *WFStep, rootWFId string, rootWFRunID string, objectId string) error {
	logger := workflow.GetLogger(ctx).With(
		zap.String("objectId", objectId))

	logger.Info(">>>>>>> Branching Child Workflow Begin for step " + step.Name + " <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")

	if step == nil {
		logger.Info(">>>>> Step is NULL RETURNING <<<<<<<<<<<<<<<<<")
		return nil
	}

	// we got a leaf with the exact definition
	if len(step.Steps) == 0 {
		logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [START]  Blocking Execution for Single  Leaf :  " + step.Name + " <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,")
		result, err := runLeafChildWFSync(ctx, step, rootWFId, rootWFId, objectId)

		if err != nil {
			logger.Error(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [ERROR] Blocking Execution for Single  Leaf :  "+step.Name+" <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,", zap.Error(err))
			return err
		}

		logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [END] Blocking Execution for Single  Leaf :  "+step.Name+" <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,",
			zap.Int("result", result.Result),
			zap.String("Token", result.Token),
			zap.String("Request UUID", result.RequestUUID),
		)

		return nil
	}

	if len(step.Steps) > 0 {

		if step.TerminationStrategy == AndMergeTerminationStrategy {
			logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [PARALLEL-AND-MERGE] for Branch :  " + step.Name + " <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,")
			return andMergeParallelTermination(ctx, logger, step.Steps, rootWFId, rootWFRunID, objectId)
		}

		//if step.TerminationStrategy == OrMergeTerminationStrategy {
		//	// TODO
		//}
	}

	return nil

}

func runLeafChildWFSync(ctx workflow.Context, step *WFStep, rootWFId string, rootWFRunID string, objectId string) (*ExecutionResult, error) {

	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 10 * time.Minute,
		TaskStartToCloseTimeout:      1 * time.Minute,
	}
	childCtx := workflow.WithChildOptions(ctx, cwo)

	processingFuture := workflow.ExecuteChildWorkflow(childCtx, LeafStepWorkflow, step, rootWFId, rootWFRunID, objectId)

	var result ExecutionResult
	err := processingFuture.Get(childCtx, &result)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func runBranchChildWFAsync(ctx workflow.Context, step WFStep, rootWFId string, rootWFRunID string, objectId string) workflow.Future {

	cwo := workflow.ChildWorkflowOptions{
		ExecutionStartToCloseTimeout: 15 * time.Minute,
		TaskStartToCloseTimeout:      1 * time.Minute,
	}
	childCtx := workflow.WithChildOptions(ctx, cwo)
	processingFuture := workflow.ExecuteChildWorkflow(childCtx, BranchingWorkflow, &step, rootWFId, rootWFRunID, objectId)
	return processingFuture
}


func LeafStepWorkflow(ctx workflow.Context, step *WFStep, rootWFId string, rootWFRunID string, objectId string) (*ExecutionResult, error) {
	logger := workflow.GetLogger(ctx).With(
		zap.String("LeafStep", step.Name),
		zap.String("Assignee", step.Assignee),
		zap.String("ObjectId", objectId),
	)

	logger.Info("Leaf Workflow Strated.")

	var uuid4 string
	err := workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		uuidV4 := uuid.NewV4().String()
		return uuidV4
	}).Get(&uuid4)

	if err != nil {
		logger.Error("failed to generate the fucking UUID4!", zap.Error(err))
		return nil, cadence.NewCustomError("failed to generate the UUID4", zap.Error(err))
	}

	wfInfo := workflow.GetInfo(ctx)

	execRequest := ExecutionRequest{
		UUID:            uuid4,
		StepId:          step.Name,
		Assignee:        step.Assignee,
		NumA:            0,
		NumB:            0,
		TaskRootWFId:    rootWFId,
		TaskRootWFRunId: rootWFRunID,
		TaskWFId:        wfInfo.WorkflowExecution.ID,
		TaskWFRunId:     wfInfo.WorkflowExecution.RunID,
	}

	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Second * 20,
		StartToCloseTimeout:    time.Minute * 5,
	}

	logger.Info("############### Starting to send request !!!!")
	actx := workflow.WithActivityOptions(ctx, ao)
	var result ExecutionResult
	err = workflow.ExecuteActivity(actx, SendTaskRequestActivity, execRequest).Get(actx, &result)

	if err != nil {
		logger.Error("########## Leaf activity failed.", zap.Error(err))
		return nil, cadence.NewCustomError(fmt.Sprintf(">>>>>> child workflow failed %v", err))
	}

	log.Info("Activity Completed with status", zap.String("RETURNED_STATUS", result.Status))

	return &result, nil
}

func andMergeParallelTermination(ctx workflow.Context, logger *zap.Logger, steps []WFStep, rootWFId string, rootWFRunID string, objectId string) error {

	logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [CREATING BRANCH] Creating Branch. Number of steps =" + string(len(steps)) + " <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,")
	childCtx, _ := workflow.WithCancel(ctx)
	//selector := workflow.NewSelector(ctx)
	futureList := []workflow.Future{}
	//var activityErr error
	for _, s := range steps {
		logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [START NEW BRANCH Async] Starting new branch for  :  " + s.Name + " <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,")

		f := runBranchChildWFAsync(childCtx, s, rootWFId, rootWFRunID, objectId)
		logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [IN-PROGRESS  BRANCH EXECUTION] Branch become in progress  :  " + s.Name + " <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,")
		futureList = append(futureList, f)
	}

	logger.Info(">>>>>>>>>>>>>>>>>>>>>>>>>>>> [ALL BRANCH RUN] ALL BRANCHES IN PROGRESS   <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<,")

	logger.Info(">>>>>>>>>>>>>>>>> [WAITING FOR ALL BRANCHES TO FINISH] <<<<<<<<<<<<<<<<<<<<")
	for i := 0; i < len(futureList); i++ {
		curF := futureList[i]
		err := curF.Get(childCtx, nil)

		if err != nil {
			logger.Info(">>>>>>>>>>>>>>>>> [** Branch Error **]  at "+steps[i].Name+"<<<<<<<<<<<<<<<<<<<< ", zap.Error(err))
			return err
		}
	}

	return nil
}

func parseWorkflowTemplate(ctx workflow.Context, logger *zap.Logger, templateName string, path string) (*WFStep, error) {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: time.Minute,
		StartToCloseTimeout:    time.Minute,
	}

	actx := workflow.WithActivityOptions(ctx, ao)
	fu := workflow.ExecuteActivity(actx, LoadWorkflowTemplateActivity, templateName, path)
	var template WFStep
	err := fu.Get(actx, &template)
	if err != nil {
		logger.Error("failed to get workflow template", zap.Error(err), zap.String("path", path))
		return nil, cadence.NewCustomError("parseWorkflowTemplate: failed to parse workflow template",
			zap.String("templateName", templateName),
			zap.Error(err),
		)
	}

	return &template, nil
}
