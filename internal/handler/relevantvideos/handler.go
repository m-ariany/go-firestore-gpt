package relevantvideos

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-firestore-gpt/internal/eventpublisher"
	"go-firestore-gpt/internal/eventpublisher/event"
	"go-firestore-gpt/internal/handler/relevantvideos/instructor"
	"go-firestore-gpt/internal/model"
	relevantVideosRepository "go-firestore-gpt/internal/repository/relevantvideos"
	"go-firestore-gpt/internal/utils"
	"go-firestore-gpt/internal/youtube"

	gpt "go-firestore-gpt/internal/gpt"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type Handler struct {
	productEventPublisher eventpublisher.Publisher
	relevantVideosRepo    relevantVideosRepository.IRepository
	gptFactory            gpt.ClientFactory
	youtubeClient         youtube.YouTubeAPI
	productSubscriptionCh event.EventChannel
}

func New(
	productEventPublisher eventpublisher.Publisher,
	relevantVideosRepo relevantVideosRepository.IRepository,
	gptFactory gpt.ClientFactory,
	youtubeClient youtube.YouTubeAPI) *Handler {
	return &Handler{
		productEventPublisher: productEventPublisher,
		relevantVideosRepo:    relevantVideosRepo,
		gptFactory:            gptFactory,
		youtubeClient:         youtubeClient,
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

func (h *Handler) Handle(ctx context.Context) error {

	h.subscribeToEvents()
	defer h.unsubscribeFromEvents()

	// if either of the handlers fail the process will be incomplete,
	// so return an error
	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return h.videoEventHandler(ctx)
	})

	group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event, ok := <-h.productSubscriptionCh:
				if !ok {
					return nil
				}

				if event.Err != nil {
					log.Error().Err(event.Err).Msg("relevant video (EventHandler): error reading events")
					return event.Err
				}

				product, ok := event.Message.(model.Product)
				if !ok {
					continue
				}

				go h.handleProduct(ctx, product)
			}
		}
	})

	err := group.Wait()
	if err != nil {
		log.Error().Err(err).Msg("relevant video handler stopped")
	}
	return err
}

func (h *Handler) videoEventHandler(ctx context.Context) error {

	events := h.relevantVideosRepo.NotifyOnAdded(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}

			if event.Err != nil {
				log.Error().Err(event.Err).Msg("relevant video (videoEventHandler): error reading events")
				return event.Err
			}
			go h.handleVideos(ctx, event.RelevantVideos)
		}
	}
}

func (h Handler) handleProduct(ctx context.Context, product model.Product) error {
	if product.Id == nil || product.Name == nil {
		return nil
	}

	err := h.relevantVideosRepo.CreateIfNotExist(ctx, model.RelevantVideos{
		ProductId:   product.Id,
		ProductName: product.Name,
		Ready:       utils.BoolToPointer(false),
	})
	if err != nil {
		log.Error().Err(err).Msgf("create related video, id: %s", *product.Id)
	}
	return err
}

func (h *Handler) handleVideos(ctx context.Context, relevantVideo model.RelevantVideos) error {

	suggestedVideos, err := h.searchYoutube(ctx, relevantVideo)
	if err != nil {
		return err
	}

	relevantVideos, err := h.evaluateSuggestedVideos(ctx, relevantVideo, suggestedVideos)
	if err != nil {
		return err
	}
	if len(relevantVideos) == 0 {
		log.Debug().Msgf("No video found for productId: %s", *relevantVideo.ProductId)
		return nil
	}

	relevantVideo.Videos = relevantVideos
	relevantVideo.Ready = utils.BoolToPointer(true)
	err = h.relevantVideosRepo.Update(ctx, relevantVideo)

	if err != nil {
		log.Error().Err(err).Msgf("failed to update relevant video for productId: %s", *relevantVideo.ProductId)
	}

	return err
}

func (h *Handler) searchYoutube(ctx context.Context, relevantVideo model.RelevantVideos) ([]youtube.Video, error) {
	suggestedVideos := []youtube.Video{}

	gptClient, err := h.gptFactory.Client()
	if err != nil {
		return nil, err
	}

	gptClient.Instruct(instructor.GetProductNameExtractionInstruction())
	productName, err := gptClient.Prompt(ctx, *relevantVideo.ProductName)
	if err != nil {
		log.Error().Err(err).Msg("failed to create search term for YouTube")
		return suggestedVideos, err
	}

	searchTerm := fmt.Sprintf("%s", productName)
	suggestedVideos, err = h.youtubeClient.Search(searchTerm, 10)
	if err != nil {
		log.Error().Err(err).Msg("failed to call YouTube")
	}
	return suggestedVideos, err
}

func (h *Handler) evaluateSuggestedVideos(ctx context.Context, relevantVideo model.RelevantVideos, suggestedVideos []youtube.Video) ([]model.Video, error) {

	selectedVideos := []model.Video{}
	suggestedVideosAsJson, err := suggestedVideosToJson(suggestedVideos)
	if err != nil {
		log.Error().Err(err).Msg("failed to conver suggested videos to json")
		return selectedVideos, err
	}

	gptClient, err := h.gptFactory.Client()
	if err != nil {
		return nil, err
	}

	// Use the full product name since it includes more details about the product
	gptClient.Instruct(instructor.GetVideoEvaluationInstruction())
	prompt := fmt.Sprintf("Product name: '%s'\nVideos: '%s'", *relevantVideo.ProductName, suggestedVideosAsJson)
	relatedVideoIDs, err := gptClient.Prompt(ctx, prompt)
	if err != nil {
		log.Error().Err(err).Msg("failed to evaluate suggested videos")
		return selectedVideos, err
	}

	relevantVideos := filterSuggestedVideos(relatedVideoIDs, suggestedVideos)
	if len(relevantVideos) == 0 {
		log.Debug().Msgf("Could not find any relevant video for productId %s", *relevantVideo.ProductId)
	}

	return relevantVideos, err
}

func suggestedVideosToJson(videos []youtube.Video) (string, error) {

	type videoFormat struct {
		Id          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	first500Chars := func(s string) string {
		limit := 500
		min := func(a, b int) int {
			if a < b {
				return a
			}
			return b
		}
		end := min(limit, len(s))
		return s[:end]
	}

	vs := []videoFormat{}
	for _, item := range videos {
		vs = append(vs,
			videoFormat{item.ID,
				item.Title,
				first500Chars(item.Description)})
	}

	b, err := json.Marshal(vs)
	if err != nil {
		log.Error().Err(err).Msgf("suggested videos to json %s", videos)
		return "", err
	}

	return string(b), nil
}

// Return the suggested videos that are included in the relatedVideoIDs
func filterSuggestedVideos(relatedVideoIDs string, suggestedVideos []youtube.Video) []model.Video {
	videos := []model.Video{}
	// The instruction suggest that the relatedVideoIDs are comma separated
	relatedVideoIDsList := strings.Split(relatedVideoIDs, ",")

	for _, videoId := range relatedVideoIDsList {
		for _, suggested := range suggestedVideos {
			if videoId == suggested.ID {
				videos = append(videos, model.Video{Url: suggested.URL})
			}
		}
	}

	return videos
}
