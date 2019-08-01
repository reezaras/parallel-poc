package main

const (
	ParallelExecutionStrategy = ExecutionStrategy("Parallel")
	SequenceExecutionStrategy = ExecutionStrategy("Sequence")

	AndMergeTerminationStrategy = TerminationStrategy("And-Merge")
	OrMergeTerminationStrategy  = TerminationStrategy("Or-Merge")
)

type (
	ExecutionStrategy   string
	TerminationStrategy string

	WFStep struct {
		Name                string              `json:"name"`
		Assignee            string              `json:"assignee"`
		ExecutionStrategy   ExecutionStrategy   `json:"execute"`
		TerminationStrategy TerminationStrategy `json:"terminate"`
		Steps               []WFStep            `json:"steps"`
	}

	// the execution request which travels outside of wf engine
	ExecutionRequest struct {
		UUID         string
		NumA         int
		NumB         int
		Assignee     string
		Token        string
		StepId       string
		TaskWFId     string
		TaskWFRunId  string
		TaskRootWFId string
		TaskRootWFRunId string
	}

	// the execution request returned to the execution engine
	ExecutionResult struct {
		RequestUUID string
		Result      int
		Status      string
		Token       string
		Error       string
	}
)
