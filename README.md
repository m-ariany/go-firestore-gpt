# go-firestore-gpt

This project implements a worker written in Go that integrates with Firebase Firestore to enhance product data using ChatGPT. The worker listens for new products added to the database, and performs the following enrichments:

- Product Review Sentiment Analysis: Utilizes natural language processing to analyze and classify the sentiment of product reviews.

- YouTube Related Videos: Finds and associates relevant YouTube videos based on product information.

**Features**

- Real-time Data Processing: Listens to Firestore database changes in real-time, ensuring timely updates to product information.

- Natural Language Processing: Uses ChatGPT to perform sentiment analysis on product reviews, helping to gauge customer satisfaction.

- YouTube Integration: Finds and links YouTube videos related to the products added, enhancing customer engagement and information.

## Running the Backend
To run the backend, you have two options: download the latest released executable or build the project yourself.

#### Build
If you choose to build the project yourself, use the `build.sh` file.

#### Environment Variables
Before running the backend, make sure to set the following environment variables:

```
# Gilas.io API
export GILAS_API_KEY=
export GILAS_API_URL=https://api.gilas.io/v1
export GILAS_GPT_MODEL=gpt-3.5-turbo

# Firebase
FIREBASE_TYPE=<type_value>
FIREBASE_PROJECT_ID=<project_id_value>
FIREBASE_PRIVATE_KEY_ID=<private_key_id_value>
FIREBASE_PRIVATE_KEY=<base64_encoded_private_key_value>
FIREBASE_CLIENT_EMAIL=<client_email_value>
FIREBASE_CLIENT_ID=<client_id_value>
FIREBASE_AUTH_URI=<auth_uri_value>
FIREBASE_TOKEN_URI=<token_uri_value>
FIREBASE_AUTH_PROVIDER_X509_CERT_URL=<auth_provider_x509_cert_url_value>
FIREBASE_CLIENT_X509_CERT_URL=<client_x509_cert_url_value>
FIREBASE_WRITE_TIMEOUT_SECOND=10s

# Youtube
export YOUTUBE_API_KEY=<api_key_value>
```

#### Run
To run the backend, make sure the required environment variables are set as described above. Then,

```sh
<os>-<arch>-buywise-go
```
