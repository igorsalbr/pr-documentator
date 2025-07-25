package postman

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sony/gobreaker"

	"github.com/igorsal/pr-documentator/internal/config"
	"github.com/igorsal/pr-documentator/internal/interfaces"
	"github.com/igorsal/pr-documentator/internal/models"
	pkgerrors "github.com/igorsal/pr-documentator/pkg/errors"
)

type Client struct {
	httpClient     *http.Client
	config         config.PostmanConfig
	logger         interfaces.Logger
	circuitBreaker interfaces.CircuitBreaker
	metrics        interfaces.MetricsCollector
}

// NewClient creates a new Postman API client with circuit breaker
func NewClient(cfg config.PostmanConfig, logger interfaces.Logger, metrics interfaces.MetricsCollector) *Client {
	// Configure HTTP client
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	// Configure circuit breaker
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "postman-api",
		MaxRequests: 3,
		Interval:    30 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Info("Postman API circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	})

	// Wrap circuit breaker
	cbWrapper := &postmanCircuitBreakerWrapper{cb: cb}

	return &Client{
		httpClient:     client,
		config:         cfg,
		logger:         logger,
		circuitBreaker: cbWrapper,
		metrics:        metrics,
	}
}

// postmanCircuitBreakerWrapper implements interfaces.CircuitBreaker
type postmanCircuitBreakerWrapper struct {
	cb *gobreaker.CircuitBreaker
}

func (w *postmanCircuitBreakerWrapper) Execute(req func() (any, error)) (any, error) {
	return w.cb.Execute(req)
}

func (w *postmanCircuitBreakerWrapper) Name() string {
	return w.cb.Name()
}

func (w *postmanCircuitBreakerWrapper) State() string {
	return w.cb.State().String()
}

// GetCollection retrieves a Postman collection
func (c *Client) GetCollection(ctx context.Context) (*models.PostmanCollection, error) {
	startTime := time.Now()
	labels := map[string]string{
		"service":   "postman",
		"operation": "get_collection",
	}

	result, err := c.circuitBreaker.Execute(func() (any, error) {
		return c.executeGetCollection(ctx)
	})

	duration := time.Since(startTime).Seconds()
	c.metrics.RecordDuration("postman_request_duration_seconds", duration, labels)

	if err != nil {
		labels["status"] = "error"
		c.metrics.IncrementCounter("postman_requests_total", labels)
		return nil, err
	}

	labels["status"] = "success"
	c.metrics.IncrementCounter("postman_requests_total", labels)
	return result.(*models.PostmanCollection), nil
}

func (c *Client) executeGetCollection(ctx context.Context) (*models.PostmanCollection, error) {
	url := fmt.Sprintf("%s/collections/%s", c.config.BaseURL, c.config.CollectionID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, pkgerrors.NewExternalError("postman", "failed to create request").WithCause(err)
	}

	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, pkgerrors.NewExternalError("postman", err.Error()).WithCause(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, pkgerrors.NewExternalError("postman", "failed to read response").WithCause(err)
	}

	if resp.StatusCode >= 400 {
		switch resp.StatusCode {
		case 401:
			return nil, pkgerrors.NewUnauthorizedError("Invalid Postman API key")
		case 404:
			return nil, pkgerrors.NewNotFoundError("Collection not found")
		case 429:
			return nil, pkgerrors.NewRateLimitError("postman")
		default:
			return nil, pkgerrors.NewExternalError("postman", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)))
		}
	}

	var collectionResp models.PostmanCollectionResponse
	if err := json.Unmarshal(respBody, &collectionResp); err != nil {
		return nil, pkgerrors.NewExternalError("postman", "failed to parse response").WithCause(err)
	}

	return &collectionResp.Collection, nil
}

// UpdateCollection updates a Postman collection with new API routes
func (c *Client) UpdateCollection(ctx context.Context, analysisResp *models.AnalysisResponse) (*models.PostmanUpdate, error) {
	c.logger.Info("Starting Postman collection update", "collection_id", c.config.CollectionID)

	// First, get the current collection
	collection, err := c.GetCollection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	// Update the collection with new routes
	updated, err := c.updateCollectionWithRoutes(collection, analysisResp)
	if err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}

	// Send the updated collection back to Postman
	if err := c.putCollection(ctx, collection); err != nil {
		return nil, fmt.Errorf("failed to save updated collection: %w", err)
	}

	c.logger.Info("Successfully updated Postman collection",
		"collection_id", c.config.CollectionID,
		"items_added", updated.ItemsAdded,
		"items_modified", updated.ItemsModified,
		"items_deleted", updated.ItemsDeleted,
	)

	return updated, nil
}

func (c *Client) putCollection(ctx context.Context, collection *models.PostmanCollection) error {
	startTime := time.Now()
	labels := map[string]string{
		"service":   "postman",
		"operation": "put_collection",
	}

	_, err := c.circuitBreaker.Execute(func() (any, error) {
		return nil, c.executePutCollection(ctx, collection)
	})

	duration := time.Since(startTime).Seconds()
	c.metrics.RecordDuration("postman_request_duration_seconds", duration, labels)

	if err != nil {
		labels["status"] = "error"
		c.metrics.IncrementCounter("postman_requests_total", labels)
		return err
	}

	labels["status"] = "success"
	c.metrics.IncrementCounter("postman_requests_total", labels)
	return nil
}

func (c *Client) executePutCollection(ctx context.Context, collection *models.PostmanCollection) error {
	updateReq := models.PostmanUpdateRequest{
		Collection: *collection,
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		return pkgerrors.NewExternalError("postman", "failed to marshal request").WithCause(err)
	}

	url := fmt.Sprintf("%s/collections/%s", c.config.BaseURL, c.config.CollectionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return pkgerrors.NewExternalError("postman", "failed to create request").WithCause(err)
	}

	req.Header.Set("X-API-Key", c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return pkgerrors.NewExternalError("postman", err.Error()).WithCause(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		switch resp.StatusCode {
		case 401:
			return pkgerrors.NewUnauthorizedError("Invalid Postman API key")
		case 404:
			return pkgerrors.NewNotFoundError("Collection not found")
		case 429:
			return pkgerrors.NewRateLimitError("postman")
		default:
			return pkgerrors.NewExternalError("postman", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)))
		}
	}

	return nil
}

func (c *Client) updateCollectionWithRoutes(collection *models.PostmanCollection, analysis *models.AnalysisResponse) (*models.PostmanUpdate, error) {
	update := &models.PostmanUpdate{
		CollectionID: c.config.CollectionID,
		Status:       "success",
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	// Add new routes
	for _, route := range analysis.NewRoutes {
		item := c.convertRouteToPostmanItem(route)
		collection.Items = append(collection.Items, item)
		update.ItemsAdded++
	}

	// Update modified routes
	for _, route := range analysis.ModifiedRoutes {
		if c.updateExistingItem(collection, route) {
			update.ItemsModified++
		} else {
			// If route not found, add as new
			item := c.convertRouteToPostmanItem(route)
			collection.Items = append(collection.Items, item)
			update.ItemsAdded++
		}
	}

	// Mark deleted routes (we don't actually delete, just mark as deprecated)
	for _, route := range analysis.DeletedRoutes {
		if c.markItemAsDeprecated(collection, route) {
			update.ItemsModified++
		}
	}

	return update, nil
}

func (c *Client) convertRouteToPostmanItem(route models.APIRoute) models.PostmanItem {
	// Convert path to Postman URL format
	pathSegments := []string{}
	if route.Path != "" && route.Path != "/" {
		// Remove leading slash and split
		path := route.Path
		if path[0] == '/' {
			path = path[1:]
		}
		pathSegments = []string{"{{baseUrl}}", path}
	} else {
		pathSegments = []string{"{{baseUrl}}"}
	}

	// Convert parameters to headers and query params
	var headers []models.PostmanHeader
	var queryParams []models.PostmanQueryParam

	// Add default headers
	headers = append(headers, models.PostmanHeader{
		Key:   "Content-Type",
		Value: "application/json",
		Type:  "text",
	})

	// Add route-specific headers
	for _, header := range route.Headers {
		headers = append(headers, models.PostmanHeader{
			Key:         header.Name,
			Value:       fmt.Sprintf("%v", header.Example),
			Type:        "text",
			Description: header.Description,
		})
	}

	// Add parameters as query params or path variables
	for _, param := range route.Parameters {
		if param.In == "query" {
			queryParams = append(queryParams, models.PostmanQueryParam{
				Key:         param.Name,
				Value:       fmt.Sprintf("%v", param.Example),
				Description: param.Description,
				Disabled:    !param.Required,
			})
		}
	}

	// Create request body
	var body *models.PostmanBody
	if route.RequestBody != nil && len(route.RequestBody) > 0 {
		bodyJSON, _ := json.MarshalIndent(route.RequestBody, "", "  ")
		body = &models.PostmanBody{
			Mode: "raw",
			Raw:  string(bodyJSON),
			Options: map[string]any{
				"raw": map[string]any{
					"language": "json",
				},
			},
		}
	}

	// Create example response
	var responses []models.PostmanResponse
	if route.Response != nil && len(route.Response) > 0 {
		respJSON, _ := json.MarshalIndent(route.Response, "", "  ")
		responses = append(responses, models.PostmanResponse{
			Name:   "Success Response",
			Status: "OK",
			Code:   200,
			Header: []models.PostmanHeader{
				{
					Key:   "Content-Type",
					Value: "application/json",
				},
			},
			Body: string(respJSON),
		})
	}

	return models.PostmanItem{
		Name:        fmt.Sprintf("%s %s", route.Method, route.Path),
		Description: route.Description,
		Request: &models.PostmanRequest{
			Method: route.Method,
			Header: headers,
			Body:   body,
			URL: models.PostmanURL{
				Raw:   fmt.Sprintf("{{baseUrl}}%s", route.Path),
				Host:  []string{"{{baseUrl}}"},
				Path:  pathSegments,
				Query: queryParams,
			},
			Description: route.Description,
		},
		Response: responses,
	}
}

func (c *Client) updateExistingItem(collection *models.PostmanCollection, route models.APIRoute) bool {
	routeName := fmt.Sprintf("%s %s", route.Method, route.Path)

	for i, item := range collection.Items {
		if item.Name == routeName || (item.Request != nil &&
			item.Request.Method == route.Method &&
			item.Request.URL.Raw == fmt.Sprintf("{{baseUrl}}%s", route.Path)) {

			// Update the existing item
			collection.Items[i] = c.convertRouteToPostmanItem(route)
			return true
		}
	}
	return false
}

func (c *Client) markItemAsDeprecated(collection *models.PostmanCollection, route models.APIRoute) bool {
	routeName := fmt.Sprintf("%s %s", route.Method, route.Path)

	for i, item := range collection.Items {
		if item.Name == routeName || (item.Request != nil &&
			item.Request.Method == route.Method &&
			item.Request.URL.Raw == fmt.Sprintf("{{baseUrl}}%s", route.Path)) {

			// Mark as deprecated by adding to description
			if collection.Items[i].Description == "" {
				collection.Items[i].Description = "[DEPRECATED] This endpoint is deprecated."
			} else {
				collection.Items[i].Description = "[DEPRECATED] " + collection.Items[i].Description
			}

			// Also update the name
			if collection.Items[i].Name != "" && collection.Items[i].Name[:12] != "[DEPRECATED]" {
				collection.Items[i].Name = "[DEPRECATED] " + collection.Items[i].Name
			}

			return true
		}
	}
	return false
}
