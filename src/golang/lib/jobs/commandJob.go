package jobs

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/shreyasksrao/jobmanager/lib/core"
	"github.com/shreyasksrao/jobmanager/lib/utils"
)

type CommandJob struct {
	CommonJobFields core.CommonJobFields
	Command         string      `json:"Command"`   // Command to run
	Args            []string    `json:"Args"`      // Arguments for the command
	CronExpr        string      `json:"CronExpr"`  // Cron expression
	RunAsUser       string      `json:"RunAsUser"` // Username under which the command will be run
	Logger          core.Logger `json:"-"`
	cmd             *exec.Cmd   `json:"-"`
	SaveFile        string      `json:"-"` // Full path of the JSON file where the jobs can be saved.
}

func (job *CommandJob) GetCommonJobFields() (commonJobFields *core.CommonJobFields) {
	commonJobFields = &job.CommonJobFields
	return
}

// Save saves the Job object to the resource file (resources/jobs.json)
func (job *CommandJob) Save() (saved bool, err error) {
	var errMsg string
	var jobsMap map[string]CommandJob
	job.Logger.Infof("Saving the job with ID - %v to the resource file.", job.CommonJobFields.ID)
	exists := utils.CheckFileExistance(job.Logger, job.SaveFile)
	if exists {
		job.Logger.Infof("Jobs resource file exists. Loading the existing jobs.")
		var fileData []byte
		// Open the file in append mode, create it if it doesn't exist, with write-only permissions
		fileData, err = os.ReadFile(job.SaveFile)
		if err != nil {
			errMsg = "Failed to save the job - " + string(job.CommonJobFields.ID) + "to the file. Error reading file - " + job.SaveFile + ". Error - " + err.Error()
			job.Logger.Errorf(errMsg)
			err = fmt.Errorf(errMsg)
			return false, err
		}
		// Unmarshal into map
		err = json.Unmarshal(fileData, &jobsMap)
		if err != nil {
			errMsg = "Failed to save the job - " + string(job.CommonJobFields.ID) + " to the file. Error unmarshaling file - " + job.SaveFile + ". Error - " + err.Error()
			job.Logger.Errorf(errMsg)
			err = fmt.Errorf(errMsg)
			return false, err
		}
	} else {
		job.Logger.Infof("Jobs resources file doesn't exist. Creating an empty jobMap.")
		jobsMap = make(map[string]CommandJob)
	}

	job.Logger.Debugf("Before update Job - %v", jobsMap[string(job.CommonJobFields.ID)])
	// Create or update entry
	jobsMap[string(job.CommonJobFields.ID)] = CommandJob{
		CommonJobFields: core.CommonJobFields{
			ID:      job.CommonJobFields.ID,
			LastRun: job.CommonJobFields.LastRun,
			NextRun: job.CommonJobFields.NextRun,
		},
		Command:   job.Command,
		Args:      job.Args,
		CronExpr:  job.CronExpr,
		RunAsUser: job.RunAsUser,
	}
	// Marshal back to JSON
	jsonData, err := json.MarshalIndent(jobsMap, "", "  ")
	if err != nil {
		errMsg = "Error marshaling the JSON. Error : " + err.Error()
		job.Logger.Errorf(errMsg)
		err = fmt.Errorf(errMsg)
		return false, err
	}
	// Write to file (overwrite with updated map)
	err = os.WriteFile(job.SaveFile, jsonData, 0644)
	if err != nil {
		errMsg = "Error writing to file. Error : " + err.Error()
		job.Logger.Errorf(errMsg)
		err = fmt.Errorf(errMsg)
		return false, err
	}
	job.Logger.Debugf("After update Job - %v", jobsMap[string(job.CommonJobFields.ID)])
	job.Logger.Infof("Successfully saved the Job with ID - %v to the resource file.", string(job.CommonJobFields.ID))
	return true, nil
}

// Execute runs the specified command. If the "RunAsUser" field is specified,
// then this func tries to run the command as that user. Else the command will
// be run as the default user (root)
func (job *CommandJob) Execute() (err error) {
	job.Logger.Infof("---------------------------------EXECUTION START------------------------------------")
	defer job.Logger.Infof("---------------------------------EXECUTION STOP------------------------------------")
	if job.RunAsUser != "" {
		job.Logger.Infof("Fetching the user details for the username - %v", job.RunAsUser)
		runUser, err := user.Lookup(job.RunAsUser)
		if err != nil {
			job.Logger.Errorf("Failed to fetch the user details for the username - %v. Error - %v", job.RunAsUser, err)
			return err
		}
		uid, err := strconv.Atoi(runUser.Uid)
		if err != nil {
			job.Logger.Errorf("Invalid UID. Error - %v.")
			return err
		}
		gid, err := strconv.Atoi(runUser.Gid)
		if err != nil {
			job.Logger.Errorf("Invalid GID. Error - %v.")
			return err
		}
		job.Logger.Infof("Executing the command - %v with arguments - %v", job.Command, job.Args)
		job.cmd = exec.Command(job.Command, job.Args...)
		// Set UID and GID of the target user
		job.cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(uid), // replace with target user's UID
				Gid: uint32(gid), // replace with target user's GID
			},
		}
	} else {
		job.Logger.Infof("RunAsUser field is empty, going with the default user.")
		job.Logger.Infof("Executing the command - %v with arguments - %v", job.Command, job.Args)
		job.cmd = exec.Command(job.Command, job.Args...)
	}
	if err = job.cmd.Start(); err != nil {
		job.Logger.Errorf("Error executing job %s: %v", string(job.CommonJobFields.ID), err)
		return err
	}
	job.Logger.Infof("Process ID - %v", job.cmd.Process.Pid)
	if err := job.cmd.Wait(); err != nil {
		job.Logger.Errorf("Process exited with error: %v", err)
	} else {
		job.Logger.Infof("Process exited cleanly")
	}
	job.Logger.Infof("Job %s executed successfully.", string(job.CommonJobFields.ID))
	return nil
}

func (job *CommandJob) Stop() {
	job.Logger.Infof("Stopping the Job - %v", job.CommonJobFields.ID)
	if err := job.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		job.Logger.Errorf("failed to send SIGTERM: %v", err)
	} else {
		job.Logger.Infof("SIGTERM sent for the process with ID - %v.", job.cmd.Process.Pid)
	}
}

func (job *CommandJob) GetNextScheduleTime(now time.Time) (nextRun time.Time, err error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(job.CronExpr)
	return schedule.Next(now), err
}

func ValidatePostPayload(log core.Logger, job *CommandJob) (isValid bool, err error) {
	if job.Command == "" {
		log.Errorf("invalid request. Command is not specified in the payload")
		err = fmt.Errorf("invalid request. Command is not specified in the payload")
		return false, err
	}
	if job.CronExpr == "" {
		log.Errorf("invalid request. CronExpr is not specified in the payload")
		err = fmt.Errorf("invalid request. CronExpr is not specified in the payload")
		return false, err
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err = parser.Parse(job.CronExpr)
	if err != nil {
		log.Errorf("invalid request. Failed to parse the CronExpr - %v. Error - %v", job.CronExpr, err.Error())
		return false, err
	}
	log.Infof("Successfully validated the POST payload")
	return true, nil
}

func LoadCommandJobsFromJsonFile(log core.Logger, jobFilePath string, jobLogger core.Logger) (jobs []*CommandJob, err error) {
	log.Infof("Loading the existing Jobs from the file - %v.", jobFilePath)
	fileContent, err := os.ReadFile(jobFilePath)
	if err != nil {
		log.Errorf("Error reading the JSON file : %v. Error : %v", jobFilePath, err.Error())
		return
	}
	var jobsMap map[string]CommandJob
	err = json.Unmarshal([]byte(fileContent), &jobsMap)
	if err != nil {
		log.Errorf("Error parsing the JSON file : %v. Error : %v", jobFilePath, err.Error())
		return
	}
	// Add the fields which are not persisted.
	for _, job := range jobsMap {
		job.Logger = jobLogger
		job.SaveFile = jobFilePath
		jobs = append(jobs, &job)
	}
	log.Infof("Successfully loaded the jobs from the file - %v", jobFilePath)
	return
}
