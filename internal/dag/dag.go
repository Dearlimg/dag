package dag

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrCircularDependency = errors.New("circular dependency detected")
	ErrTaskNotFound       = errors.New("task not found")
	ErrTaskAlreadyExists  = errors.New("task already exists")
	ErrDependencyNotFound = errors.New("dependency not found")
)

type TaskStatus int

const (
	StatusPending TaskStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusSkipped
)

func (s TaskStatus) String() string {
	switch s {
	case StatusPending:
		return "pending"
	case StatusRunning:
		return "running"
	case StatusCompleted:
		return "completed"
	case StatusFailed:
		return "failed"
	case StatusSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

type Task struct {
	ID           string
	Name         string
	Executor     func() error
	Dependencies []string
	Status       TaskStatus
	Err          error
	StartTime    time.Time
	EndTime      time.Time
}

func NewTask(id, name string, executor func() error, dependencies ...string) *Task {
	return &Task{
		ID:           id,
		Name:         name,
		Executor:     executor,
		Dependencies: dependencies,
		Status:       StatusPending,
	}
}

func (t *Task) Duration() time.Duration {
	if t.StartTime.IsZero() || t.EndTime.IsZero() {
		return 0
	}
	return t.EndTime.Sub(t.StartTime)
}

type RetryPolicy struct {
	MaxRetries    int
	RetryDelay    time.Duration
	RetryableFunc func(error) bool
}

type ExecuteOptions struct {
	MaxWorkers      int
	ContinueOnError bool
	Timeout         time.Duration
	RetryPolicy     *RetryPolicy
}

func DefaultExecuteOptions() ExecuteOptions {
	return ExecuteOptions{
		MaxWorkers:      4,
		ContinueOnError: false,
		Timeout:         0,
	}
}

type ExecutionError struct {
	TaskID   string
	TaskName string
	Err      error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("task %q (%s) failed: %v", e.TaskName, e.TaskID, e.Err)
}

type PipelineError struct {
	Errors []*ExecutionError
}

func (e *PipelineError) Error() string {
	if len(e.Errors) == 0 {
		return "pipeline executed successfully"
	}
	return fmt.Sprintf("pipeline failed with %d error(s)", len(e.Errors))
}

func (e *PipelineError) Add(err *ExecutionError) {
	e.Errors = append(e.Errors, err)
}

type ExecutionSummary struct {
	TotalTasks     int
	CompletedTasks int
	FailedTasks    int
	SkippedTasks   int
	ExecutionTime  time.Duration
	Errors         []*ExecutionError
}

type Pipeline struct {
	tasks                  map[string]*Task
	dependencyGraph        map[string][]string
	reverseDependencyGraph map[string][]string
	mu                     sync.RWMutex
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		tasks:                  make(map[string]*Task),
		dependencyGraph:        make(map[string][]string),
		reverseDependencyGraph: make(map[string][]string),
	}
}

func (p *Pipeline) AddTask(task *Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.tasks[task.ID]; exists {
		return fmt.Errorf("%w: %s", ErrTaskAlreadyExists, task.ID)
	}

	for _, depID := range task.Dependencies {
		if _, exists := p.tasks[depID]; !exists {
			return fmt.Errorf("%w: %s", ErrDependencyNotFound, depID)
		}
	}

	p.tasks[task.ID] = task
	p.dependencyGraph[task.ID] = make([]string, len(task.Dependencies))
	copy(p.dependencyGraph[task.ID], task.Dependencies)

	for _, depID := range task.Dependencies {
		p.reverseDependencyGraph[depID] = append(p.reverseDependencyGraph[depID], task.ID)
	}

	return nil
}

func (p *Pipeline) GetTask(taskID string) (*Task, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}
	return task, nil
}

func (p *Pipeline) topologicalSort() ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	inDegree := make(map[string]int)
	for id := range p.tasks {
		inDegree[id] = 0
	}

	for id, deps := range p.dependencyGraph {
		inDegree[id] = len(deps)
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var result []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, dependent := range p.reverseDependencyGraph[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(p.tasks) {
		return nil, ErrCircularDependency
	}

	return result, nil
}

func (p *Pipeline) Execute() error {
	return p.ExecuteWithOptions(DefaultExecuteOptions())
}

func (p *Pipeline) ExecuteWithOptions(options ExecuteOptions) error {
	if _, err := p.topologicalSort(); err != nil {
		return err
	}

	p.mu.Lock()
	for _, task := range p.tasks {
		task.Status = StatusPending
		task.Err = nil
		task.StartTime = time.Time{}
		task.EndTime = time.Time{}
	}
	p.mu.Unlock()

	errors := &PipelineError{}
	taskChan := make(chan *Task, len(p.tasks))
	resultChan := make(chan *Task, len(p.tasks))

	workers := options.MaxWorkers
	if workers <= 0 {
		workers = 1
	}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go p.worker(taskChan, resultChan, &wg, options)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(resultChan)
		close(done)
	}()

	initialReady := p.getReadyTasks()
	for _, task := range initialReady {
		p.mu.Lock()
		task.Status = StatusRunning
		p.mu.Unlock()
		taskChan <- task
	}

	activeCount := len(initialReady)
	totalTasks := len(p.tasks)

	for activeCount > 0 {
		select {
		case task := <-resultChan:
			activeCount--

			if task.Status == StatusFailed {
				errors.Add(&ExecutionError{
					TaskID:   task.ID,
					TaskName: task.Name,
					Err:      task.Err,
				})

				if !options.ContinueOnError {
					close(taskChan)
					<-done
					if len(errors.Errors) > 0 {
						return errors
					}
					return nil
				}
			}

			skippedTasks := p.checkAndMarkDependents(task)
			for _, t := range skippedTasks {
				if t.Status == StatusSkipped {
					continue
				}
				p.mu.Lock()
				t.Status = StatusRunning
				p.mu.Unlock()
				taskChan <- t
				activeCount++
			}

			completedCount := 0
			p.mu.RLock()
			for _, t := range p.tasks {
				if t.Status == StatusCompleted || t.Status == StatusFailed || t.Status == StatusSkipped {
					completedCount++
				}
			}
			p.mu.RUnlock()

			if completedCount == totalTasks {
				close(taskChan)
			}
		}
	}

	<-done

	if len(errors.Errors) > 0 {
		return errors
	}

	return nil
}

func (p *Pipeline) getReadyTasks() []*Task {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var ready []*Task
	for _, task := range p.tasks {
		if task.Status == StatusPending {
			allDepsComplete := true
			for _, depID := range task.Dependencies {
				dep := p.tasks[depID]
				if dep.Status != StatusCompleted {
					allDepsComplete = false
					break
				}
			}
			if allDepsComplete {
				ready = append(ready, task)
			}
		}
	}
	return ready
}

func (p *Pipeline) checkAndMarkDependents(completedTask *Task) []*Task {
	p.mu.Lock()
	defer p.mu.Unlock()

	var result []*Task
	dependents := p.reverseDependencyGraph[completedTask.ID]

	for _, depID := range dependents {
		task := p.tasks[depID]
		if task.Status != StatusPending {
			continue
		}

		if completedTask.Status == StatusFailed {
			task.Status = StatusSkipped
			result = append(result, task)
			continue
		}

		allDepsComplete := true
		for _, depID2 := range task.Dependencies {
			dep := p.tasks[depID2]
			if dep.Status != StatusCompleted {
				allDepsComplete = false
				break
			}
		}
		if allDepsComplete {
			result = append(result, task)
		}
	}

	return result
}

func (p *Pipeline) worker(taskChan <-chan *Task, resultChan chan<- *Task, wg *sync.WaitGroup, options ExecuteOptions) {
	defer wg.Done()

	for task := range taskChan {
		p.executeTask(task, options)
		resultChan <- task
	}
}

func (p *Pipeline) executeTask(task *Task, options ExecuteOptions) {
	p.mu.Lock()
	task.StartTime = time.Now()
	p.mu.Unlock()

	var err error
	retries := 0

	for {
		err = task.Executor()
		if err == nil {
			break
		}

		if options.RetryPolicy == nil {
			break
		}

		if options.RetryPolicy.RetryableFunc != nil && !options.RetryPolicy.RetryableFunc(err) {
			break
		}

		retries++
		if retries >= options.RetryPolicy.MaxRetries {
			break
		}

		if options.RetryPolicy.RetryDelay > 0 {
			time.Sleep(options.RetryPolicy.RetryDelay)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	task.EndTime = time.Now()
	task.Err = err

	if err != nil {
		task.Status = StatusFailed
	} else {
		task.Status = StatusCompleted
	}
}

func (p *Pipeline) GetExecutionSummary() *ExecutionSummary {
	p.mu.RLock()
	defer p.mu.RUnlock()

	summary := &ExecutionSummary{
		TotalTasks: len(p.tasks),
	}

	var earliestStart, latestEnd time.Time

	for _, task := range p.tasks {
		switch task.Status {
		case StatusCompleted:
			summary.CompletedTasks++
		case StatusFailed:
			summary.FailedTasks++
			summary.Errors = append(summary.Errors, &ExecutionError{
				TaskID:   task.ID,
				TaskName: task.Name,
				Err:      task.Err,
			})
		case StatusSkipped:
			summary.SkippedTasks++
		}

		if !task.StartTime.IsZero() && (earliestStart.IsZero() || task.StartTime.Before(earliestStart)) {
			earliestStart = task.StartTime
		}
		if !task.EndTime.IsZero() && (latestEnd.IsZero() || task.EndTime.After(latestEnd)) {
			latestEnd = task.EndTime
		}
	}

	if !earliestStart.IsZero() && !latestEnd.IsZero() {
		summary.ExecutionTime = latestEnd.Sub(earliestStart)
	}

	return summary
}

func (p *Pipeline) GetTaskStatus(taskID string) (TaskStatus, error) {
	task, err := p.GetTask(taskID)
	if err != nil {
		return StatusPending, err
	}
	return task.Status, nil
}
