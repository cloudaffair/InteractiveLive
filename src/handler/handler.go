package handler

import (
	"common"
	"context"
	"net/http"
	"time"
	"github.com/gin-gonic/gin"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"model"
	"elementallive"
	"es"
	"github.com/elastic/go-elasticsearch/esapi"
	"strings"
	"strconv"
	"bytes"
	"medialive"
)

const HealthZStatusOK                = "OK"
const HealthZStatusFailed            = "FAILED"

type Handler struct {
}

type HealthZResponse struct {
	Error        string              `json:"error,omitempty"`
	ServerStatus string              `json:"serverStatus"`
	Status       string              `json:"status"`
	TimeStamp    time.Time           `json:"time"`
}

func NewHandler() *Handler {
	h := new(Handler)
	return h
}

func (h *Handler) SetupContext() gin.HandlerFunc {
	return func(cgin *gin.Context) {
		common.CreateLoadGoogleContext(cgin)
	}
}

func (h *Handler) HandleHealthRequest(ctx context.Context) error {
	c, _ := common.GinContext(ctx)

	healthzResp := HealthZResponse{}

	healthzResp.Status = "Health Ok"
	healthzResp.TimeStamp = time.Now()

	c.JSON(http.StatusOK, healthzResp)
	return nil
}

func (h *Handler) ErrorCatch(f func(context.Context) error) gin.HandlerFunc {
	return func(cgin *gin.Context) {
		ctx, _ := common.GoogleContext(cgin)
		httpErr := f(ctx)
		if httpErr != nil {
			cgin.Error(httpErr)
			cgin.String(http.StatusInternalServerError, httpErr.Error()+"\n")
		}
	}
}

func (h *Handler) HandleOpinionRequest(ctx context.Context) error {
	c, _ := common.GinContext(ctx)

	config,_ := common.GetConfig()
	dataIn, _ := ioutil.ReadAll(c.Request.Body)

	var opinion model.Opinion
	if err := json.Unmarshal(dataIn, &opinion); err != nil {
		fmt.Errorf( "Bad Input: %s :%s", err.Error(), string(dataIn))
		return common.HttpErrorfCode(400, 8001, "%s", err.Error())
	}

	fmt.Println( "Opinion request ", opinion)

	switch opinion.Partner {
	case "elemental_live":
		response, err := elementallive.HandleOpinionRequest(ctx, opinion, config)
		if err != nil {
			return common.HttpErrorfCode(http.StatusInternalServerError, 8001, "%s", err)
		}
		c.JSON(http.StatusCreated, response)
	case "media_live":
		response, err := medialive.HandleOpinionRequest(ctx, opinion, config)
		if err != nil {
			return common.HttpErrorfCode(http.StatusInternalServerError, 8001, "%s", err)
		}
		c.JSON(http.StatusCreated, response)
	default:
		fmt.Errorf("Unknown partner  ", opinion.Partner)
		return common.HttpErrorfCode(http.StatusBadRequest, 8001, "%s", "Unknown partner", opinion.Partner)
	}

	return nil
}

func (h *Handler) HandleOpinionFeedbackRequest(ctx context.Context) error {
	c, _ := common.GinContext(ctx)

	config,_ := common.GetConfig()
	elasticServer := config.Dependencies["elastic_server"]
	dataIn, _ := ioutil.ReadAll(c.Request.Body)

	var feedback model.OpinionFeedback
	if err := json.Unmarshal(dataIn, &feedback); err != nil {
		fmt.Errorf( "Bad Input: %s :%s", err.Error(), string(dataIn))
		return common.HttpErrorfCode(400, 8001, "%s", err.Error())
	}

	opinionId := c.Params.ByName("id")

	fmt.Printf("Opinion feedback for Id  ", opinionId,
		" elastic server: ", elasticServer, "feedback data :", feedback)
	elasticServers := elasticServer.([]interface{})

	elasticServersStr := make([]string, len(elasticServers))
	for i, v := range elasticServers {
		elasticServersStr[i] = v.(string)
	}

	go writeToEs(feedback, opinionId, elasticServersStr)
	return nil
}

func writeToEs(feedback model.OpinionFeedback, opinionId string, elasticServersStr []string){
	es, err := es.GetClient(elasticServersStr)

	if err != nil {
		return
	}

	// Set up the request object directly.
	feedback.OpinionId = opinionId

	body, _ := json.Marshal(feedback)
	req := esapi.IndexRequest{
		Index:      "ila",
		DocumentID: "ila_" + opinionId + "_"+ strconv.Itoa(int(time.Now().UnixNano())),
		Body:       strings.NewReader(string(body)),
		Refresh: "true",
		Pretty:     true,
		Human:      true,
		ErrorTrace: true,
	}

	// Perform the request with the client.
	res, err := req.Do(context.Background(), es)
	if err != nil {
		fmt.Errorf("Error getting response: %s", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		s := buf.String() // Does a complete copy of the bytes in the buffer.

		fmt.Printf("[%s] Error indexing document [%s]", res.Status(), s)
	} else {
		// Deserialize the response into a map.
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			fmt.Printf("Error parsing the response body: %s", err)
		} else {
			// Print the response status and indexed document version.
			fmt.Printf("[%s] %s; version=%d", res.Status(), r["result"], int(r["_version"].(float64)))
		}
	}
}