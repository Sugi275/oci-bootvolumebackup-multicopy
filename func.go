package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Sugi275/oci-objectstorage-multiregioner/loglib"
	"github.com/fnproject/fdk-go"
	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/common/auth"
	"github.com/oracle/oci-go-sdk/core"
)

const (
	envBucketName        = "OCI_BUCKETNAME"
	envSourceRegion      = "OCI_SOURCE_REGION"
	envDestinationRegion = "OCI_DESTINATION_REGION"
	actionTypeCreate     = "com.oraclecloud.blockvolumes.createbootvolumebackup.end"
	actionTypeDelete     = "com.oraclecloud.blockvolumes.deletebootvolumebackup.end"
)

// EventsInput EventsInput
type EventsInput struct {
	EventType          string    `json:"eventType"`
	CloudEventsVersion string    `json:"cloudEventsVersion"`
	EventTypeVersion   string    `json:"eventTypeVersion"`
	Source             string    `json:"source"`
	EventTime          time.Time `json:"eventTime"`
	ContentType        string    `json:"contentType"`
	Data               struct {
		CompartmentID     string `json:"compartmentId"`
		CompartmentName   string `json:"compartmentName"`
		ResourceName      string `json:"resourceName"`
		ResourceID        string `json:"resourceId"`
		AdditionalDetails struct {
			SourceType string `json:"sourceType"`
			VolumeID   string `json:"volumeId"`
		} `json:"additionalDetails"`
		DefinedTags struct {
			Organization struct {
				Fy20Level2 string `json:"fy20_level_2"`
				Fy20Level1 string `json:"fy20_level_1"`
			} `json:"organization"`
			Basic struct {
				Owner     string `json:"owner"`
				CreatedBy string `json:"created_by"`
			} `json:"basic"`
		} `json:"definedTags"`
	} `json:"data"`
	EventID    string `json:"eventID"`
	Extensions struct {
		CompartmentID string `json:"compartmentId"`
	} `json:"extensions"`
}

// Action Action
type Action struct {
	BootVolumeBackupOCID string
	BootVolumeBackupName string
	SourceRegion         string
	DestinationRegion    string
	ActionType           string
	ctx                  context.Context
}

func main() {
	fdk.Handle(fdk.HandlerFunc(fnMain))

	// ------- local development ---------
	// reader := os.Stdin
	// writer := os.Stdout
	// fnMain(context.TODO(), reader, writer)
}

func fnMain(ctx context.Context, in io.Reader, out io.Writer) {
	loglib.InitSugar()
	defer loglib.Sugar.Sync()

	// Events から受け取るパラメータ
	input := &EventsInput{}
	json.NewDecoder(in).Decode(input)

	action, err := generateAction(ctx, *input)

	if err != nil {
		loglib.Sugar.Error(err)
		return
	}

	err = runAction(action)

	if err != nil {
		loglib.Sugar.Error(err)
		return
	}

	out.Write([]byte("Done!"))
}

func generateAction(ctx context.Context, input EventsInput) (Action, error) {
	action := Action{}
	var ok bool

	// ActionType
	action.ActionType = input.EventType

	// BootVolumeBackupOCID
	action.BootVolumeBackupOCID = input.Data.ResourceID

	// BootVolumeBackupName
	action.BootVolumeBackupName = input.Data.ResourceName

	// SourceRegion
	var sourceRegion string
	if sourceRegion, ok = os.LookupEnv(envSourceRegion); !ok {
		err := fmt.Errorf("can not read envSourceRegion from environment variable %s", envSourceRegion)
		return action, err
	}
	action.SourceRegion = sourceRegion

	// DestinationRegions
	var destinationRegionString string
	if destinationRegionString, ok = os.LookupEnv(envDestinationRegion); !ok {
		err := fmt.Errorf("can not read envDestinationRegion from environment variable %s", envDestinationRegion)
		return action, err
	}
	action.DestinationRegion = destinationRegionString

	// Context
	action.ctx = ctx

	return action, nil
}

func runAction(action Action) error {
	var err error

	fmt.Println(action)

	switch action.ActionType {
	case actionTypeCreate:
		err = copyBlockVolumeBackup(action)
	case actionTypeDelete:
		err = deleteBlockVolumeBackupInAnotherRegion(action)
	default:
		err = fmt.Errorf("do nothing. ActionType : %s", action.ActionType)
	}

	if err != nil {
		return err
	}

	return nil
}

func copyBlockVolumeBackup(action Action) error {

	provider, err := auth.ResourcePrincipalConfigurationProvider()
	if err != nil {
		loglib.Sugar.Error(err)
		return err
	}
	// provider := common.DefaultConfigProvider()
	client, err := core.NewBlockstorageClientWithConfigurationProvider(provider)
	client.SetRegion(string(action.SourceRegion))

	if err != nil {
		loglib.Sugar.Error(err)
		return err
	}

	copyBootVolumeBackupDetails := core.CopyBootVolumeBackupDetails{
		DestinationRegion: common.String(action.DestinationRegion),
		DisplayName:       common.String(action.BootVolumeBackupName),
	}

	request := core.CopyBootVolumeBackupRequest{
		CopyBootVolumeBackupDetails: copyBootVolumeBackupDetails,
		BootVolumeBackupId:          common.String(action.BootVolumeBackupOCID),
	}

	_, err = client.CopyBootVolumeBackup(action.ctx, request)

	if err != nil {
		loglib.Sugar.Error(err)
		return err
	}

	return nil
}

func deleteBlockVolumeBackupInAnotherRegion(action Action) error {
	return nil // 何か処理を入れてもいいかも。今のところは未実装にしておく
}
