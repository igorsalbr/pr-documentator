package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
)

type AnalyzerService struct {
	claudeClient  interfaces.ClaudeClient
	postmanClient interfaces.PostmanClient
	logger        interfaces.Logger
	metrics       interfaces.MetricsCollector
}

// NewAnalyzerService creates a new analyzer service
func NewAnalyzerService(claudeClient interfaces.ClaudeClient, postmanClient interfaces.PostmanClient, logger interfaces.Logger, metrics interfaces.MetricsCollector) *AnalyzerService {
	return &AnalyzerService{
		claudeClient:  claudeClient,
		postmanClient: postmanClient,
		logger:        logger,
		metrics:       metrics,
	}
}

// AnalyzePR analyzes a pull request and updates Postman documentation
func (s *AnalyzerService) AnalyzePR(ctx context.Context, payload models.GitHubPRPayload) (*models.AnalysisResponse, error) {
	s.logger.Info("Starting PR analysis",
		"pr_number", payload.PullRequest.Number,
		"repo", payload.Repository.FullName,
		"action", payload.Action,
	)

	// Only process opened, synchronize, or reopened PRs
	if !s.shouldProcessAction(payload.Action) {
		s.logger.Info("Skipping PR action", "action", payload.Action)
		return &models.AnalysisResponse{
			Summary: fmt.Sprintf("Skipped action: %s", payload.Action),
		}, nil
	}

	// Fetch the PR diff
	// diff, err := s.fetchPRDiff(ctx, payload.PullRequest.DiffURL)
	// if err != nil {
	// 	s.logger.Error("Failed to fetch PR diff", err, "diff_url", payload.PullRequest.DiffURL)
	// 	return nil, fmt.Errorf("failed to fetch PR diff: %w", err)
	// }

	diff := `diff --git a/real_estate/controllers/etl/etl.go b/real_estate/controllers/etl/etl.go
index cad779dc7..a8267638a 100644
--- a/real_estate/controllers/etl/etl.go
+++ b/real_estate/controllers/etl/etl.go
@@ -128,76 +128,78 @@ func (e *etl) InitV2(c *components.HTTPComponents) {
 		return
 	}
 
+	req := &models.RealEstateAgencyValidateXML{}
+	if err := adapter.ValidateRequest(c, req); err != nil {
+		adapter.HttpErrorResponse(c, http.StatusBadRequest, err)
+		return
+	}
+
 	agency, err := db.RealEstateAgency.FindByID(c, agencyID)
 	if err != nil {
 		adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
 		return
 	}
 
-	req := &models.RealEstateAgencyValidateXML{}
-	if err := adapter.ValidateRequest(c, req); err != nil {
-		adapter.HttpErrorResponse(c, http.StatusBadRequest, err)
+	existingFeed, err := db.RealEstateAgencyFeed.FindByAgencyIDAndURL(c, agencyID, req.URL)
+	if err != nil {
+		adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
+		return
+	}
+
+	if existingFeed != nil {
+		adapter.HttpErrorResponse(c, http.StatusConflict, errors.New("feed URL already exists for this agency"))
 		return
 	}
 
-	exists, err := db.RealEstateAgencyFeed.ExistsByRealEstateAgencyID(c, agency.ID.String())
+	_, err = db.RealEstateAgencyFeed.Create(c, &models.RealEstateAgencyFeed{
+		AgencyID:  agency.ID,
+		URL:       req.URL,
+		Status:    models.FeedStatusEtlV2,
+		EtlStatus: models.EtlStatusCompleted,
+	})
+
 	if err != nil {
 		adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
 		return
 	}
 
-	if !exists {
-		_, err = db.RealEstateAgencyFeed.Create(c, &models.RealEstateAgencyFeed{
-			AgencyID:  agency.ID,
-			URL:       req.URL,
-			Status:    models.FeedStatusEtlV2,
-			EtlStatus: models.EtlStatusCompleted,
-		})
-		if err != nil {
-			loggerV2.AddFlow(c.HttpRequest.Context(), loggerV2.Flow{
-				Message: "[ETL][InitV2][ERROR] - Create Feed",
-				Type:    loggerV2.FlowTypeError,
-				Arguments: map[string]interface{}{
-					"agency_id": agency.ID.String(),
-					"feed_url":  req.URL,
-				},
-			})
-			adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
-			return
-		}
+	processableStatuses := []string{
+		string(models.EtlStatusCompleted),
+		string(models.EtlStatusFailed),
 	}
 
-	feeds, err := db.RealEstateAgencyFeed.FindByAgencyIDAndEtlStatusList(c, agencyID, []string{string(models.EtlStatusCompleted), string(models.EtlStatusFailed)})
+	feedsToProcess, err := db.RealEstateAgencyFeed.FindByAgencyIDAndEtlStatusList(c, agencyID, processableStatuses)
 	if err != nil {
 		adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
 		return
 	}
 
-	if len(feeds) == 0 {
+	if len(feedsToProcess) == 0 {
 		adapter.HttpErrorResponse(c, http.StatusInternalServerError, errors.New("no feeds to process"))
 		return
 	}
 
-	err = db.RealEstateAgencyFeed.UpdateEtlStatusBatch(c, feeds, models.EtlStatusProcessing)
+	err = db.RealEstateAgencyFeed.UpdateEtlStatusBatch(c, feedsToProcess, models.EtlStatusProcessing)
 	if err != nil {
 		adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
 		return
 	}
 
-	returnProcess := map[string]interface{}{}
-	for _, feed := range feeds {
+	processResults := make(map[string]interface{}, len(feedsToProcess))
+	for _, feed := range feedsToProcess {
 		response, err := grupoZapProcessV2(c, agency, feed)
 		if err != nil {
 			adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
 			return
 		}
-		returnProcess[feed.ID.String()] = map[string]interface{}{
+
+		processResults[feed.ID.String()] = map[string]interface{}{
 			"body":   response.String(),
 			"status": response.Status(),
 		}
 	}
 
-	adapter.HttpResponseWithPayload(c, returnProcess, http.StatusOK)
+	adapter.HttpResponseWithPayload(c, processResults, http.StatusOK)
 }
 
 func (e *etl) GrupoZapByAgencyID(c *components.HTTPComponents) {
diff --git a/real_estate/controllers/real_estate_agency_validate_xml.go b/real_estate/controllers/real_estate_agency_validate_xml.go
index 8e54ba014..24c136428 100644
--- a/real_estate/controllers/real_estate_agency_validate_xml.go
+++ b/real_estate/controllers/real_estate_agency_validate_xml.go
@@ -5,6 +5,7 @@ import (
 	"bytes"
 	"compress/gzip"
 	"context"
+	"core/real_estate/db"
 	"core/real_estate/models"
 	"core/tools/components"
 	"core/tools/helpers/adapter"
@@ -15,6 +16,8 @@ import (
 	"net/http"
 	"sync"
 	"time"
+
+	"github.com/gofrs/uuid"
 )
 
 type realEstateAgencyValidateXML struct{}
@@ -59,8 +62,13 @@ var (
 	ErrNotFound         = errors.New("link not found")
 )
 
-// ValidateXML endpoint
 func (r realEstateAgencyValidateXML) ValidateXML(c *components.HTTPComponents) {
+	id, err := adapter.GetIDFromURLParam(c, RealEstateAgencyAccessCode.Key)
+	if err != nil {
+		adapter.HttpErrorResponse(c, http.StatusBadRequest, err)
+		return
+	}
+
 	req := &models.RealEstateAgencyValidateXML{}
 	if err := adapter.ValidateRequest(c, req); err != nil {
 		adapter.HttpErrorResponse(c, http.StatusBadRequest, err)
@@ -72,8 +80,13 @@ func (r realEstateAgencyValidateXML) ValidateXML(c *components.HTTPComponents) {
 
 	_, count, err := fetchAndCount(ctx, req.URL)
 	response := buildValidateXMLResponse(err, count)
+
+	if err := saveXMLWithStatusIfNew(c, response, req.URL, id); err != nil {
+		adapter.HttpErrorResponse(c, http.StatusInternalServerError, err)
+		return
+	}
+
 	adapter.HttpResponseWithPayload(c, response, http.StatusOK)
-	return
 }
 
 func buildValidateXMLResponse(err error, count int) models.RealEstateAgencyValidateXMLResponse {
@@ -222,3 +235,44 @@ func parseMinimal(r io.Reader) (string, int, error) {
 
 	return feedType, total, nil
 }
+
+func saveXMLWithStatusIfNew(c *components.HTTPComponents, response models.RealEstateAgencyValidateXMLResponse, url, id string) error {
+
+	xmls, err := db.RealEstateAgencyFeed.FindByAgencyID(c, id)
+	if err != nil {
+		return err
+	}
+
+	for _, xml := range xmls {
+		if xml.URL == url {
+			return nil
+		}
+	}
+
+	agencyID := uuid.FromStringOrNil(id)
+	status, etlStatus := feedStatusFromResponseSlug(response.Slug)
+	feed := &models.RealEstateAgencyFeed{
+		URL:       url,
+		Status:    status,
+		AgencyID:  agencyID,
+		EtlStatus: etlStatus,
+	}
+
+	_, err = db.RealEstateAgencyFeed.Create(c, feed)
+	if err != nil {
+		return fmt.Errorf("failed to save XML feed: %w", err)
+	}
+
+	return nil
+}
+
+func feedStatusFromResponseSlug(slug string) (models.FeedStatus, models.EtlStatus) {
+	switch slug {
+	case "success":
+		return models.FeedStatusMigratedToEtl, models.EtlStatusCompleted
+	case "error":
+		return models.FeedStatusBlocked, models.EtlStatusFailed
+	default:
+		return models.FeedStatusBlocked, models.EtlStatusFailed
+	}
+}
diff --git a/real_estate/db/real_estate_agency_feed.go b/real_estate/db/real_estate_agency_feed.go
index 523492d1b..c314b9e24 100644
--- a/real_estate/db/real_estate_agency_feed.go
+++ b/real_estate/db/real_estate_agency_feed.go
@@ -124,3 +124,14 @@ func (re *realEstateAgencyFeed) AvailableVersions(c *components.HTTPComponents)
 	return versions, nil
 
 }
+
+func (re *realEstateAgencyFeed) FindByAgencyIDAndURL(c *components.HTTPComponents, agencyID, url string) (*models.RealEstateAgencyFeed, error) {
+	var feed models.RealEstateAgencyFeed
+	err := c.Components.Postgres.WithContext(c.HttpRequest.Context()).Model(&feed).
+		Where("agency_id = ? AND url = ?", agencyID, url).
+		Select()
+	if err != nil {
+		return nil, err
+	}
+	return &feed, nil
+}
diff --git a/service/routes.go b/service/routes.go
index c8b7db522..c8189eef6 100644
--- a/service/routes.go
+++ b/service/routes.go
@@ -1966,8 +1966,8 @@ func realEstateAgencyRouter(c *components.Components) http.Handler {
 		}
 	})
 
-	// POST /api/v1/real-estate-agency/validate-xml
-	subRouter.Post("/validate-xml", func(w http.ResponseWriter, r *http.Request) {
+	// POST /api/v1/real-estate-agency/{{realEstateAgencyID}}/validate-xml
+	subRouter.Post("/{realEstateAgencyID}/validate-xml", func(w http.ResponseWriter, r *http.Request) {
 		if auth.Validate(adapter.Components(w, r, c), "admin,bubble") {
 			controllers.RealEstateAgencyValidateXML.ValidateXML(adapter.Components(w, r, c))
 		}`

	// Create analysis request
	analysisReq := models.AnalysisRequest{
		PullRequest: payload.PullRequest,
		Repository:  payload.Repository,
		Diff:        diff,
	}

	// Analyze with Claude
	analysisResp, err := s.claudeClient.AnalyzePR(ctx, analysisReq)
	if err != nil {
		s.logger.Error("Failed to analyze PR with Claude", err, "pr_number", payload.PullRequest.Number)
		return nil, fmt.Errorf("claude analysis failed: %w", err)
	}

	// Only update Postman if there are changes
	if s.hasAPIChanges(analysisResp) {
		s.logger.Info("API changes detected, updating Postman collection",
			"new_routes", len(analysisResp.NewRoutes),
			"modified_routes", len(analysisResp.ModifiedRoutes),
			"deleted_routes", len(analysisResp.DeletedRoutes),
		)

		postmanUpdate, err := s.postmanClient.UpdateCollection(ctx, analysisResp)
		if err != nil {
			s.logger.Error("Failed to update Postman collection", err, "pr_number", payload.PullRequest.Number)
			// Don't fail the entire operation if Postman update fails
			analysisResp.PostmanUpdate = models.PostmanUpdate{
				Status:       "error",
				ErrorMessage: err.Error(),
				UpdatedAt:    time.Now().Format(time.RFC3339),
			}
		} else {
			analysisResp.PostmanUpdate = *postmanUpdate
		}
	} else {
		s.logger.Info("No API changes detected, skipping Postman update")
		analysisResp.PostmanUpdate = models.PostmanUpdate{
			Status:    "skipped",
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
	}

	s.logger.Info("PR analysis completed successfully",
		"pr_number", payload.PullRequest.Number,
		"confidence", analysisResp.Confidence,
		"postman_status", analysisResp.PostmanUpdate.Status,
	)

	return analysisResp, nil
}

func (s *AnalyzerService) shouldProcessAction(action string) bool {
	processableActions := []string{"opened", "synchronize", "reopened"}
	for _, a := range processableActions {
		if a == action {
			return true
		}
	}
	return false
}

func (s *AnalyzerService) fetchPRDiff(ctx context.Context, diffURL string) (string, error) {
	if diffURL == "" {
		return "", fmt.Errorf("diff URL is empty")
	}

	s.logger.Debug("Fetching PR diff", "diff_url", diffURL)

	req, err := http.NewRequestWithContext(ctx, "GET", diffURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// GitHub returns plain text diff
	req.Header.Set("Accept", "text/plain")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch diff, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	diff := string(body)
	s.logger.Debug("Successfully fetched PR diff",
		"diff_size_bytes", len(body),
		"diff_size_chars", len(diff),
	)

	return diff, nil
}

func (s *AnalyzerService) hasAPIChanges(resp *models.AnalysisResponse) bool {
	return len(resp.NewRoutes) > 0 || len(resp.ModifiedRoutes) > 0 || len(resp.DeletedRoutes) > 0
}
