package infra

import (
	"context"
)

// DB interface wraps the databse
type DB interface {
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
	EnsureIndices(ctx context.Context, table string, indices []DbIndex) error
	DropIndices(ctx context.Context, table string, indices []DbIndex) error
	Insert(ctx context.Context, table string, docs interface{}) error
	List(ctx context.Context, table string, filter DbQuery, skip, limit int, v interface{}, sorts []string, opts ...*ListOptions) error
	FindOne(ctx context.Context, table string, q DbQuery, preloads []string, v interface{}, sorts ...string) error
	PartialUpdateMany(ctx context.Context, table string, filter DbQuery, data interface{}) error
	BulkUpdate(ctx context.Context, table string, models []BulkWriteModel) error
	Aggregate(ctx context.Context, col string, q []DbQuery, v interface{}) error
	Distinct(ctx context.Context, table string, field string, q DbQuery, v interface{}, opts ...*DistinctOptions) error
	DeleteMany(ctx context.Context, table string, filter interface{}, opts ...DeleteOptions) error
	UpdateOne(ctx context.Context, table string, filter interface{}, update interface{}, opts ...UpdateOptions) error
	//PartialUpdateManyByQuery(ctx context.Context, col string, filter DbQuery, query UnorderedDbQuery) error
	//AggregateWithDiskUse(ctx context.Context, col string, q []DbQuery, v interface{}) error
	//InsertMany(ctx context.Context, tab string, v []interface{}) error
}

// DbIndex holds PostgreSQL index configuration
/* Example usage:
Creating a GIN index on JSONB column with concurrent build
idx: = DbIndex{
    Name: "idx_user_metadata",
    Columns: []string{"metadata"},
    Method: "gin",
    Concurrent: true,
}
*/

type DbIndex struct {
	Name       string   // Index name
	Columns    []string // Column names to index
	Unique     bool     // Whether index is unique
	Method     string   // Index method (e.g., "btree", "hash", "gin", "gist")
	Where      string   // Partial index condition (e.g., "deleted_at IS NULL")
	Include    []string // Columns to include (INCLUDE clause for covering indexes)
	With       string   // WITH clause for index storage parameters
	Tablespace string   // Tablespace for the index
	Concurrent bool     // Whether to create index concurrently
	OpClass    string   // Operator class for the index
}

// DbQuery holds a database query
// DbQuery represents a database query condition for PostgreSQL
type DbQuery map[string]interface{}

// ListOptions provides additional query options
type ListOptions struct {
	Select  []string // Fields to select
	Preload []string // Associations to preload
	GroupBy string   // GROUP BY clause
	Having  DbQuery  // HAVING conditions
}

// DistinctOptions provides additional query options
type DistinctOptions struct {
	OrderBy string // Sorting clause
	Limit   int    // Maximum number of distinct values
}

// BulkWriteModel represents a single write operation
type BulkWriteModel struct {
	Operation string           // "insert", "update", "delete"
	Filter    DbQuery          // For update/delete
	Document  interface{}      // For insert/update
	Update    UnorderedDbQuery // For update operations
}

type DeleteOptions struct {
	HardDelete bool // If false, does soft delete (if model has DeletedAt)
	BatchSize  int  // Batch size for large deletes
}

type UpdateOptions struct {
	ReturnUpdated bool     // Whether to return the updated document
	OmitFields    []string // Fields to exclude from update
}

type UnorderedDbQuery map[string]interface{}
