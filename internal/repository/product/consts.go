package product

import "time"

const (
	// collection name
	productNode string = "products"
	qasNode     string = "qas"
	reviewNode  string = "reviews"

	// Fields' name and path
	IdFieldPath                string = "id"
	NameFieldPath              string = "name"
	DescriptionFieldPath       string = "description"
	SentimentAnalizedFieldPath string = "sentimentAnalized"
	RelatedVideosAnalized      string = "relatedVideosAnalized"
	CreatedAtFieldPath         string = "createdAt"
	UpdatedAtFieldPath         string = "updatedAt"

	// It must not exceed the write timeout of the database.firestore.notifyOnChanges
	channelWriteTimeout time.Duration = time.Second * 3
)
