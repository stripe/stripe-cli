package cmd

// overview_alerts.go fetches integration health alerts for `stripe overview`
// through the GraphQL gateway's ListIntegHlthAlerts query, and each alert's
// doctor doc through RetrieveDoctorRule.

import (
	"context"
	"slices"

	genqlient "github.com/Khan/genqlient/graphql"
	log "github.com/sirupsen/logrus"

	"github.com/stripe/stripe-cli/pkg/graphql"
)

var integHlthAlertStatuses = []string{"open", "resolved"}

// integHlthAlertPredicates limits `stripe overview` to alerts for these
// predicates during the initial rollout.
var integHlthAlertPredicates = []string{"LEGACY_PAYMENT_METHODS"}

type integHlthAlertsResponse struct {
	Alerts []integHlthAlert `json:"alerts"`
}

type integHlthAlert struct {
	HealthInsightID       string                 `json:"health_insight_id"`
	PredicateName         string                 `json:"predicate_name"`
	CompartmentID         string                 `json:"compartment_id"`
	RecentIntegrationData []string               `json:"recent_integration_data,omitempty"`
	CreatedAt             string                 `json:"created_at"`
	LastUpdated           string                 `json:"last_updated"`
	HealthInsight         *integHlthAlertInsight `json:"health_insight,omitempty"`
	Status                string                 `json:"status"`
	DoctorDoc             *string                `json:"doctor_doc,omitempty"`
	DoctorTopic           *string                `json:"doctor_topic,omitempty"`
}

type integHlthAlertInsight struct {
	Summary           *string                     `json:"summary,omitempty"`
	Severity          *string                     `json:"severity,omitempty"`
	WhyIsItImportant  *string                     `json:"why_is_it_important,omitempty"`
	RecommendedAction *integHlthRecommendedAction `json:"recommended_action,omitempty"`
}

type integHlthRecommendedAction struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func fetchIntegHlthAlerts(ctx context.Context, client genqlient.Client, limit int) (*integHlthAlertsResponse, error) {
	alerts := make([]integHlthAlert, 0)
	page := ""
	for {
		resp, err := graphql.ListIntegHlthAlerts(ctx, client, integHlthAlertStatuses, 0, page, "", "")
		if err != nil {
			return nil, overviewGraphQLError(err)
		}

		list := resp.V2ListIntegHlthAlerts
		for _, a := range list.Alerts {
			if !slices.Contains(integHlthAlertPredicates, a.Predicate_name) {
				continue
			}
			alerts = append(alerts, integHlthAlertFromGraphQL(a))
			if limit > 0 && len(alerts) == limit {
				return &integHlthAlertsResponse{Alerts: alerts}, nil
			}
		}

		if list.Next_page == "" || list.Next_page == page {
			return &integHlthAlertsResponse{Alerts: alerts}, nil
		}
		page = list.Next_page
	}
}

func integHlthAlertFromGraphQL(a graphql.ListIntegHlthAlertsV2ListIntegHlthAlertsV2IntegrationHealthListIntegHlthAlertsResponseAlertsV2IntegrationHealthAlertDetail) integHlthAlert {
	alert := integHlthAlert{
		HealthInsightID:       a.Health_insight_id,
		PredicateName:         a.Predicate_name,
		CompartmentID:         a.Compartment_id,
		RecentIntegrationData: a.Recent_integration_data,
		CreatedAt:             a.Created_at,
		LastUpdated:           a.Last_updated,
		Status:                a.Status,
		DoctorTopic:           nonEmpty(a.Doctor_topic),
	}

	hi := a.Health_insight
	insight := integHlthAlertInsight{
		Summary:          nonEmpty(hi.Summary),
		Severity:         nonEmpty(hi.Severity),
		WhyIsItImportant: nonEmpty(hi.Why_is_it_important),
	}
	if ra := hi.Recommended_action; ra.Title != "" || ra.Description != "" {
		insight.RecommendedAction = &integHlthRecommendedAction{Title: ra.Title, Description: ra.Description}
	}
	if insight != (integHlthAlertInsight{}) {
		alert.HealthInsight = &insight
	}
	return alert
}

// addIntegHlthAlertDoctorDocs sets each alert's DoctorDoc to the first doc of
// the doctor rule for its doctor topic, looking up each topic once.
func addIntegHlthAlertDoctorDocs(ctx context.Context, client genqlient.Client, alerts []integHlthAlert) {
	docs := make(map[string]*string)
	for i, a := range alerts {
		if a.DoctorTopic == nil {
			continue
		}
		topic := *a.DoctorTopic
		doc, ok := docs[topic]
		if !ok {
			doc = fetchDoctorRuleDoc(ctx, client, topic)
			docs[topic] = doc
		}
		alerts[i].DoctorDoc = doc
	}
}

func fetchDoctorRuleDoc(ctx context.Context, client genqlient.Client, topic string) *string {
	resp, err := graphql.RetrieveDoctorRule(ctx, client, topic)
	if err != nil {
		log.Debugf("Skipping doctor doc for topic %q: %v", topic, err)
		return nil
	}
	ruleDocs := resp.V2RetrieveDoctorRule.Doctor_rule.Docs
	if len(ruleDocs) == 0 {
		return nil
	}
	return nonEmpty(ruleDocs[0])
}
