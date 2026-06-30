package service

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const seedanceProxyKBasePath = "/api/saas/proxy-k/v1"

func seedanceHTTPTimeout() time.Duration {
	if common.RelayTimeout > 0 {
		return time.Duration(common.RelayTimeout) * time.Second
	}
	return 60 * time.Second
}

type SeedanceAssetClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

type seedanceResponseMetadata struct {
	Action    string `json:"Action"`
	Region    string `json:"Region"`
	RequestId string `json:"RequestId"`
	Service   string `json:"Service"`
	Version   string `json:"Version"`
}

type SeedanceCreateGroupResult struct {
	Id string `json:"Id"`
}

type SeedanceCreateAssetResult struct {
	Id string `json:"Id"`
}

type SeedanceGetAssetResult struct {
	Id          string `json:"Id"`
	GroupId     string `json:"GroupId"`
	Name        string `json:"Name"`
	AssetType   string `json:"AssetType"`
	URL         string `json:"URL"`
	Status      string `json:"Status"`
	ProjectName string `json:"ProjectName"`
	CreateTime  string `json:"CreateTime"`
	UpdateTime  string `json:"UpdateTime"`
}

func NewSeedanceAssetClient(baseURL, apiKey string) *SeedanceAssetClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = "https://agentrs.jd.com"
	}
	return &SeedanceAssetClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: seedanceHTTPTimeout(),
		},
	}
}

func (c *SeedanceAssetClient) post(action string, body any, result any) error {
	if c == nil {
		return fmt.Errorf("seedance asset client is nil")
	}
	payload, err := common.Marshal(body)
	if err != nil {
		return err
	}
	url := c.BaseURL + seedanceProxyKBasePath + "/" + action
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("seedance upstream returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var envelope struct {
		ResponseMetadata seedanceResponseMetadata `json:"ResponseMetadata"`
		Result           any                      `json:"Result"`
		Error            *struct {
			Code    string `json:"Code"`
			Message string `json:"Message"`
		} `json:"Error"`
	}
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("decode seedance response failed: %w", err)
	}
	if envelope.Error != nil && envelope.Error.Message != "" {
		return fmt.Errorf("seedance upstream error: %s", envelope.Error.Message)
	}
	if result == nil || envelope.Result == nil {
		return nil
	}
	resultBody, err := common.Marshal(envelope.Result)
	if err != nil {
		return err
	}
	if err := common.Unmarshal(resultBody, result); err != nil {
		return fmt.Errorf("decode seedance result failed: %w", err)
	}
	return nil
}

func (c *SeedanceAssetClient) CreateAssetGroup(name, description, groupType string) (string, error) {
	var result SeedanceCreateGroupResult
	err := c.post("CreateAssetGroup", map[string]string{
		"Name":        name,
		"Description": description,
		"GroupType":   groupType,
	}, &result)
	if err != nil {
		return "", err
	}
	if result.Id == "" {
		return "", fmt.Errorf("seedance CreateAssetGroup returned empty id")
	}
	return result.Id, nil
}

func (c *SeedanceAssetClient) UpdateAssetGroup(id, name, description string) error {
	var result SeedanceCreateGroupResult
	return c.post("UpdateAssetGroup", map[string]string{
		"Id":          id,
		"Name":        name,
		"Description": description,
	}, &result)
}

func (c *SeedanceAssetClient) CreateAsset(groupId, assetURL, assetType, name string) (string, error) {
	var result SeedanceCreateAssetResult
	body := map[string]string{
		"GroupId":   groupId,
		"URL":       assetURL,
		"AssetType": assetType,
	}
	if name != "" {
		body["Name"] = name
	}
	err := c.post("CreateAsset", body, &result)
	if err != nil {
		return "", err
	}
	if result.Id == "" {
		return "", fmt.Errorf("seedance CreateAsset returned empty id")
	}
	return result.Id, nil
}

func (c *SeedanceAssetClient) GetAsset(id string) (*SeedanceGetAssetResult, error) {
	var result SeedanceGetAssetResult
	err := c.post("GetAsset", map[string]string{"Id": id}, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *SeedanceAssetClient) UpdateAsset(id, name string) error {
	var result SeedanceCreateAssetResult
	return c.post("UpdateAsset", map[string]string{
		"Id":   id,
		"Name": name,
	}, &result)
}

func (c *SeedanceAssetClient) DeleteAsset(id string) error {
	return c.post("DeleteAsset", map[string]string{"Id": id}, nil)
}
