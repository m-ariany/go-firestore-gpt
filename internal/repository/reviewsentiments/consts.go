package reviewsentiments

import "time"

const (
	// collection name
	reviewSentimentsNode string = "reviewSentiments"
	sentimentsNode       string = "sentiments"

	// relevantVideos's Field names and paths
	ProductIdFieldPath string = "productId"
	CreatedAtFieldPath string = "createdAt"
	UpdatedAtFieldPath string = "updatedAt"

	// videos's Field names and paths
	SentimentLabelFieldPath    string = "label"
	SentimentPositiveFieldPath string = "positive"
	SentimentNegativeFieldPath string = "negative"
	VideoCreatedAtFieldPath    string = "createdAt"

	// It must not exceed the write timeout of the database.firestore.notifyOnChanges
	channelWriteTimeout time.Duration = time.Second * 3
)
