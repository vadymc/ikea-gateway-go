package gateway_handler

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/eriklupander/tradfri-go/model"
	"github.com/stretchr/testify/assert"
)

type MockTradfriClient struct {
	deviceJsonPath string
}

func (m *MockTradfriClient) GetGroupIds() ([]int, error) {
	return []int{1}, nil
}

func (m *MockTradfriClient) GetGroup(id string) (model.Group, error) {
	return model.Group{Name: "TestGroup"}, nil
}

func (m *MockTradfriClient) GetGroupDevices(group model.Group) ([]model.Device, error) {
	device := &model.Device{}
	b, _ := ioutil.ReadFile("test_data/" + m.deviceJsonPath)
	json.Unmarshal(b, &device)
	return []model.Device{*device}, nil
}

type MockDBStorage struct {
	invocationCount int
}

func (s *MockDBStorage) SaveGroupState(l []LightState) {
	s.invocationCount++
}

func TestLightStateChange(t *testing.T) {
	testData := []struct {
		name string
		data []string
	}{
		{"bulbs", []string{"bulb_on.json", "bulb_off.json"}},
		{"brightness", []string{"brightness_low.json", "brightness_high.json"}},
		{"RGB", []string{"rgb_warm.json", "rgb_white.json"}},
	}

	tc := &MockTradfriClient{}
	for _, td := range testData {
		t.Run(td.name, func(t *testing.T) {
			s := &MockDBStorage{}
			h := NewHandler(tc, s)

			// initial state
			tc.deviceJsonPath = td.data[0]
			h.PollAndSaveDevicesState()

			// updated state
			tc.deviceJsonPath = td.data[1]
			h.PollAndSaveDevicesState()

			assert.Equal(t, 1, s.invocationCount)
		})
	}
}
