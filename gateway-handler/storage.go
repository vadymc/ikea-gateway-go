package gateway_handler

import log "github.com/sirupsen/logrus"

type IStorage interface {
	SaveGroupState(l []LightState)
}

type DBStorage struct {
}

func (s *DBStorage) SaveGroupState(l []LightState) {
	log.WithField("arg", l).Info("Saving group state")
}
