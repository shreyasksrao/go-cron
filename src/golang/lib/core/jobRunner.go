package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DEFAULT_MAX_RUNNING_JOBS    = 100
	DEFAULT_JOB_RUN_CHAN_BUFFER = 50
	DEFAULT_MONITOR_TICKER      = 60 * time.Second
)

type JobRunner struct {
	MaxRunningJobCount int16
	RunningJobCount    int16
	RunningJobCountMu  sync.Mutex
	RunningJobs        []*JobRun
	RunningJobsMu      sync.Mutex
	Logger             Logger
	stopChan           chan struct{}
	JobRunChan         chan *JobRun
}

type JobRun struct {
	ID          string
	Job         Job
	ScheduledAt time.Time
	RanAt       time.Time
	CompletedAt time.Time
	Running     bool
	Logger      Logger
}

func NewJobRunner(logger Logger, maxRunningJobs int16, jobRunnerChan chan *JobRun) (jobRunner *JobRunner) {
	logger.Infof("Creating a new instance of JobRunner...")
	if maxRunningJobs == 0 {
		logger.Infof("Max running jobs set to default value - %v", DEFAULT_MAX_RUNNING_JOBS)
		maxRunningJobs = DEFAULT_MAX_RUNNING_JOBS
	}
	jobRunner = &JobRunner{
		MaxRunningJobCount: maxRunningJobs,
		RunningJobCount:    0,
		RunningJobs:        make([]*JobRun, 0),
		RunningJobCountMu:  sync.Mutex{},
		Logger:             logger,
		stopChan:           make(chan struct{}),
		JobRunChan:         jobRunnerChan,
	}
	logger.Infof("Successfully created the instance of Job Runner.")
	return
}

// CreateJobRun creates an instance of the JobRun struct and populate the fields.
func (jr *JobRunner) CreateJobRun(job Job) (jobRun *JobRun) {
	jr.Logger.Infof("Creating a new JobRun instance for the Job - %v, Schedule time - %v",
		job.GetCommonJobFields().ID, job.GetCommonJobFields().NextRun)
	jobRun = &JobRun{
		ID:          uuid.New().String(),
		Job:         job,
		Logger:      jr.Logger,
		ScheduledAt: job.GetCommonJobFields().NextRun,
		Running:     false,
	}
	return
}

func (jr *JobRunner) Start() (err error) {
	jr.Logger.Infof("Starting the Job runner...")
	go jr.monitorJobRunner()
	for {
		select {
		case jobRun := <-jr.JobRunChan:
			jr.Logger.Infof("Recieved job on the job run channel.")
			jr.runJob(jobRun)
		case <-jr.stopChan:
			jr.Logger.Infof("Recieved signal on stop channel.")
			return
		}
	}
}

func (jr *JobRunner) Stop() (err error) {
	jr.Logger.Infof("Stopping the Job runner...")
	defer jr.Logger.Infof("Stopped the Job runner.")
	jr.stopChan <- struct{}{}
	for _, jobRun := range jr.RunningJobs {
		jr.Logger.Infof("[Stop] STOPPING the running job - %v, job run ID - %v",
			jobRun.Job.GetCommonJobFields().ID, jobRun.ID)
		jobRun.Job.Stop()
		jr.Logger.Infof("[Stop] STOPPED the job - %v, job run ID - %v",
			jobRun.Job.GetCommonJobFields().ID, jobRun.ID)
	}
	if len(jr.RunningJobs) > 0 {
		jr.Logger.Errorf("Few jobs are running even after Stop() method invoke.")
		err = fmt.Errorf("few jobs are running after Stop() method invoke")
		return err
	}
	return nil
}

func (jr *JobRunner) runJob(jobRun *JobRun) {
	jr.Logger.Infof("In a go-routine, running the job - %v. Job run ID - %v.",
		jobRun.Job.GetCommonJobFields().ID, jobRun.ID)
	go func() {
		jr.RunningJobCountMu.Lock()
		jr.RunningJobCount++
		jr.RunningJobCountMu.Unlock()
		jobRun.Job.GetCommonJobFields().LastRun = time.Now()
		jr.RunningJobsMu.Lock()
		jr.RunningJobs = append(jr.RunningJobs, jobRun)
		jr.RunningJobsMu.Unlock()
		jr.Logger.Infof("[runJob] Execution of the Job - %v, JobRun - %v STARTED.",
			jobRun.Job.GetCommonJobFields().ID, jobRun.ID)
		jobRun.Job.Execute()
		jr.Logger.Infof("[runJob] Execution of the Job - %v, JobRun - %v COMPLETED.",
			jobRun.Job.GetCommonJobFields().ID, jobRun.ID)
		jr.RunningJobCountMu.Lock()
		jr.RunningJobCount--
		jr.RunningJobCountMu.Unlock()
		jr.removeRunEntry(jobRun.ID)
	}()
}

func (jr *JobRunner) removeRunEntry(runId string) {
	for i, j := range jr.RunningJobs {
		if j.ID == runId {
			jr.Logger.Infof("[removeRunEntry] Removing the JobRun entry for the run ID - %v", runId)
			// Locking here works only when there is only ONE go-routine removes entry.
			jr.RunningJobsMu.Lock()
			defer jr.RunningJobsMu.Unlock()
			jr.RunningJobs = append(jr.RunningJobs[:i], jr.RunningJobs[i+1:]...)
			jr.Logger.Infof("[removeRunEntry] Successfully removed the job with ID - %v", runId)
			return
		}
	}
	// If we don't find the desired job, but somebody is calling remove entry...
	jr.Logger.Infof("[removeRunEntry] Failed to get the job run with ID - %v", runId)
	jr.syncRunningCount()
}

func (jr *JobRunner) syncRunningCount() {
	jr.Logger.Infof("[syncRunningCount] Syncing the job runner's RunningJobCount.")
	runningJobsSize := len(jr.RunningJobs)
	if runningJobsSize != int(jr.RunningJobCount) {
		jr.Logger.Warnf("[syncRunningCount] Syncing RunningJobCount. Actual running jobs - %v, RunningJobCount - %v", runningJobsSize, jr.RunningJobCount)
		jr.RunningJobCountMu.Lock()
		jr.RunningJobCount = int16(runningJobsSize)
		jr.RunningJobCountMu.Unlock()
		return
	}
	jr.Logger.Infof("[syncRunningCount] Running count is in sync with number of running jobs.")
}

func (jr *JobRunner) monitorJobRunner() {
	monitorTicker := time.NewTicker(DEFAULT_MONITOR_TICKER)
	defer func() {
		monitorTicker.Stop()
		jr.Logger.Infof("Stopped the Job runner monitor.")
	}()
	for {
		select {
		case <-jr.stopChan:
			return
		case <-monitorTicker.C:
			jr.monitor()
		}
	}
}

func (jr *JobRunner) monitor() {
	jr.Logger.Infof("-------------JobRunner MONITOR START-------------")
	defer jr.Logger.Infof("-------------JobRunner MONITOR STOP-------------")
	jr.Logger.Infof("Number of running jobs - %v", jr.RunningJobCount)
	for i, j := range jr.RunningJobs {
		jr.Logger.Infof("| #%v : Jobrun ID - %v :: Job ID - %v :: Scheduled at - %v :: Ran at - %v |",
			i, j.ID, j.Job.GetCommonJobFields().ID, j.ScheduledAt, j.RanAt)
	}
}
