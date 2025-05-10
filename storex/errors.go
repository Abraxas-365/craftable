package storex

import "github.com/Abraxas-365/craftale/errx"

// Error registry for storex
var (
	storeErrors = errx.NewRegistry("STORE")

	// Common errors
	ErrInvalidQuery     = storeErrors.Register("INVALID_QUERY", errx.TypeBadRequest, 400, "Invalid query")
	ErrRecordNotFound   = storeErrors.Register("NOT_FOUND", errx.TypeNotFound, 404, "Record not found")
	ErrConnectionFailed = storeErrors.Register("CONNECTION_FAILED", errx.TypeUnavailable, 503, "Database connection failed")
	ErrCreateFailed     = storeErrors.Register("CREATE_FAILED", errx.TypeInternal, 500, "Failed to create record")
	ErrUpdateFailed     = storeErrors.Register("UPDATE_FAILED", errx.TypeInternal, 500, "Failed to update record")
	ErrDeleteFailed     = storeErrors.Register("DELETE_FAILED", errx.TypeInternal, 500, "Failed to delete record")
	ErrTxBeginFailed    = storeErrors.Register("TX_BEGIN_FAILED", errx.TypeInternal, 500, "Failed to begin transaction")
	ErrTxCommitFailed   = storeErrors.Register("TX_COMMIT_FAILED", errx.TypeInternal, 500, "Failed to commit transaction")
	ErrTxRollbackFailed = storeErrors.Register("TX_ROLLBACK_FAILED", errx.TypeInternal, 500, "Failed to rollback transaction")
	ErrBulkOpFailed     = storeErrors.Register("BULK_OPERATION_FAILED", errx.TypeInternal, 500, "Bulk operation failed")
	ErrSearchFailed     = storeErrors.Register("SEARCH_FAILED", errx.TypeInternal, 500, "Search operation failed")

	// SQL-specific errors
	ErrSQLScanFailed  = storeErrors.Register("SQL_SCAN_FAILED", errx.TypeInternal, 500, "Failed to scan SQL results")
	ErrSQLQueryFailed = storeErrors.Register("SQL_QUERY_FAILED", errx.TypeInternal, 500, "SQL query execution failed")
	ErrSQLCountFailed = storeErrors.Register("SQL_COUNT_FAILED", errx.TypeInternal, 500, "Failed to count SQL records")
	ErrSQLExecFailed  = storeErrors.Register("SQL_EXEC_FAILED", errx.TypeInternal, 500, "SQL exec operation failed")

	// MongoDB-specific errors
	ErrMongoFindFailed   = storeErrors.Register("MONGO_FIND_FAILED", errx.TypeInternal, 500, "MongoDB find operation failed")
	ErrMongoCountFailed  = storeErrors.Register("MONGO_COUNT_FAILED", errx.TypeInternal, 500, "Failed to count MongoDB records")
	ErrMongoDecodeFailed = storeErrors.Register("MONGO_DECODE_FAILED", errx.TypeInternal, 500, "Failed to decode MongoDB document")
	ErrMongoInsertFailed = storeErrors.Register("MONGO_INSERT_FAILED", errx.TypeInternal, 500, "MongoDB insert operation failed")
	ErrMongoUpdateFailed = storeErrors.Register("MONGO_UPDATE_FAILED", errx.TypeInternal, 500, "MongoDB update operation failed")
	ErrMongoDeleteFailed = storeErrors.Register("MONGO_DELETE_FAILED", errx.TypeInternal, 500, "MongoDB delete operation failed")
	ErrInvalidID         = storeErrors.Register("INVALID_ID", errx.TypeBadRequest, 400, "Invalid ID format")
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
