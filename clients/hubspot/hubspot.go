package hubspot

import (
	"strconv"
	"time"
)

// HubSpotErrorResponse represents the standard HubSpot API error response
type HubSpotErrorResponse struct {
	Status        string                   `json:"status"`
	Message       string                   `json:"message"`
	CorrelationID string                   `json:"correlationId"`
	Category      string                   `json:"category"`
	SubCategory   string                   `json:"subCategory,omitempty"`
	Errors        []HubSpotValidationError `json:"errors,omitempty"`
	Context       map[string]any           `json:"context,omitempty"`
	Links         map[string]string        `json:"links,omitempty"`
}

// HubSpotValidationError represents a validation error from HubSpot
type HubSpotValidationError struct {
	Message string         `json:"message"`
	In      string         `json:"in,omitempty"`
	Code    string         `json:"code,omitempty"`
	Context map[string]any `json:"context,omitempty"`
}

// Paging represents pagination information from HubSpot API responses
type Paging struct {
	Next *PagingNext `json:"next,omitempty"`
	Prev *PagingPrev `json:"prev,omitempty"`
}

// PagingNext represents next page information
type PagingNext struct {
	After string `json:"after"`
	Link  string `json:"link,omitempty"`
}

// PagingPrev represents previous page information
type PagingPrev struct {
	Before string `json:"before"`
	Link   string `json:"link,omitempty"`
}

// SearchRequest represents a generic search request
type SearchRequest struct {
	Query        string        `json:"query,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	After        string        `json:"after,omitempty"`
	Sorts        []string      `json:"sorts,omitempty"`
	Properties   []string      `json:"properties,omitempty"`
	FilterGroups []FilterGroup `json:"filterGroups,omitempty"`
}

// FilterGroup represents a group of filters
type FilterGroup struct {
	Filters []Filter `json:"filters"`
}

// Filter represents a single filter
type Filter struct {
	PropertyName string `json:"propertyName"`
	Operator     string `json:"operator"`
	Value        any    `json:"value"`
}

// BatchRequest represents a generic batch request
type BatchRequest struct {
	Inputs []any `json:"inputs"`
}

// BatchResponse represents a generic batch response
type BatchResponse struct {
	Status      string            `json:"status"`
	Results     []any             `json:"results,omitempty"`
	NumErrors   int               `json:"numErrors,omitempty"`
	Errors      []BatchError      `json:"errors,omitempty"`
	RequestedAt *int64            `json:"requestedAt,omitempty"`
	StartedAt   *int64            `json:"startedAt,omitempty"`
	CompletedAt *int64            `json:"completedAt,omitempty"`
	Links       map[string]string `json:"links,omitempty"`
}

// BatchError represents an error in a batch operation
type BatchError struct {
	Status        string         `json:"status"`
	Message       string         `json:"message"`
	CorrelationID string         `json:"correlationId"`
	Context       map[string]any `json:"context,omitempty"`
	Category      string         `json:"category"`
	SubCategory   string         `json:"subCategory,omitempty"`
}

// Property represents a HubSpot property
type Property struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// Properties represents a collection of HubSpot properties
type Properties map[string]any

// Association represents a HubSpot association
type Association struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// AssociationInput represents input for creating associations
type AssociationInput struct {
	From AssociationSpec `json:"from"`
	To   AssociationSpec `json:"to"`
	Type string          `json:"type"`
}

// AssociationSpec represents an association specification
type AssociationSpec struct {
	ID string `json:"id"`
}

// ============================================================================
// WORKFLOW TYPES (Updated to match actual HubSpot API response)
// ============================================================================

// Workflow represents a HubSpot workflow (matches actual API response)
type Workflow struct {
	ID                   int                `json:"id"`
	Name                 string             `json:"name"`
	Type                 string             `json:"type"`
	Enabled              bool               `json:"enabled"`
	Description          string             `json:"description,omitempty"`
	Actions              []WorkflowAction   `json:"actions,omitempty"`
	Enrollment           WorkflowEnrollment `json:"enrollment,omitzero"`
	PortalID             int                `json:"portalId,omitempty"`
	CreatedAt            *int64             `json:"createdAt,omitempty"`
	UpdatedAt            *int64             `json:"updatedAt,omitempty"`
	InsertedAt           *int64             `json:"insertedAt,omitempty"`
	MigrationStatus      *MigrationStatus   `json:"migrationStatus,omitempty"`
	CreationSource       *CreationSource    `json:"creationSource,omitempty"`
	UpdateSource         *UpdateSource      `json:"updateSource,omitempty"`
	ContactListIds       *ContactListIds    `json:"contactListIds,omitempty"`
	PersonaTagIds        []string           `json:"personaTagIds,omitempty"`
	ContactCounts        *ContactCounts     `json:"contactCounts,omitempty"`
	LastUpdatedByUserID  int                `json:"lastUpdatedByUserId,omitempty"`
	OriginalAuthorUserID int                `json:"originalAuthorUserId,omitempty"`
}

// WorkflowAction represents an action within a workflow
type WorkflowAction struct {
	ID             string         `json:"id,omitempty"`
	Type           string         `json:"type"`
	Settings       map[string]any `json:"settings,omitempty"`
	DelayMillis    int            `json:"delayMillis,omitempty"`
	ActionMetadata map[string]any `json:"actionMetadata,omitempty"`
}

// WorkflowEnrollment represents workflow enrollment settings
type WorkflowEnrollment struct {
	CriteriaType         string         `json:"criteriaType,omitempty"`
	Criteria             map[string]any `json:"criteria,omitempty"`
	ReenrollmentCriteria map[string]any `json:"reenrollmentCriteria,omitempty"`
	SuppressionLists     []int          `json:"suppressionLists,omitempty"`
}

// MigrationStatus represents the migration status of a workflow
type MigrationStatus struct {
	PortalID                         int    `json:"portalId"`
	WorkflowID                       int    `json:"workflowId"`
	MigrationStatus                  string `json:"migrationStatus"`
	EnrollmentMigrationStatus        string `json:"enrollmentMigrationStatus"`
	PlatformOwnsActions              bool   `json:"platformOwnsActions"`
	LastSuccessfulMigrationTimestamp *int64 `json:"lastSuccessfulMigrationTimestamp"`
	EnrollmentMigrationTimestamp     *int64 `json:"enrollmentMigrationTimestamp"`
	FlowID                           int    `json:"flowId"`
}

// CreationSource represents the source of workflow creation
type CreationSource struct {
	SourceApplication SourceApplication `json:"sourceApplication"`
	CreatedByUser     *User             `json:"createdByUser,omitempty"`
	CreatedAt         int64             `json:"createdAt"`
}

// UpdateSource represents the source of workflow updates
type UpdateSource struct {
	SourceApplication SourceApplication `json:"sourceApplication"`
	UpdatedByUser     *User             `json:"updatedByUser,omitempty"`
	UpdatedAt         int64             `json:"updatedAt"`
}

// SourceApplication represents the application source
type SourceApplication struct {
	Source      string `json:"source"` // This matches the actual API response
	ServiceName string `json:"serviceName,omitempty"`
}

// User represents a HubSpot user (matches actual API response)
type User struct {
	UserID    int    `json:"userId"`    // This matches the actual API response
	UserEmail string `json:"userEmail"` // This matches the actual API response
}

// ContactListIds represents contact list IDs for workflows (matches actual API response)
type ContactListIds struct {
	Enrolled  int `json:"enrolled"`
	Active    int `json:"active"`
	Completed int `json:"completed"`
	Succeeded int `json:"succeeded"`
}

// ContactCounts represents contact counts for workflows (matches actual API response)
type ContactCounts struct {
	Enrolled int `json:"enrolled"`
	Active   int `json:"active"`
	// Note: The API response shows these fields, but they might not always be present
	Completed int `json:"completed,omitempty"`
	Succeeded int `json:"succeeded,omitempty"`
}

// WorkflowListResponse represents a list of workflows response
type WorkflowListResponse struct {
	Workflows []Workflow `json:"workflows"`
	Total     int        `json:"total,omitempty"`
	Paging    *Paging    `json:"paging,omitempty"`
}

// WorkflowCreateRequest represents a request to create a workflow
type WorkflowCreateRequest struct {
	Name       string             `json:"name"`
	Type       string             `json:"type"`
	Enabled    *bool              `json:"enabled,omitempty"`
	Actions    []WorkflowAction   `json:"actions,omitempty"`
	Enrollment WorkflowEnrollment `json:"enrollment"`
}

// WorkflowUpdateRequest represents a request to update a workflow
type WorkflowUpdateRequest struct {
	Name       *string             `json:"name,omitempty"`
	Enabled    *bool               `json:"enabled,omitempty"`
	Actions    *[]WorkflowAction   `json:"actions,omitempty"`
	Enrollment *WorkflowEnrollment `json:"enrollment,omitempty"`
}

// WorkflowFilter represents filters for workflow queries
type WorkflowFilter struct {
	Type          string `json:"type,omitempty"`
	Enabled       *bool  `json:"enabled,omitempty"`
	NamePattern   string `json:"namePattern,omitempty"`
	PortalID      int    `json:"portalId,omitempty"`
	CreatedAfter  *int64 `json:"createdAfter,omitempty"`
	CreatedBefore *int64 `json:"createdBefore,omitempty"`
	UpdatedAfter  *int64 `json:"updatedAfter,omitempty"`
	UpdatedBefore *int64 `json:"updatedBefore,omitempty"`
}

// ============================================================================
// CONTACT TYPES
// ============================================================================

// Contact represents a HubSpot contact
type Contact struct {
	ID           string         `json:"id"`
	Properties   Properties     `json:"properties"`
	CreatedAt    *int64         `json:"createdAt,omitempty"`
	UpdatedAt    *int64         `json:"updatedAt,omitempty"`
	Archived     bool           `json:"archived,omitempty"`
	ArchivedAt   *int64         `json:"archivedAt,omitempty"`
	Associations map[string]any `json:"associations,omitempty"`
}

// ContactInput represents input for creating/updating a contact
type ContactInput struct {
	Properties   Properties    `json:"properties"`
	Associations []Association `json:"associations,omitempty"`
}

// ContactSearchResponse represents a contact search response
type ContactSearchResponse struct {
	Total   int       `json:"total"`
	Results []Contact `json:"results"`
	Paging  *Paging   `json:"paging,omitempty"`
}

// ContactListResponse represents a contact list response
type ContactListResponse struct {
	Results []Contact `json:"results"`
	Paging  *Paging   `json:"paging,omitempty"`
}

// ============================================================================
// COMPANY TYPES
// ============================================================================

// Company represents a HubSpot company
type Company struct {
	ID           string         `json:"id"`
	Properties   Properties     `json:"properties"`
	CreatedAt    *int64         `json:"createdAt,omitempty"`
	UpdatedAt    *int64         `json:"updatedAt,omitempty"`
	Archived     bool           `json:"archived,omitempty"`
	ArchivedAt   *int64         `json:"archivedAt,omitempty"`
	Associations map[string]any `json:"associations,omitempty"`
}

// CompanyInput represents input for creating/updating a company
type CompanyInput struct {
	Properties   Properties    `json:"properties"`
	Associations []Association `json:"associations,omitempty"`
}

// CompanySearchResponse represents a company search response
type CompanySearchResponse struct {
	Total   int       `json:"total"`
	Results []Company `json:"results"`
	Paging  *Paging   `json:"paging,omitempty"`
}

// CompanyListResponse represents a company list response
type CompanyListResponse struct {
	Results []Company `json:"results"`
	Paging  *Paging   `json:"paging,omitempty"`
}

// ============================================================================
// DEAL TYPES
// ============================================================================

// Deal represents a HubSpot deal
type Deal struct {
	ID           string         `json:"id"`
	Properties   Properties     `json:"properties"`
	CreatedAt    *int64         `json:"createdAt,omitempty"`
	UpdatedAt    *int64         `json:"updatedAt,omitempty"`
	Archived     bool           `json:"archived,omitempty"`
	ArchivedAt   *int64         `json:"archivedAt,omitempty"`
	Associations map[string]any `json:"associations,omitempty"`
}

// DealInput represents input for creating/updating a deal
type DealInput struct {
	Properties   Properties    `json:"properties"`
	Associations []Association `json:"associations,omitempty"`
}

// DealSearchResponse represents a deal search response
type DealSearchResponse struct {
	Total   int     `json:"total"`
	Results []Deal  `json:"results"`
	Paging  *Paging `json:"paging,omitempty"`
}

// DealListResponse represents a deal list response
type DealListResponse struct {
	Results []Deal  `json:"results"`
	Paging  *Paging `json:"paging,omitempty"`
}

// ============================================================================
// TICKET TYPES
// ============================================================================

// Ticket represents a HubSpot ticket
type Ticket struct {
	ID           string         `json:"id"`
	Properties   Properties     `json:"properties"`
	CreatedAt    *int64         `json:"createdAt,omitempty"`
	UpdatedAt    *int64         `json:"updatedAt,omitempty"`
	Archived     bool           `json:"archived,omitempty"`
	ArchivedAt   *int64         `json:"archivedAt,omitempty"`
	Associations map[string]any `json:"associations,omitempty"`
}

// TicketInput represents input for creating/updating a ticket
type TicketInput struct {
	Properties   Properties    `json:"properties"`
	Associations []Association `json:"associations,omitempty"`
}

// TicketSearchResponse represents a ticket search response
type TicketSearchResponse struct {
	Total   int      `json:"total"`
	Results []Ticket `json:"results"`
	Paging  *Paging  `json:"paging,omitempty"`
}

// TicketListResponse represents a ticket list response
type TicketListResponse struct {
	Results []Ticket `json:"results"`
	Paging  *Paging  `json:"paging,omitempty"`
}

// ============================================================================
// EMAIL TYPES
// ============================================================================

// Email represents a HubSpot email
type Email struct {
	ID           string         `json:"id"`
	Properties   Properties     `json:"properties"`
	CreatedAt    *int64         `json:"createdAt,omitempty"`
	UpdatedAt    *int64         `json:"updatedAt,omitempty"`
	Archived     bool           `json:"archived,omitempty"`
	ArchivedAt   *int64         `json:"archivedAt,omitempty"`
	Associations map[string]any `json:"associations,omitempty"`
}

// EmailInput represents input for creating/updating an email
type EmailInput struct {
	Properties   Properties    `json:"properties"`
	Associations []Association `json:"associations,omitempty"`
}

// ============================================================================
// PIPELINE TYPES
// ============================================================================

// Pipeline represents a HubSpot pipeline
type Pipeline struct {
	ID           string          `json:"id"`
	Label        string          `json:"label"`
	DisplayOrder int             `json:"displayOrder"`
	Stages       []PipelineStage `json:"stages"`
	CreatedAt    *int64          `json:"createdAt,omitempty"`
	UpdatedAt    *int64          `json:"updatedAt,omitempty"`
	Archived     bool            `json:"archived,omitempty"`
}

// PipelineStage represents a stage in a pipeline
type PipelineStage struct {
	ID           string         `json:"id"`
	Label        string         `json:"label"`
	DisplayOrder int            `json:"displayOrder"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	CreatedAt    *int64         `json:"createdAt,omitempty"`
	UpdatedAt    *int64         `json:"updatedAt,omitempty"`
	Archived     bool           `json:"archived,omitempty"`
}

// ============================================================================
// OWNER TYPES
// ============================================================================

// Owner represents a HubSpot owner
type Owner struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	UserID    int    `json:"userId,omitempty"`
	CreatedAt *int64 `json:"createdAt,omitempty"`
	UpdatedAt *int64 `json:"updatedAt,omitempty"`
	Archived  bool   `json:"archived,omitempty"`
	Teams     []Team `json:"teams,omitempty"`
}

// Team represents a HubSpot team
type Team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// OwnerListResponse represents an owner list response
type OwnerListResponse struct {
	Results []Owner `json:"results"`
	Paging  *Paging `json:"paging,omitempty"`
}

// ============================================================================
// LIST TYPES
// ============================================================================

// List represents a HubSpot list
type List struct {
	ListID    int            `json:"listId"`
	Name      string         `json:"name"`
	ListType  string         `json:"listType"`
	CreatedAt *int64         `json:"createdAt,omitempty"`
	UpdatedAt *int64         `json:"updatedAt,omitempty"`
	Filters   []ListFilter   `json:"filters,omitempty"`
	MetaData  map[string]any `json:"metaData,omitempty"`
	Archived  bool           `json:"archived,omitempty"`
}

// ListFilter represents a filter for a HubSpot list
type ListFilter struct {
	FilterFamily string `json:"filterFamily"`
	Property     string `json:"property"`
	Type         string `json:"type"`
	Operation    string `json:"operation"`
	Value        any    `json:"value"`
}

// ============================================================================
// FILE TYPES
// ============================================================================

// File represents a HubSpot file
type File struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Path              string         `json:"path"`
	Size              int64          `json:"size"`
	Height            int            `json:"height,omitempty"`
	Width             int            `json:"width,omitempty"`
	Encoding          string         `json:"encoding,omitempty"`
	Type              string         `json:"type"`
	Extension         string         `json:"extension"`
	DefaultHostingUrl string         `json:"defaultHostingUrl"`
	URL               string         `json:"url"`
	CreatedAt         *int64         `json:"createdAt,omitempty"`
	UpdatedAt         *int64         `json:"updatedAt,omitempty"`
	ArchivedAt        *int64         `json:"archivedAt,omitempty"`
	Archived          bool           `json:"archived,omitempty"`
	Access            string         `json:"access"`
	TTL               string         `json:"ttl,omitempty"`
	Options           map[string]any `json:"options,omitempty"`
}

// FileUploadOptions represents options for file upload
type FileUploadOptions struct {
	Access                      string `json:"access,omitempty"`
	TTL                         string `json:"ttl,omitempty"`
	Overwrite                   bool   `json:"overwrite,omitempty"`
	DuplicateValidationStrategy string `json:"duplicateValidationStrategy,omitempty"`
	DuplicateValidationScope    string `json:"duplicateValidationScope,omitempty"`
}

// ============================================================================
// ANALYTICS TYPES
// ============================================================================

// AnalyticsQuery represents an analytics query
type AnalyticsQuery struct {
	StartDate  string   `json:"startDate"`
	EndDate    string   `json:"endDate"`
	TimeFormat string   `json:"timeFormat,omitempty"`
	Frequency  string   `json:"frequency,omitempty"`
	Properties []string `json:"properties,omitempty"`
	Breakdowns []string `json:"breakdowns,omitempty"`
	Filters    []Filter `json:"filters,omitempty"`
	Sorts      []string `json:"sorts,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	After      string   `json:"after,omitempty"`
}

// AnalyticsResponse represents an analytics response
type AnalyticsResponse struct {
	Total   int              `json:"total"`
	Results []map[string]any `json:"results"`
	Paging  *Paging          `json:"paging,omitempty"`
}

// UnixMillisToTime converts Unix timestamp (milliseconds) to time.Time
func UnixMillisToTime(unixMillis *int64) *time.Time {
	if unixMillis == nil {
		return nil
	}
	t := time.Unix(*unixMillis/1000, (*unixMillis%1000)*1000000)
	return &t
}

// UnixMillisToTimeValue converts Unix timestamp (milliseconds) to time.Time (non-pointer)
func UnixMillisToTimeValue(unixMillis int64) time.Time {
	return time.Unix(unixMillis/1000, (unixMillis%1000)*1000000)
}

// TimeToUnixMillis converts time.Time to Unix timestamp in milliseconds
func TimeToUnixMillis(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// TimeToUnixMillisPtr converts time.Time to Unix timestamp in milliseconds (pointer)
func TimeToUnixMillisPtr(t *time.Time) *int64 {
	if t == nil {
		return nil
	}
	millis := t.UnixNano() / int64(time.Millisecond)
	return &millis
}

// StringToInt safely converts string to int
func StringToInt(s string) (int, error) {
	return strconv.Atoi(s)
}

// StringToInt64 safely converts string to int64
func StringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// IntToString converts int to string
func IntToString(i int) string {
	return strconv.Itoa(i)
}

// Int64ToString converts int64 to string
func Int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// BoolToString converts bool to string
func BoolToString(b bool) string {
	return strconv.FormatBool(b)
}

// StringToBool safely converts string to bool
func StringToBool(s string) (bool, error) {
	return strconv.ParseBool(s)
}

// ============================================================================
// PROPERTY TYPES
// ============================================================================

// PropertyDefinition represents a HubSpot property definition
type PropertyDefinition struct {
	Name                 string                `json:"name"`
	Label                string                `json:"label"`
	Type                 string                `json:"type"`
	FieldType            string                `json:"fieldType"`
	GroupName            string                `json:"groupName,omitempty"`
	Description          string                `json:"description,omitempty"`
	Options              []PropertyOption      `json:"options,omitempty"`
	DisplayOrder         int                   `json:"displayOrder,omitempty"`
	HasUniqueValue       bool                  `json:"hasUniqueValue,omitempty"`
	Hidden               bool                  `json:"hidden,omitempty"`
	FormField            bool                  `json:"formField,omitempty"`
	Deleted              bool                  `json:"deleted,omitempty"`
	CalculationFormula   string                `json:"calculationFormula,omitempty"`
	ExternalOptions      bool                  `json:"externalOptions,omitempty"`
	ReferencedObjectType string                `json:"referencedObjectType,omitempty"`
	CreatedAt            *int64                `json:"createdAt,omitempty"`
	UpdatedAt            *int64                `json:"updatedAt,omitempty"`
	CreatedUserId        string                `json:"createdUserId,omitempty"`
	UpdatedUserId        string                `json:"updatedUserId,omitempty"`
	ModificationMetadata *ModificationMetadata `json:"modificationMetadata,omitempty"`
}

// PropertyOption represents an option for enumeration properties
type PropertyOption struct {
	Label        string `json:"label"`
	Value        string `json:"value"`
	Description  string `json:"description,omitempty"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
	Hidden       bool   `json:"hidden,omitempty"`
}

// ModificationMetadata represents metadata about property modifications
type ModificationMetadata struct {
	Archivable         bool `json:"archivable"`
	ReadOnlyValue      bool `json:"readOnlyValue"`
	ReadOnlyDefinition bool `json:"readOnlyDefinition"`
}

// PropertyGroup represents a HubSpot property group
type PropertyGroup struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	DisplayOrder int    `json:"displayOrder"`
	Archived     bool   `json:"archived,omitempty"`
}

// PropertyCreateRequest represents a request to create a property
type PropertyCreateRequest struct {
	Name                 string           `json:"name"`
	Label                string           `json:"label"`
	Type                 string           `json:"type"`
	FieldType            string           `json:"fieldType"`
	GroupName            string           `json:"groupName,omitempty"`
	Description          string           `json:"description,omitempty"`
	Options              []PropertyOption `json:"options,omitempty"`
	DisplayOrder         int              `json:"displayOrder,omitempty"`
	HasUniqueValue       bool             `json:"hasUniqueValue,omitempty"`
	Hidden               bool             `json:"hidden,omitempty"`
	FormField            bool             `json:"formField,omitempty"`
	CalculationFormula   string           `json:"calculationFormula,omitempty"`
	ExternalOptions      bool             `json:"externalOptions,omitempty"`
	ReferencedObjectType string           `json:"referencedObjectType,omitempty"`
}

// PropertyUpdateRequest represents a request to update a property
type PropertyUpdateRequest struct {
	Label                *string           `json:"label,omitempty"`
	GroupName            *string           `json:"groupName,omitempty"`
	Description          *string           `json:"description,omitempty"`
	Options              *[]PropertyOption `json:"options,omitempty"`
	DisplayOrder         *int              `json:"displayOrder,omitempty"`
	Hidden               *bool             `json:"hidden,omitempty"`
	FormField            *bool             `json:"formField,omitempty"`
	CalculationFormula   *string           `json:"calculationFormula,omitempty"`
	ExternalOptions      *bool             `json:"externalOptions,omitempty"`
	ReferencedObjectType *string           `json:"referencedObjectType,omitempty"`
}

// PropertyListResponse represents a list of properties response
type PropertyListResponse struct {
	Results []PropertyDefinition `json:"results"`
	Paging  *Paging              `json:"paging,omitempty"`
}

// PropertyGroupCreateRequest represents a request to create a property group
type PropertyGroupCreateRequest struct {
	Name         string `json:"name"`
	Label        string `json:"label"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
}

// PropertyGroupUpdateRequest represents a request to update a property group
type PropertyGroupUpdateRequest struct {
	Label        *string `json:"label,omitempty"`
	DisplayOrder *int    `json:"displayOrder,omitempty"`
}

// PropertyGroupListResponse represents a list of property groups response
type PropertyGroupListResponse struct {
	Results []PropertyGroup `json:"results"`
	Paging  *Paging         `json:"paging,omitempty"`
}

// ============================================================================
// FORM TYPES
// ============================================================================

// Form represents a HubSpot form
type Form struct {
	ID                    string           `json:"id"`
	Name                  string           `json:"name"`
	Action                string           `json:"action,omitempty"`
	Method                string           `json:"method,omitempty"`
	CssClass              string           `json:"cssClass,omitempty"`
	Redirect              string           `json:"redirect,omitempty"`
	SubmitText            string           `json:"submitText,omitempty"`
	FollowUpAction        string           `json:"followUpActionType,omitempty"`
	NotifyRecipients      string           `json:"notifyRecipients,omitempty"`
	LeadNurturingCampaign string           `json:"leadNurturingCampaignId,omitempty"`
	FormFieldGroups       []FormFieldGroup `json:"formFieldGroups"`
	CreatedAt             *int64           `json:"createdAt,omitempty"`
	UpdatedAt             *int64           `json:"updatedAt,omitempty"`
	Archived              bool             `json:"archived,omitempty"`
	Published             bool             `json:"published,omitempty"`
}

// FormFieldGroup represents a group of form fields
type FormFieldGroup struct {
	Fields       []FormField    `json:"fields"`
	Default      bool           `json:"default,omitempty"`
	IsSmartGroup bool           `json:"isSmartGroup,omitempty"`
	RichText     map[string]any `json:"richText,omitempty"`
	GroupType    string         `json:"groupType,omitempty"`
}

// FormField represents a single form field
type FormField struct {
	Name                 string            `json:"name"`
	Label                string            `json:"label,omitempty"`
	FieldType            string            `json:"fieldType"`
	ObjectTypeId         string            `json:"objectTypeId,omitempty"`
	Description          string            `json:"description,omitempty"`
	GroupName            string            `json:"groupName,omitempty"`
	DisplayOrder         int               `json:"displayOrder,omitempty"`
	Required             bool              `json:"required,omitempty"`
	Enabled              bool              `json:"enabled,omitempty"`
	Hidden               bool              `json:"hidden,omitempty"`
	DefaultValue         string            `json:"defaultValue,omitempty"`
	Placeholder          string            `json:"placeholder,omitempty"`
	Options              []FormFieldOption `json:"options,omitempty"`
	Validation           map[string]any    `json:"validation,omitempty"`
	DependentFields      []map[string]any  `json:"dependentFields,omitempty"`
	UseCountryCodeSelect bool              `json:"useCountryCodeSelect,omitempty"`
	AllowMultipleFiles   bool              `json:"allowMultipleFiles,omitempty"`
	LabelHidden          bool              `json:"labelHidden,omitempty"`
	PropertyObjectType   string            `json:"propertyObjectType,omitempty"`
	MetaData             []map[string]any  `json:"metaData,omitempty"`
}

// FormFieldOption represents an option for select/radio/checkbox fields
type FormFieldOption struct {
	Label        string `json:"label"`
	Value        string `json:"value"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
	Selected     bool   `json:"selected,omitempty"`
}

// FormCreateRequest represents a request to create a form
type FormCreateRequest struct {
	Name                  string           `json:"name"`
	FormFieldGroups       []FormFieldGroup `json:"formFieldGroups"`
	SubmitText            string           `json:"submitText,omitempty"`
	FollowUpAction        string           `json:"followUpActionType,omitempty"`
	NotifyRecipients      string           `json:"notifyRecipients,omitempty"`
	LeadNurturingCampaign string           `json:"leadNurturingCampaignId,omitempty"`
	Configuration         map[string]any   `json:"configuration,omitempty"`
	DisplayOptions        map[string]any   `json:"displayOptions,omitempty"`
	Style                 map[string]any   `json:"style,omitempty"`
}

// FormUpdateRequest represents a request to update a form
type FormUpdateRequest struct {
	Name                  *string           `json:"name,omitempty"`
	FormFieldGroups       *[]FormFieldGroup `json:"formFieldGroups,omitempty"`
	SubmitText            *string           `json:"submitText,omitempty"`
	FollowUpAction        *string           `json:"followUpActionType,omitempty"`
	NotifyRecipients      *string           `json:"notifyRecipients,omitempty"`
	LeadNurturingCampaign *string           `json:"leadNurturingCampaignId,omitempty"`
	Archived              *bool             `json:"archived,omitempty"`
	Configuration         *map[string]any   `json:"configuration,omitempty"`
	DisplayOptions        *map[string]any   `json:"displayOptions,omitempty"`
	Style                 *map[string]any   `json:"style,omitempty"`
}

// FormListResponse represents a list of forms response
type FormListResponse struct {
	Results []Form  `json:"results"`
	Paging  *Paging `json:"paging,omitempty"`
}

// FormSubmission represents a form submission
type FormSubmission struct {
	SubmittedAt *int64                `json:"submittedAt,omitempty"`
	Values      []FormSubmissionValue `json:"values"`
	PageUrl     string                `json:"pageUrl,omitempty"`
	PageName    string                `json:"pageName,omitempty"`
	ContactID   string                `json:"contactId,omitempty"`
	FormID      string                `json:"formId"`
}

// FormSubmissionValue represents a submitted form field value
type FormSubmissionValue struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Selected bool   `json:"selected,omitempty"`
}

// FormSubmissionListResponse represents form submission list response
type FormSubmissionListResponse struct {
	Results []FormSubmission `json:"results"`
	Paging  *Paging          `json:"paging,omitempty"`
}
