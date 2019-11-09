package ila

import (
	"common"
	"github.com/gin-gonic/gin"
	"handler"
	"time"
	"os"
	"net/http"
	"fmt"
)

type ila struct {
	config              *common.Config
}

func NewIla(config *common.Config) (*ila, error) {
	n := new(ila)
	n.config = config
	return n, nil
}

func (ila *ila) Run(sv *gin.Engine) {
	fmt.Println("Interactive Live App Run")
	sv.Use(gin.Recovery())

	hl := handler.NewHandler()

	simple := sv.Group("/")
	simple.Use(hl.SetupContext())

	simple.GET("/healthz", (hl.ErrorCatch(hl.HandleHealthRequest)))
	simple.POST("/v1/opinion", (hl.ErrorCatch(hl.HandleOpinionRequest)))

	simple.POST("/v1/opinion/:id/feedback", (hl.ErrorCatch(hl.HandleOpinionFeedbackRequest)))

	httpPort := ila.config.Dependencies["port"].(string)
	httpListenPort := "0.0.0.0:" + httpPort
	fmt.Println("http listening on port=", httpPort)

	err := listenAndServe(httpListenPort, sv)
	if err != nil {
		fmt.Errorf("Failed to start HTTP server on port %s. Error=%s", httpPort, err.Error())
		time.Sleep(time.Second * 2) // allow time for logger threads to write errors
		os.Exit(2)
	}
	fmt.Errorf("Exiting ila.Run()...this should NEVER happen")
}

var listenAndServe = func(listenPort string, sv *gin.Engine) error {
	return http.ListenAndServe(listenPort, sv)
}

