package core

import (
	"time"
)

type JobId string

// All the implementations of Job interface should contain these fields.
// These fields are used in the JobManager to set the next run and last run.
type CommonJobFields struct {
	ID      JobId     `json:"ID"`      // Unique job identifier
	NextRun time.Time `json:"NextRun"` // NextRun at which this job will run
	LastRun time.Time `json:"LastRun"` // Command execution start time in Epoch millis
}

type Job interface {
	// Execute() function will be called inside a separate go routine in the Job runner's Run().
	// Implementation should handle the cleanup of the resources otherwise
	// resource leak may happen.
	Execute() (err error)
	// Stop() will be called on all the running Jobs when the JobManager recieves Stop signal.
	Stop()
	Save() (saved bool, err error)
	// GetNextScheduleTime() should return the next run of a job wrt "now"
	GetNextScheduleTime(now time.Time) (nextRun time.Time, err error)
	// Implementation of Job interface must contains "NextRun", "LastRun" and "ID" fields.
	// These fields are used to computing the Schedule time. GetCommonJobFields()
	// should return the pointer to CommonJobFields
	GetCommonJobFields() (commonFields *CommonJobFields)
}
