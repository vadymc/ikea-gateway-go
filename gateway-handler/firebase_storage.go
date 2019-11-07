package gateway_handler

import (
	"context"

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

func (s *FirebaseStorage) SaveGroupState(lightGroup []LightState) {
	r, _, err := s.statDataCollection.Add(context.Background(), map[string][]LightState{"lightState": lightGroup})
	log.WithField("Document ID", r.ID).Info("Stored update to firebase")
	if err != nil {
		log.WithError(err).WithField("light Group", lightGroup).Error("Failed to insert document")
	}
}
