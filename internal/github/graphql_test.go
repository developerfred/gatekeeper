package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGraphQLClient_Query(t *testing.T) {
	var receivedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		if r.Method != "POST" {
			t.Errorf("method = %q, want %q", r.Method, "POST")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		var body GraphQLQuery
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Query == "" {
			t.Error("Query = empty, want non-empty")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"pullRequest": map[string]interface{}{
						"number": 123,
						"title":  "Test PR",
					},
				},
			},
		})
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	var result struct {
		Repository struct {
			PullRequest struct {
				Number int    `json:"number"`
				Title  string `json:"title"`
			} `json:"pullRequest"`
		} `json:"repository"`
	}

	err := client.Query(ctx, `query($owner: String!, $repo: String!) {
		repository(owner: $owner, name: $repo) {
			pullRequest(number: 123) {
				number
				title
			}
		}
	}`, map[string]string{"owner": "myorg", "repo": "myrepo"}, &result)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	if result.Repository.PullRequest.Number != 123 {
		t.Errorf("PullRequest.Number = %d, want %d", result.Repository.PullRequest.Number, 123)
	}
	if receivedReq == nil {
		t.Fatal("receivedReq = nil")
	}
}

func TestGraphQLClient_Query_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"errors": []map[string]string{
				{"message": "Field 'pullRequest' not found"},
			},
		})
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	var result map[string]interface{}
	err := client.Query(ctx, `query { invalid }`, nil, &result)
	if err == nil {
		t.Error("Query() error = nil, want error")
	}
}

func TestGraphQLClient_GetPRDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"pullRequest": {
						"number": 456,
						"title": "feat: amazing feature",
						"body": "This PR adds an amazing feature",
						"state": "OPEN",
						"author": {
							"login": "contributor",
							"createdAt": "2020-01-01T00:00:00Z"
						},
						"additions": 200,
						"deletions": 50,
						"changedFiles": 5,
						"commits": {
							"nodes": [
								{
									"oid": "abc123",
									"message": "feat: start",
									"author": {"name": "Dev", "date": "2024-01-01T00:00:00Z"}
								}
							]
						},
						"reviews": {
							"nodes": [
								{
									"state": "APPROVED",
									"author": {"login": "reviewer1"}
								}
							]
						},
						"labels": {
							"nodes": [
								{"name": "enhancement"}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	pr, err := client.GetPRDetails(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("GetPRDetails() error = %v", err)
	}
	if pr.Number != 456 {
		t.Errorf("PR.Number = %d, want %d", pr.Number, 456)
	}
	if pr.Title != "feat: amazing feature" {
		t.Errorf("PR.Title = %q, want %q", pr.Title, "feat: amazing feature")
	}
	if pr.Additions != 200 {
		t.Errorf("PR.Additions = %d, want %d", pr.Additions, 200)
	}
	if pr.Deletions != 50 {
		t.Errorf("PR.Deletions = %d, want %d", pr.Deletions, 50)
	}
	if pr.ChangedFiles != 5 {
		t.Errorf("PR.ChangedFiles = %d, want %d", pr.ChangedFiles, 5)
	}
	if pr.Author.Login != "contributor" {
		t.Errorf("PR.Author.Login = %q, want %q", pr.Author.Login, "contributor")
	}
	if len(pr.Commits.Nodes) != 1 {
		t.Errorf("len(PR.Commits.Nodes) = %d, want %d", len(pr.Commits.Nodes), 1)
	}
	if len(pr.Reviews.Nodes) != 1 {
		t.Errorf("len(PR.Reviews.Nodes) = %d, want %d", len(pr.Reviews.Nodes), 1)
	}
	if len(pr.Labels.Nodes) != 1 {
		t.Errorf("len(PR.Labels.Nodes) = %d, want %d", len(pr.Labels.Nodes), 1)
	}
}

func TestGraphQLClient_GetIssueDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"issue": {
						"number": 789,
						"title": "Bug: something broken",
						"body": "It doesn't work when...",
						"state": "OPEN",
						"author": {
							"login": "reporter"
						},
						"labels": {
							"nodes": [
								{"name": "bug"},
								{"name": "priority"}
							]
						},
						"assignees": {
							"nodes": [
								{"login": "maintainer"}
							]
						},
						"comments": {
							"nodes": [
								{
									"body": "Comment text",
									"author": {"login": "helper"}
								}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	issue, err := client.GetIssueDetails(ctx, "owner", "repo", 789)
	if err != nil {
		t.Fatalf("GetIssueDetails() error = %v", err)
	}
	if issue.Number != 789 {
		t.Errorf("Issue.Number = %d, want %d", issue.Number, 789)
	}
	if issue.Title != "Bug: something broken" {
		t.Errorf("Issue.Title = %q, want %q", issue.Title, "Bug: something broken")
	}
	if len(issue.Labels.Nodes) != 2 {
		t.Errorf("len(Issue.Labels.Nodes) = %d, want %d", len(issue.Labels.Nodes), 2)
	}
	if len(issue.Assignees.Nodes) != 1 {
		t.Errorf("len(Issue.Assignees.Nodes) = %d, want %d", len(issue.Assignees.Nodes), 1)
	}
	if len(issue.Comments.Nodes) != 1 {
		t.Errorf("len(Issue.Comments.Nodes) = %d, want %d", len(issue.Comments.Nodes), 1)
	}
}

func TestGraphQLClient_SearchIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body GraphQLQuery
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Query == "" {
			t.Error("Query = empty, want non-empty")
		}

		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"search": {
					"nodes": [
						{
							"__typename": "Issue",
							"number": 100,
							"title": "First issue",
							"state": "OPEN"
						},
						{
							"__typename": "Issue",
							"number": 101,
							"title": "Second issue",
							"state": "CLOSED"
						}
					],
					"issueCount": 2
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	results, err := client.SearchIssues(ctx, "repo:owner/repo is:issue is:open", 10)
	if err != nil {
		t.Fatalf("SearchIssues() error = %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want %d", len(results), 2)
	}
}

func TestGraphQLClient_GetRepositoryStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"name": "myrepo",
					"owner": {"login": "myorg"},
					"openIssues": {"totalCount": 42},
					"closedIssues": {"totalCount": 158},
					"openPullRequests": {"totalCount": 5},
					"closedPullRequests": {"totalCount": 95},
					"mergedPullRequests": {"totalCount": 300}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	stats, err := client.GetRepositoryStats(ctx, "myorg", "myrepo")
	if err != nil {
		t.Fatalf("GetRepositoryStats() error = %v", err)
	}
	if stats.OpenIssues.TotalCount != 42 {
		t.Errorf("stats.OpenIssues.TotalCount = %d, want %d", stats.OpenIssues.TotalCount, 42)
	}
	if stats.ClosedIssues.TotalCount != 158 {
		t.Errorf("stats.ClosedIssues.TotalCount = %d, want %d", stats.ClosedIssues.TotalCount, 158)
	}
	if stats.OpenPullRequests.TotalCount != 5 {
		t.Errorf("stats.OpenPullRequests.TotalCount = %d, want %d", stats.OpenPullRequests.TotalCount, 5)
	}
	if stats.MergedPullRequests.TotalCount != 300 {
		t.Errorf("stats.MergedPullRequests.TotalCount = %d, want %d", stats.MergedPullRequests.TotalCount, 300)
	}
}

func TestGraphQLClient_GetFileContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"object": {
						"__typename": "Blob",
						"text": "package main\n\nfunc main() {}\n"
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	content, err := client.GetFileContent(ctx, "owner", "repo", "src/main.go", "main")
	if err != nil {
		t.Fatalf("GetFileContent() error = %v", err)
	}
	expected := "package main\n\nfunc main() {}\n"
	if content != expected {
		t.Errorf("content = %q, want %q", content, expected)
	}
}

func TestGraphQLClient_GetCodeReviewComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"pullRequest": {
						"reviews": {
							"nodes": [
								{
									"comments": {
										"nodes": [
											{
												"path": "src/main.go",
												"line": 10,
												"body": "Consider using a constant here",
												"state": "PENDING"
											}
										]
									}
								}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	comments, err := client.GetCodeReviewComments(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("GetCodeReviewComments() error = %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("len(comments) = %d, want %d", len(comments), 1)
	}
	if comments[0].Path != "src/main.go" {
		t.Errorf("comments[0].Path = %q, want %q", comments[0].Path, "src/main.go")
	}
	if comments[0].Line != 10 {
		t.Errorf("comments[0].Line = %d, want %d", comments[0].Line, 10)
	}
}

func TestGraphQLClient_CheckRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"rateLimit": {
					"limit": 5000,
					"remaining": 4999,
					"resetAt": "2024-01-01T12:00:00Z"
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	limit, remaining, resetAt, err := client.CheckRateLimit(ctx)
	if err != nil {
		t.Fatalf("CheckRateLimit() error = %v", err)
	}
	if limit != 5000 {
		t.Errorf("limit = %d, want %d", limit, 5000)
	}
	if remaining != 4999 {
		t.Errorf("remaining = %d, want %d", remaining, 4999)
	}
	if resetAt.IsZero() {
		t.Error("resetAt is zero")
	}
}

func TestGraphQLClient_GetRecentCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"defaultBranchRef": {
						"target": {
							"history": {
								"nodes": [
									{
										"oid": "abc123",
										"message": "feat: add feature",
										"author": {
											"name": "Dev",
											"email": "dev@example.com",
											"date": "2024-01-01T10:00:00Z"
										}
									},
									{
										"oid": "def456",
										"message": "fix: bug",
										"author": {
											"name": "Dev",
											"email": "dev@example.com",
											"date": "2024-01-02T10:00:00Z"
										}
									}
								]
							}
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	commits, err := client.GetRecentCommits(ctx, "owner", "repo", 2)
	if err != nil {
		t.Fatalf("GetRecentCommits() error = %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("len(commits) = %d, want %d", len(commits), 2)
	}
	if commits[0].SHA != "abc123" {
		t.Errorf("commits[0].SHA = %q, want %q", commits[0].SHA, "abc123")
	}
}

func TestGraphQLClient_GetAuthorAssociation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"viewerPermission": "ADMIN"
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	perm, err := client.GetAuthorAssociation(ctx, "owner", "repo")
	if err != nil {
		t.Fatalf("GetAuthorAssociation() error = %v", err)
	}
	if perm != "ADMIN" {
		t.Errorf("permission = %q, want %q", perm, "ADMIN")
	}
}

func TestNewGraphQLClient(t *testing.T) {
	client := NewGraphQLClient("https://api.github.com/graphql", "v1.ghp_xxxxx")

	if client.endpoint != "https://api.github.com/graphql" {
		t.Errorf("endpoint = %q, want %q", client.endpoint, "https://api.github.com/graphql")
	}
	if client.httpClient == nil {
		t.Error("httpClient = nil, want non-nil")
	}
}

func TestGraphQLClient_QueryRequestFormat(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{}})
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	var result map[string]interface{}
	client.Query(ctx, `query { viewer { login } }`, map[string]string{"owner": "test"}, &result)

	if receivedBody["query"] == nil {
		t.Error("query not in request body")
	}
	if receivedBody["variables"] == nil {
		t.Error("variables not in request body")
	}
}

func TestGraphQLClient_GetPullRequestChecks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"pullRequest": {
						"commits": {
							"nodes": [
								{
									"commit": {
										"statusCheckRollup": {
											"state": "SUCCESS",
											"checks": {
												"nodes": [
													{
														"name": "CI",
														"state": "SUCCESS"
													},
													{
														"name": "lint",
														"state": "SUCCESS"
													}
												]
											}
										}
									}
								}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	checks, err := client.GetPullRequestChecks(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("GetPullRequestChecks() error = %v", err)
	}
	if len(checks) != 2 {
		t.Errorf("len(checks) = %d, want %d", len(checks), 2)
	}
	if checks[0].Name != "CI" {
		t.Errorf("checks[0].Name = %q, want %q", checks[0].Name, "CI")
	}
}

func TestGraphQLClient_QueryWithVariables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body GraphQLQuery
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body.Variables == nil {
			t.Error("Variables = nil, want non-nil")
		}
		if body.Variables["owner"] != "myorg" {
			t.Errorf("Variables[owner] = %v, want %q", body.Variables["owner"], "myorg")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]interface{}{
				"repository": map[string]interface{}{
					"name": "myrepo",
				},
			},
		})
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	var result struct {
		Repository struct {
			Name string `json:"name"`
		} `json:"repository"`
	}

	err := client.Query(ctx, `query($owner: String!) {
		repository(owner: $owner, name: "myrepo") {
			name
		}
	}`, map[string]string{"owner": "myorg"}, &result)
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
}

func TestGraphQLClient_GetIssueTimeline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"issue": {
						"timeline": {
							"nodes": [
								{
									"__typename": "IssueComment",
									"body": "Comment 1"
								},
								{
									"__typename": "LabeledEvent",
									"label": {"name": "bug"}
								},
								{
									"__typename": "ClosedEvent",
									"actor": {"login": "closer"}
								}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	events, err := client.GetIssueTimeline(ctx, "owner", "repo", 789)
	if err != nil {
		t.Fatalf("GetIssueTimeline() error = %v", err)
	}
	if len(events) != 3 {
		t.Errorf("len(events) = %d, want %d", len(events), 3)
	}
	if events[0].Type != "IssueComment" {
		t.Errorf("events[0].Type = %q, want %q", events[0].Type, "IssueComment")
	}
	if events[1].Type != "LabeledEvent" {
		t.Errorf("events[1].Type = %q, want %q", events[1].Type, "LabeledEvent")
	}
	if events[2].Type != "ClosedEvent" {
		t.Errorf("events[2].Type = %q, want %q", events[2].Type, "ClosedEvent")
	}
}

func TestGraphQLClient_GetFirstPRCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"pullRequest": {
						"commits": {
							"nodes": [
								{
									"commit": {
										"oid": "firstcommit123",
										"message": "Initial commit",
										"author": {
											"name": "Dev",
											"date": "2024-01-01T00:00:00Z"
										}
									}
								}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	commit, err := client.GetFirstPRCommit(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("GetFirstPRCommit() error = %v", err)
	}
	if commit.SHA != "firstcommit123" {
		t.Errorf("commit.SHA = %q, want %q", commit.SHA, "firstcommit123")
	}
}

func TestGraphQLClient_ExtractRepositoryLabels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"labels": {
						"nodes": [
							{"name": "bug"},
							{"name": "enhancement"},
							{"name": "documentation"}
						]
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	labels, err := client.ExtractRepositoryLabels(ctx, "owner", "repo")
	if err != nil {
		t.Fatalf("ExtractRepositoryLabels() error = %v", err)
	}
	if len(labels) != 3 {
		t.Errorf("len(labels) = %d, want %d", len(labels), 3)
	}
}

func TestGraphQLClient_CheckDuplicateIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"search": {
					"nodes": [
						{
							"number": 100,
							"title": "Same issue title",
							"body": "Similar content"
						}
					],
					"issueCount": 1
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	issues, err := client.CheckDuplicateIssues(ctx, "owner", "repo", "Same issue title")
	if err != nil {
		t.Fatalf("CheckDuplicateIssues() error = %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("len(issues) = %d, want %d", len(issues), 1)
	}
	if issues[0].Number != 100 {
		t.Errorf("issues[0].Number = %d, want %d", issues[0].Number, 100)
	}
}

func TestGraphQLClient_GetUserPastPRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"search": {
					"nodes": [
						{
							"__typename": "PullRequest",
							"number": 10,
							"title": "First PR",
							"mergedAt": "2024-01-01T00:00:00Z"
						},
						{
							"__typename": "PullRequest",
							"number": 20,
							"title": "Second PR",
							"mergedAt": "2024-01-15T00:00:00Z"
						}
					],
					"issueCount": 2
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	prs, err := client.GetUserPastPRs(ctx, "owner", "repo", "contributor")
	if err != nil {
		t.Fatalf("GetUserPastPRs() error = %v", err)
	}
	if len(prs) != 2 {
		t.Errorf("len(prs) = %d, want %d", len(prs), 2)
	}
}

func TestGraphQLClient_GetPRFilesChanged(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"pullRequest": {
						"files": {
							"nodes": [
								{
									"path": "src/main.go",
									"additions": 10,
									"deletions": 5
								},
								{
									"path": "README.md",
									"additions": 50,
									"deletions": 0
								}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	files, err := client.GetPRFilesChanged(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("GetPRFilesChanged() error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("len(files) = %d, want %d", len(files), 2)
	}
	if files[0].Filename != "src/main.go" {
		t.Errorf("files[0].Filename = %q, want %q", files[0].Filename, "src/main.go")
	}
}

func TestGraphQLClient_GetDiffHunks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		response := `{
			"data": {
				"repository": {
					"pullRequest": {
						"files": {
							"nodes": [
								{
									"path": "src/main.go",
									"patch": "@@ -1,3 +1,4 @@\n func main() {\n+\tfmt.Println(\"hello\")\n }"
								}
							]
						}
					}
				}
			}
		}`
		w.Write([]byte(response))
	}))
	defer server.Close()

	client := &GraphQLClient{
		endpoint:   server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	hunks, err := client.GetDiffHunks(ctx, "owner", "repo", 456, "src/main.go")
	if err != nil {
		t.Fatalf("GetDiffHunks() error = %v", err)
	}
	if len(hunks) != 1 {
		t.Errorf("len(hunks) = %d, want %d", len(hunks), 1)
	}
	if len(hunks) > 0 && !strings.Contains(hunks[0], "@@ -1,3 +1,4 @@") {
		t.Errorf("hunks[0] doesn't contain expected hunk header, got: %q", hunks[0])
	}
}
