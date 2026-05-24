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

// JiraIssueResponse represents the enhanced Jira issue response
type JiraIssueResponse struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	URL         string `json:"url"`
	TicketTitle string `json:"ticketTitle"`
	Stakeholder string `json:"stakeholder"`
	Status      string `json:"status"`
	StatusColor string `json:"statusColor"`
}

// JiraIssueDetailResponse represents the detailed Jira issue response
type JiraIssueDetailResponse struct {
	ID            string `json:"id"`
	TicketNumber  string `json:"ticketNumber"`
	SelfLink      string `json:"selfLink"`
	TicketTitle   string `json:"ticketTitle"`
	Stakeholder   string `json:"stakeholder"`
	Status        string `json:"status"`
	StatusColor   string `json:"statusColor"`
	LastViewed    string `json:"lastViewed"`
	CreatorName   string `json:"creatorName"`
	CreatorEmail  string `json:"creatorEmail"`
	AssigneeName  string `json:"assigneeName"`
	AssigneeEmail string `json:"assigneeEmail"`
}
