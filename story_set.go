package main

import (
	"log"
	"time"

	"github.com/xoebus/go-tracker"
)

type StorySet []tracker.Story

func (set StorySet) AllAccepted() bool {
	allAccepted := true
	for _, story := range set {
		if story.State != "accepted" {
			allAccepted = false
			break
		}
	}

	return allAccepted
}

func (set StorySet) LastAccepted() time.Time {
	lastAccepted := time.Unix(0, 0)

	for _, story := range set {
		if story.AcceptedAt.After(lastAccepted) {
			lastAccepted = *story.AcceptedAt
		}
	}

	return lastAccepted
}

func (set StorySet) IssueLabels() []string {
	var labels []string

	var hasBugs bool
	var hasFeatures bool
	for _, story := range set {
		if story.Type == "feature" {
			hasFeatures = true
		} else if story.Type == "bug" {
			hasBugs = true
		}
	}

	if hasFeatures {
		labels = append(labels, IssueLabelEnhancement)
	} else if hasBugs {
		labels = append(labels, IssueLabelBug)
	}

	if set.AllAccepted() {
		// everything is accepted; only set labels for types of stories, not status
		return labels
	}

	allUnscheduled := true
	for _, story := range set {
		switch story.State {
		case "accepted":
			// ignore accepted stories; if some are accepted but the rest are
			// unscheduled, it's still unscheduled

		case "unscheduled":
			// only mark if all are unscheduled

		case "started", "finished", "delivered", "rejected":
			// a story is in-progress; report as in-flight
			labels = append(labels, IssueLabelInFlight)
			return labels

		case "unstarted", "planned":
			// something is scheduled
			allUnscheduled = false

		default:
			log.Fatalln("unknown story state:", story.State)
		}
	}

	if allUnscheduled {
		labels = append(labels, IssueLabelUnscheduled)
	} else {
		labels = append(labels, IssueLabelScheduled)
	}

	return labels
}
