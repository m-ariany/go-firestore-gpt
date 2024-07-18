package reviewsentiment

type response struct {
	Data []sentimentScore `json:"data"`
}

type sentimentScore struct {
	Label string `json:"label"`
	Score int    `json:"score"`
}

const (
	SENTIMENT_ANALYSIS_INSTRUCTION string = `Analyze a list of reviews enclosed within <rev> </rev> tags and separated by '~' character. 
	For each review, assign a single label from the provided list of labels that accurately represents a product feature 
	or specification mentioned in the text. 
	Additionally, give a sentiment score between 0 and 5, where 0 is very negative and 5 is very positive to the review.
	Generate a JSON formated response, containing a list of items under the 'data' key, and each item should have 'label,' and 'score' keys.
	Example:
	{
		"data": [
			{
				"label": lable,
				"score": score,
			},
			...
			{
				"label": lable,
				"score": score,
			}
		]
	}

	<labels>Size, Quality, Value, Durability, Design, Performance, Material, Safety, Reliability, Ease of Use, Features, Warranty, Customer Service, Packaging, Compatibility, Versatility, Sustainability, User-Friendliness, Appearance
	</labels>

	<rev>%s</rev>`
)
