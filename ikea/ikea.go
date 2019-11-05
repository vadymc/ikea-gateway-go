package ikea

import (
	"encoding/json"
	"fmt"
	"strconv"

	gocoap "github.com/dustin/go-coap"
	"github.com/eriklupander/tradfri-go/model"
	"github.com/vadymc/ikea-gateway-go/m/ikea/coap"
	log "github.com/sirupsen/logrus"
)

type ITradfriClient interface {
	GetGroupIds() ([]int, error)
	GetGroup(id string) (model.Group, error)
	GetGroupDevices(group model.Group) ([]model.Device, error)
}

type TradfriClient struct {
	dtlsclient *coap.DtlsClient
}

// Creates an instance of TradfriClient.
// Based on https://github.com/eriklupander/tradfri-go/blob/master/tradfri/tradfri-client.go
func NewTradfriClient(gatewayAddress, clientID, psk string) *TradfriClient {
	client := &TradfriClient{}
	client.dtlsclient = coap.NewDtlsClient(gatewayAddress, clientID, psk)
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

	resp, err := tc.Call(tc.dtlsclient.BuildGETMessage("/15004"))
	if err != nil {
		log.WithError(err).Error("Unable to call Trådfri")
		return groupIds, err
	}

	err = json.Unmarshal(resp.Payload, &groupIds)
	return groupIds, err
}

func (tc *TradfriClient) GetGroup(id string) (model.Group, error) {
	resp, err := tc.Call(tc.dtlsclient.BuildGETMessage("/15004/" + id))
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

	resp, err := tc.Call(tc.dtlsclient.BuildGETMessage("/15001/" + id))
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

	req := tc.dtlsclient.BuildPOSTMessage("/15011/9063", fmt.Sprintf(`{"9090":"%s"}`, clientId))

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
	return tc.dtlsclient.Call(msg)
}
