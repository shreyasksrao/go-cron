package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shreyasksrao/jobmanager/app/config"
	appContext "github.com/shreyasksrao/jobmanager/app/context"
	"github.com/shreyasksrao/jobmanager/app/logger"
	"github.com/shreyasksrao/jobmanager/app/rest"
	"github.com/shreyasksrao/jobmanager/lib/core"
	"github.com/shreyasksrao/jobmanager/lib/jobs"
)

func main() {
	// Command line flag parsing.
	configFilePath := flag.String("configFilePath", "", "Configuration file path (absolute path)")
	restServerPort := flag.Int("port", config.DEFAULT_REST_SERVER_PORT, "Job manager REST server port")
	flag.Parse()

	if *configFilePath == "" {
		fmt.Print("Config file path is not specified...")
		return
	}
	appConfig, err := config.ReadConfig(*configFilePath)
	if err != nil {
		fmt.Printf("error while reading the config - %v", err)
		return
	}
	// Initialize the logger
	logger.Initialize(&appConfig)
	appLogger := logger.GetAppLogger()
	defer logger.CleanUpLoggers()
	// Print the CLI args
	appLogger.Infof("---------- Command line flags ----------")
	appLogger.Infof("Config file path : %s", configFilePath)
	appLogger.Infof("REST server port : %d", restServerPort)
	appLogger.Infof("----------------------------------------")

	// Channel to listen for interrupt or terminate signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	// This channel listens to any server error for graceful shutdown
	serverErr := make(chan error, 1)

	// Server initializing
	appLogger.Infof("Starting the application - JOB MANAGER")

	// Create the new instance of CronManager and start the Cron scheduler.
	jmConfig := core.JobManagerConfig{
		Location:            time.Local,
		JobManagerLogger:    logger.GetJobManagerLogger(),
		JobRunnerLogger:     logger.GetJobRunnerLogger(),
		MaxRunningJobsCount: 100,
	}
	manager := core.NewJobManager(&jmConfig)
	manager.Start()
	defer manager.Stop()

	// Load the existing Jobs from the jobs.json file.
	appLogger.Infof("Getting the existing jobs from the resource file - %v", appConfig.GetJobResourceFilePath())
	commandJobs, err := jobs.LoadCommandJobsFromJsonFile(
		appLogger,
		appConfig.GetJobResourceFilePath(),
		logger.GetJobRunnerLogger(),
	)
	for _, commandJob := range commandJobs {
		appLogger.Infof("Adding the Job - %v to the Job manager.", commandJob.CommonJobFields.ID)
		manager.AddJob(commandJob)
		time.Sleep(2 * time.Second)
	}

	ctx := appContext.NewContext(appLogger, &appConfig)
	ctx.SetCronManager(manager)

	server := rest.CreateRestServer(ctx, *restServerPort)
	go func() {
		appLogger.Infof("Starting the REST server in a go-routine.")
		err = rest.StartServer(ctx, server)
		if err != nil && err != http.ErrServerClosed {
			appLogger.Errorf("Server error: %v", err)
			serverErr <- err // Put the server error into the channel to stop the application
		}
	}()

	// Wait for stop signal
	select {
	case <-stop:
		appLogger.Infof("Shutting down the server as SIGTERM signal recieved...")
	case err := <-serverErr:
		appLogger.Errorf("Shutting down the server due to the error - %v", err.Error())
	}

	// Context with timeout for graceful shutdown
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer func() {
		appLogger.Infof("Calling cancel() func on the application context.")
		cancel()
		appLogger.Infof("Successfully executed the cancel function.")
	}()
	// Perform server shutdown
	if err := server.Shutdown(timeoutCtx); err != nil {
		appLogger.Errorf("Server forced to shutdown : %v", err)
	}
}
