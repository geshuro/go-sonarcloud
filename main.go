package main

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"github.com/reinoudk/go-sonarcloud/sonarcloud"
	"github.com/reinoudk/go-sonarcloud/sonarcloud/project_branches"
	"github.com/reinoudk/go-sonarcloud/sonarcloud/project_pull_requests"
	"github.com/reinoudk/go-sonarcloud/sonarcloud/projects"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type Config struct {
	sonarCloudOrg      string
	sonarCloudToken    string
	confluencePageId   string
	confluenceOrgUrl   string
	confluenceApiKey   string
	confluenceUsername string
}

// Define Record struct
type Record struct {
	Project           string
	Branch            string
	Contributors      string
	QualityGateStatus string
	Bugs              int
	Vulnerabilities   int
	CodeSmells        int
	AnalysisDate      string
	URL               string
}

func configFromEnv() (Config, error) {
	sonarCloudOrg, ok := os.LookupEnv("SONARCLOUD_ORG")
	if !ok {
		log.Fatalf("missing SONARCLOUD_ORG environment variable")
	}
	sonarCloudToken, ok := os.LookupEnv("SONARCLOUD_TOKEN")
	if !ok {
		log.Fatalf("mising SONARCLOUD_TOKEN environment variable")
	}
	confluencePageId, ok := os.LookupEnv("CONFLUENCE_PAGEID")
	if !ok {
		log.Fatalf("missing CONFLUENCE_PAGEID environment variable")
	}
	confluenceOrgUrl, ok := os.LookupEnv("CONFLUENCE_ORG_URL")
	if !ok {
		log.Fatalf("missing CONFLUENCE_ORG_URL environment variable")
	}
	confluenceApiKey, ok := os.LookupEnv("CONFLUENCE_API_KEY")
	if !ok {
		log.Fatalf("missing CONFLUENCE_API_KEY environment variable")
	}
	confluenceUsername, ok := os.LookupEnv("CONFLUENCE_USERNAME")
	if !ok {
		log.Fatalf("missing CONFLUENCE_USERNAME environment variable")
	}

	config := Config{
		sonarCloudOrg:      sonarCloudOrg,
		sonarCloudToken:    sonarCloudToken,
		confluencePageId:   confluencePageId,
		confluenceOrgUrl:   confluenceOrgUrl,
		confluenceApiKey:   confluenceApiKey,
		confluenceUsername: confluenceUsername,
	}
	return config, nil
}

func main() {
	config, err := configFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	client := sonarcloud.NewClient(config.sonarCloudOrg, config.sonarCloudToken, nil)
	req := projects.SearchRequest{}

	prj, err := client.Projects.SearchAll(req)
	if err != nil {
		log.Fatalf("Could not search projects: %+v", err)
	}

	// Use []Record instead of [][]string
	var records []Record
	// Add a header row separately for CSV generation
	header := []string{"Project", "Branch", "Contributors", "QualityGateStatus", "Bugs", "Vulnerabilities", "CodeSmells", "AnalysisDate", "URL"}
	// Add a header row to the CSV
	for _, c := range prj.Components[:5] {
		prjBr, err := client.ProjectBranches.List(project_branches.ListRequest{
			Project: c.Key,
		})
		if err != nil {
			log.Fatalf("Could not search projects branches: %+v", err)
		}
		for _, b := range prjBr.Branches {
			if b.IsMain {
				prjPr, err := client.ProjectPullRequests.List(project_pull_requests.ListRequest{
					Project: c.Key,
				})
				if err != nil {
					log.Fatalf("Could not search projects pullrequest: %+v", err)
				}
				contributorsName := ""
				urlPullRequest := ""
				if len(prjPr.PullRequests) > 0 {
					if len(prjPr.PullRequests[0].Contributors) > 0 {
						contributorsName = prjPr.PullRequests[0].Contributors[0].Name
					}
					if prjPr.PullRequests[0].Url != "" {
						urlPullRequest = prjPr.PullRequests[0].Url
					}
				}
				analysisDate := ""
				if b.AnalysisDate != "" && b.AnalysisDate != "null" {
					dt, err := time.Parse("2006-01-02T15:04:05-0700", b.AnalysisDate)
					if err != nil {
						log.Fatalf("invalid date format: %v", err)
					}
					analysisDate = dt.Format("02-01-2006")
				}
				// Append as Record struct
				records = append(records, Record{
					Project:           c.Key,
					Branch:            b.Name,
					Contributors:      contributorsName,
					QualityGateStatus: b.Status.QualityGateStatus,
					Bugs:              int(b.Status.Bugs),
					Vulnerabilities:   int(b.Status.Vulnerabilities),
					CodeSmells:        int(b.Status.CodeSmells),
					AnalysisDate:      analysisDate,
					URL:               urlPullRequest,
				})
			}
		}
	}
	// Generate a file name with the current date and time
	nameFile := generateFileName("sonarcloud", "csv")
	// Generate and upload the CSV file using the new records struct
	err = generateAndUploadCSVFromStruct(records, header, nameFile, config)
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

// generateAndUploadCSVFromStruct generates a CSV file from struct records and uploads it to a Confluence page
func generateAndUploadCSVFromStruct(records []Record, header []string, fileName string, config Config) error {
	// Generate the CSV file
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write the records
	for _, record := range records {
		row := []string{
			record.Project,
			record.Branch,
			record.Contributors,
			record.QualityGateStatus,
			fmt.Sprintf("%v", record.Bugs),
			fmt.Sprintf("%v", record.Vulnerabilities),
			fmt.Sprintf("%v", record.CodeSmells),
			record.AnalysisDate,
			record.URL,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	// Upload the CSV file to Confluence
	return uploadToConfluence(fileName, config)
}

func uploadToConfluence(fileName string, config Config) error {
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

	url := fmt.Sprintf("%s/wiki/rest/api/content/%s/child/attachment", config.confluenceOrgUrl, config.confluencePageId)
	req, err := http.NewRequest("POST", url, &requestBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+basicAuth(config.confluenceUsername, config.confluenceApiKey))
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
