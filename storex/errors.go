package storex

import "github.com/Abraxas-365/craftable/errx"

// Error registry for storex
var (
	StoreErrors = errx.NewRegistry("STORE")

	// Common errors
	ErrInvalidQuery     = StoreErrors.Register("INVALID_QUERY", errx.TypeBadRequest, 400, "Invalid query")
	ErrRecordNotFound   = StoreErrors.Register("NOT_FOUND", errx.TypeNotFound, 404, "Record not found")
	ErrConnectionFailed = StoreErrors.Register("CONNECTION_FAILED", errx.TypeUnavailable, 503, "Database connection failed")
	ErrCreateFailed     = StoreErrors.Register("CREATE_FAILED", errx.TypeInternal, 500, "Failed to create record")
	ErrUpdateFailed     = StoreErrors.Register("UPDATE_FAILED", errx.TypeInternal, 500, "Failed to update record")
	ErrDeleteFailed     = StoreErrors.Register("DELETE_FAILED", errx.TypeInternal, 500, "Failed to delete record")
	ErrTxBeginFailed    = StoreErrors.Register("TX_BEGIN_FAILED", errx.TypeInternal, 500, "Failed to begin transaction")
	ErrTxCommitFailed   = StoreErrors.Register("TX_COMMIT_FAILED", errx.TypeInternal, 500, "Failed to commit transaction")
	ErrTxRollbackFailed = StoreErrors.Register("TX_ROLLBACK_FAILED", errx.TypeInternal, 500, "Failed to rollback transaction")
	ErrBulkOpFailed     = StoreErrors.Register("BULK_OPERATION_FAILED", errx.TypeInternal, 500, "Bulk operation failed")
	ErrSearchFailed     = StoreErrors.Register("SEARCH_FAILED", errx.TypeInternal, 500, "Search operation failed")

	// SQL-specific errors
	ErrSQLScanFailed  = StoreErrors.Register("SQL_SCAN_FAILED", errx.TypeInternal, 500, "Failed to scan SQL results")
	ErrSQLQueryFailed = StoreErrors.Register("SQL_QUERY_FAILED", errx.TypeInternal, 500, "SQL query execution failed")
	ErrSQLCountFailed = StoreErrors.Register("SQL_COUNT_FAILED", errx.TypeInternal, 500, "Failed to count SQL records")
	ErrSQLExecFailed  = StoreErrors.Register("SQL_EXEC_FAILED", errx.TypeInternal, 500, "SQL exec operation failed")

	// MongoDB-specific errors
	ErrMongoFindFailed   = StoreErrors.Register("MONGO_FIND_FAILED", errx.TypeInternal, 500, "MongoDB find operation failed")
	ErrMongoCountFailed  = StoreErrors.Register("MONGO_COUNT_FAILED", errx.TypeInternal, 500, "Failed to count MongoDB records")
	ErrMongoDecodeFailed = StoreErrors.Register("MONGO_DECODE_FAILED", errx.TypeInternal, 500, "Failed to decode MongoDB document")
	ErrMongoInsertFailed = StoreErrors.Register("MONGO_INSERT_FAILED", errx.TypeInternal, 500, "MongoDB insert operation failed")
	ErrMongoUpdateFailed = StoreErrors.Register("MONGO_UPDATE_FAILED", errx.TypeInternal, 500, "MongoDB update operation failed")
	ErrMongoDeleteFailed = StoreErrors.Register("MONGO_DELETE_FAILED", errx.TypeInternal, 500, "MongoDB delete operation failed")
	ErrInvalidID         = StoreErrors.Register("INVALID_ID", errx.TypeBadRequest, 400, "Invalid ID format")
)

// Helper functions
func IsRecordNotFound(err error) bool {
	return errx.IsCode(err, ErrRecordNotFound)
}

func IsConnectionFailed(err error) bool {
	return errx.IsCode(err, ErrConnectionFailed)
}

func IsInvalidQuery(err error) bool {
	return errx.IsCode(err, ErrInvalidQuery)
}

func IsInvalidID(err error) bool {
	return errx.IsCode(err, ErrInvalidID)
}
