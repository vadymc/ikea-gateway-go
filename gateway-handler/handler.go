package gateway_handler

import (
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/vadymc/ikea-gateway-go/m/ikea"
)

type LightState struct {
	power  int
	dimmer int
	rgb    string
	group  string
}

type Handler struct {
	tc ikea.ITradfriClient
	s  IStorage
	m  map[string][]LightState
}

// Creates new instance of a Handler.
func NewHandler(tc ikea.ITradfriClient, s IStorage) Handler {
	return Handler{
		tc: tc,
		s:  s,
		m:  make(map[string][]LightState),
	}
}

func (h *Handler) PollAndSaveDevicesState() {
	groupIds, err := h.tc.GetGroupIds()
	if err != nil {
		log.WithError(err).Error("Failed to get group IDs")
		return
	}
	for _, groupId := range groupIds {
		group, err := h.tc.GetGroup(strconv.Itoa(groupId))
		if err != nil {
			log.WithError(err).
				WithField("group ID", groupId).
				Error("Failed to get group")
			return
		}

		devices, err := h.tc.GetGroupDevices(group)
		if err != nil {
			log.WithError(err).
				WithField("group ID", groupId).
				Error("Failed to get group devices")
			return
		}

		var l []LightState
		for _, d := range devices {
			if len(d.LightControl) > 0 {
				lc := d.LightControl[0]
				ls := LightState{lc.Power, lc.Dimmer, lc.RGBHex, group.Name}
				l = append(l, ls)
			}
		}

		if prevL, ok := h.m[group.Name]; ok && !h.equal(prevL, l) {
			log.WithField("old state", prevL).WithField("new state", l).Info("Device state changed")
			h.m[group.Name] = l
			h.persistStateChange(l)
		} else if !ok {
			h.m[group.Name] = l
		}
	}
}

func (h *Handler) persistStateChange(l []LightState) {
	h.s.SaveGroupState(l)
}

func (h *Handler) equal(l1, l2 []LightState) bool {
	for i, v := range l1 {
		if v != l2[i] {
			return false
		}
	}
	return true
}
