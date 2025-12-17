package kiket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

const slaPath = "/api/v1/ext/sla/events"

// slaEventsClient implements the SLAEventsClient interface.
type slaEventsClient struct {
	client    Client
	projectID interface{}
}

// NewSLAEventsClient creates a new SLA events client.
func NewSLAEventsClient(client Client, projectID interface{}) SLAEventsClient {
	return &slaEventsClient{
		client:    client,
		projectID: projectID,
	}
}

func (c *slaEventsClient) buildParams(opts *SLAEventsListOptions) map[string]string {
	params := map[string]string{
		"project_id": fmt.Sprintf("%v", c.projectID),
	}

	if opts != nil {
		if opts.IssueID != nil {
			params["issue_id"] = fmt.Sprintf("%v", opts.IssueID)
		}
		if opts.State != "" {
			params["state"] = opts.State
		}
		if opts.Limit > 0 {
			params["limit"] = strconv.Itoa(opts.Limit)
		}
	}

	return params
}

func (c *slaEventsClient) List(ctx context.Context, opts *SLAEventsListOptions) (*SLAEventsListResponse, error) {
	if c.projectID == nil || c.projectID == "" {
		return nil, errors.New("projectID is required for SLA events")
	}

	resp, err := c.client.Get(ctx, slaPath, &RequestOptions{
		Params: c.buildParams(opts),
	})
	if err != nil {
		return nil, err
	}

	var result SLAEventsListResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}
