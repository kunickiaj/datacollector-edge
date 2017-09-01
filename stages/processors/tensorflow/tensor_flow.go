package tensorflow

import (
	"errors"
	"fmt"
	"github.com/streamsets/datacollector-edge/api"
	"github.com/streamsets/datacollector-edge/container/common"
	"github.com/streamsets/datacollector-edge/stages/stagelibrary"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"log"
	"os"
)

const (
	LIBRARY           = "streamsets-datacollector-tensorflow-lib"
	STAGE_NAME        = "com_streamsets_pipeline_stage_processor_tensorflow_TensorFlowDProcessor"
	MODEL_PATH_CONFIG = "modelPath"
	N_TIME_STEPS      = 200
	N_CLASSES         = 6
)

type TensorFlowProcessor struct {
	*common.BaseStage
	model           *tf.SavedModel
	windowRecords   [][]float32
	numberOfRecords int
}

func init() {
	stagelibrary.SetCreator(LIBRARY, STAGE_NAME, func() api.Stage {
		return &TensorFlowProcessor{BaseStage: &common.BaseStage{}}
	})
}

func (tfp *TensorFlowProcessor) Init(stageContext api.StageContext) error {
	tfp.BaseStage.Init(stageContext)
	stageConfig := tfp.GetStageConfig()
	for _, config := range stageConfig.Configuration {
		switch config.Name {
		case MODEL_PATH_CONFIG:
			modelPath := stageContext.GetResolvedValue(config.Value.(string)).(string)
			if _, err := os.Stat(modelPath); os.IsNotExist(err) {
				return errors.New(fmt.Sprintf("Model Path %s does not exist", modelPath))
			}
			var err error

			if tfp.model, err = tf.LoadSavedModel(modelPath, []string{"activity_tracker"}, nil); err != nil {
				return errors.New("Error when loading model from path : " + modelPath + "\n" + err.Error())
			}
		default:
			return errors.New("Unsupported Config : " + config.Name)
		}
		tfp.windowRecords = make([][]float32, N_TIME_STEPS)
		tfp.numberOfRecords = 0
	}

	return nil
}

func (tfp *TensorFlowProcessor) Process(batch api.Batch, batchMaker api.BatchMaker) error {
	log.Println("[DEBUG] Tensor Flow - Process method")

	X := tfp.model.Graph.Operation("input").Output(0)
	Y := tfp.model.Graph.Operation("y_").Output(0)

	for _, record := range batch.GetRecords() {
		tfp.windowRecords[tfp.numberOfRecords] = convertRecordToXYZFloatSlice(record)
		tfp.numberOfRecords = tfp.numberOfRecords + 1
		if tfp.numberOfRecords < N_TIME_STEPS {
			setClassificationForRecord(record, -1)
		} else {
			final_slice := [][][]float32{tfp.windowRecords}
			tfp.numberOfRecords = 0
			if tensor, err := tf.NewTensor(final_slice); err != nil {
				log.Printf("[ERROR] Error creating tensor for record %s", err.Error())
				return err
			} else {
				res, runErr := tfp.model.Session.Run(
					map[tf.Output]*tf.Tensor{X: tensor},
					[]tf.Output{Y},
					nil,
				)
				if runErr != nil {
					log.Printf("[ERROR] Error running tensor flow record %s", runErr.Error())
					return runErr
				}
				result := res[0].Value().([][]float32)[0]
				log.Println("[DEBUG] Predictions", result)

				max := float32(-1)
				class := -1
				if len(result) != N_CLASSES {
					log.Printf(
						"[ERROR] Predicted more than required number of classes,"+
							" expected :%d, actual : %d\n",
						N_CLASSES,
						len(result),
					)
					return errors.New("Something wrong with prediction")
				} else {
					for idx, r := range result {
						val := r
						if val > max {
							max = val
							class = idx
						}
					}
					setClassificationForRecord(record, class)
					otherRecords := tfp.windowRecords[1:]
					tfp.windowRecords = make([][]float32, N_TIME_STEPS)
					for d, otherRecord := range otherRecords {
						tfp.windowRecords[d] = otherRecord
						tfp.numberOfRecords = d + 1
					}
				}
			}
		}
		batchMaker.AddRecord(record)
	}
	return nil
}

func convertRecordToXYZFloatSlice(record api.Record) []float32 {
	rootField := record.Get().Value.(map[string]api.Field)
	acceleroMeter := rootField["CMAccelerometer"].Value.(map[string]api.Field)
	return []float32{float32(acceleroMeter["x"].Value.(float64)), float32(acceleroMeter["y"].Value.(float64)), float32(acceleroMeter["z"].Value.(float64))}
}

func setClassificationForRecord(record api.Record, classification int) {
	log.Printf("[DEBUG] Predicted %d for record\n", classification)
	rootField := record.Get().Value.(map[string]api.Field)
	if val, err := api.CreateField(classification); err == nil {
		rootField["classification"] = *val
	} else {
		log.Println("[ERROR] Error happened when setting root field", err.Error())
	}
}

func (tfp *TensorFlowProcessor) Destroy() error {
	log.Println("[DEBUG] Tensor Flow Runner Destroyed")
	return tfp.model.Session.Close()
}
