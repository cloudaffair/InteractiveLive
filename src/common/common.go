package common

import (
	"model"
	"github.com/rs/xid"
	"net/http"
	"context"
	"bytes"
	"encoding/binary"
)

const(
	ID3_FIELD = "TXXX"
	ID3_DESCRIPTION = "poll"
	ENCODING_UTF8 = 3
)

func GetOpinionToBurn(c context.Context, opinion model.Opinion, port string, env string) (model.OpinionBurnData ,error) {
	var OpinionToBurn = model.OpinionBurnData{}
	opinionId := xid.New().String()
	OpinionToBurn.OpinionId = opinionId
	OpinionToBurn.Text = opinion.Text
	OpinionToBurn.ServiceUrl = opinion.ServiceUrl
	OpinionToBurn.Values = opinion.Values
	OpinionToBurn.PollType = opinion.PollType

	domain_name := ""
	if (env != "dev") {
		aws_url := "http://169.254.169.254/latest/meta-data/public-hostname"
		hd := make(http.Header)
		content, _, err := HttpSubmitData(c, "GET", aws_url, &hd, nil)
		if (err == nil) {
			domain_name = string(content)
		}
	} else {
		domain_name = "localhost"
	}
	OpinionToBurn.ServiceUrl = "http://" + domain_name +":" + port + "/v1/opinion/" + OpinionToBurn.OpinionId +  "/feedback"
	return OpinionToBurn,nil
}

func GetId3Tag(value string) []byte {
	var id3 bytes.Buffer

	id3.WriteString("ID3")
	id3.WriteByte(0x04) // version
	id3.WriteByte(0x00)

	id3.WriteByte(0x00) // flags

	id3.Write(getID3Field(ID3_FIELD, ID3_DESCRIPTION, value))

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
