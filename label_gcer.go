package main

import (
	"log"

	tracker "github.com/xoebus/go-tracker"
)

type LabelGCer struct {
	ProjectClient tracker.ProjectClient
}

func (gcer LabelGCer) GC() {
	labels, err := gcer.ProjectClient.Labels()
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
