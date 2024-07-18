package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"go-firestore-gpt/internal/config"
	"go-firestore-gpt/internal/database"
	model "go-firestore-gpt/internal/model"
	productRepository "go-firestore-gpt/internal/repository/product"

	Firestore "firebase.google.com/go/v4"

	"google.golang.org/api/option"
)

func main() {

	cnf := config.LoadConfigOrPanic()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app := createFirestoreAppOrPanic(ctx, cnf.Firebase)
	firestoreClient := createFirestoreClientOrPanic(ctx, app)
	defer firestoreClient.Close()

	productRepo := productRepository.New(&firestoreClient)

	go func() {
		for p := range productRepo.NotifyOnAdded(ctx, nil) {
			if p.Err != nil {
				fmt.Println(p.Err)
				continue
			}
			fmt.Println("Newly added product:", *p.Product.Id)
		}
	}()

	if err := readProductFromJsonAndSaveToFirestore(ctx, productRepo, "./products/B06X1G5YGN.json"); err != nil {
		panic(err)
	}

	// if err := deleteProduct(ctx, productRepo, "B06X1G5YGN"); err != nil {
	// 	panic(err)
	// }

	//<-ctx.Done()
}

func createFirestoreAppOrPanic(ctx context.Context, cnf config.Firebase) *Firestore.App {
	FirestoreCreds, err := json.Marshal(cnf)
	if err != nil {
		panic(err)
	}

	sa := option.WithCredentialsJSON(FirestoreCreds)
	app, err := Firestore.NewApp(ctx, nil, sa)
	if err != nil {
		panic(err)
	}
	return app
}

func createFirestoreClientOrPanic(ctx context.Context, app *Firestore.App) database.FirestoreClient {
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		panic(err)
	}
	return database.New(firestoreClient)
}

func saveProductAsJson(ctx context.Context, productRepo productRepository.ProductRepository, productId string) error {
	p, err := productRepo.GetById(ctx, productId)
	if err != nil {
		panic(err)
	}

	jsonData, err := json.MarshalIndent(*p, "", "    ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return err
	}

	// Write JSON data to a file
	file, err := os.Create(fmt.Sprintf("%s.json", productId))
	if err != nil {
		fmt.Println("Error creating file:", err)
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonData)

	return err
}

func readProductFromJsonAndSaveToFirestore(ctx context.Context, productRepo productRepository.ProductRepository, filePath string) error {
	// Read JSON data from a file
	jsonFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return err
	}
	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return err
	}

	var product model.Product
	if err := json.Unmarshal(byteValue, &product); err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return err
	}

	// Save product to Firestore
	err = productRepo.Create(ctx, product)
	if err != nil {
		fmt.Println("Error saving product to Firestore:", err)
		return err
	}

	fmt.Println("Product saved to Firestore successfully.")
	return nil
}

func deleteProduct(ctx context.Context, productRepo productRepository.ProductRepository, productId string) error {
	return productRepo.Delete(ctx, productId)
}
