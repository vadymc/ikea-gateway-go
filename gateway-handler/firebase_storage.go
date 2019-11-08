package gateway_handler

import (
	"context"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	log "github.com/sirupsen/logrus"
)

type FirebaseStorage struct {
	statDataCollection *firestore.CollectionRef
}

func NewFirebaseStorage() *FirebaseStorage {
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "ikea-gw")
	if err != nil {
		log.WithError(err).Error("Failed to create firestore client")
		return nil
	}
	s := FirebaseStorage{}
	s.statDataCollection = client.Collection("stat_data")
	return &s
}

func (s *FirebaseStorage) SaveGroupState(ctx context.Context, lightGroup []LightState, wg *sync.WaitGroup) {
	start := time.Now()
	defer func() {
		wg.Done()
		log.WithField("SaveGroupState", "firebase storage").WithField("elapsed time", time.Since(start)).Info("Done")
	}()
	_, _, err := s.statDataCollection.Add(ctx, map[string][]LightState{"lightState": lightGroup})
	if err != nil {
		log.WithError(err).WithField("light Group", lightGroup).Error("Failed to insert document")
	}
}
