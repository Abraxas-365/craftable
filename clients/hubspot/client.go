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

// ============================================================================
// PROPERTY METHODS
// ============================================================================

// GetAllProperties fetches all properties for an object type
func (c *Client) GetAllProperties(ctx context.Context, objectType string, archived bool) (*PropertyListResponse, error) {
	logx.Debug("Fetching all properties for object type: %s", objectType)

	params := make(map[string]string)
	if archived {
		params["archived"] = "true"
	}

	var response PropertyListResponse
	endpoint := fmt.Sprintf("/crm/v3/properties/%s", objectType)
	err := c.Get(ctx, endpoint, params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetPropertyByName fetches a property by name for an object type
func (c *Client) GetPropertyByName(ctx context.Context, objectType, propertyName string, archived bool) (*PropertyDefinition, error) {
	logx.Debug("Fetching property %s for object type: %s", propertyName, objectType)

	params := make(map[string]string)
	if archived {
		params["archived"] = "true"
	}

	var property PropertyDefinition
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/%s", objectType, propertyName)
	err := c.Get(ctx, endpoint, params, &property)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("property", propertyName)
		}
		return nil, err
	}

	return &property, nil
}

// CreateProperty creates a new property for an object type
func (c *Client) CreateProperty(ctx context.Context, objectType string, property *PropertyCreateRequest) (*PropertyDefinition, error) {
	logx.Debug("Creating property %s for object type: %s", property.Name, objectType)

	var result PropertyDefinition
	endpoint := fmt.Sprintf("/crm/v3/properties/%s", objectType)
	err := c.Post(ctx, endpoint, property, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateProperty updates an existing property
func (c *Client) UpdateProperty(ctx context.Context, objectType, propertyName string, property *PropertyUpdateRequest) (*PropertyDefinition, error) {
	logx.Debug("Updating property %s for object type: %s", propertyName, objectType)

	var result PropertyDefinition
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/%s", objectType, propertyName)
	err := c.Patch(ctx, endpoint, property, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("property", propertyName)
		}
		return nil, err
	}

	return &result, nil
}

// DeleteProperty deletes a property (archives it)
func (c *Client) DeleteProperty(ctx context.Context, objectType, propertyName string) error {
	logx.Debug("Deleting property %s for object type: %s", propertyName, objectType)

	endpoint := fmt.Sprintf("/crm/v3/properties/%s/%s", objectType, propertyName)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("property", propertyName)
		}
		return err
	}

	return nil
}

// BatchCreateProperties creates multiple properties in a batch
func (c *Client) BatchCreateProperties(ctx context.Context, objectType string, properties []PropertyCreateRequest) (*BatchResponse, error) {
	logx.Debug("Batch creating %d properties for object type: %s", len(properties), objectType)

	inputs := make([]any, len(properties))
	for i, property := range properties {
		inputs[i] = property
	}

	batchReq := &BatchRequest{Inputs: inputs}
	var response BatchResponse
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/batch/create", objectType)
	err := c.BatchRequest(ctx, endpoint, batchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// BatchUpdateProperties updates multiple properties in a batch
func (c *Client) BatchUpdateProperties(ctx context.Context, objectType string, properties []PropertyUpdateRequest) (*BatchResponse, error) {
	logx.Debug("Batch updating %d properties for object type: %s", len(properties), objectType)

	inputs := make([]any, len(properties))
	for i, property := range properties {
		inputs[i] = property
	}

	batchReq := &BatchRequest{Inputs: inputs}
	var response BatchResponse
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/batch/update", objectType)
	err := c.BatchRequest(ctx, endpoint, batchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// BatchArchiveProperties archives multiple properties in a batch
func (c *Client) BatchArchiveProperties(ctx context.Context, objectType string, propertyNames []string) (*BatchResponse, error) {
	logx.Debug("Batch archiving %d properties for object type: %s", len(propertyNames), objectType)

	inputs := make([]any, len(propertyNames))
	for i, name := range propertyNames {
		inputs[i] = map[string]string{"name": name}
	}

	batchReq := &BatchRequest{Inputs: inputs}
	var response BatchResponse
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/batch/archive", objectType)
	err := c.BatchRequest(ctx, endpoint, batchReq, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ============================================================================
// PROPERTY GROUP METHODS
// ============================================================================

// GetAllPropertyGroups fetches all property groups for an object type
func (c *Client) GetAllPropertyGroups(ctx context.Context, objectType string) (*PropertyGroupListResponse, error) {
	logx.Debug("Fetching all property groups for object type: %s", objectType)

	var response PropertyGroupListResponse
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/groups", objectType)
	err := c.Get(ctx, endpoint, nil, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetPropertyGroupByName fetches a property group by name for an object type
func (c *Client) GetPropertyGroupByName(ctx context.Context, objectType, groupName string) (*PropertyGroup, error) {
	logx.Debug("Fetching property group %s for object type: %s", groupName, objectType)

	var group PropertyGroup
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/groups/%s", objectType, groupName)
	err := c.Get(ctx, endpoint, nil, &group)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("property group", groupName)
		}
		return nil, err
	}

	return &group, nil
}

// CreatePropertyGroup creates a new property group for an object type
func (c *Client) CreatePropertyGroup(ctx context.Context, objectType string, group *PropertyGroupCreateRequest) (*PropertyGroup, error) {
	logx.Debug("Creating property group %s for object type: %s", group.Name, objectType)

	var result PropertyGroup
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/groups", objectType)
	err := c.Post(ctx, endpoint, group, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdatePropertyGroup updates an existing property group
func (c *Client) UpdatePropertyGroup(ctx context.Context, objectType, groupName string, group *PropertyGroupUpdateRequest) (*PropertyGroup, error) {
	logx.Debug("Updating property group %s for object type: %s", groupName, objectType)

	var result PropertyGroup
	endpoint := fmt.Sprintf("/crm/v3/properties/%s/groups/%s", objectType, groupName)
	err := c.Patch(ctx, endpoint, group, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("property group", groupName)
		}
		return nil, err
	}

	return &result, nil
}

// DeletePropertyGroup deletes a property group (archives it)
func (c *Client) DeletePropertyGroup(ctx context.Context, objectType, groupName string) error {
	logx.Debug("Deleting property group %s for object type: %s", groupName, objectType)

	endpoint := fmt.Sprintf("/crm/v3/properties/%s/groups/%s", objectType, groupName)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("property group", groupName)
		}
		return err
	}

	return nil
}

// ============================================================================
// PROPERTY CONVENIENCE METHODS
// ============================================================================

// GetContactProperties gets all properties for contacts
func (c *Client) GetContactProperties(ctx context.Context, archived bool) (*PropertyListResponse, error) {
	return c.GetAllProperties(ctx, "contacts", archived)
}

// GetCompanyProperties gets all properties for companies
func (c *Client) GetCompanyProperties(ctx context.Context, archived bool) (*PropertyListResponse, error) {
	return c.GetAllProperties(ctx, "companies", archived)
}

// GetDealProperties gets all properties for deals
func (c *Client) GetDealProperties(ctx context.Context, archived bool) (*PropertyListResponse, error) {
	return c.GetAllProperties(ctx, "deals", archived)
}

// GetTicketProperties gets all properties for tickets
func (c *Client) GetTicketProperties(ctx context.Context, archived bool) (*PropertyListResponse, error) {
	return c.GetAllProperties(ctx, "tickets", archived)
}

// CreateContactProperty creates a new contact property
func (c *Client) CreateContactProperty(ctx context.Context, property *PropertyCreateRequest) (*PropertyDefinition, error) {
	return c.CreateProperty(ctx, "contacts", property)
}

// CreateCompanyProperty creates a new company property
func (c *Client) CreateCompanyProperty(ctx context.Context, property *PropertyCreateRequest) (*PropertyDefinition, error) {
	return c.CreateProperty(ctx, "companies", property)
}

// CreateDealProperty creates a new deal property
func (c *Client) CreateDealProperty(ctx context.Context, property *PropertyCreateRequest) (*PropertyDefinition, error) {
	return c.CreateProperty(ctx, "deals", property)
}

// CreateTicketProperty creates a new ticket property
func (c *Client) CreateTicketProperty(ctx context.Context, property *PropertyCreateRequest) (*PropertyDefinition, error) {
	return c.CreateProperty(ctx, "tickets", property)
}

// SearchPropertiesByName searches for properties by name pattern
func (c *Client) SearchPropertiesByName(ctx context.Context, objectType, namePattern string) ([]*PropertyDefinition, error) {
	response, err := c.GetAllProperties(ctx, objectType, false)
	if err != nil {
		return nil, err
	}

	var filtered []*PropertyDefinition
	pattern := strings.ToLower(namePattern)

	for i := range response.Results {
		property := &response.Results[i]
		if strings.Contains(strings.ToLower(property.Name), pattern) ||
			strings.Contains(strings.ToLower(property.Label), pattern) {
			filtered = append(filtered, property)
		}
	}

	return filtered, nil
}

// GetPropertiesByGroup gets all properties in a specific group
func (c *Client) GetPropertiesByGroup(ctx context.Context, objectType, groupName string) ([]*PropertyDefinition, error) {
	response, err := c.GetAllProperties(ctx, objectType, false)
	if err != nil {
		return nil, err
	}

	var filtered []*PropertyDefinition
	for i := range response.Results {
		property := &response.Results[i]
		if property.GroupName == groupName {
			filtered = append(filtered, property)
		}
	}

	return filtered, nil
}

// GetPropertiesByType gets all properties of a specific type
func (c *Client) GetPropertiesByType(ctx context.Context, objectType, propertyType string) ([]*PropertyDefinition, error) {
	response, err := c.GetAllProperties(ctx, objectType, false)
	if err != nil {
		return nil, err
	}

	var filtered []*PropertyDefinition
	for i := range response.Results {
		property := &response.Results[i]
		if property.Type == propertyType {
			filtered = append(filtered, property)
		}
	}

	return filtered, nil
}

// ============================================================================
// FORM METHODS
// ============================================================================

// GetAllForms fetches all forms from HubSpot
func (c *Client) GetAllForms(ctx context.Context) (*FormListResponse, error) {
	logx.Debug("Fetching all forms from HubSpot")

	var hsResponse any // Changed from map[string]interface{} to interface{}
	err := c.Get(ctx, "/forms/v2/forms", nil, &hsResponse)
	if err != nil {
		return nil, err
	}

	// Convert response - HubSpot forms API can return different formats
	var forms []Form

	// Check if response is a map with "results" key
	if responseMap, ok := hsResponse.(map[string]any); ok {
		if results, ok := responseMap["results"].([]any); ok {
			forms = make([]Form, 0, len(results))
			for _, result := range results {
				if formMap, ok := result.(map[string]any); ok {
					form := c.convertHubSpotForm(formMap)
					forms = append(forms, form)
				}
			}
		}
	} else if formsArray, ok := hsResponse.([]any); ok {
		// Sometimes HubSpot returns direct array
		forms = make([]Form, 0, len(formsArray))
		for _, result := range formsArray {
			if formMap, ok := result.(map[string]any); ok {
				form := c.convertHubSpotForm(formMap)
				forms = append(forms, form)
			}
		}
	}

	response := &FormListResponse{
		Results: forms,
		Paging:  nil, // HubSpot forms API v2 doesn't use standard pagination
	}

	logx.Debug("Successfully fetched %d forms", len(forms))
	return response, nil
}

// GetFormByID fetches a single form by ID
func (c *Client) GetFormByID(ctx context.Context, formID string) (*Form, error) {
	logx.Debug("Fetching form by ID: %s", formID)

	var hsForm map[string]any
	endpoint := fmt.Sprintf("/forms/v2/forms/%s", formID)
	err := c.Get(ctx, endpoint, nil, &hsForm)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("form", formID)
		}
		return nil, err
	}

	form := c.convertHubSpotForm(hsForm)
	return &form, nil
}

// CreateForm creates a new form
func (c *Client) CreateForm(ctx context.Context, form *FormCreateRequest) (*Form, error) {
	logx.Debug("Creating form: %s", form.Name)

	var hsForm map[string]any
	err := c.Post(ctx, "/forms/v2/forms", form, &hsForm)
	if err != nil {
		return nil, err
	}

	domainForm := c.convertHubSpotForm(hsForm)
	return &domainForm, nil
}

// UpdateForm updates an existing form
func (c *Client) UpdateForm(ctx context.Context, formID string, form *FormUpdateRequest) (*Form, error) {
	logx.Debug("Updating form: %s", formID)

	var hsForm map[string]any
	endpoint := fmt.Sprintf("/forms/v2/forms/%s", formID)
	err := c.Put(ctx, endpoint, form, &hsForm)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("form", formID)
		}
		return nil, err
	}

	domainForm := c.convertHubSpotForm(hsForm)
	return &domainForm, nil
}

// DeleteForm deletes a form
func (c *Client) DeleteForm(ctx context.Context, formID string) error {
	logx.Debug("Deleting form: %s", formID)

	endpoint := fmt.Sprintf("/forms/v2/forms/%s", formID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("form", formID)
		}
		return err
	}

	return nil
}

// GetFormSubmissions retrieves form submissions
func (c *Client) GetFormSubmissions(ctx context.Context, formID string, limit int, after string) (*FormSubmissionListResponse, error) {
	logx.Debug("Fetching form submissions for form: %s", formID)

	params := make(map[string]string)
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	if after != "" {
		params["after"] = after
	}

	var hsResponse map[string]any
	endpoint := fmt.Sprintf("/form-integrations/v1/submissions/forms/%s", formID)
	err := c.Get(ctx, endpoint, params, &hsResponse)
	if err != nil {
		return nil, err
	}

	// Convert submissions
	var submissions []FormSubmission
	if results, ok := hsResponse["results"].([]any); ok {
		submissions = make([]FormSubmission, 0, len(results))
		for _, result := range results {
			if submissionMap, ok := result.(map[string]any); ok {
				submission := c.convertFormSubmission(submissionMap)
				submissions = append(submissions, submission)
			}
		}
	}

	response := &FormSubmissionListResponse{
		Results: submissions,
		Paging:  nil, // Convert if available in response
	}

	return response, nil
}

// SubmitForm submits a form
func (c *Client) SubmitForm(ctx context.Context, formID string, submission *FormSubmission) error {
	logx.Debug("Submitting form: %s", formID)

	endpoint := fmt.Sprintf("/forms/v3/forms/%s/submit", formID)
	err := c.Post(ctx, endpoint, submission, nil)
	if err != nil {
		return err
	}

	return nil
}

// ============================================================================
// FORM HELPER METHODS
// ============================================================================

// convertHubSpotForm converts HubSpot form response to Form struct
func (c *Client) convertHubSpotForm(hsForm map[string]any) Form {
	form := Form{
		ID:   c.getStringFromMap(hsForm, "guid"),
		Name: c.getStringFromMap(hsForm, "name"),
	}

	// Convert timestamps
	if createdAt := c.getInt64FromMap(hsForm, "createdAt"); createdAt != nil {
		form.CreatedAt = createdAt
	}
	if updatedAt := c.getInt64FromMap(hsForm, "updatedAt"); updatedAt != nil {
		form.UpdatedAt = updatedAt
	}

	// Convert other form properties
	form.SubmitText = c.getStringFromMap(hsForm, "submitText")
	form.Redirect = c.getStringFromMap(hsForm, "redirect")
	form.CssClass = c.getStringFromMap(hsForm, "cssClass")
	form.NotifyRecipients = c.getStringFromMap(hsForm, "notifyRecipients")
	form.LeadNurturingCampaign = c.getStringFromMap(hsForm, "leadNurturingCampaignId")
	form.FollowUpAction = c.getStringFromMap(hsForm, "followUpActionType")

	// Convert form field groups
	if formFieldGroups, ok := hsForm["formFieldGroups"].([]any); ok {
		form.FormFieldGroups = c.convertFormFieldGroups(formFieldGroups)
	}

	return form
}

// convertFormFieldGroups converts HubSpot form field groups
func (c *Client) convertFormFieldGroups(hsGroups []any) []FormFieldGroup {
	groups := make([]FormFieldGroup, 0, len(hsGroups))

	for _, groupInterface := range hsGroups {
		if groupMap, ok := groupInterface.(map[string]any); ok {
			group := FormFieldGroup{
				Default:      c.getBoolFromMap(groupMap, "default"),
				IsSmartGroup: c.getBoolFromMap(groupMap, "isSmartGroup"),
				GroupType:    c.getStringFromMap(groupMap, "groupType"),
			}

			// Convert fields
			if fields, ok := groupMap["fields"].([]any); ok {
				group.Fields = c.convertFormFields(fields)
			}

			// Convert rich text
			if richText, ok := groupMap["richText"].(map[string]any); ok {
				group.RichText = richText
			}

			groups = append(groups, group)
		}
	}

	return groups
}

// convertFormFields converts HubSpot form fields
func (c *Client) convertFormFields(hsFields []any) []FormField {
	fields := make([]FormField, 0, len(hsFields))

	for _, fieldInterface := range hsFields {
		if fieldMap, ok := fieldInterface.(map[string]any); ok {
			field := FormField{
				Name:                 c.getStringFromMap(fieldMap, "name"),
				Label:                c.getStringFromMap(fieldMap, "label"),
				FieldType:            c.getStringFromMap(fieldMap, "fieldType"),
				ObjectTypeId:         c.getStringFromMap(fieldMap, "objectTypeId"),
				Description:          c.getStringFromMap(fieldMap, "description"),
				GroupName:            c.getStringFromMap(fieldMap, "groupName"),
				DisplayOrder:         c.getIntFromMap(fieldMap, "displayOrder"),
				Required:             c.getBoolFromMap(fieldMap, "required"),
				Enabled:              c.getBoolFromMap(fieldMap, "enabled"),
				Hidden:               c.getBoolFromMap(fieldMap, "hidden"),
				DefaultValue:         c.getStringFromMap(fieldMap, "defaultValue"),
				Placeholder:          c.getStringFromMap(fieldMap, "placeholder"),
				UseCountryCodeSelect: c.getBoolFromMap(fieldMap, "useCountryCodeSelect"),
				AllowMultipleFiles:   c.getBoolFromMap(fieldMap, "allowMultipleFiles"),
				LabelHidden:          c.getBoolFromMap(fieldMap, "labelHidden"),
				PropertyObjectType:   c.getStringFromMap(fieldMap, "propertyObjectType"),
			}

			// Convert options
			if options, ok := fieldMap["options"].([]any); ok {
				field.Options = c.convertFormFieldOptions(options)
			}

			// Convert validation, dependent fields, metadata as maps
			if validation, ok := fieldMap["validation"].(map[string]any); ok {
				field.Validation = validation
			}
			if dependentFields, ok := fieldMap["dependentFields"].([]map[string]any); ok {
				field.DependentFields = dependentFields
			}
			if metaData, ok := fieldMap["metaData"].([]map[string]any); ok {
				field.MetaData = metaData
			}

			fields = append(fields, field)
		}
	}

	return fields
}

// convertFormFieldOptions converts HubSpot form field options
func (c *Client) convertFormFieldOptions(hsOptions []any) []FormFieldOption {
	options := make([]FormFieldOption, 0, len(hsOptions))

	for _, optionInterface := range hsOptions {
		if optionMap, ok := optionInterface.(map[string]any); ok {
			option := FormFieldOption{
				Label:        c.getStringFromMap(optionMap, "label"),
				Value:        c.getStringFromMap(optionMap, "value"),
				DisplayOrder: c.getIntFromMap(optionMap, "displayOrder"),
				Selected:     c.getBoolFromMap(optionMap, "selected"),
			}
			options = append(options, option)
		}
	}

	return options
}

// convertFormSubmission converts HubSpot form submission
func (c *Client) convertFormSubmission(hsSubmission map[string]any) FormSubmission {
	submission := FormSubmission{
		FormID:   c.getStringFromMap(hsSubmission, "formGuid"),
		PageUrl:  c.getStringFromMap(hsSubmission, "pageUrl"),
		PageName: c.getStringFromMap(hsSubmission, "pageName"),
	}

	if submittedAt := c.getInt64FromMap(hsSubmission, "submittedAt"); submittedAt != nil {
		submission.SubmittedAt = submittedAt
	}

	// Convert values
	if values, ok := hsSubmission["values"].([]any); ok {
		submission.Values = make([]FormSubmissionValue, 0, len(values))
		for _, valueInterface := range values {
			if valueMap, ok := valueInterface.(map[string]any); ok {
				value := FormSubmissionValue{
					Name:     c.getStringFromMap(valueMap, "name"),
					Value:    c.getStringFromMap(valueMap, "value"),
					Selected: c.getBoolFromMap(valueMap, "selected"),
				}
				submission.Values = append(submission.Values, value)
			}
		}
	}

	return submission
}

// Helper methods for safe map access
func (c *Client) getStringFromMap(m map[string]any, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func (c *Client) getBoolFromMap(m map[string]any, key string) bool {
	if val, ok := m[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (c *Client) getIntFromMap(m map[string]any, key string) int {
	if val, ok := m[key]; ok {
		if i, ok := val.(float64); ok {
			return int(i)
		}
		if i, ok := val.(int); ok {
			return i
		}
	}
	return 0
}

func (c *Client) getInt64FromMap(m map[string]any, key string) *int64 {
	if val, ok := m[key]; ok {
		if i, ok := val.(float64); ok {
			result := int64(i)
			return &result
		}
		if i, ok := val.(int64); ok {
			return &i
		}
	}
	return nil
}

// ============================================================================
// LIST METHODS
// ============================================================================

// GetAllLists fetches all lists from HubSpot
func (c *Client) GetAllLists(ctx context.Context, limit int, offset int) (*ListResponse, error) {
	logx.Debug("Fetching all lists from HubSpot")

	params := make(map[string]string)
	if limit > 0 {
		params["count"] = strconv.Itoa(limit)
	}
	if offset > 0 {
		params["offset"] = strconv.Itoa(offset)
	}

	var response ListResponse
	err := c.Get(ctx, "/contacts/v1/lists", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetListByID fetches a single list by ID
func (c *Client) GetListByID(ctx context.Context, listID int) (*List, error) {
	logx.Debug("Fetching list by ID: %d", listID)

	var list List
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d", listID)
	err := c.Get(ctx, endpoint, nil, &list)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &list, nil
}

// GetStaticLists fetches all static lists
func (c *Client) GetStaticLists(ctx context.Context, limit int, offset int) (*ListResponse, error) {
	logx.Debug("Fetching static lists from HubSpot")

	params := make(map[string]string)
	params["listType"] = "STATIC"
	if limit > 0 {
		params["count"] = strconv.Itoa(limit)
	}
	if offset > 0 {
		params["offset"] = strconv.Itoa(offset)
	}

	var response ListResponse
	err := c.Get(ctx, "/contacts/v1/lists", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// GetDynamicLists fetches all dynamic lists
func (c *Client) GetDynamicLists(ctx context.Context, limit int, offset int) (*ListResponse, error) {
	logx.Debug("Fetching dynamic lists from HubSpot")

	params := make(map[string]string)
	params["listType"] = "DYNAMIC"
	if limit > 0 {
		params["count"] = strconv.Itoa(limit)
	}
	if offset > 0 {
		params["offset"] = strconv.Itoa(offset)
	}

	var response ListResponse
	err := c.Get(ctx, "/contacts/v1/lists", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// CreateList creates a new list
func (c *Client) CreateList(ctx context.Context, list *ListCreateRequest) (*List, error) {
	logx.Debug("Creating list: %s", list.Name)

	var result List
	err := c.Post(ctx, "/contacts/v1/lists", list, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// UpdateList updates an existing list
func (c *Client) UpdateList(ctx context.Context, listID int, list *ListUpdateRequest) (*List, error) {
	logx.Debug("Updating list: %d", listID)

	var result List
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d", listID)
	err := c.Post(ctx, endpoint, list, &result)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &result, nil
}

// DeleteList deletes a list
func (c *Client) DeleteList(ctx context.Context, listID int) error {
	logx.Debug("Deleting list: %d", listID)

	endpoint := fmt.Sprintf("/contacts/v1/lists/%d", listID)
	err := c.Delete(ctx, endpoint)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("list", listID)
		}
		return err
	}

	return nil
}

// GetListContacts fetches contacts in a list
func (c *Client) GetListContacts(ctx context.Context, listID int, limit int, vidOffset int, propertiesToReturn []string) (*ListContactsResponse, error) {
	logx.Debug("Fetching contacts for list: %d", listID)

	params := make(map[string]string)
	if limit > 0 {
		params["count"] = strconv.Itoa(limit)
	}
	if vidOffset > 0 {
		params["vidOffset"] = strconv.Itoa(vidOffset)
	}
	if len(propertiesToReturn) > 0 {
		params["property"] = strings.Join(propertiesToReturn, "&property=")
	}

	var response ListContactsResponse
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/contacts/all", listID)
	err := c.Get(ctx, endpoint, params, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &response, nil
}

// GetRecentListContacts fetches recently added contacts in a list
func (c *Client) GetRecentListContacts(ctx context.Context, listID int, limit int, timeOffset int64, propertiesToReturn []string) (*ListContactsResponse, error) {
	logx.Debug("Fetching recent contacts for list: %d", listID)

	params := make(map[string]string)
	if limit > 0 {
		params["count"] = strconv.Itoa(limit)
	}
	if timeOffset > 0 {
		params["timeOffset"] = strconv.FormatInt(timeOffset, 10)
	}
	if len(propertiesToReturn) > 0 {
		params["property"] = strings.Join(propertiesToReturn, "&property=")
	}

	var response ListContactsResponse
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/contacts/recent", listID)
	err := c.Get(ctx, endpoint, params, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &response, nil
}

// AddContactsToList adds contacts to a static list by contact IDs
func (c *Client) AddContactsToList(ctx context.Context, listID int, contactIDs []int) (*ListMembershipResponse, error) {
	logx.Debug("Adding %d contacts to list: %d", len(contactIDs), listID)

	request := &ListMembershipRequest{
		VIDs: contactIDs,
	}

	var response ListMembershipResponse
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/add", listID)
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &response, nil
}

// AddContactsToListByEmail adds contacts to a static list by email addresses
func (c *Client) AddContactsToListByEmail(ctx context.Context, listID int, emails []string) (*ListMembershipResponse, error) {
	logx.Debug("Adding %d contacts to list by email: %d", len(emails), listID)

	request := &ListMembershipRequest{
		Emails: emails,
	}

	var response ListMembershipResponse
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/add", listID)
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &response, nil
}

// RemoveContactsFromList removes contacts from a static list by contact IDs
func (c *Client) RemoveContactsFromList(ctx context.Context, listID int, contactIDs []int) (*ListMembershipResponse, error) {
	logx.Debug("Removing %d contacts from list: %d", len(contactIDs), listID)

	request := &ListMembershipRequest{
		VIDs: contactIDs,
	}

	var response ListMembershipResponse
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/remove", listID)
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &response, nil
}

// RemoveContactsFromListByEmail removes contacts from a static list by email addresses
func (c *Client) RemoveContactsFromListByEmail(ctx context.Context, listID int, emails []string) (*ListMembershipResponse, error) {
	logx.Debug("Removing %d contacts from list by email: %d", len(emails), listID)

	request := &ListMembershipRequest{
		Emails: emails,
	}

	var response ListMembershipResponse
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/remove", listID)
	err := c.Post(ctx, endpoint, request, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &response, nil
}

// RefreshList refreshes a dynamic list (triggers re-evaluation)
func (c *Client) RefreshList(ctx context.Context, listID int) error {
	logx.Debug("Refreshing list: %d", listID)

	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/refresh", listID)
	err := c.Post(ctx, endpoint, nil, nil)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return NewResourceNotFoundError("list", listID)
		}
		return err
	}

	return nil
}

// GetListMemberships gets all list memberships for a contact
func (c *Client) GetListMemberships(ctx context.Context, contactID int) (*ContactListMembershipsResponse, error) {
	logx.Debug("Fetching list memberships for contact: %d", contactID)

	var response ContactListMembershipsResponse
	endpoint := fmt.Sprintf("/contacts/v1/contact/vid/%d/lists", contactID)
	err := c.Get(ctx, endpoint, nil, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("contact", contactID)
		}
		return nil, err
	}

	return &response, nil
}

// GetListMembershipsByEmail gets all list memberships for a contact by email
func (c *Client) GetListMembershipsByEmail(ctx context.Context, email string) (*ContactListMembershipsResponse, error) {
	logx.Debug("Fetching list memberships for contact email: %s", email)

	var response ContactListMembershipsResponse
	endpoint := fmt.Sprintf("/contacts/v1/contact/email/%s/lists", url.QueryEscape(email))
	err := c.Get(ctx, endpoint, nil, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("contact", email)
		}
		return nil, err
	}

	return &response, nil
}

// SearchLists searches for lists by name
func (c *Client) SearchLists(ctx context.Context, query string, limit int, offset int) (*ListResponse, error) {
	logx.Debug("Searching lists with query: %s", query)

	// HubSpot doesn't have native search for lists, so we get all and filter
	response, err := c.GetAllLists(ctx, 0, 0)
	if err != nil {
		return nil, err
	}

	var filtered []List
	queryLower := strings.ToLower(query)

	for _, list := range response.Lists {
		if strings.Contains(strings.ToLower(list.Name), queryLower) {
			filtered = append(filtered, list)
		}
	}

	// Apply limit and offset
	if offset > len(filtered) {
		filtered = []List{}
	} else {
		filtered = filtered[offset:]
		if limit > 0 && limit < len(filtered) {
			filtered = filtered[:limit]
		}
	}

	return &ListResponse{
		Lists:      filtered,
		HasMore:    len(filtered) == limit,
		TotalCount: len(filtered),
	}, nil
}

// GetListsByType gets lists filtered by type
func (c *Client) GetListsByType(ctx context.Context, listType string, limit int, offset int) (*ListResponse, error) {
	logx.Debug("Fetching lists by type: %s", listType)

	params := make(map[string]string)
	params["listType"] = listType
	if limit > 0 {
		params["count"] = strconv.Itoa(limit)
	}
	if offset > 0 {
		params["offset"] = strconv.Itoa(offset)
	}

	var response ListResponse
	err := c.Get(ctx, "/contacts/v1/lists", params, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// CloneList creates a copy of an existing list
func (c *Client) CloneList(ctx context.Context, listID int, newListName string) (*List, error) {
	logx.Debug("Cloning list %d with new name: %s", listID, newListName)

	// Get the original list
	originalList, err := c.GetListByID(ctx, listID)
	if err != nil {
		return nil, err
	}

	// Create clone request based on original list
	cloneRequest := &ListCreateRequest{
		Name:     newListName,
		ListType: originalList.ListType,
		Filters:  originalList.Filters,
	}

	// Create the new list
	return c.CreateList(ctx, cloneRequest)
}

// GetListSize gets the count of contacts in a list
func (c *Client) GetListSize(ctx context.Context, listID int) (*ListSizeResponse, error) {
	logx.Debug("Getting size for list: %d", listID)

	var response ListSizeResponse
	endpoint := fmt.Sprintf("/contacts/v1/lists/%d/size", listID)
	err := c.Get(ctx, endpoint, nil, &response)
	if err != nil {
		if errx.IsCode(err, ErrHubSpotNotFound) {
			return nil, NewResourceNotFoundError("list", listID)
		}
		return nil, err
	}

	return &response, nil
}

// BatchAddContactsToLists adds contacts to multiple lists
func (c *Client) BatchAddContactsToLists(ctx context.Context, contactIDs []int, listIDs []int) error {
	logx.Debug("Batch adding %d contacts to %d lists", len(contactIDs), len(listIDs))

	for _, listID := range listIDs {
		_, err := c.AddContactsToList(ctx, listID, contactIDs)
		if err != nil {
			return err
		}
	}

	return nil
}

// BatchRemoveContactsFromLists removes contacts from multiple lists
func (c *Client) BatchRemoveContactsFromLists(ctx context.Context, contactIDs []int, listIDs []int) error {
	logx.Debug("Batch removing %d contacts from %d lists", len(contactIDs), len(listIDs))

	for _, listID := range listIDs {
		_, err := c.RemoveContactsFromList(ctx, listID, contactIDs)
		if err != nil {
			return err
		}
	}

	return nil
}
