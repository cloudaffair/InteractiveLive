package medialive

import (
	"context"
	"common"
	"time"
	"model"
	"fmt"
	"encoding/json"
	"encoding/base64"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/medialive"
	"strings"
)

func HandleOpinionRequest(c context.Context, opinion model.Opinion, config *common.Config) (model.OpinionResponse, error) {
	mediaLiveRegion := opinion.Deployment

	var access_key string
	var secret_key string

	opinionResponse := model.OpinionResponse{}
	if time.Now().UTC().After(opinion.EndTime) {
		return opinionResponse, common.NewError("Poll End time already elapsed. Nothing to schedule")
	}

	deps := config.Dependencies["medialive"].([]interface{})
	if deps != nil && len(deps) > 0 {
		for _, d := range deps {
			d := d.(map[string]interface{})
			if d["region"] == mediaLiveRegion {
				access_key = d["access_key"].(string)
				secret_key = d["secret_key"].(string)
			}
		}
	}
	fmt.Println( "Setting opinion request for ", mediaLiveRegion, opinion.EventId)

	port := config.Dependencies["port"].(string)
	if mediaLiveRegion == "" || opinion.EventId == "" || access_key == "" || secret_key == "" {
		return opinionResponse, common.NewError("Media Live deployment info missing in configuration")
	}

	env := config.Dependencies["environment"].(string)

	OpinionToBurn, err := common.GetOpinionToBurn(c, opinion, port, env)

	if err != nil {
		fmt.Println(err)
		return opinionResponse, common.NewError("Error while computing the opinion to burn")
	}

	fmt.Println("Opinion to Burn ", OpinionToBurn)
	go scheduleID3Inserts(c, OpinionToBurn, opinion.SegmentLength, mediaLiveRegion, opinion.EventId, access_key, secret_key, opinion.StartTime, opinion.EndTime)
	opinionResponse.OpinionId = OpinionToBurn.OpinionId

	return opinionResponse, nil
}

func DeleteOpinionRequest(c context.Context, opinionId string, config *common.Config, event model.LiveEvent) error {

	var access_key string
	var secret_key string

	region := event.Deployment
	eventId := event.EventId

	deps := config.Dependencies["medialive"].([]interface{})
	if deps != nil && len(deps) > 0 {
		for _, d := range deps {
			d := d.(map[string]interface{})
			if d["region"] == region {
				access_key = d["access_key"].(string)
				secret_key = d["secret_key"].(string)
			}
		}
	}
	fmt.Println( "Deleting opinion request for ", region, eventId, opinionId)

	if region == "" || eventId == "" || access_key == "" || secret_key == "" {
		return common.NewError("Media Live deployment info missing in configuration")
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(access_key, secret_key, ""),
	})

	if err != nil {
		fmt.Errorf("%s", "error creating AWS session", err)
		return err
	}
	// Create a MediaLive client from just a session.
	client := medialive.New(sess)
	input := medialive.DescribeScheduleInput{}
	input.ChannelId = &eventId
	response, error := client.DescribeSchedule(&input)
	if error != nil {
		return error
	}
	fmt.Println("Describe schedule response", response)

	var actionNames []*string
	for _, action := range response.ScheduleActions {
		if strings.HasPrefix(*action.ActionName, opinionId){
			actionNames = append(actionNames, action.ActionName)
		}
	}

	scheduleActionDeleteRequest := medialive.BatchScheduleActionDeleteRequest{}
	scheduleActionDeleteRequest.ActionNames = actionNames

	delete_input := medialive.BatchUpdateScheduleInput{}
	delete_input.SetChannelId(eventId)
	delete_input.SetDeletes(&scheduleActionDeleteRequest)

	output, error := client.BatchUpdateSchedule(&delete_input)
	if error != nil {
		fmt.Errorf("error while deleting scheduled timed metadatas %s", error)
		return error
	}
	fmt.Println("Schedule deleted successfully", output.GoString())
	return nil
}

func scheduleID3Inserts(c context.Context, OpinionToBurn model.OpinionBurnData, burnInterval int, elementalLive string, eventId string, key string, username string, startTime time.Time, endTime time.Time) {
	now := time.Now().UTC()
	if now.After(startTime) {
		fmt.Println("%s", "Start time already expired, nothing to schedule for start time; schedule for future")
		startTime = time.Now().UTC()
	} else {
		// Schedule one for the start time
		writeID3Tag (c, startTime, OpinionToBurn, elementalLive, eventId, key, username)
	}

	var lastSetTime = startTime
	for lastSetTime.Before(endTime) {
		var nextEpochTime = (lastSetTime.Unix())+(int64(burnInterval))
		lastSetTime = time.Unix(nextEpochTime, 0).UTC()
		if endTime.Sub(time.Now().UTC()).Seconds() <= float64(burnInterval) {
			fmt.Printf("Probably the last ID3 to burn for ", OpinionToBurn.OpinionId, ";Setting poll as false")
			OpinionToBurn.OpinionStart = false
		} else {
			OpinionToBurn.OpinionStart = true
		}

		writeID3Tag (c, lastSetTime, OpinionToBurn, elementalLive, eventId, key, username)
	}
}

func writeID3Tag(c context.Context, t time.Time, OpinionToBurn model.OpinionBurnData, mediaLive string, eventId string, key string, secret string) {
	OpinionToBurn.OpinionStart = true

	id3Value, err := json.Marshal(OpinionToBurn)
	if err != nil {
		return
	}

	fmt.Printf("%s", "Stringified ID3 data ", string(id3Value))
	id3Tag := common.GetId3Tag(string(id3Value))
	base64Id3 := base64.StdEncoding.EncodeToString(id3Tag)

	fmt.Println( "Base64 Encoded ID3 content ", base64Id3)

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(mediaLive),
		Credentials: credentials.NewStaticCredentials(key, secret, ""),
	})

	if err != nil {
		fmt.Errorf( "%s", "error creating AWS session", err)
		return
	}
	time := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d.000Z",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second())

	// Create a MediaLive client from just a session.
	client := medialive.New(sess)

	hlsTimedMetaScheduleActionSettings := medialive.HlsTimedMetadataScheduleActionSettings{}
	hlsTimedMetaScheduleActionSettings.SetId3(base64Id3)

	scheduleActionSettings := medialive.ScheduleActionSettings{}
	scheduleActionSettings.SetHlsTimedMetadataSettings(&hlsTimedMetaScheduleActionSettings)

	fixedModeScheduleActionStartSettings := medialive.FixedModeScheduleActionStartSettings{}

	fixedModeScheduleActionStartSettings.SetTime(time)

	scheduleActionStartSettings := medialive.ScheduleActionStartSettings{}
	scheduleActionStartSettings.SetFixedModeScheduleActionStartSettings(&fixedModeScheduleActionStartSettings)

	scheduleAction := medialive.ScheduleAction{}
	scheduleAction.SetScheduleActionSettings(&scheduleActionSettings)
	scheduleAction.SetActionName(OpinionToBurn.OpinionId + "-" + time)
	scheduleAction.SetScheduleActionStartSettings(&scheduleActionStartSettings)

	scheduleActions := []*medialive.ScheduleAction{}
	scheduleActions = append(scheduleActions, &scheduleAction)

	scheduleActionCreateRequest := medialive.BatchScheduleActionCreateRequest{}
	scheduleActionCreateRequest.SetScheduleActions(scheduleActions)

	input := medialive.BatchUpdateScheduleInput{}
	input.SetChannelId(eventId)
	input.SetCreates(&scheduleActionCreateRequest)
	fmt.Println("Time meta set as ", input.GoString())

	output, error := client.BatchUpdateSchedule(&input)
	if error != nil {
		fmt.Errorf("error while updating timed metadata", error)
	}
	fmt.Println("Time meta set successfully", output.GoString())
}