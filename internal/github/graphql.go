package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GraphQLClient struct {
	endpoint   string
	httpClient *http.Client
	authToken  string
}

type GraphQLQuery struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type GraphQLResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

func NewGraphQLClient(endpoint, authToken string) *GraphQLClient {
	return &GraphQLClient{
		endpoint:   endpoint,
		authToken:  authToken,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *GraphQLClient) Query(ctx context.Context, query string, variables map[string]string, result interface{}) error {
	vars := make(map[string]interface{})
	for k, v := range variables {
		vars[k] = v
	}

	payload := GraphQLQuery{
		Query:     query,
		Variables: vars,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	dataBytes, err := json.Marshal(gqlResp.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, result); err != nil {
		return fmt.Errorf("failed to unmarshal into result: %w", err)
	}

	return nil
}

type PRDetails struct {
	Number       int        `json:"number"`
	Title        string     `json:"title"`
	Body         string     `json:"body"`
	State        string     `json:"state"`
	Author       GitHubUser `json:"author"`
	Additions    int        `json:"additions"`
	Deletions    int        `json:"deletions"`
	ChangedFiles int        `json:"changedFiles"`
	Commits      PRCommits  `json:"commits"`
	Reviews      PRReviews  `json:"reviews"`
	Labels       PRLabels   `json:"labels"`
}

type PRCommits struct {
	Nodes []Commit `json:"nodes"`
}

type PRReviews struct {
	Nodes []Review `json:"nodes"`
}

type PRLabels struct {
	Nodes []Label `json:"nodes"`
}

type Review struct {
	State  string     `json:"state"`
	Author GitHubUser `json:"author"`
}

type Commit struct {
	SHA     string       `json:"oid"`
	Message string       `json:"message"`
	Author  GitHubAuthor `json:"author"`
}

type IssueDetails struct {
	Number    int            `json:"number"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	State     string         `json:"state"`
	Author    GitHubUser     `json:"author"`
	Labels    IssueLabels    `json:"labels"`
	Assignees IssueAssignees `json:"assignees"`
	Comments  IssueComments  `json:"comments"`
}

type IssueLabels struct {
	Nodes []Label `json:"nodes"`
}

type IssueAssignees struct {
	Nodes []GitHubUser `json:"nodes"`
}

type IssueComments struct {
	Nodes []GQLIssueComment `json:"nodes"`
}

type GQLIssueComment struct {
	Body   string     `json:"body"`
	Author GitHubUser `json:"author"`
}

type SearchResult struct {
	Type     string `json:"__typename"`
	Number   int    `json:"number"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	State    string `json:"state"`
	MergedAt string `json:"mergedAt"`
}

type RepositoryStats struct {
	OpenIssues         IssueCount `json:"openIssues"`
	ClosedIssues       IssueCount `json:"closedIssues"`
	OpenPullRequests   PRCount    `json:"openPullRequests"`
	ClosedPullRequests PRCount    `json:"closedPullRequests"`
	MergedPullRequests PRCount    `json:"mergedPullRequests"`
}

type IssueCount struct {
	TotalCount int `json:"totalCount"`
}

type PRCount struct {
	TotalCount int `json:"totalCount"`
}

type Check struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

type TimelineEvent struct {
	Type  string      `json:"__typename"`
	Body  string      `json:"body,omitempty"`
	Label *Label      `json:"label,omitempty"`
	Actor *GitHubUser `json:"actor,omitempty"`
}

type FileContent struct {
	Path string `json:"path"`
	Text string `json:"text"`
}

func (c *GraphQLClient) GetPRDetails(ctx context.Context, owner, repo string, prNumber int) (*PRDetails, error) {
	query := `query($owner: String!, $repo: String!, $prNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $prNumber) {
				number
				title
				body
				state
				author { login createdAt }
				additions
				deletions
				changedFiles
				commits(first: 100) {
					nodes { oid message author { name date } }
				}
				reviews(first: 100) {
					nodes { state author { login } }
				}
				labels(first: 100) {
					nodes { name }
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":    owner,
		"repo":     repo,
		"prNumber": fmt.Sprintf("%d", prNumber),
	}

	var response struct {
		Repository struct {
			PullRequest PRDetails `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return &response.Repository.PullRequest, nil
}

func (c *GraphQLClient) GetIssueDetails(ctx context.Context, owner, repo string, issueNumber int) (*IssueDetails, error) {
	query := `query($owner: String!, $repo: String!, $issueNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			issue(number: $issueNumber) {
				number
				title
				body
				state
				author { login }
				labels(first: 100) {
					nodes { name }
				}
				assignees(first: 100) {
					nodes { login }
				}
				comments(first: 100) {
					nodes { body author { login } }
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":       owner,
		"repo":        repo,
		"issueNumber": fmt.Sprintf("%d", issueNumber),
	}

	var response struct {
		Repository struct {
			Issue IssueDetails `json:"issue"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return &response.Repository.Issue, nil
}

func (c *GraphQLClient) SearchIssues(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	gqlQuery := `query($query: String!, $limit: Int!) {
		search(query: $query, type: ISSUE, first: $limit) {
			nodes {
				__typename
				... on Issue {
					number
					title
					body
					state
				}
				... on PullRequest {
					number
					title
					body
					state
					mergedAt
				}
			}
			issueCount
		}
	}`

	variables := map[string]string{
		"query": query,
		"limit": fmt.Sprintf("%d", limit),
	}

	var response struct {
		Search struct {
			Nodes      []SearchResult `json:"nodes"`
			IssueCount int            `json:"issueCount"`
		} `json:"search"`
	}

	if err := c.Query(ctx, gqlQuery, variables, &response); err != nil {
		return nil, err
	}

	return response.Search.Nodes, nil
}

func (c *GraphQLClient) GetRepositoryStats(ctx context.Context, owner, repo string) (*RepositoryStats, error) {
	query := `query($owner: String!, $repo: String!) {
		repository(owner: $owner, name: $repo) {
			name
			owner { login }
			openIssues: issues(states: OPEN) { totalCount }
			closedIssues: issues(states: CLOSED) { totalCount }
			openPullRequests: pullRequests(states: OPEN) { totalCount }
			closedPullRequests: pullRequests(states: CLOSED) { totalCount }
			mergedPullRequests: pullRequests(states: MERGED) { totalCount }
		}
	}`

	variables := map[string]string{
		"owner": owner,
		"repo":  repo,
	}

	var response struct {
		Repository RepositoryStats `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return &response.Repository, nil
}

func (c *GraphQLClient) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	query := `query($owner: String!, $repo: String!, $path: String!, $ref: String!) {
		repository(owner: $owner, name: $repo) {
			object(expression: $ref) {
				__typename
				... on Blob {
					text
				}
			}
		}
	}`

	variables := map[string]string{
		"owner": owner,
		"repo":  repo,
		"path":  path,
		"ref":   ref + ":" + path,
	}

	var response struct {
		Repository struct {
			Object struct {
				Typename string `json:"__typename"`
				Text     string `json:"text"`
			} `json:"object"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return "", err
	}

	return response.Repository.Object.Text, nil
}

func (c *GraphQLClient) GetCodeReviewComments(ctx context.Context, owner, repo string, prNumber int) ([]ReviewCommentResponse, error) {
	query := `query($owner: String!, $repo: String!, $prNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $prNumber) {
				reviews(first: 100) {
					nodes {
						comments(first: 100) {
							nodes {
								path
								line
								body
								state
							}
						}
					}
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":    owner,
		"repo":     repo,
		"prNumber": fmt.Sprintf("%d", prNumber),
	}

	var response struct {
		Repository struct {
			PullRequest struct {
				Reviews struct {
					Nodes []struct {
						Comments struct {
							Nodes []ReviewCommentResponse `json:"nodes"`
						} `json:"comments"`
					} `json:"nodes"`
				} `json:"reviews"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	var allComments []ReviewCommentResponse
	for _, review := range response.Repository.PullRequest.Reviews.Nodes {
		allComments = append(allComments, review.Comments.Nodes...)
	}

	return allComments, nil
}

func (c *GraphQLClient) CheckRateLimit(ctx context.Context) (limit, remaining int, resetAt time.Time, err error) {
	query := `query {
		rateLimit {
			limit
			remaining
			resetAt
		}
	}`

	var response struct {
		RateLimit struct {
			Limit     int    `json:"limit"`
			Remaining int    `json:"remaining"`
			ResetAt   string `json:"resetAt"`
		} `json:"rateLimit"`
	}

	if err := c.Query(ctx, query, nil, &response); err != nil {
		return 0, 0, time.Time{}, err
	}

	resetAt, err = time.Parse(time.RFC3339, response.RateLimit.ResetAt)
	if err != nil {
		return 0, 0, time.Time{}, err
	}

	return response.RateLimit.Limit, response.RateLimit.Remaining, resetAt, nil
}

func (c *GraphQLClient) GetRecentCommits(ctx context.Context, owner, repo string, limit int) ([]Commit, error) {
	query := `query($owner: String!, $repo: String!, $limit: Int!) {
		repository(owner: $owner, name: $repo) {
			defaultBranchRef {
				target {
					... on Commit {
						history(first: $limit) {
							nodes {
								oid
								message
								author {
									name
									email
									date
								}
							}
						}
					}
				}
			}
		}
	}`

	variables := map[string]string{
		"owner": owner,
		"repo":  repo,
		"limit": fmt.Sprintf("%d", limit),
	}

	var response struct {
		Repository struct {
			DefaultBranchRef struct {
				Target struct {
					History struct {
						Nodes []Commit `json:"nodes"`
					} `json:"history"`
				} `json:"target"`
			} `json:"defaultBranchRef"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response.Repository.DefaultBranchRef.Target.History.Nodes, nil
}

func (c *GraphQLClient) GetAuthorAssociation(ctx context.Context, owner, repo string) (string, error) {
	query := `query($owner: String!, $repo: String!) {
		repository(owner: $owner, name: $repo) {
			viewerPermission
		}
	}`

	variables := map[string]string{
		"owner": owner,
		"repo":  repo,
	}

	var response struct {
		Repository struct {
			ViewerPermission string `json:"viewerPermission"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return "", err
	}

	return response.Repository.ViewerPermission, nil
}

func (c *GraphQLClient) GetPullRequestChecks(ctx context.Context, owner, repo string, prNumber int) ([]Check, error) {
	query := `query($owner: String!, $repo: String!, $prNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $prNumber) {
				commits(first: 100) {
					nodes {
						commit {
							statusCheckRollup {
								state
								checks(first: 100) {
									nodes { name state }
								}
							}
						}
					}
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":    owner,
		"repo":     repo,
		"prNumber": fmt.Sprintf("%d", prNumber),
	}

	var response struct {
		Repository struct {
			PullRequest struct {
				Commits struct {
					Nodes []struct {
						Commit struct {
							StatusCheckRollup struct {
								State  string `json:"state"`
								Checks struct {
									Nodes []Check `json:"nodes"`
								} `json:"checks"`
							} `json:"statusCheckRollup"`
						} `json:"commit"`
					} `json:"nodes"`
				} `json:"commits"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	var allChecks []Check
	for _, commit := range response.Repository.PullRequest.Commits.Nodes {
		if commit.Commit.StatusCheckRollup.Checks.Nodes != nil {
			allChecks = append(allChecks, commit.Commit.StatusCheckRollup.Checks.Nodes...)
		}
	}

	return allChecks, nil
}

func (c *GraphQLClient) GetIssueTimeline(ctx context.Context, owner, repo string, issueNumber int) ([]TimelineEvent, error) {
	query := `query($owner: String!, $repo: String!, $issueNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			issue(number: $issueNumber) {
				timeline(first: 100) {
					nodes {
						__typename
						... on IssueComment { body }
						... on LabeledEvent { label { name } }
						... on ClosedEvent { actor { login } }
					}
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":       owner,
		"repo":        repo,
		"issueNumber": fmt.Sprintf("%d", issueNumber),
	}

	var response struct {
		Repository struct {
			Issue struct {
				Timeline struct {
					Nodes []TimelineEvent `json:"nodes"`
				} `json:"timeline"`
			} `json:"issue"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response.Repository.Issue.Timeline.Nodes, nil
}

func (c *GraphQLClient) GetFirstPRCommit(ctx context.Context, owner, repo string, prNumber int) (*Commit, error) {
	query := `query($owner: String!, $repo: String!, $prNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $prNumber) {
				commits(first: 1) {
					nodes {
						commit {
							oid
							message
							author { name date }
						}
					}
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":    owner,
		"repo":     repo,
		"prNumber": fmt.Sprintf("%d", prNumber),
	}

	var response struct {
		Repository struct {
			PullRequest struct {
				Commits struct {
					Nodes []struct {
						Commit Commit `json:"commit"`
					} `json:"nodes"`
				} `json:"commits"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	if len(response.Repository.PullRequest.Commits.Nodes) == 0 {
		return nil, fmt.Errorf("no commits found")
	}

	return &response.Repository.PullRequest.Commits.Nodes[0].Commit, nil
}

func (c *GraphQLClient) ExtractRepositoryLabels(ctx context.Context, owner, repo string) ([]Label, error) {
	query := `query($owner: String!, $repo: String!) {
		repository(owner: $owner, name: $repo) {
			labels(first: 100) {
				nodes { name }
			}
		}
	}`

	variables := map[string]string{
		"owner": owner,
		"repo":  repo,
	}

	var response struct {
		Repository struct {
			Labels struct {
				Nodes []Label `json:"nodes"`
			} `json:"labels"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response.Repository.Labels.Nodes, nil
}

func (c *GraphQLClient) CheckDuplicateIssues(ctx context.Context, owner, repo, title string) ([]SearchResult, error) {
	query := `query($owner: String!, $repo: String!, $title: String!) {
		search(query: $query, type: ISSUE, first: 10) {
			nodes {
				__typename
				... on Issue {
					number
					title
					body
					state
				}
			}
			issueCount
		}
	}`

	fullQuery := fmt.Sprintf("repo:%s/%s is:issue %s", owner, repo, title)

	variables := map[string]string{
		"owner": owner,
		"repo":  repo,
		"title": title,
		"query": fullQuery,
	}

	var response struct {
		Search struct {
			Nodes      []SearchResult `json:"nodes"`
			IssueCount int            `json:"issueCount"`
		} `json:"search"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response.Search.Nodes, nil
}

func (c *GraphQLClient) GetUserPastPRs(ctx context.Context, owner, repo, author string) ([]SearchResult, error) {
	query := `query($query: String!) {
		search(query: $query, type: ISSUE, first: 100) {
			nodes {
				__typename
				... on PullRequest {
					number
					title
					mergedAt
				}
			}
			issueCount
		}
	}`

	fullQuery := fmt.Sprintf("repo:%s/%s is:pr author:%s is:merged", owner, repo, author)

	variables := map[string]string{
		"query": fullQuery,
	}

	var response struct {
		Search struct {
			Nodes      []SearchResult `json:"nodes"`
			IssueCount int            `json:"issueCount"`
		} `json:"search"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	return response.Search.Nodes, nil
}

func (c *GraphQLClient) GetPRFilesChanged(ctx context.Context, owner, repo string, prNumber int) ([]PRFile, error) {
	query := `query($owner: String!, $repo: String!, $prNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $prNumber) {
				files(first: 100) {
					nodes {
						path
						additions
						deletions
					}
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":    owner,
		"repo":     repo,
		"prNumber": fmt.Sprintf("%d", prNumber),
	}

	var response struct {
		Repository struct {
			PullRequest struct {
				Files struct {
					Nodes []struct {
						Path      string `json:"path"`
						Additions int    `json:"additions"`
						Deletions int    `json:"deletions"`
					} `json:"nodes"`
				} `json:"files"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	result := make([]PRFile, len(response.Repository.PullRequest.Files.Nodes))
	for i, f := range response.Repository.PullRequest.Files.Nodes {
		result[i] = PRFile{Filename: f.Path, Additions: f.Additions, Deletions: f.Deletions}
	}
	return result, nil
}

func (c *GraphQLClient) GetDiffHunks(ctx context.Context, owner, repo string, prNumber int, path string) ([]string, error) {
	query := `query($owner: String!, $repo: String!, $prNumber: Int!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: $prNumber) {
				files(first: 100) {
					nodes {
						path
						patch
					}
				}
			}
		}
	}`

	variables := map[string]string{
		"owner":    owner,
		"repo":     repo,
		"prNumber": fmt.Sprintf("%d", prNumber),
	}

	var response struct {
		Repository struct {
			PullRequest struct {
				Files struct {
					Nodes []struct {
						Path  string `json:"path"`
						Patch string `json:"patch"`
					} `json:"nodes"`
				} `json:"files"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	if err := c.Query(ctx, query, variables, &response); err != nil {
		return nil, err
	}

	for _, file := range response.Repository.PullRequest.Files.Nodes {
		if file.Path == path {
			if file.Patch == "" {
				return nil, nil
			}
			return splitPatchIntoHunks(file.Patch), nil
		}
	}

	return nil, nil
}

func splitPatchIntoHunks(patch string) []string {
	var hunks []string
	var currentHunk string

	lines := splitLines(patch)
	for _, line := range lines {
		if len(line) > 0 && line[0] == '@' {
			if currentHunk != "" {
				hunks = append(hunks, currentHunk)
			}
			currentHunk = line + "\n"
		} else {
			currentHunk += line + "\n"
		}
	}

	if currentHunk != "" {
		hunks = append(hunks, currentHunk)
	}

	return hunks
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
