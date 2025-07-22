package context

import (
	"github.com/shreyasksrao/jobmanager/app/config"
	"github.com/shreyasksrao/jobmanager/lib/core"
)

type AppContext struct {
	Logger     core.Logger
	AppConfig  *config.Config
	JobManager *core.JobManager
}

func NewContext(logger core.Logger, appConfig *config.Config) (ctx *AppContext) {
	logger.Infof("Creating new AppContext.")
	ctx = &AppContext{
		Logger:    logger,
		AppConfig: appConfig,
	}
	return
}

func (appCtx *AppContext) SetCronManager(jm *core.JobManager) {
	appCtx.Logger.Infof("Setting the Cron manager object in the application context instance.")
	appCtx.JobManager = jm
}
