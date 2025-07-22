package rest

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/shreyasksrao/jobmanager/app/context"
	"github.com/shreyasksrao/jobmanager/app/handlers/job"
)

const (
	API_PREFIX = "/api/v1"
)

func CreateRestServer(ctx *context.AppContext, restServerPort int) (server *http.Server) {
	router := registerRoutes(ctx)
	logger := ctx.Logger
	address := fmt.Sprintf(":%v", restServerPort)
	logger.Infof("Creating webserver on - %v", address)
	server = &http.Server{
		Addr:    address,
		Handler: router,
	}
	return
}

func StartServer(ctx *context.AppContext, server *http.Server) (err error) {
	logger := ctx.Logger
	logger.Infof("Starting the webserevr on - %v", server.Addr)
	err = server.ListenAndServe()
	if err == nil {
		logger.Infof("Successfully started the webserver...")
	}
	return
}

func registerRoutes(ctx *context.AppContext) (router *httprouter.Router) {
	router = httprouter.New()
	router.GET(API_PREFIX+"/jobs", job.GetAllJobs(ctx))
	router.POST(API_PREFIX+"/jobs", job.CreateJob(ctx))
	return
}
