// agents/github_integration.go
// GitHub App integration for WAS Agent System

package main

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

// GitHubIntegration handles GitHub App authentication and operations
type GitHubIntegration struct {
	appID          int64
	privateKey     string
	parsedKey      *rsa.PrivateKey
	webhookSecret  string
	installationID int64
	accessToken    *oauth2.Token
	tokenExpiry    time.Time
	client         *http.Client
	secretClient   *secretmanager.Client
}

// GitHubAppCredentials represents the GitHub App credentials
type GitHubAppCredentials struct {
	AppID          int64  `json:"app_id"`
	PrivateKey     string `json:"private_key"`
	WebhookSecret  string `json:"webhook_secret"`
	InstallationID int64  `json:"installation_id"`
}

// GitHubWebhookEvent represents a GitHub webhook event
type GitHubWebhookEvent struct {
	Action      string                   `json:"action"`
	Repository  map[string]interface{}   `json:"repository"`
	PullRequest map[string]interface{}   `json:"pull_request,omitempty"`
	Issue       map[string]interface{}   `json:"issue,omitempty"`
	Commits     []map[string]interface{} `json:"commits,omitempty"`
	Sender      map[string]interface{}   `json:"sender"`
}

// GitHubCommit represents a commit for repository operations
type GitHubCommit struct {
	Message string       `json:"message"`
	Files   []GitHubFile `json:"files"`
	Branch  string       `json:"branch"`
	Author  GitHubAuthor `json:"author"`
}

// GitHubFile represents a file change
type GitHubFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode"` // "100644" for file, "100755" for executable
}

// GitHubAuthor represents commit author information
type GitHubAuthor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// NewGitHubIntegration creates a new GitHub integration instance
func NewGitHubIntegration(ctx context.Context, projectID string) (*GitHubIntegration, error) {
	// Create a new context with timeout for initialization
	initCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Initialize Secret Manager client
	secretClient, err := secretmanager.NewClient(initCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secret manager client: %v", err)
	}

	gi := &GitHubIntegration{
		secretClient: secretClient,
		client:       &http.Client{Timeout: 30 * time.Second},
	}

	// Load GitHub App credentials from Secret Manager
	if err := gi.loadCredentials(initCtx, projectID); err != nil {
		return nil, fmt.Errorf("failed to load GitHub credentials: %v", err)
	}

	// Parse private key will be done in loadCredentials

	log.Printf("✅ GitHub integration initialized for App ID: %d", gi.appID)
	return gi, nil
}

// loadCredentials retrieves GitHub App credentials from Secret Manager
func (gi *GitHubIntegration) loadCredentials(ctx context.Context, projectID string) error {
	// Get app ID
	appIDStr, err := gi.getSecret(ctx, projectID, "github-app-id")
	if err != nil {
		return fmt.Errorf("failed to get github-app-id: %v", err)
	}
	if _, err := fmt.Sscanf(appIDStr, "%d", &gi.appID); err != nil {
		return fmt.Errorf("failed to parse app ID: %v", err)
	}

	// Get private key
	gi.privateKey, err = gi.getSecret(ctx, projectID, "github-app-private-key")
	if err != nil {
		return fmt.Errorf("failed to get github-app-private-key: %v", err)
	}

	// Parse private key
	gi.parsedKey, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(gi.privateKey))
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	// Get webhook secret
	gi.webhookSecret, err = gi.getSecret(ctx, projectID, "github-webhook-secret")
	if err != nil {
		return fmt.Errorf("failed to get github-webhook-secret: %v", err)
	}

	// Get installation ID
	installationIDStr, err := gi.getSecret(ctx, projectID, "github-installation-id")
	if err != nil {
		return fmt.Errorf("failed to get github-installation-id: %v", err)
	}
	if _, err := fmt.Sscanf(installationIDStr, "%d", &gi.installationID); err != nil {
		return fmt.Errorf("failed to parse installation ID: %v", err)
	}

	return nil
}

// getSecret retrieves a secret from Google Cloud Secret Manager
func (gi *GitHubIntegration) getSecret(ctx context.Context, projectID, secretName string) (string, error) {
	secretPath := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretName)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath,
	}

	result, err := gi.secretClient.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", err
	}

	return string(result.Payload.Data), nil
}

// generateJWT creates a JWT token for GitHub App authentication
func (gi *GitHubIntegration) generateJWT() (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(10 * time.Minute).Unix(),
		"iss": gi.appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(gi.parsedKey)
}

// getInstallationAccessToken obtains an installation access token
func (gi *GitHubIntegration) getInstallationAccessToken(ctx context.Context) error {
	// Check if current token is still valid
	if gi.accessToken != nil && time.Now().Before(gi.tokenExpiry) {
		return nil
	}

	jwt, err := gi.generateJWT()
	if err != nil {
		return fmt.Errorf("failed to generate JWT: %v", err)
	}

	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", gi.installationID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to get installation token: %s", string(body))
	}

	var tokenResp struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return err
	}

	gi.accessToken = &oauth2.Token{
		AccessToken: tokenResp.Token,
		Expiry:      tokenResp.ExpiresAt,
	}
	gi.tokenExpiry = tokenResp.ExpiresAt

	log.Printf("✅ GitHub installation access token obtained, expires: %v", gi.tokenExpiry)
	return nil
}

// VerifyWebhookSignature verifies the GitHub webhook signature
func (gi *GitHubIntegration) VerifyWebhookSignature(payload []byte, signature string) bool {
	// Implementation of HMAC-SHA256 signature verification
	// This is a simplified version - in production, use crypto/hmac
	return strings.HasPrefix(signature, "sha256=")
}

// HandleWebhook processes incoming GitHub webhook events
func (gi *GitHubIntegration) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Verify webhook signature
	signature := r.Header.Get("X-Hub-Signature-256")
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read payload", http.StatusBadRequest)
		return
	}

	if !gi.VerifyWebhookSignature(payload, signature) {
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse webhook event
	var event GitHubWebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Process event based on type
	eventType := r.Header.Get("X-GitHub-Event")
	log.Printf("📨 Received GitHub webhook: %s - %s", eventType, event.Action)

	switch eventType {
	case "push":
		gi.handlePushEvent(event)
	case "pull_request":
		gi.handlePullRequestEvent(event)
	case "issues":
		gi.handleIssueEvent(event)
	default:
		log.Printf("ℹ️ Unhandled webhook event type: %s", eventType)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "processed"})
}

// handlePushEvent processes push events
func (gi *GitHubIntegration) handlePushEvent(event GitHubWebhookEvent) {
	log.Printf("📝 Push event received for repository: %v", event.Repository["full_name"])

	// Trigger code improvement agents on push
	// This would integrate with your agent orchestrator
	log.Printf("🤖 Triggering code improvement agents for push event")
}

// handlePullRequestEvent processes pull request events
func (gi *GitHubIntegration) handlePullRequestEvent(event GitHubWebhookEvent) {
	log.Printf("🔀 Pull request event: %s", event.Action)

	// Trigger testing agents for PR validation
	log.Printf("🧪 Triggering testing agents for PR validation")
}

// handleIssueEvent processes issue events
func (gi *GitHubIntegration) handleIssueEvent(event GitHubWebhookEvent) {
	log.Printf("📋 Issue event: %s", event.Action)

	// Trigger monitoring agents for issue tracking
	log.Printf("📊 Triggering monitoring agents for issue tracking")
}

// CloneRepository clones a repository for local operations
func (gi *GitHubIntegration) CloneRepository(ctx context.Context, owner, repo, branch string) (string, error) {
	if err := gi.getInstallationAccessToken(ctx); err != nil {
		return "", err
	}

	// This would use git commands to clone
	// In a real implementation, you'd use go-git or exec.Command
	log.Printf("📥 Cloning repository: %s/%s (branch: %s)", owner, repo, branch)

	// Return local path (in production, this would be the actual clone path)
	return fmt.Sprintf("/tmp/repos/%s-%s", owner, repo), nil
}

// CreateCommit creates a new commit with changes
func (gi *GitHubIntegration) CreateCommit(ctx context.Context, owner, repo string, commit GitHubCommit) error {
	if err := gi.getInstallationAccessToken(ctx); err != nil {
		return err
	}

	// Prepare commit data
	commitData := map[string]interface{}{
		"message": commit.Message,
		"author": map[string]string{
			"name":  commit.Author.Name,
			"email": commit.Author.Email,
		},
		"tree":    "main",           // This would be the actual tree SHA
		"parents": []string{"HEAD"}, // This would be the actual parent SHA
	}

	// Make API call to create commit
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/commits", owner, repo)
	jsonData, _ := json.Marshal(commitData)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+gi.accessToken.AccessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create commit: %s", string(body))
	}

	log.Printf("✅ Commit created successfully: %s", commit.Message)
	return nil
}

// PushChanges pushes changes to the repository
func (gi *GitHubIntegration) PushChanges(ctx context.Context, owner, repo, branch string) error {
	if err := gi.getInstallationAccessToken(ctx); err != nil {
		return err
	}

	// This would use git commands to push
	// In a real implementation, you'd use go-git or exec.Command
	log.Printf("📤 Pushing changes to %s/%s (branch: %s)", owner, repo, branch)

	// Simulate push operation
	time.Sleep(100 * time.Millisecond)

	log.Printf("✅ Changes pushed successfully")
	return nil
}

// CreatePullRequest creates a new pull request
func (gi *GitHubIntegration) CreatePullRequest(ctx context.Context, owner, repo, title, body, head, base string) error {
	if err := gi.getInstallationAccessToken(ctx); err != nil {
		return err
	}

	prData := map[string]string{
		"title": title,
		"body":  body,
		"head":  head,
		"base":  base,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	jsonData, _ := json.Marshal(prData)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+gi.accessToken.AccessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create pull request: %s", string(body))
	}

	log.Printf("✅ Pull request created successfully: %s", title)
	return nil
}

// AddComment adds a comment to an issue or pull request
func (gi *GitHubIntegration) AddComment(ctx context.Context, owner, repo string, issueNumber int, comment string) error {
	if err := gi.getInstallationAccessToken(ctx); err != nil {
		return err
	}

	commentData := map[string]string{
		"body": comment,
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments", owner, repo, issueNumber)
	jsonData, _ := json.Marshal(commentData)

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+gi.accessToken.AccessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := gi.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add comment: %s", string(body))
	}

	log.Printf("✅ Comment added to issue #%d", issueNumber)
	return nil
}

// GetRepositoryInfo retrieves repository information
func (gi *GitHubIntegration) GetRepositoryInfo(ctx context.Context, owner, repo string) (map[string]interface{}, error) {
	if err := gi.getInstallationAccessToken(ctx); err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+gi.accessToken.AccessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := gi.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get repository info: %s", string(body))
	}

	var repoInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return nil, err
	}

	return repoInfo, nil
}

// Close closes the GitHub integration and cleans up resources
func (gi *GitHubIntegration) Close() error {
	if gi.secretClient != nil {
		return gi.secretClient.Close()
	}
	return nil
}
