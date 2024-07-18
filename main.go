package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-firestore-gpt/internal/config"
	"go-firestore-gpt/internal/database"
	relevantVideoHandler "go-firestore-gpt/internal/handler/relevantvideos"
	reviewSentimentHandler "go-firestore-gpt/internal/handler/reviewsentiment"
	productRepository "go-firestore-gpt/internal/repository/product"
	relevantVideoRepository "go-firestore-gpt/internal/repository/relevantvideos"
	reviewSentimentsRepository "go-firestore-gpt/internal/repository/reviewsentiments"
	"go-firestore-gpt/internal/utils"
	youtubeApi "go-firestore-gpt/internal/youtube"

	gpt "go-firestore-gpt/internal/gpt"
	gptutils "go-firestore-gpt/internal/gpt/utils"

	Firestore "firebase.google.com/go/v4"

	productEventPublisher "go-firestore-gpt/internal/eventpublisher/product"

	"golang.org/x/sync/errgroup"
	"google.golang.org/api/option"
)

func main() {

	cnf := config.LoadConfigOrPanic()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	defer close(sigs)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	app := createFirestoreAppOrPanic(ctx, cnf.Firebase)
	firestoreClient := createFirestoreClientOrPanic(ctx, app)
	defer firestoreClient.Close()

	tokenizer, err := gptutils.NewTokenzier()
	if err != nil {
		panic(err)
	}

	gptFactory, err := gpt.NewClientFactory(gpt.ClientConfig{
		ApiUrl:      cnf.GilasAI.ApiUrl,
		ApiKey:      cnf.GilasAI.ApiKey,
		Model:       cnf.GilasAI.Model,
		Temperature: utils.Float32ToPointer(0.1),
	})

	if err != nil {
		panic(err)
	}

	productRepo := productRepository.New(&firestoreClient)
	reviewSentimentRepo := reviewSentimentsRepository.New(&firestoreClient)
	relevantVideoRepo := relevantVideoRepository.New(&firestoreClient)
	youtubeClient := youtubeApi.NewYouTubeClient(ctx, cnf.Youtube)
	if youtubeClient == nil {
		panic(fmt.Errorf("failed to create a youtube client"))
	}
	productSentimentPublisher := productEventPublisher.ProductPublisherFactory(productRepo).OnProductReviewSentimentAnalysis()
	productVideoPublisher := productEventPublisher.ProductPublisherFactory(productRepo).OnProductVideoAnalysis()

	rv := relevantVideoHandler.New(productVideoPublisher, relevantVideoRepo, gptFactory, youtubeClient)
	rs := reviewSentimentHandler.New(productSentimentPublisher, productRepo, reviewSentimentRepo, gptFactory, tokenizer)

	group, gctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return rv.Handle(gctx)
	})
	group.Go(func() error {
		return rs.EventHandler(gctx)
	})
	group.Go(func() error {
		return productSentimentPublisher.Start(gctx)
	})
	group.Go(func() error {
		return productVideoPublisher.Start(gctx)
	})

	select {
	case <-sigs:
		// Received a termination signal, continue to shutdown
	case <-gctx.Done():
		// errgroup encountered an error, continue to shutdown
	}

	cancel() // cancel the root context to signal all the consumers

	select {
	case <-time.After(time.Second * 5):
		// Give enough time to close all the pending resources
	case <-sigs:
		// Forcefully terminate the app with a signal
	}

	os.Exit(1)
}

func createFirestoreAppOrPanic(ctx context.Context, cnf config.Firebase) *Firestore.App {
	FirestoreCreds, err := json.Marshal(cnf)
	if err != nil {
		panic(err)
	}

	sa := option.WithCredentialsJSON(FirestoreCreds)
	app, err := Firestore.NewApp(ctx, nil, sa)
	if err != nil {
		panic(err)
	}
	return app
}

func createFirestoreClientOrPanic(ctx context.Context, app *Firestore.App) database.FirestoreClient {
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		panic(err)
	}
	return database.New(firestoreClient)
}
