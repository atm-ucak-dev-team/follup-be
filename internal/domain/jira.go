package domain

// JiraTicket represents a Jira ticket
type JiraTicket struct {
	ID      string `json:"id"`
	Key     string `json:"key"`
	Summary string `json:"summary"`
	Status  string `json:"status"`
}

// JiraIssue represents a Jira issue with stakeholder information
type JiraIssue struct {
	ID           string   `json:"id"`
	Key          string   `json:"key"`
	Summary      string   `json:"summary"`
	Status       string   `json:"status"`
	Stakeholders []string `json:"stakeholders"`
}
