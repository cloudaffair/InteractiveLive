package model

import (
	"time"
)

type Opinion struct {
	Text             string      `json:"text"`
	StartTime		 time.Time   `json:"start_time"`
	EndTime			 time.Time   `json:"end_time"`
	Values           interface{} `json:"values"`
	ServiceUrl       string      `json:"service_url"`
	SegmentLength    int       `json:"segment_length,omitempty"`
	Partner			 string      `json:"partner,omitempty"`
	Deployment       string      `json:"deployment,omitempty"`
	EventId			 string      `json:"event_id,omitempty"`
	PollType		 string	     `json:"poll_type"`
}

type OpinionResponse struct {
	OpinionId	string `json:"opinion_id"`
}

type OpinionBurnData struct {
	Text             string      `json:"text"`
	Values           interface{} `json:"values"`
	ServiceUrl       string      `json:"url"`
	OpinionId		 string		 `json:"id"`
	OpinionStart     bool		 `json:"start"`
	PollType		 string	     `json:"poll_type"`
}

type ElementalTimedMetaTemplate struct {
	TimedMetadata struct {
		ID3 struct {
			Encoding string `xml:"encoding,attr"`
			Text     string `xml:""`
		} `xml:"id3"`
	} `xml:"timed_metadata"`
}

type ElementalTimedMetaJsonTemplate struct {
	TimedMetadata struct {
		ID3 struct {
			Encoding string `json:"-encoding"`
			Text     string `json:"#text"`
		} `json:"id3"`
	} `json:"timed_metadata"`
}

type OpinionFeedback struct {
	Values           interface{} `json:"client_data"`
	OpinionId		 string		 `json:"opinion_id"`
}
