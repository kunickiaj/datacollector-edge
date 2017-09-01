package httpserver

import (
	"bytes"
	"github.com/streamsets/datacollector-edge/api"
	"github.com/streamsets/datacollector-edge/container/common"
	"github.com/streamsets/datacollector-edge/container/recordio/jsonrecord"
	"github.com/streamsets/datacollector-edge/stages/stagelibrary"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

const (
	LIBRARY    = "streamsets-datacollector-basic-lib"
	STAGE_NAME = "com_streamsets_pipeline_stage_origin_httpserver_HttpServerDPushSource"
)

type HttpServerOrigin struct {
	*common.BaseStage
	port         int64
	appId        string
	httpServer   *http.Server
	incomingData chan interface{}
}

func init() {
	stagelibrary.SetCreator(LIBRARY, STAGE_NAME, func() api.Stage {
		return &HttpServerOrigin{BaseStage: &common.BaseStage{}}
	})
}

func (h *HttpServerOrigin) Init(stageContext api.StageContext) error {
	if err := h.BaseStage.Init(stageContext); err != nil {
		return err
	}
	stageConfig := h.GetStageConfig()
	for _, config := range stageConfig.Configuration {
		if config.Name == "httpConfigs.port" {
			h.port = stageContext.GetResolvedValue(config.Value).(int64)
		}
		if config.Name == "httpConfigs.appId" {
			h.appId = stageContext.GetResolvedValue(config.Value).(string)
		}
	}

	h.httpServer = h.startHttpServer()
	h.incomingData = make(chan interface{})
	return nil
}

func (h *HttpServerOrigin) Destroy() error {
	if err := h.httpServer.Shutdown(nil); err != nil {
		return err
	}
	log.Println("[DEBUG] HTTP Server - server shutdown successfully")
	return nil
}

func (h *HttpServerOrigin) Produce(
	lastSourceOffset string,
	maxBatchSize int,
	batchMaker api.BatchMaker,
) (string, error) {
	log.Println("[DEBUG] HTTP Server - Produce method")
	recordReaderFactoryImpl := &jsonrecord.JsonReaderFactoryImpl{}

	for recordCnt := 0; recordCnt < 1000; recordCnt++ {
		value := <-h.incomingData
		log.Printf("[DEBUG] Incoming Data: %s, Record Count : %s \n", value, strconv.FormatInt(int64(recordCnt), 10))
		bufferWriter := bytes.NewBuffer([]byte{})
		bufferWriter.WriteString(value.(string))

		if recordReader, err := recordReaderFactoryImpl.CreateReader(h.GetStageContext(), bufferWriter); err == nil {
			record, er := recordReader.ReadRecord()
			if er == nil {
				batchMaker.AddRecord(record)
			} else {
				return "", er
			}
		} else {
			log.Println("[Error] cannot deserialize json", value)
			return "", err
		}
	}
	return "", nil
}

func (h *HttpServerOrigin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("[DEBUG] HTTP Server error reading request body : ", err)
		h.GetStageContext().ReportError(err)
	} else {
		h.incomingData <- string(body)
	}
}

func (h *HttpServerOrigin) startHttpServer() *http.Server {
	srv := &http.Server{
		Addr:    ":" + strconv.FormatInt(h.port, 10),
		Handler: h,
	}

	go func() {
		log.Println("[DEBUG] HTTP Server - Running on URI : http://localhost:", h.port)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("[ERROR] Httpserver: ListenAndServe() error: %s", err)
			h.GetStageContext().ReportError(err)
		}
	}()

	return srv
}
