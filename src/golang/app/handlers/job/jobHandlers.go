package job

import (
	"encoding/json"
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

func GetAllJobs(ctx *context.AppContext) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		logger := ctx.Logger
		logger.Infof("Inside GetAllJobs function")
		fileContent, err := os.ReadFile(ctx.AppConfig.GetJobResourceFilePath())
		if os.IsNotExist(err) {
			logger.Infof("Job file not exist, returning empty map.")
			emptyMap := map[string]interface{}{}
			res, _ := json.Marshal(emptyMap)
			common.WriteOkResponse(w, res)
			return
		}
		if err != nil {
			errMsg := "Error reading the JSON file : " + ctx.AppConfig.GetJobResourceFilePath() + err.Error()
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Internal Error", http.StatusInternalServerError)
			return
		}
		var result map[string]core.Job
		err = json.Unmarshal([]byte(fileContent), &result)
		if err != nil {
			errMsg := "Error parsing the JSON file : " + ctx.AppConfig.GetJobResourceFilePath() + err.Error()
			logger.Errorf(errMsg)
			common.WriteErrorResponse(w, errMsg, "Internal Error", http.StatusInternalServerError)
			return
		}
		logger.Infof("Successfully fetched all the Jobs.")
		common.WriteOkResponse(w, result)
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
