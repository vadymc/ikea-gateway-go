package ikea

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	gocoap "github.com/dustin/go-coap"
	"github.com/eriklupander/tradfri-go/model"
	log "github.com/sirupsen/logrus"
	"github.com/vadymc/telegram-client-go/v2"

	"github.com/vadymc/ikea-gateway-go/m/ikea/coap"
)

type ITradfriClient interface {
	GetGroupIds() ([]int, error)
	GetGroup(id string) (model.Group, error)
	GetGroupDevices(group model.Group) ([]model.Device, error)
}

type TradfriClient struct {
	dtlsClient     *coap.DtlsClient
	telegramClient *telegram.TelegramClient
}

var (
	errorCount     int
	errorThreshold = 5
)

// Creates an instance of TradfriClient.
// Based on https://github.com/eriklupander/tradfri-go/blob/master/tradfri/tradfri-client.go
func NewTradfriClient(gatewayAddress, clientID, psk string, telegramClient *telegram.TelegramClient) *TradfriClient {
	client := &TradfriClient{}
	client.dtlsClient = coap.NewDtlsClient(gatewayAddress, clientID, psk)
	client.telegramClient = telegramClient
	return client
}

func (tc *TradfriClient) ListGroups() ([]model.Group, error) {
	groups := make([]model.Group, 0)

	groupIds, err := tc.GetGroupIds()
	if err != nil {
		return groups, err
	}
	for _, id := range groupIds {
		group, _ := tc.GetGroup(strconv.Itoa(id))
		groups = append(groups, group)
	}
	return groups, nil
}

func (tc *TradfriClient) GetGroupIds() ([]int, error) {
	groupIds := make([]int, 0)

	resp, err := tc.Call(tc.dtlsClient.BuildGETMessage("/15004"))
	if err != nil {
		if errorCount >= errorThreshold {
			log.WithError(err).WithField("Error threshold", errorThreshold).Error("Failed to call Trådfri, stopping application")
			body := fmt.Sprintf("Failed to call Trådfri, stopping application. Retried %v times. Error [%v]", errorCount, err.Error())
			tc.telegramClient.SendMessage("Ikea GW", body)
			os.Exit(1)
		}
		errorCount++
		log.WithError(err).Error("Unable to call Trådfri")
		return groupIds, err
	} else if errorCount != 0 {
		errorCount = 0
	}

	err = json.Unmarshal(resp.Payload, &groupIds)
	return groupIds, err
}

func (tc *TradfriClient) GetGroup(id string) (model.Group, error) {
	resp, err := tc.Call(tc.dtlsClient.BuildGETMessage("/15004/" + id))
	group := &model.Group{}
	if err != nil {
		return *group, err
	}
	err = json.Unmarshal(resp.Payload, &group)
	if err != nil {
		return *group, err
	}
	return *group, nil
}

func (tc *TradfriClient) GetGroupDevices(group model.Group) ([]model.Device, error) {
	deviceIds := group.Content.DeviceList.DeviceIds
	devices := make([]model.Device, len(deviceIds))
	for _, id := range deviceIds {
		d, err := tc.GetDevice(strconv.Itoa(id))
		if err != nil {
			return devices, err
		}
		devices = append(devices, d)
	}
	return devices, nil
}

func (tc *TradfriClient) GetDevice(id string) (model.Device, error) {
	device := &model.Device{}

	resp, err := tc.Call(tc.dtlsClient.BuildGETMessage("/15001/" + id))
	if err != nil {
		return *device, err
	}
	err = json.Unmarshal(resp.Payload, &device)
	if err != nil {
		return *device, err
	}
	return *device, nil
}

func (tc *TradfriClient) AuthExchange(clientId string) (model.TokenExchange, error) {

	req := tc.dtlsClient.BuildPOSTMessage("/15011/9063", fmt.Sprintf(`{"9090":"%s"}`, clientId))

	// Send CoAP message for token exchange
	resp, _ := tc.Call(req)

	// Handle response and return
	token := model.TokenExchange{}
	err := json.Unmarshal(resp.Payload, &token)
	if err != nil {
		panic(err)
	}
	return token, nil
}

// A proxy to the underlying DtlsClient Call.
func (tc *TradfriClient) Call(msg gocoap.Message) (gocoap.Message, error) {
	return tc.dtlsClient.Call(msg)
}
