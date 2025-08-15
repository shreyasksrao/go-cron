package core

import (
	"sort"
	"sync"
	"time"
)

type JobManager struct {
	Jobs       []Job
	stopChan   chan struct{}
	addChan    chan Job
	removeChan chan JobId
	running    bool
	runningMu  sync.Mutex
	Logger     Logger
	jobLock    sync.Mutex
	Location   *time.Location
	jobRunner  *JobRunner
	jobRunChan chan *JobRun
}

type JobManagerConfig struct {
	Location         *time.Location
	JobManagerLogger Logger
	JobRunnerLogger  Logger
	// Maximum number of Jobs the runner can handle in parallel.
	// Each Job run will spawn a new go-routine and call the Job's Execute() function.
	MaxRunningJobsCount int16
}

func NewJobManager(config *JobManagerConfig) (jobManager *JobManager) {
	config.JobManagerLogger.Infof("Creating the new instance of JobManager")
	location := config.Location
	if location == nil {
		location = time.Local
	}
	jobRunChan := make(chan *JobRun, DEFAULT_JOB_RUN_CHAN_BUFFER)
	stopChan := make(chan struct{})
	jobManager = &JobManager{
		Jobs:       nil,
		Logger:     config.JobManagerLogger,
		stopChan:   stopChan,
		addChan:    make(chan Job),
		removeChan: make(chan JobId),
		running:    false,
		Location:   location,
		jobRunner:  NewJobRunner(config.JobRunnerLogger, config.MaxRunningJobsCount, jobRunChan),
		jobRunChan: jobRunChan,
	}
	config.JobManagerLogger.Infof("Successfully created the JobManager instance.")
	return
}

func (manager *JobManager) Start() (err error) {
	manager.Logger.Infof("============================ STARTING Job manager ============================")
	defer manager.Logger.Infof("============================ STARTED Job manager ============================")
	manager.runningMu.Lock()
	defer manager.runningMu.Unlock()
	manager.running = true
	go manager.runScheduler()
	go manager.jobRunner.Start()
	return
}

func (manager *JobManager) Stop() (err error) {
	manager.Logger.Infof("============================ STOPPING Job manager ============================")
	defer manager.Logger.Infof("============================ STOPPED Job manager ============================")
	if manager.running {
		manager.runningMu.Lock()
		defer manager.runningMu.Unlock()
		manager.running = false
		manager.stopChan <- struct{}{}
		manager.jobRunner.Stop()
	}
	return
}

func (manager *JobManager) AddJob(j Job) (jobId JobId) {
	manager.Logger.Infof("Adding the job to the job manager.")
	jobId = j.GetCommonJobFields().ID
	manager.jobLock.Lock()
	defer manager.jobLock.Unlock()
	if !manager.running {
		manager.Logger.Infof("Job manager is not running, simply adding the job to the entry list.")
		manager.Jobs = append(manager.Jobs, j)
	} else {
		manager.addChan <- j
	}
	return
}

func (manager *JobManager) RemoveJob(jobId string) {
	manager.Logger.Infof("Adding the job to the job manager.")
	manager.jobLock.Lock()
	defer manager.jobLock.Unlock()
	if !manager.running {
		manager.Logger.Infof("Job manager is not running, simply removing the job from the entry list.")
		manager.removeEntry(JobId(jobId))
	} else {
		manager.removeChan <- JobId(jobId)
	}
}

func (manager *JobManager) runScheduler() {
	manager.Logger.Infof("Running the scheduler.")
	now := time.Now()
	manager.Logger.Infof("Populatinng the next job run ffor all the jobs.")
	for _, job := range manager.Jobs {
		nextRun, _ := job.GetNextScheduleTime(now)
		job.GetCommonJobFields().NextRun = nextRun
		job.Save()
	}
	for {
		// Sort the Jobs based on the next schedule time.
		sortByNextScheduleTime := func(a, b int) bool {
			aNext := manager.Jobs[a].GetCommonJobFields().NextRun
			bNext := manager.Jobs[b].GetCommonJobFields().NextRun
			if aNext.IsZero() {
				return false
			}
			if bNext.IsZero() {
				return true
			}
			return aNext.Before(bNext)
		}
		sort.Slice(manager.Jobs, sortByNextScheduleTime)
		manager.Logger.Debugf("Joobs after sort by next schedule time - %v", manager.Jobs)

		var timer *time.Timer
		if len(manager.Jobs) == 0 || manager.Jobs[0].GetCommonJobFields().NextRun.IsZero() {
			// If there are no jobs yet, just sleep - it still handles new jobs and stop requests.
			timer = time.NewTimer(100000 * time.Hour)
		} else {
			timer = time.NewTimer(manager.Jobs[0].GetCommonJobFields().NextRun.Sub(now))
		}
		// Listen for any requests on the channels...
		for {
			select {
			case now = <-timer.C:
				now = now.In(manager.Location)
				manager.Logger.Infof("Timer expired at - %v.", now)
				// Run every entry whose next time was less than now
				for _, job := range manager.Jobs {
					manager.Logger.Infof("Next run - %v", job.GetCommonJobFields().NextRun)
					if job.GetCommonJobFields().NextRun.After(now) || job.GetCommonJobFields().NextRun.IsZero() {
						break
					}
					// send the job to the JobRunner as JobRun object.
					jobRun := manager.jobRunner.CreateJobRun(job)
					manager.jobRunChan <- jobRun
					job.GetCommonJobFields().NextRun, _ = job.GetNextScheduleTime(now)
				}

			case newEntry := <-manager.addChan:
				timer.Stop()
				now = time.Now().In(manager.Location)
				newEntry.GetCommonJobFields().NextRun, _ = newEntry.GetNextScheduleTime(now)
				manager.Jobs = append(manager.Jobs, newEntry)
				manager.Logger.Infof("Added the job with ID - %v. Current time - %v, Next run at - %v",
					newEntry.GetCommonJobFields().ID, now, newEntry.GetCommonJobFields().NextRun)

			case <-manager.stopChan:
				timer.Stop()
				manager.Logger.Infof("Recieved signal on Stop channel. Stopped the scheduler...")
				return

			case id := <-manager.removeChan:
				timer.Stop()
				now = now.In(manager.Location)
				manager.removeEntry(id)
				manager.Logger.Infof("Removed the job with ID - %v", id)
			}
			break
		}
	}
}

func (manager *JobManager) removeEntry(id JobId) {
	for i, job := range manager.Jobs {
		if job.GetCommonJobFields().ID == id {
			manager.Logger.Infof("Found the element to remove at the index - %v", i)
			manager.jobLock.Lock()
			defer manager.jobLock.Unlock()
			manager.Jobs = append(manager.Jobs[:i], manager.Jobs[i+1:]...)
			manager.Logger.Infof("Successfully removed the job with ID - %v", id)
			break
		}
	}
}
