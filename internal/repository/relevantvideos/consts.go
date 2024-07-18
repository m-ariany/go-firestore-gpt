package relevantvideos

import "time"

const (
	// collection name
	relevantVideosNode string = "relevantVideos"
	videosNode         string = "videos"

	// relevantVideos's Field names and paths
	ProductIdFieldPath string = "productId"
	ReadyFieldPath     string = "ready"
	CreatedAtFieldPath string = "createdAt"
	UpdatedAtFieldPath string = "updatedAt"

	// videos's Field names and paths
	VideoIdFieldPath        string = "id"
	VideoUrlFieldPath       string = "url"
	VideoThumbUpFieldPath   string = "thumbup"
	VideoThumbDownFieldPath string = "thumbdown"
	VideoCreatedAtFieldPath string = "createdAt"
	VideoUpdatedAtFieldPath string = "updatedAt"

	// It must not exceed the write timeout of the database.firestore.notifyOnChanges
	channelWriteTimeout time.Duration = time.Second * 3
)
