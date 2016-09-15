package main

import (
	"log"

	tracker "github.com/xoebus/go-tracker"
)

type LabelGCer struct {
	ProjectClient tracker.ProjectClient
}

func (gcer LabelGCer) GC() {
	labels, err := gcer.allLabels()
	if err != nil {
		log.Println("failed to fetch errors:", err)
		return
	}

	for _, label := range labels {
		if label.Counts.NumberOfStoriesByState.Total() > 0 {
			continue
		}

		log.Println("deleting label:", label.Name)
		log.Println("but not really")
		// err := gcer.ProjectClient.DeleteLabel(label.ID)
		// if err != nil {
		// 	log.Println("failed to delete:", err)
		// }
	}
}

func (gcer LabelGCer) allLabels() ([]tracker.Label, error) {
	query := tracker.LabelsQuery{}

	allLabels := []tracker.Label{}

	for {
		labels, _, err := gcer.ProjectClient.Labels(query)
		if err != nil {
			return nil, err
		}

		if len(labels) == 0 {
			break
		}

		allLabels = append(allLabels, labels...)

		query.Offset = len(allLabels)
	}

	return allLabels, nil
}
