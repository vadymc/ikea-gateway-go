package handler

import (
	"context"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vadymc/ikea-gateway-go/m/ikea"
	"github.com/vadymc/ikea-gateway-go/m/sql"
)

type Handler struct {
	tc ikea.ITradfriClient
	s  []sql.IStorage
	m  map[string][]sql.LightState
}

// Creates new instance of a Handler.
func NewHandler(tc ikea.ITradfriClient, s ...sql.IStorage) Handler {
	return Handler{
		tc: tc,
		s:  s,
		m:  make(map[string][]sql.LightState),
	}
}

func (h *Handler) PollAndSaveDevicesState() {
	groupIds, err := h.tc.GetGroupIds()
	if err != nil {
		log.WithError(err).Error("Failed to get Group IDs")
		return
	}
	for _, groupId := range groupIds {
		group, err := h.tc.GetGroup(strconv.Itoa(groupId))
		if err != nil {
			log.WithError(err).
				WithField("Group ID", groupId).
				Error("Failed to get Group")
			return
		}

		devices, err := h.tc.GetGroupDevices(group)
		if err != nil {
			log.WithError(err).
				WithField("Group ID", groupId).
				Error("Failed to get Group devices")
			return
		}

		var l []sql.LightState
		for _, d := range devices {
			if len(d.LightControl) > 0 {
				lc := d.LightControl[0]
				ls := sql.LightState{lc.Power, lc.Dimmer, lc.RGBHex, group.Name, time.Now()}
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

func (h *Handler) persistStateChange(l []sql.LightState) {
	timeout := time.Duration(4 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	var wg sync.WaitGroup

	for _, storage := range h.s {
		wg.Add(1)
		go storage.SaveGroupState(ctx, l, &wg)
	}

	go func() {
		wg.Wait()
		cancel()
	}()

	select {
	case <-ctx.Done():
		log.WithField("LightState", l).Info("Successfully saved")
	case t := <-time.After(timeout):
		cancel()
		log.WithField("LightState", l).WithField("timeout", t).Warn("Timeout in persistStateChange")
	}

}

func (h *Handler) equal(l1, l2 []sql.LightState) bool {
	for i, l := range l1 {
		if l.Power != l2[i].Power || l.RGB != l2[i].RGB || l.Dimmer != l2[i].Dimmer {
			return false
		}
	}
	return true
}
