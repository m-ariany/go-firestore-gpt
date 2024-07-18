package reviewsentiment

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"go-firestore-gpt/internal/eventpublisher"
	"go-firestore-gpt/internal/eventpublisher/event"
	gptutils "go-firestore-gpt/internal/gpt/utils"
	"go-firestore-gpt/internal/model"
	productRepository "go-firestore-gpt/internal/repository/product"
	sentimentRepository "go-firestore-gpt/internal/repository/reviewsentiments"
	"go-firestore-gpt/internal/utils"

	gpt "go-firestore-gpt/internal/gpt"

	"github.com/rs/zerolog/log"
)

type Handler struct {
	productEventPublisher eventpublisher.Publisher
	productRepo           productRepository.IRepository
	sentimentRepo         sentimentRepository.IRepository
	gptFactory            gpt.ClientFactory
	tokenizer             gptutils.Tokenizer
	productSubscriptionCh event.EventChannel
}

func New(
	productEventPublisher eventpublisher.Publisher,
	productRepo productRepository.IRepository,
	sentimentRepo sentimentRepository.IRepository,
	gptFactory gpt.ClientFactory,
	tokenizer gptutils.Tokenizer) *Handler {

	return &Handler{
		productEventPublisher: productEventPublisher,
		productRepo:           productRepo,
		sentimentRepo:         sentimentRepo,
		gptFactory:            gptFactory,
		tokenizer:             tokenizer,
		productSubscriptionCh: make(event.EventChannel),
	}
}

func (h *Handler) subscribeToEvents() {
	h.productEventPublisher.Subscribe(h.eventChannel())
}

func (h *Handler) unsubscribeFromEvents() {
	h.productEventPublisher.Unsubscribe(h.eventChannel())
}

func (h *Handler) eventChannel() chan<- event.Event {
	return h.productSubscriptionCh
}

func (h *Handler) EventHandler(ctx context.Context) error {

	h.subscribeToEvents()
	defer h.unsubscribeFromEvents()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-h.productSubscriptionCh:
			if !ok {
				return nil
			}

			if event.Err != nil {
				log.Error().Err(event.Err).Msg("sentiment handler: error reading events")
				return event.Err
			}

			product, ok := event.Message.(model.Product)
			if !ok {
				continue
			}

			go h.handle(ctx, product)
		}
	}
}

func (h *Handler) handle(ctx context.Context, product model.Product) error {

	// if sentiment analysis is already done, skip
	s, _ := h.sentimentRepo.GetById(ctx, *product.Id)
	if s != nil {
		log.Debug().Msgf("sentiment is already analyzed - productId %s", *product.Id)
		return nil
	}

	log.Debug().Msgf("sentiment analysis - productId %s", *product.Id)
	sentimentScores, err := h.generateSentimentScores(ctx, product)
	if err != nil {
		log.Error().Err(err).Msgf("review sentiment handler: failed to generate sentiments for %s", *product.Id)
		return err
	}

	top5Sentiments := selectTop5FrequentlyMentionedSentiments(sentimentScores)

	if err := h.sentimentRepo.Create(ctx, model.ReviewSentiments{
		ProductId:  product.Id,
		Sentiments: top5Sentiments,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}); err != nil {
		log.Error().Err(err).Msgf("review sentiment handler: failed to persist %s", *product.Id)
		return err
	}

	product.SentimentAnalized = utils.BoolToPointer(true)
	h.productRepo.Update(ctx, *product.Id, product)

	return nil
}

func (h *Handler) generateSentimentScores(ctx context.Context, product model.Product) ([]sentimentScore, error) {

	callGPT := func(ctx context.Context, instruction string) (string, error) {
		gptClient, err := h.gptFactory.Client()

		if err != nil {
			return "", err
		}

		gptClient.Instruct(instruction)
		return gptClient.Prompt(ctx, "")
	}

	response, err := callGPT(ctx, fmt.Sprintf(SENTIMENT_ANALYSIS_INSTRUCTION, h.productReviews(product)))
	if err != nil {
		return nil, err
	}

	return responseToSentimentScore(response)
}

func (h *Handler) productReviews(product model.Product) string {
	sb := strings.Builder{}
	for _, review := range product.Reviews {
		sb.WriteString(fmt.Sprintf("~%s\n", *review.Comment))
	}

	return sb.String()
}

func responseToSentimentScore(responseAsString string) ([]sentimentScore, error) {

	data := response{}
	err := json.Unmarshal([]byte(responseAsString), &data)
	if err != nil {
		return []sentimentScore{}, err
	}

	return data.Data, nil
}

func selectTop5FrequentlyMentionedSentiments(data []sentimentScore) []model.Sentiment {

	top5 := []model.Sentiment{}
	if len(data) == 0 {
		return top5
	}

	// Create a map to store scores and counts for each label
	sentimentData := make(map[string]struct {
		Total int
		Count int
	})

	// Iterate through the list and update the map
	for _, item := range data {
		data, exists := sentimentData[item.Label]
		if !exists {
			data = struct {
				Total int
				Count int
			}{}
		}
		data.Total += item.Score
		data.Count++
		sentimentData[item.Label] = data
	}

	// Calculate averages and frequencies
	result := make([]struct {
		Label     string
		Average   int
		Frequency int
	}, len(sentimentData))

	i := 0
	for label, data := range sentimentData {
		average := int(data.Total) / int(data.Count)
		result[i] = struct {
			Label     string
			Average   int
			Frequency int
		}{
			Label:     label,
			Average:   average,
			Frequency: data.Count,
		}
		i++
	}

	// Sort the result slice by frequency in descending order
	sort.Slice(result, func(i, j int) bool {
		return result[j].Frequency < result[i].Frequency
	})

	// Select top 5 frequently mentioned sentiments
	for i := 0; i < min(5, len(result)); i++ {
		top5 = append(top5, model.Sentiment{
			Label: result[i].Label,
			Score: result[i].Average,
		})
	}

	return top5
}

func min(a, b int) int {
	if a <= b {
		return a
	}

	return b
}
