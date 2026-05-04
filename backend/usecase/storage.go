package usecase

import (
	"context"
	"encoding/json"

	appcrypto "github.com/aethernet-0g/aethernet/backend/infrastructure/crypto"
)

type StorageService struct {
	Client ZGStorageClient
}

type PersonalityUpload struct {
	AgentID     string         `json:"agentId"`
	Name        string         `json:"name"`
	Archetype   string         `json:"archetype"`
	Prompt      string         `json:"prompt"`
	Preferences map[string]any `json:"preferences,omitempty"`
}

func (s StorageService) UploadPersonality(ctx context.Context, personality PersonalityUpload) (string, error) {
	return s.Client.UploadJSON(ctx, personality)
}

func (s StorageService) UploadGeneratedImage(ctx context.Context, imageBytes []byte) (string, error) {
	return s.Client.UploadBytes(ctx, "image/webp", imageBytes)
}

func (s StorageService) AppendEncryptedMemory(
	ctx context.Context,
	agentID string,
	key []byte,
	existing []byte,
	nextEntry map[string]any,
) (string, error) {
	entryBytes, err := json.Marshal(nextEntry)
	if err != nil {
		return "", err
	}

	plain := append(append([]byte(nil), existing...), '\n')
	plain = append(plain, entryBytes...)

	encrypted, err := appcrypto.EncryptAESGCM(key, plain, []byte(agentID))
	if err != nil {
		return "", err
	}
	return s.Client.UploadJSON(ctx, encrypted)
}

func (s StorageService) FetchByPointer(ctx context.Context, pointer string) ([]byte, error) {
	return s.Client.Fetch(ctx, pointer)
}
