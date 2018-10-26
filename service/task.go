package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// https://github.com/docker/cli/blob/master/cli/command/service/progress/progress.go
// The same structure as `progress.go` without stdout

var (
	numberedStates = map[swarm.TaskState]int64{
		swarm.TaskStateNew:       1,
		swarm.TaskStateAllocated: 2,
		swarm.TaskStatePending:   3,
		swarm.TaskStateAssigned:  4,
		swarm.TaskStateAccepted:  5,
		swarm.TaskStatePreparing: 6,
		swarm.TaskStateReady:     7,
		swarm.TaskStateStarting:  8,
		swarm.TaskStateRunning:   9,

		// The following states are not actually shown in progress
		// output, but are used internally for ordering.
		swarm.TaskStateComplete: 10,
		swarm.TaskStateShutdown: 11,
		swarm.TaskStateFailed:   12,
		swarm.TaskStateRejected: 13,
	}

	longestState int
)

func init() {
	for state := range numberedStates {
		if !terminalState(state) && len(state) > longestState {
			longestState = len(state)
		}
	}
}

func terminalState(state swarm.TaskState) bool {
	return numberedStates[state] > numberedStates[swarm.TaskStateRunning]
}

func stateToProgress(state swarm.TaskState, rollback bool) int64 {
	if !rollback {
		return numberedStates[state]
	}
	return numberedStates[swarm.TaskStateRunning] - numberedStates[state]
}

func getActiveNodes(ctx context.Context, client *client.Client) (map[string]struct{}, error) {
	nodes, err := client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}

	activeNodes := make(map[string]struct{})
	for _, n := range nodes {
		if n.Status.State != swarm.NodeStateDown {
			activeNodes[n.ID] = struct{}{}
		}
	}
	return activeNodes, nil
}

func initializeUpdater(service swarm.Service) (progressUpdater, error) {
	if service.Spec.Mode.Replicated != nil && service.Spec.Mode.Replicated.Replicas != nil {
		return &replicatedProgressUpdater{}, nil
	}
	if service.Spec.Mode.Global != nil {
		return &globalProgressUpdater{}, nil
	}
	return nil, errors.New("unrecognized service mode")
}

type progressUpdater interface {
	update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error)
}

// GetTaskList returns tasks when it is the service is converged
func GetTaskList(ctx context.Context, client *client.Client, serviceID string) ([]swarm.Task, error) {

	taskFilter := filters.NewArgs()
	taskFilter.Add("service", serviceID)
	taskFilter.Add("_up-to-date", "true")
	taskFilter.Add("desired-state", "running")
	taskFilter.Add("desired-state", "accepted")

	getUpToDateTasks := func() ([]swarm.Task, error) {
		return client.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
	}

	var (
		updater     progressUpdater
		converged   bool
		convergedAt time.Time
		monitor     = 5 * time.Second
		rollback    bool
	)

	taskList, err := getUpToDateTasks()
	if err != nil {
		return taskList, err
	}

	for {
		service, _, err := client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
		if err != nil {
			return taskList, err
		}

		if service.Spec.UpdateConfig != nil && service.Spec.UpdateConfig.Monitor != 0 {
			monitor = service.Spec.UpdateConfig.Monitor
		}

		if updater == nil {
			updater, err = initializeUpdater(service)
			if err != nil {
				return taskList, err
			}
		}

		if service.UpdateStatus != nil {
			switch service.UpdateStatus.State {
			case swarm.UpdateStateUpdating:
				rollback = false
			case swarm.UpdateStateCompleted:
				if !converged {
					return taskList, nil
				}
			case swarm.UpdateStatePaused:
				return taskList, fmt.Errorf("service update paused: %s", service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackStarted:
				rollback = true
			case swarm.UpdateStateRollbackPaused:
				return taskList, fmt.Errorf("service rollback paused %s", service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackCompleted:
				if !converged {
					return taskList, fmt.Errorf("service rolled back: %s", service.UpdateStatus.Message)
				}
			}
		}
		if converged && time.Since(convergedAt) >= monitor {
			return taskList, nil
		}

		taskList, err = getUpToDateTasks()
		if err != nil {
			return taskList, err
		}

		activeNodes, err := getActiveNodes(ctx, client)
		if err != nil {
			return taskList, err
		}

		converged, err = updater.update(service, taskList, activeNodes, rollback)
		if err != nil {
			return taskList, err
		}
		if converged {
			if convergedAt.IsZero() {
				convergedAt = time.Now()
			}
		} else {
			convergedAt = time.Time{}
		}

		<-time.After(200 * time.Millisecond)
	}

}

// TasksAllRunning checks if a service is currently up and running
func TasksAllRunning(ctx context.Context, cli *client.Client, serviceID string) (bool, error) {

	service, _, err := cli.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return false, err
	}
	updater, err := initializeUpdater(service)
	if err != nil {
		return false, err
	}

	taskFilter := filters.NewArgs()
	taskFilter.Add("service", serviceID)
	taskFilter.Add("_up-to-date", "true")
	taskFilter.Add("desired-state", "running")
	taskFilter.Add("desired-state", "accepted")

	tasks, err := cli.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
	if err != nil {
		return false, err
	}

	activeNodes, err := getActiveNodes(ctx, cli)
	if err != nil {
		return false, err
	}

	return updater.update(service, tasks, activeNodes, false)
}

type replicatedProgressUpdater struct {
	initialized bool
	done        bool
}

func (u *replicatedProgressUpdater) update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {

	if service.Spec.Mode.Replicated == nil || service.Spec.Mode.Replicated.Replicas == nil {
		return false, errors.New("no replica count")
	}

	replicas := *service.Spec.Mode.Replicated.Replicas

	if !u.initialized {
		u.initialized = true
	}

	tasksBySlot := u.tasksBySlot(tasks, activeNodes)

	// If we had reached a converged state, check if we are still converged.
	if u.done {
		for _, task := range tasksBySlot {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := uint64(0)

	for _, task := range tasksBySlot {
		if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
			running++
		}
	}

	if !u.done && running == replicas {
		u.done = true
	}

	return u.done == true, nil
}

func (u *replicatedProgressUpdater) tasksBySlot(tasks []swarm.Task, activeNodes map[string]struct{}) map[int]swarm.Task {
	// If there are multiple tasks with the same slot number, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksBySlot := make(map[int]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksBySlot[task.Slot]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}
			// If the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}
		}
		if task.NodeID != "" {
			if _, nodeActive := activeNodes[task.NodeID]; !nodeActive {
				continue
			}
		}
		tasksBySlot[task.Slot] = task
	}

	return tasksBySlot
}

type globalProgressUpdater struct {
	initialized bool
	done        bool
}

func (u *globalProgressUpdater) update(service swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {
	tasksByNode := u.tasksByNode(tasks)
	// We don't have perfect knowledge of how many nodes meet the
	// constraints for this service. But the orchestrator creates tasks
	// for all eligible nodes at the same time, so we should see all those
	// nodes represented among the up-to-date tasks.
	nodeCount := len(tasksByNode)

	if !u.initialized {
		if nodeCount == 0 {
			// Two possibilities: either the orchestrator hasn't created
			// the tasks yet, or the service doesn't meet constraints for
			// any node. Either way, we wait.
			return false, nil
		}

		u.initialized = true
	}

	// If we had reached a converged state, check if we are still converged.
	if u.done {
		for _, task := range tasksByNode {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := 0

	for _, task := range tasksByNode {
		if _, nodeActive := activeNodes[task.NodeID]; nodeActive {
			if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
				running++
			}
		}
	}

	if !u.done && running == nodeCount {
		u.done = true
	}

	return running == nodeCount, nil
}

func (u *globalProgressUpdater) tasksByNode(tasks []swarm.Task) map[string]swarm.Task {
	// If there are multiple tasks with the same node ID, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksByNode := make(map[string]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksByNode[task.NodeID]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}

			// If the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}

		}
		tasksByNode[task.NodeID] = task
	}

	return tasksByNode
}
