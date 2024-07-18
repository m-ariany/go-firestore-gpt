package utils

import (
	"encoding/json"
	"fmt"

	"cloud.google.com/go/firestore"
)

func DocSnapToType(doc *firestore.DocumentSnapshot, v interface{}) error {
	if doc == nil {
		return fmt.Errorf("doc is nil")
	}

	jsonStr, err := json.Marshal(doc.Data())
	if err != nil {
		return err
	}

	if err := json.Unmarshal(jsonStr, v); err != nil {
		return err
	}

	return nil
}
