package instructor

const (
	PRODUCT_NAME_EXTRACTION_INSTRUCTION string = `You are a product specialist. Extract the product name and model, if mentioned,
	 from the given product description. If the model is not mentioned, return an empty string. Format your response as 'name model'.`

	VIDEO_EVALUATION_INSTRUCTION string = `Given a product name and a JSON list of YouTube video info, 
	first understand the product type and brand. Then use the product name, type, and brand to identify relevant video IDs. 
	Analyze each video's details, noting IDs related to the product and its type. 
	Respond with a comma-separated list of these IDs, or return -1 if no videos are relevant. Do not include any other text in your response.`
)
