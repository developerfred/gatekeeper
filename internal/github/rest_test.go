package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewGitHubAppClient(t *testing.T) {
	client := NewGitHubAppClient("app-id", "private-key", "https://github.com/myorg/myrepo")

	if client.appID != "app-id" {
		t.Errorf("appID = %q, want %q", client.appID, "app-id")
	}
	if client.privateKey != "private-key" {
		t.Errorf("privateKey = %q, want %q", client.privateKey, "private-key")
	}
	if client.baseURL != "https://github.com/myorg/myrepo" {
		t.Errorf("baseURL = %q, want %q", client.baseURL, "https://github.com/myorg/myrepo")
	}
	if client.httpClient == nil {
		t.Error("httpClient = nil, want non-nil")
	}
}

func TestGitHubAppClient_CreateComment(t *testing.T) {
	var receivedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		if r.Method != "POST" {
			t.Errorf("method = %q, want %q", r.Method, "POST")
		}
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/issues/123/comments") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/issues/123/comments")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":         1,
			"body":       "test comment",
			"user":       map[string]string{"login": "gatekeeper[bot]"},
			"created_at": time.Now().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	resp, err := client.CreateComment(ctx, "owner", "repo", 123, "test comment")
	if err != nil {
		t.Fatalf("CreateComment() error = %v", err)
	}
	if resp.ID != 1 {
		t.Errorf("Comment.ID = %d, want %d", resp.ID, 1)
	}
	if resp.Body != "test comment" {
		t.Errorf("Comment.Body = %q, want %q", resp.Body, "test comment")
	}
	if receivedReq == nil {
		t.Fatal("receivedReq = nil, want non-nil")
	}
}

func TestGitHubAppClient_CreateReviewComment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/456/comments") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/pulls/456/comments")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":   2,
			"body": "review comment",
			"path": "src/main.go",
			"line": 42,
			"side": "RIGHT",
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	resp, err := client.CreateReviewComment(ctx, "owner", "repo", 456, &ReviewComment{
		Body: "review comment",
		Path: "src/main.go",
		Line: 42,
		Side: "RIGHT",
	})
	if err != nil {
		t.Fatalf("CreateReviewComment() error = %v", err)
	}
	if resp.ID != 2 {
		t.Errorf("ReviewComment.ID = %d, want %d", resp.ID, 2)
	}
	if resp.Path != "src/main.go" {
		t.Errorf("ReviewComment.Path = %q, want %q", resp.Path, "src/main.go")
	}
}

func TestGitHubAppClient_AddLabels(t *testing.T) {
	var receivedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		if r.Method != "POST" {
			t.Errorf("method = %q, want %q", r.Method, "POST")
		}
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/issues/789/labels") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/issues/789/labels")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"name": "bug"}, {"name": "priority"},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	labels, err := client.AddLabels(ctx, "owner", "repo", 789, []string{"bug", "priority"})
	if err != nil {
		t.Fatalf("AddLabels() error = %v", err)
	}
	if len(labels) != 2 {
		t.Errorf("len(labels) = %d, want %d", len(labels), 2)
	}
	if receivedReq.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want %q", receivedReq.Header.Get("Content-Type"), "application/json")
	}
}

func TestGitHubAppClient_RemoveLabel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("method = %q, want %q", r.Method, "DELETE")
		}
		expectedPath := "/repos/owner/repo/issues/789/labels/bug"
		if !strings.Contains(r.URL.Path, expectedPath) {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, expectedPath)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	err := client.RemoveLabel(ctx, "owner", "repo", 789, "bug")
	if err != nil {
		t.Fatalf("RemoveLabel() error = %v", err)
	}
}

func TestGitHubAppClient_CloseIssue(t *testing.T) {
	var receivedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		if r.Method != "PATCH" {
			t.Errorf("method = %q, want %q", r.Method, "PATCH")
		}
		expectedPath := "/repos/owner/repo/issues/100"
		if !strings.Contains(r.URL.Path, expectedPath) {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, expectedPath)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"state": "closed",
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	err := client.CloseIssue(ctx, "owner", "repo", 100)
	if err != nil {
		t.Fatalf("CloseIssue() error = %v", err)
	}
	if receivedReq == nil {
		t.Fatal("receivedReq = nil")
	}
}

func TestGitHubAppClient_GetPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/456") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/pulls/456")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":        456,
			"title":         "feat: add new feature",
			"body":          "This PR adds...",
			"state":         "open",
			"additions":     150,
			"deletions":     20,
			"changed_files": 3,
			"user": map[string]string{
				"login":      "contributor",
				"created_at": "2020-01-01T00:00:00Z",
			},
			"head": map[string]string{
				"ref": "feature-branch",
				"sha": "abc123",
			},
			"base": map[string]string{
				"ref": "main",
				"sha": "def456",
			},
			"commits":   5,
			"mergeable": true,
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	pr, err := client.GetPR(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("GetPR() error = %v", err)
	}
	if pr.Number != 456 {
		t.Errorf("PR.Number = %d, want %d", pr.Number, 456)
	}
	if pr.Title != "feat: add new feature" {
		t.Errorf("PR.Title = %q, want %q", pr.Title, "feat: add new feature")
	}
	if pr.Additions != 150 {
		t.Errorf("PR.Additions = %d, want %d", pr.Additions, 150)
	}
	if pr.Deletions != 20 {
		t.Errorf("PR.Deletions = %d, want %d", pr.Deletions, 20)
	}
	if pr.ChangedFiles != 3 {
		t.Errorf("PR.ChangedFiles = %d, want %d", pr.ChangedFiles, 3)
	}
	if pr.User.Login != "contributor" {
		t.Errorf("PR.User.Login = %q, want %q", pr.User.Login, "contributor")
	}
}

func TestGitHubAppClient_GetIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/issues/789") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/issues/789")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number": 789,
			"title":  "Bug: something broken",
			"body":   "It doesn't work when...",
			"state":  "open",
			"labels": []map[string]string{
				{"name": "bug"},
				{"name": "priority"},
			},
			"user": map[string]string{
				"login":      "reporter",
				"created_at": "2020-01-01T00:00:00Z",
			},
			"assignees": []map[string]string{
				{"login": "maintainer"},
			},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	issue, err := client.GetIssue(ctx, "owner", "repo", 789)
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if issue.Number != 789 {
		t.Errorf("Issue.Number = %d, want %d", issue.Number, 789)
	}
	if issue.Title != "Bug: something broken" {
		t.Errorf("Issue.Title = %q, want %q", issue.Title, "Bug: something broken")
	}
	if len(issue.Labels) != 2 {
		t.Errorf("len(Issue.Labels) = %d, want %d", len(issue.Labels), 2)
	}
}

func TestGitHubAppClient_ListPRFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/456/files") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/pulls/456/files")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"filename":  "src/main.go",
				"status":    "modified",
				"additions": 10,
				"deletions": 5,
				"patch":     "@@ -1,5 +1,6 @@\n func main() {\n+\tfmt.Println(\"hello\")\n }",
			},
			{
				"filename":  "README.md",
				"status":    "added",
				"additions": 50,
				"deletions": 0,
				"patch":     "@@ -0,0 +1,50 @@\n # My Project",
			},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	files, err := client.ListPRFiles(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("ListPRFiles() error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("len(files) = %d, want %d", len(files), 2)
	}
	if files[0].Filename != "src/main.go" {
		t.Errorf("files[0].Filename = %q, want %q", files[0].Filename, "src/main.go")
	}
	if files[0].Status != "modified" {
		t.Errorf("files[0].Status = %q, want %q", files[0].Status, "modified")
	}
}

func TestGitHubAppClient_GetBranch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/branches/feature-branch") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/branches/feature-branch")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":      "feature-branch",
			"commit":    map[string]string{"sha": "abc123def456"},
			"protected": false,
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	branch, err := client.GetBranch(ctx, "owner", "repo", "feature-branch")
	if err != nil {
		t.Fatalf("GetBranch() error = %v", err)
	}
	if branch.Name != "feature-branch" {
		t.Errorf("Branch.Name = %q, want %q", branch.Name, "feature-branch")
	}
	if branch.Commit.SHA != "abc123def456" {
		t.Errorf("Branch.Commit.SHA = %q, want %q", branch.Commit.SHA, "abc123def456")
	}
}

func TestGitHubAppClient_ListPullRequestReviews(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/456/reviews") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/pulls/456/reviews")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":           1001,
				"state":        "APPROVED",
				"user":         map[string]string{"login": "reviewer1"},
				"body":         "LGTM!",
				"submitted_at": "2024-01-01T00:00:00Z",
			},
			{
				"id":           1002,
				"state":        "CHANGES_REQUESTED",
				"user":         map[string]string{"login": "reviewer2"},
				"body":         "Please fix this",
				"submitted_at": "2024-01-02T00:00:00Z",
			},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	reviews, err := client.ListPullRequestReviews(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("ListPullRequestReviews() error = %v", err)
	}
	if len(reviews) != 2 {
		t.Errorf("len(reviews) = %d, want %d", len(reviews), 2)
	}
	if reviews[0].State != "APPROVED" {
		t.Errorf("reviews[0].State = %q, want %q", reviews[0].State, "APPROVED")
	}
	if reviews[1].State != "CHANGES_REQUESTED" {
		t.Errorf("reviews[1].State = %q, want %q", reviews[1].State, "CHANGES_REQUESTED")
	}
}

func TestGitHubAppClient_ListIssueComments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/issues/789/comments") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/issues/789/comments")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"id":         5001,
				"body":       "This is a comment",
				"user":       map[string]string{"login": "commenter"},
				"created_at": "2024-01-01T00:00:00Z",
			},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	comments, err := client.ListIssueComments(ctx, "owner", "repo", 789)
	if err != nil {
		t.Fatalf("ListIssueComments() error = %v", err)
	}
	if len(comments) != 1 {
		t.Errorf("len(comments) = %d, want %d", len(comments), 1)
	}
}

func TestGitHubAppClient_GetAuthenticatedUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("URL path = %q, want %q", r.URL.Path, "/user")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"login": "gatekeeper[bot]",
			"id":    12345678,
			"type":  "Bot",
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	user, err := client.GetAuthenticatedUser(ctx)
	if err != nil {
		t.Fatalf("GetAuthenticatedUser() error = %v", err)
	}
	if user.Login != "gatekeeper[bot]" {
		t.Errorf("User.Login = %q, want %q", user.Login, "gatekeeper[bot]")
	}
}

func TestGitHubAppClient_ListRepoIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/issues") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/issues")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"number": 100,
				"title":  "First issue",
				"state":  "open",
			},
			{
				"number": 101,
				"title":  "Second issue",
				"state":  "closed",
			},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	issues, err := client.ListRepoIssues(ctx, "owner", "repo", "open")
	if err != nil {
		t.Fatalf("ListRepoIssues() error = %v", err)
	}
	if len(issues) != 2 {
		t.Errorf("len(issues) = %d, want %d", len(issues), 2)
	}
}

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "test-webhook-secret"
	payload := []byte(`{"action":"opened","pull_request":{"number":123}}`)

	sig := "sha256=" + computeHMACSHA256(secret, payload)

	if !VerifyWebhookSignature(secret, payload, sig) {
		t.Error("VerifyWebhookSignature() = false, want true for valid signature")
	}

	wrongSig := "sha256=0000000000000000000000000000000000000000000000000000000000000000"
	if VerifyWebhookSignature(secret, payload, wrongSig) {
		t.Error("VerifyWebhookSignature() = true, want false for invalid signature")
	}

	emptySig := ""
	if VerifyWebhookSignature(secret, payload, emptySig) {
		t.Error("VerifyWebhookSignature() = true, want false for empty signature")
	}
}

func TestGitHubAppClient_CreateInstallationToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %q, want %q", r.Method, "POST")
		}
		expectedPath := "/app/installations/123/access_tokens"
		if !strings.Contains(r.URL.Path, expectedPath) {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, expectedPath)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token":      "v1.ghp_xxxxxxxxxxxx",
			"expires_at": time.Now().Add(time.Hour).Format(time.RFC3339),
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	token, err := client.CreateInstallationToken(ctx, 123)
	if err != nil {
		t.Fatalf("CreateInstallationToken() error = %v", err)
	}
	if token != "v1.ghp_xxxxxxxxxxxx" {
		t.Errorf("token = %q, want %q", token, "v1.ghp_xxxxxxxxxxxx")
	}
}

func TestGitHubAppClient_SubmitReview(t *testing.T) {
	var receivedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		if r.Method != "POST" {
			t.Errorf("method = %q, want %q", r.Method, "POST")
		}
		expectedPath := "/repos/owner/repo/pulls/456/reviews"
		if !strings.Contains(r.URL.Path, expectedPath) {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, expectedPath)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    9001,
			"state": "APPROVED",
			"body":  "LGTM!",
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	review := &SubmitReviewInput{
		Body:  "LGTM!",
		Event: "APPROVE",
	}
	err := client.SubmitReview(ctx, "owner", "repo", 456, review)
	if err != nil {
		t.Fatalf("SubmitReview() error = %v", err)
	}
	if receivedReq == nil {
		t.Fatal("receivedReq = nil")
	}
}

func TestGitHubAppClient_DismissReview(t *testing.T) {
	var receivedReq *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedReq = r
		if r.Method != "PUT" {
			t.Errorf("method = %q, want %q", r.Method, "PUT")
		}
		expectedPath := "/repos/owner/repo/pulls/456/reviews/789/dismissals"
		if !strings.Contains(r.URL.Path, expectedPath) {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, expectedPath)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	err := client.DismissReview(ctx, "owner", "repo", 456, 789, "No longer needed")
	if err != nil {
		t.Fatalf("DismissReview() error = %v", err)
	}
	if receivedReq == nil {
		t.Fatal("receivedReq = nil")
	}
}

func TestGitHubAppClient_GetCommit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/commits/abc123") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/commits/abc123")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"sha":     "abc123",
			"message": "feat: add feature",
			"author": map[string]string{
				"name":  "Contributor",
				"email": "contributor@example.com",
				"date":  "2024-01-01T00:00:00Z",
			},
			"stats": map[string]int{
				"additions": 100,
				"deletions": 10,
			},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	commit, err := client.GetCommit(ctx, "owner", "repo", "abc123")
	if err != nil {
		t.Fatalf("GetCommit() error = %v", err)
	}
	if commit.SHA != "abc123" {
		t.Errorf("Commit.SHA = %q, want %q", commit.SHA, "abc123")
	}
	if commit.Message != "feat: add feature" {
		t.Errorf("Commit.Message = %q, want %q", commit.Message, "feat: add feature")
	}
}

func TestGitHubAppClient_ListCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repos/owner/repo/pulls/456/commits") {
			t.Errorf("URL path = %q, want containing %q", r.URL.Path, "/repos/owner/repo/pulls/456/commits")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"sha":     "commit1",
				"message": "First commit",
				"author":  map[string]string{"name": "Dev"},
			},
			{
				"sha":     "commit2",
				"message": "Second commit",
				"author":  map[string]string{"name": "Dev"},
			},
		})
	}))
	defer server.Close()

	client := &GitHubAppClient{
		appID:      "123",
		privateKey: "test-key",
		baseURL:    server.URL,
		httpClient: server.Client(),
	}

	ctx := context.Background()
	commits, err := client.ListCommits(ctx, "owner", "repo", 456)
	if err != nil {
		t.Fatalf("ListCommits() error = %v", err)
	}
	if len(commits) != 2 {
		t.Errorf("len(commits) = %d, want %d", len(commits), 2)
	}
}
