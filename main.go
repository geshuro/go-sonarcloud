package main

import (
	"encoding/csv"
	"fmt"
	"github.com/reinoudk/go-sonarcloud/sonarcloud"
	"github.com/reinoudk/go-sonarcloud/sonarcloud/project_branches"
	"github.com/reinoudk/go-sonarcloud/sonarcloud/projects"
	"log"
	"os"
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
	for _, c := range res.Components {
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

	file, err := os.Create("sonarcloud.csv")
	defer file.Close()
	if err != nil {
		log.Fatalln("failed to open file", err)
	}

	w := csv.NewWriter(file)
	defer w.Flush()
	w.WriteAll(records) // calls Flush internally

	if err := w.Error(); err != nil {
		log.Fatalln("error writing csv:", err)
	}

}
