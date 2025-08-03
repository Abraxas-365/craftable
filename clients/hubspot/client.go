package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/logx"
)

// Client represents a HubSpot API client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Config holds configuration for the HubSpot client
type Config struct {
	Token   string        `json:"token"`
	BaseURL string        `json:"baseUrl"`
	Timeout time.Duration `json:"timeout"`
}

// NewClient creates a new HubSpot API client
func NewClient(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.hubapi.com"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL: config.BaseURL,
		token:   config.Token,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// SetTimeout sets the HTTP client timeout
func (c *Client) SetTimeout(timeout time.Duration) {
	c.httpClient.Timeout = timeout
}

// TestConnection tests the API connection and token validity
func (c *Client) TestConnection(ctx context.Context) error {
	_, err := c.GetAccountInfo(ctx)
	return err
}

// GetAccountInfo retrieves account information
func (c *Client) GetAccountInfo(ctx context.Context) (map[string]any, error) {
	logx.Debug("Getting HubSpot account information")

	var result map[string]any
	err := c.Get(ctx, "/account-info/v3/details", nil, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ============================================================================
// BASIC HTTP METHODS
// ============================================================================

// Get performs a GET request to the HubSpot API
func (c *Client) Get(ctx context.Context, endpoint string, params map[string]string, result any) error {
	return c.request(ctx, "GET", endpoint, params, nil, result)
}

// Post performs a POST request to the HubSpot API
func (c *Client) Post(ctx context.Context, endpoint string, body any, result any) error {
	return c.request(ctx, "POST", endpoint, nil, body, result)
}

// Put performs a PUT request to the HubSpot API
func (c *Client) Put(ctx context.Context, endpoint string, body any, result any) error {
	return c.request(ctx, "PUT", endpoint, nil, body, result)
}

// Patch performs a PATCH request to the HubSpot API
func (c *Client) Patch(ctx context.Context, endpoint string, body any, result any) error {
	return c.request(ctx, "PATCH", endpoint, nil, body, result)
}

// Delete performs a DELETE request to the HubSpot API
func (c *Client) Delete(ctx context.Context, endpoint string) error {
	return c.request(ctx, "DELETE", endpoint, nil, nil, nil)
}

// PostWithParams performs a POST request with query parameters
func (c *Client) PostWithParams(ctx context.Context, endpoint string, params map[string]string, body any, result any) error {
	return c.request(ctx, "POST", endpoint, params, body, result)
}

// PutWithParams performs a PUT request with query parameters
func (c *Client) PutWithParams(ctx context.Context, endpoint string, params map[string]string, body any, result any) error {
	return c.request(ctx, "PUT", endpoint, params, body, result)
}

// PatchWithParams performs a PATCH request with query parameters
func (c *Client) PatchWithParams(ctx context.Context, endpoint string, params map[string]string, body any, result any) error {
	return c.request(ctx, "PATCH", endpoint, params, body, result)
}

// DeleteWithParams performs a DELETE request with query parameters
func (c *Client) DeleteWithParams(ctx context.Context, endpoint string, params map[string]string) error {
	return c.request(ctx, "DELETE", endpoint, params, nil, nil)
}

// ============================================================================
// WORKFLOW METHODS
// ============================================================================

// GetAllWorkflows fetches all workflows from HubSpot
func (c *Client) GetAllWorkflows(ctx context.Context) (*WorkflowListResponse, error) {
	logx.Debug("Fetching all workflows from HubSpot")

	var response WorkflowListResponse
	err := c.Get(ctx, "/automation/v3/workflows", nil, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetWorkflowByID fetches a single workflow by ID
func (c *Client) GetWorkflowByID(ctx context.Context, workflowID int) (*Workflow, error) {
	logx.Debug("Fetching workflow by ID: %d", workflowID)

	var workflow Workflow
	endpoint := fmt.Sprintf("/automation/v3/workflows/%d", workflowID)
	err := c.Get(ctx, endpoint, nil, &workflow)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("workflow", workflowID)
		}
		return nil, err
	}

	return &workflow, nil
}

// GetWorkflowsByIDs fetches multiple workflows by their IDs
func (c *Client) GetWorkflowsByIDs(ctx context.Context, workflowIDs []int) ([]*Workflow, error) {
	var workflows []*Workflow
	var errors []error

	for _, id := range workflowIDs {
		workflow, err := c.GetWorkflowByID(ctx, id)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		workflows = append(workflows, workflow)
	}

	// If all requests failed, return the first error
	if len(errors) == len(workflowIDs) && len(errors) > 0 {
		return nil, errors[0]
	}

	return workflows, nil
}

// CreateWorkflow creates a new workflow
func (c *Client) CreateWorkflow(ctx context.Context, workflow *WorkflowCreateRequest) (*Workflow, error) {
	logx.Debug("Creating workflow: %s", workflow.Name)

	var result Workflow
	err := c.Post(ctx, "/automation/v3/workflows", workflow, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateWorkflow updates an existing workflow
func (c *Client) UpdateWorkflow(ctx context.Context, workflowID int, workflow *WorkflowUpdateRequest) (*Workflow, error) {
	logx.Debug("Updating workflow: %d", workflowID)

	var result Workflow
	endpoint := fmt.Sprintf("/automation/v3/workflows/%d", workflowID)
	err := c.Put(ctx, endpoint, workflow, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("workflow", workflowID)
		}
		return nil, err
	}

	return &result, nil
}

// DeleteWorkflow deletes a workflow
func (c *Client) DeleteWorkflow(ctx context.Context, workflowID int) error {
	logx.Debug("Deleting workflow: %d", workflowID)

	endpoint := fmt.Sprintf("/automation/v3/workflows/%d", workflowID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("workflow", workflowID)
		}
		return err
	}

	return nil
}

// EnableWorkflow enables a workflow
func (c *Client) EnableWorkflow(ctx context.Context, workflowID int) error {
	enabled := true
	updateReq := &WorkflowUpdateRequest{Enabled: &enabled}
	_, err := c.UpdateWorkflow(ctx, workflowID, updateReq)
	return err
}

// DisableWorkflow disables a workflow
func (c *Client) DisableWorkflow(ctx context.Context, workflowID int) error {
	enabled := false
	updateReq := &WorkflowUpdateRequest{Enabled: &enabled}
	_, err := c.UpdateWorkflow(ctx, workflowID, updateReq)
	return err
}

// SearchWorkflows searches workflows by name or description
func (c *Client) SearchWorkflows(ctx context.Context, query string) ([]*Workflow, error) {
	// HubSpot doesn't have native search for workflows, so we get all and filter
	response, err := c.GetAllWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*Workflow
	queryLower := strings.ToLower(query)

	for i := range response.Workflows {
		workflow := &response.Workflows[i]
		if strings.Contains(strings.ToLower(workflow.Name), queryLower) {
			filtered = append(filtered, workflow)
		}
	}

	return filtered, nil
}

// ============================================================================
// CONTACT METHODS
// ============================================================================

// GetAllContacts fetches all contacts
func (c *Client) GetAllContacts(ctx context.Context, properties []string, limit int, after string) (*ContactListResponse, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if after != "" {
		params["after"] = after
	}

	var response ContactListResponse
	err := c.Get(ctx, "/crm/v3/objects/contacts", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetContactByID fetches a contact by ID
func (c *Client) GetContactByID(ctx context.Context, contactID string, properties []string) (*Contact, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}

	var contact Contact
	endpoint := fmt.Sprintf("/crm/v3/objects/contacts/%s", contactID)
	err := c.Get(ctx, endpoint, params, &contact)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("contact", contactID)
		}
		return nil, err
	}

	return &contact, nil
}

// CreateContact creates a new contact
func (c *Client) CreateContact(ctx context.Context, contact *ContactInput) (*Contact, error) {
	var result Contact
	err := c.Post(ctx, "/crm/v3/objects/contacts", contact, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateContact updates an existing contact
func (c *Client) UpdateContact(ctx context.Context, contactID string, contact *ContactInput) (*Contact, error) {
	var result Contact
	endpoint := fmt.Sprintf("/crm/v3/objects/contacts/%s", contactID)
	err := c.Patch(ctx, endpoint, contact, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("contact", contactID)
		}
		return nil, err
	}

	return &result, nil
}

// DeleteContact deletes a contact
func (c *Client) DeleteContact(ctx context.Context, contactID string) error {
	endpoint := fmt.Sprintf("/crm/v3/objects/contacts/%s", contactID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("contact", contactID)
		}
		return err
	}

	return nil
}

// SearchContacts searches for contacts
func (c *Client) SearchContacts(ctx context.Context, searchReq *SearchRequest) (*ContactSearchResponse, error) {
	var response ContactSearchResponse
	err := c.Post(ctx, "/crm/v3/objects/contacts/search", searchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ============================================================================
// COMPANY METHODS
// ============================================================================

// GetAllCompanies fetches all companies
func (c *Client) GetAllCompanies(ctx context.Context, properties []string, limit int, after string) (*CompanyListResponse, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if after != "" {
		params["after"] = after
	}

	var response CompanyListResponse
	err := c.Get(ctx, "/crm/v3/objects/companies", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetCompanyByID fetches a company by ID
func (c *Client) GetCompanyByID(ctx context.Context, companyID string, properties []string) (*Company, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}

	var company Company
	endpoint := fmt.Sprintf("/crm/v3/objects/companies/%s", companyID)
	err := c.Get(ctx, endpoint, params, &company)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("company", companyID)
		}
		return nil, err
	}

	return &company, nil
}

// CreateCompany creates a new company
func (c *Client) CreateCompany(ctx context.Context, company *CompanyInput) (*Company, error) {
	var result Company
	err := c.Post(ctx, "/crm/v3/objects/companies", company, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateCompany updates an existing company
func (c *Client) UpdateCompany(ctx context.Context, companyID string, company *CompanyInput) (*Company, error) {
	var result Company
	endpoint := fmt.Sprintf("/crm/v3/objects/companies/%s", companyID)
	err := c.Patch(ctx, endpoint, company, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("company", companyID)
		}
		return nil, err
	}

	return &result, nil
}

// DeleteCompany deletes a company
func (c *Client) DeleteCompany(ctx context.Context, companyID string) error {
	endpoint := fmt.Sprintf("/crm/v3/objects/companies/%s", companyID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("company", companyID)
		}
		return err
	}

	return nil
}

// SearchCompanies searches for companies
func (c *Client) SearchCompanies(ctx context.Context, searchReq *SearchRequest) (*CompanySearchResponse, error) {
	var response CompanySearchResponse
	err := c.Post(ctx, "/crm/v3/objects/companies/search", searchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ============================================================================
// DEAL METHODS
// ============================================================================

// GetAllDeals fetches all deals
func (c *Client) GetAllDeals(ctx context.Context, properties []string, limit int, after string) (*DealListResponse, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if after != "" {
		params["after"] = after
	}

	var response DealListResponse
	err := c.Get(ctx, "/crm/v3/objects/deals", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetDealByID fetches a deal by ID
func (c *Client) GetDealByID(ctx context.Context, dealID string, properties []string) (*Deal, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}

	var deal Deal
	endpoint := fmt.Sprintf("/crm/v3/objects/deals/%s", dealID)
	err := c.Get(ctx, endpoint, params, &deal)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("deal", dealID)
		}
		return nil, err
	}

	return &deal, nil
}

// CreateDeal creates a new deal
func (c *Client) CreateDeal(ctx context.Context, deal *DealInput) (*Deal, error) {
	var result Deal
	err := c.Post(ctx, "/crm/v3/objects/deals", deal, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateDeal updates an existing deal
func (c *Client) UpdateDeal(ctx context.Context, dealID string, deal *DealInput) (*Deal, error) {
	var result Deal
	endpoint := fmt.Sprintf("/crm/v3/objects/deals/%s", dealID)
	err := c.Patch(ctx, endpoint, deal, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("deal", dealID)
		}
		return nil, err
	}

	return &result, nil
}

// DeleteDeal deletes a deal
func (c *Client) DeleteDeal(ctx context.Context, dealID string) error {
	endpoint := fmt.Sprintf("/crm/v3/objects/deals/%s", dealID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("deal", dealID)
		}
		return err
	}

	return nil
}

// SearchDeals searches for deals
func (c *Client) SearchDeals(ctx context.Context, searchReq *SearchRequest) (*DealSearchResponse, error) {
	var response DealSearchResponse
	err := c.Post(ctx, "/crm/v3/objects/deals/search", searchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ============================================================================
// TICKET METHODS
// ============================================================================

// GetAllTickets fetches all tickets
func (c *Client) GetAllTickets(ctx context.Context, properties []string, limit int, after string) (*TicketListResponse, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if after != "" {
		params["after"] = after
	}

	var response TicketListResponse
	err := c.Get(ctx, "/crm/v3/objects/tickets", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetTicketByID fetches a ticket by ID
func (c *Client) GetTicketByID(ctx context.Context, ticketID string, properties []string) (*Ticket, error) {
	params := make(map[string]string)
	if len(properties) > 0 {
		params["properties"] = strings.Join(properties, ",")
	}

	var ticket Ticket
	endpoint := fmt.Sprintf("/crm/v3/objects/tickets/%s", ticketID)
	err := c.Get(ctx, endpoint, params, &ticket)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("ticket", ticketID)
		}
		return nil, err
	}

	return &ticket, nil
}

// CreateTicket creates a new ticket
func (c *Client) CreateTicket(ctx context.Context, ticket *TicketInput) (*Ticket, error) {
	var result Ticket
	err := c.Post(ctx, "/crm/v3/objects/tickets", ticket, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateTicket updates an existing ticket
func (c *Client) UpdateTicket(ctx context.Context, ticketID string, ticket *TicketInput) (*Ticket, error) {
	var result Ticket
	endpoint := fmt.Sprintf("/crm/v3/objects/tickets/%s", ticketID)
	err := c.Patch(ctx, endpoint, ticket, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("ticket", ticketID)
		}
		return nil, err
	}

	return &result, nil
}

// DeleteTicket deletes a ticket
func (c *Client) DeleteTicket(ctx context.Context, ticketID string) error {
	endpoint := fmt.Sprintf("/crm/v3/objects/tickets/%s", ticketID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("ticket", ticketID)
		}
		return err
	}

	return nil
}

// SearchTickets searches for tickets
func (c *Client) SearchTickets(ctx context.Context, searchReq *SearchRequest) (*TicketSearchResponse, error) {
	var response TicketSearchResponse
	err := c.Post(ctx, "/crm/v3/objects/tickets/search", searchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ============================================================================
// OWNER METHODS
// ============================================================================

// GetAllOwners fetches all owners
func (c *Client) GetAllOwners(ctx context.Context, limit int, after string) (*OwnerListResponse, error) {
	params := make(map[string]string)
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if after != "" {
		params["after"] = after
	}

	var response OwnerListResponse
	err := c.Get(ctx, "/crm/v3/owners", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetOwnerByID fetches an owner by ID
func (c *Client) GetOwnerByID(ctx context.Context, ownerID string) (*Owner, error) {
	var owner Owner
	endpoint := fmt.Sprintf("/crm/v3/owners/%s", ownerID)
	err := c.Get(ctx, endpoint, nil, &owner)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("owner", ownerID)
		}
		return nil, err
	}

	return &owner, nil
}

// ============================================================================
// PIPELINE METHODS
// ============================================================================

// GetAllPipelines fetches all pipelines for an object type
func (c *Client) GetAllPipelines(ctx context.Context, objectType string) ([]Pipeline, error) {
	var pipelines []Pipeline
	endpoint := fmt.Sprintf("/crm/v3/pipelines/%s", objectType)
	err := c.Get(ctx, endpoint, nil, &pipelines)
	if err != nil {
		return nil, err
	}

	return pipelines, nil
}

// GetPipelineByID fetches a pipeline by ID
func (c *Client) GetPipelineByID(ctx context.Context, objectType, pipelineID string) (*Pipeline, error) {
	var pipeline Pipeline
	endpoint := fmt.Sprintf("/crm/v3/pipelines/%s/%s", objectType, pipelineID)
	err := c.Get(ctx, endpoint, nil, &pipeline)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("pipeline", pipelineID)
		}
		return nil, err
	}

	return &pipeline, nil
}

// ============================================================================
// FILE METHODS
// ============================================================================

// UploadFile uploads a file to HubSpot
func (c *Client) UploadFile(ctx context.Context, fileName string, fileData []byte, options *FileUploadOptions) (*File, error) {
	// Create multipart form
	var requestBody bytes.Buffer
	writer := NewMultipartWriter(&requestBody)

	// Add file
	if err := writer.WriteFile("file", fileName, fileData); err != nil {
		return nil, Registry.NewWithCause(ErrHubSpotInvalidData, err)
	}

	// Add options if provided
	if options != nil {
		if options.Access != "" {
			writer.WriteField("access", options.Access)
		}
		if options.TTL != "" {
			writer.WriteField("ttl", options.TTL)
		}
		if options.Overwrite {
			writer.WriteField("overwrite", "true")
		}
		if options.DuplicateValidationStrategy != "" {
			writer.WriteField("duplicateValidationStrategy", options.DuplicateValidationStrategy)
		}
		if options.DuplicateValidationScope != "" {
			writer.WriteField("duplicateValidationScope", options.DuplicateValidationScope)
		}
	}

	contentType := writer.FormDataContentType()
	if err := writer.Close(); err != nil {
		return nil, Registry.NewWithCause(ErrHubSpotInvalidData, err)
	}

	// Create request
	reqURL := c.baseURL + "/filemanager/api/v3/files/upload"
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, &requestBody)
	if err != nil {
		return nil, Registry.NewWithCause(ErrHubSpotConnection, err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", contentType)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, Registry.NewWithCause(ErrHubSpotConnection, err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, Registry.NewWithCause(ErrHubSpotAPIError, err)
	}

	// Handle errors
	if resp.StatusCode >= 400 {
		return nil, c.handleHTTPError(resp.StatusCode, respBody, resp.Header)
	}

	// Parse response
	var file File
	if err := json.Unmarshal(respBody, &file); err != nil {
		return nil, Registry.NewWithCause(ErrHubSpotParsingError, err)
	}

	return &file, nil
}

// GetFileByID fetches a file by ID
func (c *Client) GetFileByID(ctx context.Context, fileID string) (*File, error) {
	var file File
	endpoint := fmt.Sprintf("/filemanager/api/v3/files/%s", fileID)
	err := c.Get(ctx, endpoint, nil, &file)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("file", fileID)
		}
		return nil, err
	}

	return &file, nil
}

// DeleteFile deletes a file
func (c *Client) DeleteFile(ctx context.Context, fileID string) error {
	endpoint := fmt.Sprintf("/filemanager/api/v3/files/%s", fileID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("file", fileID)
		}
		return err
	}

	return nil
}

// ============================================================================
// BATCH OPERATIONS
// ============================================================================

// BatchRequest performs a batch request to HubSpot
func (c *Client) BatchRequest(ctx context.Context, endpoint string, batchReq *BatchRequest, result any) error {
	return c.Post(ctx, endpoint, batchReq, result)
}

// BatchCreateContacts creates multiple contacts in a batch
func (c *Client) BatchCreateContacts(ctx context.Context, contacts []ContactInput) (*BatchResponse, error) {
	inputs := make([]any, len(contacts))
	for i, contact := range contacts {
		inputs[i] = contact
	}

	batchReq := &BatchRequest{Inputs: inputs}
	var response BatchResponse
	err := c.BatchRequest(ctx, "/crm/v3/objects/contacts/batch/create", batchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// BatchUpdateContacts updates multiple contacts in a batch
func (c *Client) BatchUpdateContacts(ctx context.Context, contacts []ContactInput) (*BatchResponse, error) {
	inputs := make([]any, len(contacts))
	for i, contact := range contacts {
		inputs[i] = contact
	}

	batchReq := &BatchRequest{Inputs: inputs}
	var response BatchResponse
	err := c.BatchRequest(ctx, "/crm/v3/objects/contacts/batch/update", batchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ============================================================================
// SEARCH OPERATIONS
// ============================================================================

// SearchRequest performs a search request to HubSpot
func (c *Client) SearchRequest(ctx context.Context, endpoint string, searchReq *SearchRequest, result any) error {
	return c.Post(ctx, endpoint, searchReq, result)
}

// ============================================================================
// UTILITY METHODS
// ============================================================================

// request performs the actual HTTP request
func (c *Client) request(ctx context.Context, method, endpoint string, params map[string]string, body any, result any) error {
	// Build URL
	reqURL := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		reqURL += "?" + values.Encode()
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return Registry.NewWithCause(ErrHubSpotInvalidData, err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotConnection, err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Log request
	logx.Debug("Making HubSpot API request: %s %s", method, reqURL)

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotConnection, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotAPIError, err)
	}

	// Handle HTTP errors
	if resp.StatusCode >= 400 {
		return c.handleHTTPError(resp.StatusCode, respBody, resp.Header)
	}

	// Parse response if result is provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return Registry.NewWithCause(ErrHubSpotParsingError, err).
				WithDetail("responseBody", string(respBody))
		}
	}

	logx.Debug("HubSpot API request completed successfully")
	return nil
}

// handleHTTPError converts HTTP status codes to appropriate errors
func (c *Client) handleHTTPError(statusCode int, body []byte, headers http.Header) error {
	bodyStr := string(body)

	// Try to parse HubSpot error response
	var hubspotError HubSpotErrorResponse
	if err := json.Unmarshal(body, &hubspotError); err == nil && hubspotError.Message != "" {
		bodyStr = hubspotError.Message
	}

	switch statusCode {
	case http.StatusUnauthorized:
		return Registry.New(ErrHubSpotUnauthorized).WithDetail("response", bodyStr)
	case http.StatusForbidden:
		return Registry.New(ErrHubSpotForbidden).WithDetail("response", bodyStr)
	case http.StatusTooManyRequests:
		err := Registry.New(ErrHubSpotRateLimit).WithDetail("response", bodyStr)
		if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
			err.WithDetail("retryAfter", retryAfter)
		}
		return err
	case http.StatusNotFound:
		return Registry.New(ErrHubSpotNotFound).WithDetail("response", bodyStr)
	case http.StatusConflict:
		return Registry.New(ErrHubSpotConflict).WithDetail("response", bodyStr)
	case http.StatusBadRequest:
		return Registry.New(ErrHubSpotBadRequest).WithDetail("response", bodyStr)
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return Registry.New(ErrHubSpotTimeout).WithDetail("response", bodyStr)
	case http.StatusServiceUnavailable, http.StatusBadGateway:
		return Registry.New(ErrHubSpotUnavailable).WithDetail("response", bodyStr)
	default:
		return Registry.New(ErrHubSpotAPIError).
			WithDetail("statusCode", statusCode).
			WithDetail("response", bodyStr)
	}
}

// StreamRequest performs a streaming request (useful for large responses)
func (c *Client) StreamRequest(ctx context.Context, method, endpoint string, params map[string]string, body any, callback func([]byte) error) error {
	// Build URL
	reqURL := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		reqURL += "?" + values.Encode()
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return Registry.NewWithCause(ErrHubSpotInvalidData, err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotConnection, err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotConnection, err)
	}
	defer resp.Body.Close()

	// Handle HTTP errors
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return c.handleHTTPError(resp.StatusCode, respBody, resp.Header)
	}

	// Stream response
	buffer := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			if callbackErr := callback(buffer[:n]); callbackErr != nil {
				return callbackErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return Registry.NewWithCause(ErrHubSpotAPIError, err)
		}
	}

	return nil
}

// RequestWithCustomHeaders performs a request with custom headers
func (c *Client) RequestWithCustomHeaders(ctx context.Context, method, endpoint string, params map[string]string, headers map[string]string, body any, result any) error {
	// Build URL
	reqURL := c.baseURL + endpoint
	if params != nil && len(params) > 0 {
		values := url.Values{}
		for k, v := range params {
			values.Add(k, v)
		}
		reqURL += "?" + values.Encode()
	}

	// Prepare request body
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return Registry.NewWithCause(ErrHubSpotInvalidData, err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotConnection, err)
	}

	// Set default headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotConnection, err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return Registry.NewWithCause(ErrHubSpotAPIError, err)
	}

	// Handle HTTP errors
	if resp.StatusCode >= 400 {
		return c.handleHTTPError(resp.StatusCode, respBody, resp.Header)
	}

	// Parse response if result is provided
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return Registry.NewWithCause(ErrHubSpotParsingError, err).
				WithDetail("responseBody", string(respBody))
		}
	}

	return nil
}

// GetBaseURL returns the base URL being used
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// SetBaseURL sets the base URL
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// GetToken returns the current API token (masked for security)
func (c *Client) GetToken() string {
	if len(c.token) > 8 {
		return c.token[:4] + "****" + c.token[len(c.token)-4:]
	}
	return "****"
}

// SetToken sets the API token
func (c *Client) SetToken(token string) {
	c.token = token
}

// GetHTTPClient returns the underlying HTTP client
func (c *Client) GetHTTPClient() *http.Client {
	return c.httpClient
}

// SetHTTPClient sets a custom HTTP client
func (c *Client) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// ============================================================================
// MULTIPART WRITER
// ============================================================================

// MultipartWriter wraps multipart.Writer with convenience methods
type MultipartWriter struct {
	writer *multipart.Writer
	buffer *bytes.Buffer
}

// NewMultipartWriter creates a new MultipartWriter
func NewMultipartWriter(buffer *bytes.Buffer) *MultipartWriter {
	writer := multipart.NewWriter(buffer)
	return &MultipartWriter{
		writer: writer,
		buffer: buffer,
	}
}

// WriteField writes a form field
func (mw *MultipartWriter) WriteField(fieldname, value string) error {
	return mw.writer.WriteField(fieldname, value)
}

// WriteFile writes a file field
func (mw *MultipartWriter) WriteFile(fieldname, filename string, data []byte) error {
	part, err := mw.writer.CreateFormFile(fieldname, filename)
	if err != nil {
		return err
	}
	_, err = part.Write(data)
	return err
}

// WriteReader writes a file from an io.Reader
func (mw *MultipartWriter) WriteReader(fieldname, filename string, reader io.Reader) error {
	part, err := mw.writer.CreateFormFile(fieldname, filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, reader)
	return err
}

// FormDataContentType returns the Content-Type for the form
func (mw *MultipartWriter) FormDataContentType() string {
	return mw.writer.FormDataContentType()
}

// Close closes the multipart writer
func (mw *MultipartWriter) Close() error {
	return mw.writer.Close()
}

