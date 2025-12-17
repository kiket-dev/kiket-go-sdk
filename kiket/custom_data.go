package kiket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

// customDataClient implements the CustomDataClient interface.
type customDataClient struct {
	client    Client
	projectID interface{}
}

// NewCustomDataClient creates a new custom data client.
func NewCustomDataClient(client Client, projectID interface{}) CustomDataClient {
	return &customDataClient{
		client:    client,
		projectID: projectID,
	}
}

func (c *customDataClient) buildPath(moduleKey, table string, recordID interface{}) string {
	base := fmt.Sprintf("%s/ext/custom_data/%s/%s",
		apiPrefix,
		url.PathEscape(moduleKey),
		url.PathEscape(table))

	if recordID != nil {
		return fmt.Sprintf("%s/%v", base, recordID)
	}
	return base
}

func (c *customDataClient) buildParams(limit int, filters map[string]interface{}) map[string]string {
	params := map[string]string{
		"project_id": fmt.Sprintf("%v", c.projectID),
	}

	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}

	if filters != nil && len(filters) > 0 {
		filtersJSON, _ := json.Marshal(filters)
		params["filters"] = string(filtersJSON)
	}

	return params
}

func (c *customDataClient) List(ctx context.Context, moduleKey, table string, opts *CustomDataListOptions) (*CustomDataListResponse, error) {
	if c.projectID == nil || c.projectID == "" {
		return nil, errors.New("project_id is required for custom data operations")
	}

	var limit int
	var filters map[string]interface{}
	if opts != nil {
		limit = opts.Limit
		filters = opts.Filters
	}

	path := c.buildPath(moduleKey, table, nil)
	resp, err := c.client.Get(ctx, path, &RequestOptions{
		Params: c.buildParams(limit, filters),
	})
	if err != nil {
		return nil, err
	}

	var result CustomDataListResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *customDataClient) Get(ctx context.Context, moduleKey, table string, recordID interface{}) (*CustomDataRecordResponse, error) {
	if c.projectID == nil || c.projectID == "" {
		return nil, errors.New("project_id is required for custom data operations")
	}

	path := c.buildPath(moduleKey, table, recordID)
	resp, err := c.client.Get(ctx, path, &RequestOptions{
		Params: c.buildParams(0, nil),
	})
	if err != nil {
		return nil, err
	}

	var result CustomDataRecordResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *customDataClient) Create(ctx context.Context, moduleKey, table string, record map[string]interface{}) (*CustomDataRecordResponse, error) {
	if c.projectID == nil || c.projectID == "" {
		return nil, errors.New("project_id is required for custom data operations")
	}

	path := c.buildPath(moduleKey, table, nil)
	resp, err := c.client.Post(ctx, path, map[string]interface{}{"record": record}, &RequestOptions{
		Params: c.buildParams(0, nil),
	})
	if err != nil {
		return nil, err
	}

	var result CustomDataRecordResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *customDataClient) Update(ctx context.Context, moduleKey, table string, recordID interface{}, record map[string]interface{}) (*CustomDataRecordResponse, error) {
	if c.projectID == nil || c.projectID == "" {
		return nil, errors.New("project_id is required for custom data operations")
	}

	path := c.buildPath(moduleKey, table, recordID)
	resp, err := c.client.Patch(ctx, path, map[string]interface{}{"record": record}, &RequestOptions{
		Params: c.buildParams(0, nil),
	})
	if err != nil {
		return nil, err
	}

	var result CustomDataRecordResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func (c *customDataClient) Delete(ctx context.Context, moduleKey, table string, recordID interface{}) error {
	if c.projectID == nil || c.projectID == "" {
		return errors.New("project_id is required for custom data operations")
	}

	path := c.buildPath(moduleKey, table, recordID)
	_, err := c.client.Delete(ctx, path, &RequestOptions{
		Params: c.buildParams(0, nil),
	})
	return err
}
