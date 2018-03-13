package service

import (
	"strconv"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/suite"
)

// https://github.com/docker/cli/blob/master/cli/command/service/progress/progress_test.go
// Inspired by `progress_test.go`

type TaskTestSuite struct {
	suite.Suite
	service     swarm.Service
	updater     progressUpdater
	activeNodes map[string]struct{}
	rollback    bool
}

func TestTaskUnitTestSuite(t *testing.T) {
	suite.Run(t, new(TaskTestSuite))
}

func (s *TaskTestSuite) Test_ReplicatedProcessUpdaterOneReplica() {
	replicas := uint64(1)

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{
					Replicas: &replicas,
				},
			},
		},
	}

	s.service = service
	s.updater = new(replicatedProgressUpdater)
	s.activeNodes = map[string]struct{}{"a": {}, "b": {}}

	tasks := []swarm.Task{}

	// No tasks
	s.AssertConvergence(false, tasks)

	// Tasks with DesiredState beyond running is not updated
	tasks = append(tasks,
		swarm.Task{ID: "1",
			NodeID:       "a",
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
		})
	s.AssertConvergence(false, tasks)

	// First time task reaches TaskStateRunning, service has not updated yet
	// The task is "new"
	tasks[0].DesiredState = swarm.TaskStateRunning
	s.AssertConvergence(false, tasks)

	// When an error appears, service is not updated
	tasks[0].Status.Err = "something is wrong"
	s.AssertConvergence(false, tasks)

	// When the tasks reaches running again, updated is true
	tasks[0].Status.Err = ""
	tasks[0].Status.State = swarm.TaskStateRunning
	s.AssertConvergence(true, tasks)

	// When tasks fails, update is false
	tasks[0].Status.Err = "task failed"
	tasks[0].Status.State = swarm.TaskStateFailed
	s.AssertConvergence(false, tasks)

	// If the task is restarted, update is true
	tasks[0].DesiredState = swarm.TaskStateShutdown
	tasks = append(tasks,
		swarm.Task{
			ID:           "2",
			NodeID:       "b",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		})
	s.AssertConvergence(true, tasks)

	// Add a new task while the current one is still running, to simulate
	// "start-then-stop" updates.
	tasks = append(tasks,
		swarm.Task{
			ID:           "3",
			NodeID:       "b",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
		})
	s.AssertConvergence(false, tasks)

}

func (s *TaskTestSuite) Test_ReplicatedProcessUpdaterManyReplica() {
	replicas := uint64(50)
	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Replicated: &swarm.ReplicatedService{
					Replicas: &replicas,
				},
			},
		},
	}

	s.service = service
	s.updater = new(replicatedProgressUpdater)
	s.activeNodes = map[string]struct{}{"a": {}, "b": {}}

	tasks := []swarm.Task{}

	// No tasks
	s.AssertConvergence(false, tasks)

	for i := 0; i != int(replicas); i++ {
		tasks = append(tasks,
			swarm.Task{
				ID:           strconv.Itoa(i),
				Slot:         i + 1,
				NodeID:       "a",
				DesiredState: swarm.TaskStateRunning,
				Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			})
		if i%2 == 1 {
			tasks[i].NodeID = "b"
		}
		s.AssertConvergence(false, tasks)
		tasks[i].Status.State = swarm.TaskStateRunning
		s.AssertConvergence(uint64(i) == replicas-1, tasks)
	}
}

func (s *TaskTestSuite) Test_GlobalProgressUpdaterOneNode() {

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Global: &swarm.GlobalService{},
			},
		},
	}

	s.activeNodes = map[string]struct{}{"a": {}, "b": {}}
	s.service = service
	s.updater = new(globalProgressUpdater)

	tasks := []swarm.Task{}

	// No tasks
	s.AssertConvergence(false, tasks)

	// Task with DesiredState beyond Running is ignored
	tasks = append(tasks,
		swarm.Task{
			ID:           "1",
			NodeID:       "a",
			DesiredState: swarm.TaskStateShutdown,
			Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
		})
	s.AssertConvergence(false, tasks)

	// First time task reaches TaskStateRunning, service has not converged yet
	// The task is "new"
	tasks[0].DesiredState = swarm.TaskStateRunning
	s.AssertConvergence(false, tasks)

	// If the task exposes an error, update is false
	tasks[0].Status.Err = "something is wrong"
	s.AssertConvergence(false, tasks)

	// When the task reaches running, update is true
	tasks[0].Status.Err = ""
	tasks[0].Status.State = swarm.TaskStateRunning
	s.AssertConvergence(true, tasks)

	// If the task fails, update is false
	tasks[0].Status.Err = "task failed"
	tasks[0].Status.State = swarm.TaskStateFailed
	s.AssertConvergence(false, tasks)

	// If task is restarted, update is true
	tasks[0].DesiredState = swarm.TaskStateShutdown
	tasks = append(tasks,
		swarm.Task{
			ID:           "2",
			NodeID:       "a",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStateRunning},
		})
	s.AssertConvergence(true, tasks)

	tasks = append(tasks,
		swarm.Task{
			ID:           "3",
			NodeID:       "a",
			DesiredState: swarm.TaskStateRunning,
			Status:       swarm.TaskStatus{State: swarm.TaskStatePreparing},
		})
	s.AssertConvergence(false, tasks)

}

func (s *TaskTestSuite) Test_GlobalProgressUpdaterManyNodes() {
	nodes := 50

	service := swarm.Service{
		Spec: swarm.ServiceSpec{
			Mode: swarm.ServiceMode{
				Global: &swarm.GlobalService{},
			},
		},
	}

	s.service = service
	s.updater = new(globalProgressUpdater)
	s.activeNodes = map[string]struct{}{}

	for i := 0; i != nodes; i++ {
		s.activeNodes[strconv.Itoa(i)] = struct{}{}
	}

	tasks := []swarm.Task{}

	// No tasks
	s.AssertConvergence(false, tasks)

	for i := 0; i != nodes; i++ {
		tasks = append(tasks,
			swarm.Task{
				ID:           "task" + strconv.Itoa(i),
				NodeID:       strconv.Itoa(i),
				DesiredState: swarm.TaskStateRunning,
				Status:       swarm.TaskStatus{State: swarm.TaskStateNew},
			})
	}
	// All tasks are in "new" state
	s.AssertConvergence(false, tasks)

	for i := 0; i != nodes; i++ {
		tasks[i].Status.State = swarm.TaskStateRunning
		s.AssertConvergence(i == nodes-1, tasks)
	}
}

func (s *TaskTestSuite) AssertConvergence(expectedConvergence bool, tasks []swarm.Task) {
	converged, err := s.updater.update(
		s.service, tasks, s.activeNodes, s.rollback)
	s.Require().NoError(err)
	s.Equal(expectedConvergence, converged)
}
