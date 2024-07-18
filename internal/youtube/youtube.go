package youtube

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-firestore-gpt/internal/config"
	"go-firestore-gpt/internal/utils"

	"github.com/rs/zerolog/log"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

var ErrNoResponse error = fmt.Errorf("YouTube API returned no response")

type Video struct {
	ID          string
	URL         string
	Title       string
	Description string
}

type YouTubeAPI interface {
	Search(term string, maxResult int64) ([]Video, error)
}

type YouTubeClient struct {
	Service *youtube.Service
}

var (
	once          sync.Once
	instance      *YouTubeClient
	youtubeScopes = []string{youtube.YoutubeReadonlyScope}
)

func NewYouTubeClient(ctx context.Context, cnf config.Youtube) *YouTubeClient {
	once.Do(func() {
		service, err := youtube.NewService(ctx, option.WithAPIKey(cnf.ApiKey), option.WithScopes(youtubeScopes...))
		if err != nil {
			log.Error().Err(err).Msg("Failed to create YouTube service")
			return
		}
		instance = &YouTubeClient{Service: service}
	})
	return instance
}

func (c *YouTubeClient) Search(term string, maxResult int64) ([]Video, error) {
	log.Debug().Msgf("Search YouTube for %s", term)
	call := c.Service.Search.List([]string{"id,snippet"}).
		Type("video").
		Q(term).
		MaxResults(maxResult)

	var response *youtube.SearchListResponse
	var err error

	retryHandler := utils.NewRetryHandler(time.Second*10, time.Second*3, 3)
	retryHandler.Do(func() error {
		response, err = call.Do()
		if err != nil {
			return err
		}
		if len(response.Items) == 0 {
			err = ErrNoResponse
		}
		return err
	})

	if err != nil {
		return nil, err
	}

	videos := make([]Video, 0, len(response.Items))
	for _, item := range response.Items {
		video := Video{
			ID:          item.Id.VideoId,
			URL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.Id.VideoId),
			Title:       item.Snippet.Title,
			Description: item.Snippet.Description,
		}
		videos = append(videos, video)
	}

	return videos, nil
}
