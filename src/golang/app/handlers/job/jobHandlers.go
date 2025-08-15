package job

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/shreyasksrao/jobmanager/app/common"
	"github.com/shreyasksrao/jobmanager/app/context"
	log "github.com/shreyasksrao/jobmanager/app/logger"
	"github.com/shreyasksrao/jobmanager/lib/core"
	"github.com/shreyasksrao/jobmanager/lib/jobs"
)

func getAllJobsFromFile(logger core.Logger, filePath string) (commandJobsMap map[string]jobs.CommandJob, err error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(fileContent), &commandJobsMap)
	if err != nil {
		errMsg := "Error parsing the JSON file : " + filePath + err.Error()
		logger.Errorf(errMsg)
		return
	}
	return
}

func saveJobs(logger core.Logger, filePath string, commandJobsMap map[string]jobs.CommandJob) (err error) {
	logger.Infof("Saving the Jobs to the JSON file - %v", filePath)
	indentedJson, err := json.MarshalIndent(commandJobsMap, "", "")
	if err != nil {
		logger.Errorf("Failed to encode command jobs map to a JSON. Error : %v", err.Error())
		return
	}
	err = os.WriteFile(filePath, indentedJson, 0644)
	if err != nil {
		logger.Errorf("Failed to save the jobs to the JSON file - %v", filePath)
	}
	return
}

func GetAllJobs(ctx *context.AppContext) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		logger := ctx.Logger
		logger.Infof("Inside GetAllJobs function")
		commandJobs, err := getAllJobsFromFile(logger, ctx.AppConfig.GetJobResourceFilePath())
		if os.IsNotExist(err) {
			logger.Infof("Job file not exist, returning empty map.")
			emptyMap := map[string]interface{}{}
			res, _ := json.Marshal(emptyMap)
			common.WriteOkResponse(w, res)
			return
		}
		if err != nil {
			common.WriteErrorResponse(w, err.Error(), "Internal Error", http.StatusInternalServerError)
			return
		}
		logger.Infof("Successfully fetched all the Jobs.")
		common.WriteOkResponse(w, commandJobs)
	}
}

func GetJobById(ctx *context.AppContext) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		logger := ctx.Logger
		jobId := params.ByName("id")
		logger.Infof("Inside GetJobById function for job with ID - %v", jobId)
		commandJobs, err := getAllJobsFromFile(logger, ctx.AppConfig.GetJobResourceFilePath())
		if os.IsNotExist(err) {
			errMsg := "Failed to get the job with ID - " + jobId + ". Job file doesn't exist."
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Bad Request", http.StatusBadRequest)
			return
		}
		if err != nil {
			common.WriteErrorResponse(w, err.Error(), "Internal Error", http.StatusInternalServerError)
			return
		}
		commandJob, exists := commandJobs[jobId]
		if !exists {
			errMsg := "Failed to get the job with ID " + jobId + ". Job doesn't exist."
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Bad Request", http.StatusBadRequest)
			return
		}
		common.WriteOkResponse(w, commandJob)
	}
}

func CreateJob(ctx *context.AppContext) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		logger := ctx.Logger
		logger.Infof("Inside CreateJob function")
		var job jobs.CommandJob
		payloadDecoder := json.NewDecoder(r.Body)
		if err := payloadDecoder.Decode(&job); err != nil {
			errMsg := "Invalid request. Failed to parse the JSON body. Error : " + err.Error()
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Bad Request", http.StatusBadRequest)
			return
		}
		jobId := uuid.New()
		logger.Infof("Generated the Job UUID - %v", jobId)
		job.CommonJobFields.ID = core.JobId(jobId.String())
		job.Logger = log.GetJobRunnerLogger()
		job.SaveFile = ctx.AppConfig.GetJobResourceFilePath()
		isValidRequest, err := jobs.ValidatePostPayload(logger, &job)
		if !isValidRequest {
			errMsg := "Validation failed for the request. Error : " + err.Error()
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, err.Error(), "Bad Request", http.StatusBadRequest)
			return
		}
		saved, err := job.Save()
		if !saved {
			errMsg := "Error occurred while saving the Job to the file. Error : " + err.Error()
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Internl Server Error", http.StatusInternalServerError)
			return
		}
		logger.Infof("Successfully svaed the Job.")
		logger.Infof("Adding the job to the cron manager.")
		jm := ctx.JobManager
		jm.AddJob(&job)
		logger.Infof("Successfully added the job to the cron manager.")
		common.WriteOkResponse(w, job)
	}
}

type updateCommandJob struct {
	Command   *string   `json:"Command"`   // Command to run
	Args      *[]string `json:"Args"`      // Arguments for the command
	CronExpr  *string   `json:"CronExpr"`  // Cron expression
	RunAsUser *string   `json:"RunAsUser"` // Username under which the command will be run
}

func UpdateJob(ctx *context.AppContext) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		logger := ctx.Logger
		jobId := params.ByName("id")
		logger.Infof("Inside UpdateJob function for the job - %v", jobId)
		var updateJobInput updateCommandJob
		payloadDecoder := json.NewDecoder(r.Body)
		if err := payloadDecoder.Decode(&updateJobInput); err != nil {
			errMsg := "Invalid request. Failed to parse the JSON body. Error : " + err.Error()
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Bad Request", http.StatusBadRequest)
			return
		}
		commandJobs, err := getAllJobsFromFile(logger, ctx.AppConfig.GetJobResourceFilePath())
		if err != nil {
			common.WriteErrorResponse(w, err.Error(), "Internal Error", http.StatusInternalServerError)
			return
		}
		commandJob, exists := commandJobs[jobId]
		if !exists {
			errMsg := "Failed to get the job with ID " + jobId + ". Job doesn't exist."
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Bad Request", http.StatusBadRequest)
			return
		}
		if updateJobInput.Command != nil && *updateJobInput.Command != "" {
			commandJob.Command = *updateJobInput.Command
		}
		if *updateJobInput.Args != nil {
			commandJob.Args = *updateJobInput.Args
		}
		if updateJobInput.CronExpr != nil && *updateJobInput.CronExpr != "" {
			commandJob.CronExpr = *updateJobInput.CronExpr
		}
		if updateJobInput.RunAsUser != nil && *updateJobInput.RunAsUser != "" {
			commandJob.RunAsUser = *updateJobInput.RunAsUser
		}
		commandJob.Logger = log.GetJobRunnerLogger()
		commandJob.SaveFile = ctx.AppConfig.GetJobResourceFilePath()
		commandJob.Save()
		jm := ctx.JobManager
		jm.RemoveJob(jobId)
		commandJobs, err = getAllJobsFromFile(logger, ctx.AppConfig.GetJobResourceFilePath())
		if err != nil {
			common.WriteErrorResponse(w, err.Error(), "Internal Error", http.StatusInternalServerError)
			return
		}
		updatedJob, exists := commandJobs[jobId]
		if !exists {
			err = fmt.Errorf("failed to get the job with ID - %v. Job doesn't exist", jobId)
			common.WriteErrorResponse(w, err.Error(), "Internal Error", http.StatusInternalServerError)
			return
		}
		updatedJob.Logger = log.GetJobRunnerLogger()
		updatedJob.SaveFile = ctx.AppConfig.GetJobResourceFilePath()
		jm.AddJob(&updatedJob)
		logger.Infof("Successfully updated the job in the cron manager.")
		common.WriteOkResponse(w, updatedJob)
	}
}

func DeleteJob(ctx *context.AppContext) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		logger := ctx.Logger
		jobId := params.ByName("id")
		jobFilePath := ctx.AppConfig.GetJobResourceFilePath()
		logger.Infof("Inside DeleteJob function for the job - %v", jobId)
		commandJobs, err := getAllJobsFromFile(logger, jobFilePath)
		if err != nil {
			common.WriteErrorResponse(w, err.Error(), "Internal Error", http.StatusInternalServerError)
			return
		}
		_, exists := commandJobs[jobId]
		if !exists {
			errMsg := "Failed to get the job with ID " + jobId + ". Job doesn't exist."
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Bad Request", http.StatusBadRequest)
			return
		}
		delete(commandJobs, jobId)
		err = saveJobs(logger, jobFilePath, commandJobs)
		if err != nil {
			errMsg := "Failed to save the Jobs to the JSON file '" + jobFilePath + "'." + "Error : " + err.Error()
			common.WriteErrorResponse(w, errMsg, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		statusMsg := "Successfully Deleted the job - " + jobId + " from the job file - " + jobFilePath + "."
		logger.Infof(statusMsg)
		common.WriteOkResponse(w, statusMsg)
	}
}
