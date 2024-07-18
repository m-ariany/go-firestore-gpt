package utils

func BoolToPointer(b bool) *bool {
	return &b
}

func StringToPointer(s string) *string {
	return &s
}

func IntToPointer(i int) *int {
	return &i
}

func Float32ToPointer(f float32) *float32 {
	return &f
}
