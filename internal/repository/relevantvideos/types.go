package relevantvideos

import "go-firestore-gpt/internal/model"

type RelevantVideosEvent struct {
	RelevantVideos model.RelevantVideos
	Err            error
}
