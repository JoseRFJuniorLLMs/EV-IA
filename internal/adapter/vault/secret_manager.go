package vault

import (
	"github.com/hashicorp/vault/api"
)

type SecretManager struct {
	client *api.Client
}

func NewSecretManager(address, token string) (*SecretManager, error) {
	config := api.DefaultConfig()
	config.Address = address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	client.SetToken(token)

	return &SecretManager{client: client}, nil
}

func (sm *SecretManager) GetDatabaseCredentials() (string, error) {
	secret, err := sm.client.Logical().Read("secret/data/database")
	if err != nil {
		return "", err
	}

	data := secret.Data["data"].(map[string]interface{})
	return data["connection_string"].(string), nil
}

func (sm *SecretManager) GetGeminiAPIKey() (string, error) {
	secret, err := sm.client.Logical().Read("secret/data/gemini")
	if err != nil {
		return "", err
	}

	data := secret.Data["data"].(map[string]interface{})
	return data["api_key"].(string), nil
}
