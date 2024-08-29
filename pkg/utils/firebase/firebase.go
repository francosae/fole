package firebase

import (
	"context"
	"fmt"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"firebase.google.com/go/storage"
	"google.golang.org/api/option"
)

type FirebaseApp struct {
	App       *firebase.App
	Auth      *auth.Client
	Firestore *firestore.Client
	Storage   *storage.Client
}

func NewFirebaseApp(credentialsFile string, backupPath string) (*FirebaseApp, error) {
	initializeApp := func(path string) (*FirebaseApp, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		opt := option.WithCredentialsFile(path)

		config := &firebase.Config{
			StorageBucket: "joystick-database.appspot.com",
		}

		app, err := firebase.NewApp(ctx, config, opt)
		if err != nil {
			return nil, fmt.Errorf("error initializing app: %v", err)
		}

		storage, err := app.Storage(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting Storage client: %v", err)
		}

		auth, err := app.Auth(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting Auth client: %v", err)
		}

		firestore, err := app.Firestore(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting Firestore client: %v", err)
		}

		return &FirebaseApp{
			App:       app,
			Auth:      auth,
			Firestore: firestore,
			Storage:   storage,
		}, nil
	}

	firebaseApp, err := initializeApp(credentialsFile)
	if err == nil {
		return firebaseApp, nil
	}

	firebaseApp, err = initializeApp(backupPath)
	if err == nil {
		return firebaseApp, nil
	}

	return nil, err
}

func (fa *FirebaseApp) Close() {
	if fa.Firestore != nil {
		if err := fa.Firestore.Close(); err != nil {
			log.Printf("Error closing Firestore client: %v", err)
		}
	}
}
