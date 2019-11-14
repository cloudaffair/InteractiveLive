package elementallive

import (
	"context"
	"model"
	"common"
	"encoding/base64"
	"time"
	"github.com/jasonlvhit/gocron"
	"fmt"
	"bytes"
	"encoding/binary"
	"net/http"
	"crypto/md5"
	"io"
	"github.com/clbanning/mxj"
	"encoding/json"
)

const(
 ID3_FIELD = "TXXX"
 ID3_DESCRIPTION = "poll"
 ENCODING_UTF8 = 3
)

func HandleOpinionRequest(c context.Context, opinion model.Opinion, config *common.Config) (model.OpinionResponse, error) {
	elementalDeployment := opinion.Deployment

	var elementalLive string
	var key string
	var username string

	opinionResponse := model.OpinionResponse{}
	if time.Now().UTC().After(opinion.EndTime) {
		return opinionResponse, common.NewError("Poll End time already elapsed. Nothing to schedule")
	}

	deps := config.Dependencies["elementallive"].([]interface{})
	if deps != nil && len(deps) > 0 {
		for _, d := range deps {
			d := d.(map[string]interface{})
			if (d["client_id"] == elementalDeployment) {
				elementalLive = d["elemental_live_url"].(string)
				key = d["key"].(string)
				username = d["username"].(string)
			}
		}
	}
	fmt.Println( "Setting opinion request for ", elementalDeployment, elementalLive, opinion.EventId, key, username)

	port := config.Dependencies["port"].(string)
	if elementalLive == "" || opinion.EventId == "" || key == "" || username == "" {
		return opinionResponse, common.NewError("Elemental deployment info missing in configuration")
	}

	env := config.Dependencies["environment"].(string)

	OpinionToBurn, err := common.GetOpinionToBurn(c, opinion, port, env)

	if err != nil {
		return opinionResponse, common.NewError("Error while computing the opinion to burn")
	}

	go scheduleID3Inserts(c, OpinionToBurn, opinion.SegmentLength, elementalLive, opinion.EventId, key, username, opinion.StartTime, opinion.EndTime)
	opinionResponse.OpinionId = OpinionToBurn.OpinionId
	return opinionResponse, nil
}

func writeID3Tag(c context.Context, s *gocron.Scheduler, OpinionToBurn model.OpinionBurnData, elementalLive string, eventId string, key string, username string, endTime time.Time, burnInterval uint64){

	OpinionToBurn.OpinionStart = true

	if time.Now().UTC().After(endTime) {
		fmt.Printf("Timer expired for", OpinionToBurn.OpinionId, ", Returning")
		s.Clear()
		return
	} else if endTime.Sub(time.Now().UTC()).Seconds() <= float64(burnInterval) {
		fmt.Printf("Probably the last ID3 to burn for ", OpinionToBurn.OpinionId, ";Setting poll as false")
		OpinionToBurn.OpinionStart = false
	}

	id3Value, err := json.Marshal(OpinionToBurn)
	if err != nil {
		return
	}

	fmt.Printf("%s", "Stringified ID3 data ", string(id3Value))
	id3Tag := common.GetId3Tag(string(id3Value))
	base64Id3 := base64.StdEncoding.EncodeToString(id3Tag)

	fmt.Println( "Base64 Encoded ID3 content ", base64Id3)

	path := "/live_events/" + eventId + "/timed_metadata"

	reqUrl := elementalLive + path
	hd := make(http.Header)

	expires_string := getExpiresTime()
	// string to be used in initial MD5 hash
	data_string := fmt.Sprint(path, username, key, expires_string)

	fmt.Printf( "data_string=", data_string)
	// create initial MD5 hash
	md5_hash := md5.New()
	io.WriteString(md5_hash, data_string)
	hashed_data := md5_hash.Sum(nil)

	// convert MD5 hash (type []unit8) to string
	hashed_data_string := fmt.Sprintf("%x", hashed_data)

	// concat web_dav_key with first MD5 hash to be used in final MD5 hash
	hashed_data_string2 := fmt.Sprint(key, hashed_data_string)

	// create final MD5 hash
	final_md5_hash := md5.New()
	io.WriteString(final_md5_hash, hashed_data_string2)
	final_hashed_data := final_md5_hash.Sum(nil)

	// convert MD5 hash (type []unit8) to string
	final_hashed_data_string := fmt.Sprintf("%x", final_hashed_data)

	hd.Add("Content-type", "application/xml")
	hd.Add("Accept", "application/xml")
	hd.Add("X-Auth-User", username)
	hd.Add("X-Auth-Expires", expires_string)
	hd.Add("X-Auth-Key", final_hashed_data_string)

	timedata := model.ElementalTimedMetaJsonTemplate{}

	timedata.TimedMetadata.ID3.Encoding = "base64"
	timedata.TimedMetadata.ID3.Text = base64Id3

	output, _ := json.Marshal(timedata)

	mapRaw, _ := mxj.NewMapJson(output)
	newBody, _ := mapRaw.Xml()

	fmt.Println("Request body :", string(newBody))

	content, resp, err := common.HttpSubmitData(c, "POST", reqUrl, &hd, newBody)
	fmt.Printf("%s", "Http post response ", string(content), resp, err)
}

var getExpiresTime = func() string {
	return fmt.Sprintf("%d", time.Now().Unix()+3000)
}

func GetId3Tag(field string, desc string, value string) []byte {
	var id3 bytes.Buffer

	id3.WriteString("ID3")
	id3.WriteByte(0x04) // version
	id3.WriteByte(0x00)

	id3.WriteByte(0x00) // flags

	id3.Write(getID3Field(field, desc, value))

	return id3.Bytes()
}

func getID3Field(field string, desc string, value string) []byte {
	switch field {
	case "TXXX":
		return getTXXXField(desc, value)
	}

	var fBuf bytes.Buffer
	binary.Write(&fBuf, binary.BigEndian, uint32(0)) // size
	return fBuf.Bytes()
}

func getTXXXField(desc string, value string) []byte {

	var txxx bytes.Buffer

	txxx.WriteString("TXXX")

	var size uint32
	size = uint32(1 + len(desc) + 1 + len(value) + 1)
	binary.Write(&txxx, binary.BigEndian, size)      // size
	binary.Write(&txxx, binary.BigEndian, uint16(0)) // flags

	txxx.WriteByte(ENCODING_UTF8)

	txxx.WriteString(desc)
	txxx.WriteByte(0)

	txxx.WriteString(value)
	txxx.WriteByte(0)

	var frame bytes.Buffer
	binary.Write(&frame, binary.BigEndian, uint32(txxx.Len()))
	frame.Write(txxx.Bytes())
	return frame.Bytes()
}

func scheduleID3Inserts(c context.Context, OpinionToBurn model.OpinionBurnData, burnInterval uint64, elementalLive string, eventId string, key string, username string, startTime time.Time, endTime time.Time) {
	jt := NewJobTicker(startTime)
	for {
		<-jt.t.C
		fmt.Println(time.Now(), "- ticked")
		jt.t.Stop()
		// Schedule further jobs for the defined burn interval periodically ...
		s := gocron.NewScheduler()
		s.Every(burnInterval).Seconds().Do(writeID3Tag, c, s, OpinionToBurn, elementalLive, eventId, key, username, endTime, burnInterval)
		<- s.Start()
	}
}

type jobTicker struct {
	t *time.Timer
}

func getNextTickDuration(jobStartTime time.Time) time.Duration {
	now := time.Now().UTC()
	if now.After(jobStartTime) {
		fmt.Println("%s", "Start time already expired, nothing to schedule for start time; returning")
		return 0
	}
	return jobStartTime.Sub(now)
}

func NewJobTicker(jobStartTime time.Time) jobTicker {
	return jobTicker{time.NewTimer(getNextTickDuration(jobStartTime))}
}

