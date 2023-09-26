package azure

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/appservice/armappservice/v2"
	"github.com/aws/smithy-go/time"
)

type AzureStaticCredentials struct {
	TenantId          string
	ClientId          string
	ClientSecret      string
	SubscriptionId    string
	ResourceGroupName string
}

type AzureClient struct {
	Credentials       AzureStaticCredentials
	AppServiceFactory *armappservice.ClientFactory
}

// WebAppData represents the harvested Azure Web App data
type WebAppData struct {
	WebAppName string            `json:"webAppName"`
	Digests    map[string]string `json:"digests"`
	StartedAt  int64             `json:"startedAt"`
}

// AzureWebAppsRequest represents the PUT request body to be sent to Kosli from CLI
type AzureWebAppsRequest struct {
	Artifacts []*WebAppData `json:"artifacts"`
}

func (staticCreds *AzureStaticCredentials) GetWebAppsData() ([]*WebAppData, error) {
	webAppsData := []*WebAppData{}
	azureClient, err := staticCreds.NewAzureClient()
	if err != nil {
		return nil, err
	}
	webAppInfo, err := azureClient.GetWebAppsInfo()
	if err != nil {
		return nil, err
	}
	for _, webapp := range webAppInfo {
		if strings.ToLower(*webapp.Properties.State) != "running" {
			continue
		}

		// get image name from "DOCKER|tookyregistry.azurecr.io/tookyregistry/tooky/sha256:cb29a6"
		linuxFxVersion := strings.Split(*webapp.Properties.SiteConfig.LinuxFxVersion, "|")
		imageName := linuxFxVersion[1]

		var fingerprint string
		var startedAt int64
		if linuxFxVersion[0] == "DOCKER" {
			logs, err := azureClient.GetDockerLogsForWebApp(*webapp.Name)
			if err != nil {
				return nil, err
			}
			fingerprint, startedAt, err = exractImageFingerprintAndStartedTimestampFromLogs(logs, *webapp.Name)
			if err != nil {
				return nil, err
			}
			if fingerprint == "" || startedAt == 0 {
				continue
			}
		}

		data := &WebAppData{*webapp.Name, map[string]string{imageName: fingerprint}, startedAt}
		webAppsData = append(webAppsData, data)
	}
	return webAppsData, nil
}

func (staticCreds *AzureStaticCredentials) NewAzureClient() (*AzureClient, error) {
	credentials, err := azidentity.NewClientSecretCredential(staticCreds.TenantId, staticCreds.ClientId, staticCreds.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	// Docs: https://github.com/Azure/azure-sdk-for-go/blob/main/sdk/resourcemanager/appservice/armappservice/README.md
	appserviceFactory, err := armappservice.NewClientFactory(staticCreds.SubscriptionId, credentials, nil)
	if err != nil {
		return nil, err
	}

	return &AzureClient{
		Credentials:       *staticCreds,
		AppServiceFactory: appserviceFactory,
	}, nil
}

func (azureClient *AzureClient) GetWebAppsInfo() ([]*armappservice.Site, error) {
	webAppsClient := azureClient.AppServiceFactory.NewWebAppsClient()

	ctx := context.Background()
	webappsPager := webAppsClient.NewListByResourceGroupPager(azureClient.Credentials.ResourceGroupName, nil)

	var webAppsInfo []*armappservice.Site
	for webappsPager.More() {
		response, err := webappsPager.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		webAppsInfo = append(webAppsInfo, response.Value...)
	}
	return webAppsInfo, nil
}

func (azureClient *AzureClient) GetDockerLogsForWebApp(appServiceName string) (logs []byte, error error) {
	webAppsClient := azureClient.AppServiceFactory.NewWebAppsClient()

	ctx := context.Background()

	response, err := webAppsClient.GetWebSiteContainerLogs(ctx, azureClient.Credentials.ResourceGroupName, appServiceName, nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func exractImageFingerprintAndStartedTimestampFromLogs(logs []byte, webAppName string) (fingerprint string, startedAt int64, error error) {
	logsReader := bytes.NewReader(logs)
	scanner := bufio.NewScanner(logsReader)

	searchedDigestByteArray := []byte("Digest: sha256:")
	containerStartedAtByteArray := []byte(fmt.Sprintf("for site %s initialized successfully and is ready to serve requests.", webAppName))

	var lastDigestLine []byte
	var lastStartedAtLine []byte
	for scanner.Scan() {
		line := scanner.Bytes()
		if bytes.Contains(line, searchedDigestByteArray) {
			lastDigestLine = line
		}
		if bytes.Contains(line, containerStartedAtByteArray) {
			lastStartedAtLine = line
		}
	}

	lengthOfTimestamp := 24 // example 2023-09-25T12:21:09.927Z
	var digestLoggedAt string
	if lastDigestLine != nil {
		lastDigestLineString := string(lastDigestLine)
		fingerprintStartIndex := len(lastDigestLineString) - 64
		fingerprint = lastDigestLineString[fingerprintStartIndex:]
		digestLoggedAt = lastDigestLineString[:lengthOfTimestamp]
	}

	var startedAtLoggedAt string
	if lastStartedAtLine != nil {
		startedAtLoggedAt = string(lastStartedAtLine)[:lengthOfTimestamp]
	}

	if digestLoggedAt != "" && startedAtLoggedAt != "" {
		digestLogTime, err := time.ParseDateTime(digestLoggedAt)
		if err != nil {
			return "", 0, err
		}
		startedAtLogTime, err := time.ParseDateTime(startedAtLoggedAt)
		if err != nil {
			return "", 0, err
		}

		// startedAtLoggedAt must be greater than digestLoggedAt,
		// because image pulled and build before it starts serving requests.
		// If startedAtLoggedAt is less than digestLoggedAt, then the container is not running.
		if startedAtLogTime.Before(digestLogTime) {
			return "", 0, nil
		}
		startedAt = startedAtLogTime.Unix()
	}

	return fingerprint, startedAt, nil
}
