package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	DEFAULT_BASE_DIRECTORY    = "/opt/jobManager"
	CONFIG_FILE_NAME          = "jobManager.config.json"
	APP_LOG_FILE_NAME         = "restfulCron.log"
	JOB_MANAGER_LOG_FILE_NAME = "jobManager.log"
	JOB_RUNNER_LOG_FILE_NAME  = "jobRunner.log"
	LOG_DIR_NAME              = "logs"
	RESOURCE_DIR_NAME         = "resources"
	DEFAULT_REST_SERVER_PORT  = 7000
	JOBS_FILE                 = "jobs.json"
	DEFAULT_MAX_RUNNING_JOBS  = 100
)

var AppConfig *Config

/*
Config struct holds the user provided configuration.
New config fields has to be added in this struct to take into effect.
*/
type Config struct {
	WorkingDirectory string `json:"workingDirectory"`
	LogLevel         string `json:"logLevel"`
	MaxRunningJobs   int16  `json:"maxRunningJobs"`
}

func ReadConfig(configFile string) (Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return Config{}, err
	}
	err = json.Unmarshal(data, &AppConfig)
	if err != nil {
		return Config{}, err
	}
	return *AppConfig, nil
}

func GetConfig() Config {
	if *AppConfig == (Config{}) || AppConfig == nil {
		panic("Application is not initialized properly !")
	}
	return *AppConfig
}

func (config *Config) GetBaseDirectory() (baseDir string) {
	if config == nil {
		panic("Application is not initialized properly !")
	}
	if config.WorkingDirectory == "" {
		return DEFAULT_BASE_DIRECTORY
	}
	return config.WorkingDirectory
}

func (config *Config) GetLogDirectory() (logDir string) {
	baseDir := config.GetBaseDirectory()
	logDir = filepath.Join(baseDir, LOG_DIR_NAME)
	return
}

func (config *Config) GetApplicationLogFilePath() (logFilePath string) {
	logDir := config.GetLogDirectory()
	logFilePath = filepath.Join(logDir, APP_LOG_FILE_NAME)
	return
}

func (config *Config) GetResourceDirectory() (resourceDir string) {
	baseDir := config.GetBaseDirectory()
	resourceDir = filepath.Join(baseDir, RESOURCE_DIR_NAME)
	return
}

func (config *Config) GetJobResourceFilePath() (jobResourcePath string) {
	resourceDir := config.GetResourceDirectory()
	jobResourcePath = filepath.Join(resourceDir, JOBS_FILE)
	return
}
