package coap

import (
	"bytes"
	"github.com/dustin/go-coap"
	"github.com/streamsets/datacollector-edge/api"
	"github.com/streamsets/datacollector-edge/container/common"
	"github.com/streamsets/datacollector-edge/container/recordio"
	"github.com/streamsets/datacollector-edge/container/recordio/jsonrecord"
	"github.com/streamsets/datacollector-edge/stages/stagelibrary"
	"log"
	"net/url"
)

const (
	LIBRARY            = "streamsets-datacollector-basic-lib"
	STAGE_NAME         = "com_streamsets_pipeline_stage_destination_coap_CoapClientDTarget"
	CONF_RESOURCE_URL  = "conf.resourceUrl"
	CONF_COAP_METHOD   = "conf.coapMethod"
	CONF_RESOURCE_TYPE = "conf.requestType"
	CONFIRMABLE        = "CONFIRMABLE"
	NONCONFIRMABLE     = "NONCONFIRMABLE"
	GET                = "GET"
	POST               = "POST"
	PUT                = "PUT"
	DELETE             = "DELETE"
)

type CoapClientDestination struct {
	*common.BaseStage
	Conf                ClientTargetConfig `ConfigDefBean:"conf"`
	recordWriterFactory recordio.RecordWriterFactory
}

type ClientTargetConfig struct {
	ResourceUrl string `ConfigDef:"type=STRING,required=true"`
	CoapMethod  string `ConfigDef:"type=STRING,required=true"`
	RequestType string `ConfigDef:"type=STRING,required=true"`
}

var mid uint16

func init() {
	stagelibrary.SetCreator(LIBRARY, STAGE_NAME, func() api.Stage {
		return &CoapClientDestination{BaseStage: &common.BaseStage{}}
	})
}

func (c *CoapClientDestination) Init(stageContext api.StageContext) error {
	if err := c.BaseStage.Init(stageContext); err != nil {
		return err
	}
	log.Println("[DEBUG] CoapClientDestination Init method")
	// TODO: Create RecordWriter based on configuration
	c.recordWriterFactory = &jsonrecord.JsonWriterFactoryImpl{}
	mid = 0
	return nil
}

func (c *CoapClientDestination) Write(batch api.Batch) error {
	log.Println("[DEBUG] CoapClientDestination Write method")
	for _, record := range batch.GetRecords() {
		err := c.sendRecordToSDC(record)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *CoapClientDestination) sendRecordToSDC(record api.Record) error {
	payloadBuffer := bytes.NewBuffer([]byte{})
	recordWriter, err := c.recordWriterFactory.CreateWriter(c.GetStageContext(), payloadBuffer)
	if err != nil {
		return err
	}
	err = recordWriter.WriteRecord(record)
	if err != nil {
		return err
	}
	recordWriter.Flush()
	recordWriter.Close()

	parsedURL, err := url.Parse(c.Conf.ResourceUrl)
	if err != nil {
		return err
	}

	req := coap.Message{
		Type:      getCoapType(c.Conf.RequestType),
		Code:      getCoapMethod(c.Conf.CoapMethod),
		MessageID: mid,
		Payload:   payloadBuffer.Bytes(),
	}
	req.SetPathString(parsedURL.Path)

	coapClient, err := coap.Dial("udp", parsedURL.Host)
	if err != nil {
		log.Printf("[ERROR] Error dialing: %v", err)
		return err
	}

	_, err = coapClient.Send(req)
	if err != nil {
		log.Printf("[ERROR] Error sending request: %v", err)
		return err
	}

	mid++
	return nil
}

func getCoapType(requestType string) coap.COAPType {
	switch requestType {
	case CONFIRMABLE:
		return coap.Confirmable
	case NONCONFIRMABLE:
		return coap.NonConfirmable
	}
	return coap.NonConfirmable
}

func getCoapMethod(coapMethod string) coap.COAPCode {
	switch coapMethod {
	case GET:
		return coap.GET
	case POST:
		return coap.POST
	case PUT:
		return coap.PUT
	case DELETE:
		return coap.DELETE
	}
	return coap.POST
}
