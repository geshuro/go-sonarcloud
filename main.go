package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"github.com/reinoudk/go-sonarcloud/sonarcloud"
	"github.com/reinoudk/go-sonarcloud/sonarcloud/project_branches"
	"github.com/reinoudk/go-sonarcloud/sonarcloud/projects"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

func main() {
	org, ok := os.LookupEnv("SONARCLOUD_ORG")
	if !ok {
		log.Fatalf("missing SONARCLOUD_ORG environment variable")
	}

	token, ok := os.LookupEnv("SONARCLOUD_TOKEN")
	if !ok {
		log.Fatalf("mising SONARCLOUD_TOKEN environment variable")
	}

	client := sonarcloud.NewClient(org, token, nil)
	req := projects.SearchRequest{}

	res, err := client.Projects.SearchAll(req)
	if err != nil {
		log.Fatalf("could not search projects: %+v", err)
	}

	records := [][]string{
		{"Project", "Branch", "Main", "QualityGateStatus"},
	}

	//fmt.Printf("%+v\n", res.Components[0].Key)
	for _, c := range res.Components[:1] {
		res, err := client.ProjectBranches.List(project_branches.ListRequest{
			Project: c.Key,
		})
		if err != nil {
			log.Fatalf("could not search projects: %+v", err)
		}
		//fmt.Printf("%+v\n", c.Key)
		for _, b := range res.Branches {
			if b.IsMain {
				records = append(records, []string{c.Key, b.Name, fmt.Sprintf("%t", b.IsMain), b.Status.QualityGateStatus})
				//fmt.Printf("Project: %s, Branch: %s, Main: %s, QualityGateStatus: %s\n", c.Key, b.Name, b.IsMain, b.Status.QualityGateStatus)
			}
		}
	}
	pageID, ok := os.LookupEnv("PAGEID")
	if !ok {
		log.Fatalf("missing PAGEID environment variable")
	}
	// Generate a file name with the current date and time
	nameFile := generateFileName("sonarcloud", "csv")
	// Generate and upload the CSV file
	err = generateAndUploadCSV(records, nameFile, pageID)
	if err != nil {
		log.Fatalf("failed to generate and upload CSV: %v", err)
	}
	log.Printf("CSV file generated and uploaded successfully: %s", nameFile)

}

// generateFileName generates a file name with the current date and time
func generateFileName(baseName, extension string) string {
	currentTime := time.Now().Format("2006-01-02_15-04-05") // Format: YYYY-MM-DD_HH-MM-SS
	return fmt.Sprintf("%s_%s.%s", baseName, currentTime, extension)
}

// generateAndUploadCSV generates a CSV file and uploads it to a Confluence page
func generateAndUploadCSV(records [][]string, fileName, pageID string) error {
	// Generate the CSV file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.WriteAll(records); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}

	// Upload the CSV file to Confluence
	return uploadToConfluence(fileName, pageID)
}

func uploadToConfluence(fileName, pageID string) error {
	confluenceURL := os.Getenv("CONFLUENCE_URL")
	apiKey := os.Getenv("CONFLUENCE_API_KEY")
	username := os.Getenv("CONFLUENCE_USERNAME") // Confluence requires a username for authentication

	if confluenceURL == "" || apiKey == "" || username == "" {
		return fmt.Errorf("missing Confluence configuration (CONFLUENCE_URL, CONFLUENCE_API_KEY, CONFLUENCE_USERNAME)")
	}

	file, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	_, err = io.Copy(part, file)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	err = writer.WriteField("minorEdit", "true")
	if err != nil {
		return fmt.Errorf("failed to write form field: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	url := fmt.Sprintf("%s/wiki/rest/api/content/%s/child/attachment", confluenceURL, pageID)
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+basicAuth(username, apiKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Atlassian-Token", "no-check")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload file, status: %s", resp.Status)
	}

	return nil
}

func basicAuth(username, apiKey string) string {
	return base64.StdEncoding.EncodeToString([]byte(username + ":" + apiKey))
}
