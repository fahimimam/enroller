package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"github.com/triapex/auth/config"
	"github.com/triapex/auth/internal/infra"
	defaultLogger "github.com/triapex/auth/logger"
	"github.com/triapex/auth/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"reflect"
	"strings"
	"time"
)

// Postgres holds necessery fields and
// postgres database session to connect
type Postgres struct {
	*gorm.DB
	name string
	lgr  defaultLogger.Logger
}

func NewConnection(ctx context.Context, cfgPostgres *config.Postgres) (*Postgres, error) {

	// Retrieve configuration values
	dbHost := cfgPostgres.DBHost
	dbPort := cfgPostgres.DBPort
	dbName := cfgPostgres.DBName
	dbUser := cfgPostgres.DBUser
	dbPassword := cfgPostgres.DBPassword
	dbSSLMode := cfgPostgres.DBSSLMode
	dbTimeZone := cfgPostgres.DBTimeZone
	//level := cfgPostgres.Level

	// Construct DSN
	dsn := fmt.Sprintf(
		"host=%s port=%v user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode, dbTimeZone,
	)

	// Configure GORM
	gConf := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	// Create database connection
	con, err := gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
	}), gConf)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Get underlying SQL DB to configure connection pool
	sqlDB, err := con.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Verify database connectivity with context
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close() // Close connection if ping fails
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// Configure connection pool parameters
	dbMaxOpenConn := viper.GetInt("database.max_open_connections")
	dbMaxIdleConn := viper.GetInt("database.max_idle_connections")

	if dbMaxOpenConn == 0 {
		dbMaxOpenConn = 10
	}
	if dbMaxIdleConn == 0 {
		dbMaxIdleConn = 5
	}

	sqlDB.SetConnMaxLifetime(time.Second * 10)
	sqlDB.SetMaxOpenConns(dbMaxOpenConn)
	sqlDB.SetMaxIdleConns(dbMaxIdleConn)

	db := &Postgres{
		DB:   con,
		name: dbName,
	}

	return db, nil
}

func (p *Postgres) println(args ...interface{}) {
	if p.lgr != nil {
		p.lgr.Println(args...)
	}
}

func (p *Postgres) Ping(ctx context.Context) error {
	sqlDB, err := p.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	return sqlDB.PingContext(ctx)
}

func (p *Postgres) Close(ctx context.Context) error {
	// Get the underlying sql.DB from GORM
	sqlDB, err := p.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	// Create a channel to receive the close result
	done := make(chan error, 1)

	// Run the close operation in a goroutine so we can respect context
	go func() {
		done <- sqlDB.Close()
	}()

	// Wait for either the close to complete or the context to expire
	select {
	case <-ctx.Done():
		return ctx.Err() // Return context cancellation error
	case err := <-done:
		return err // Return the close result
	}
}

// EnsureIndices with proper transaction handling
func (p *Postgres) EnsureIndices(ctx context.Context, table string, indices []infra.DbIndex) error {
	p.println("creating indices for", table)

	// Begin transaction
	tx := p.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Use defer with explicit rollback check
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after rollback
		}
	}()

	for _, index := range indices {
		// Build CREATE INDEX statement
		stmt := strings.Builder{}
		stmt.WriteString("CREATE ")

		if index.Unique {
			stmt.WriteString("UNIQUE ")
		}

		if index.Concurrent {
			stmt.WriteString("CONCURRENTLY ")
		}

		stmt.WriteString(fmt.Sprintf("INDEX %s ON %s USING %s (",
			index.Name,
			table,
			index.Method))

		// Add columns
		for i, col := range index.Columns {
			if i > 0 {
				stmt.WriteString(", ")
			}
			stmt.WriteString(col)
			if index.OpClass != "" {
				stmt.WriteString(" " + index.OpClass)
			}
		}
		stmt.WriteString(")")

		// Add INCLUDE columns if specified
		if len(index.Include) > 0 {
			stmt.WriteString(" INCLUDE (")
			stmt.WriteString(strings.Join(index.Include, ", "))
			stmt.WriteString(")")
		}

		// Add WHERE clause if specified
		if index.Where != "" {
			stmt.WriteString(" WHERE " + index.Where)
		}

		// Add WITH storage parameters if specified
		if index.With != "" {
			stmt.WriteString(" WITH (" + index.With + ")")
		}

		// Add tablespace if specified
		if index.Tablespace != "" {
			stmt.WriteString(" TABLESPACE " + index.Tablespace)
		}
		if err := tx.Exec(stmt.String()).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create index %s: %w", index.Name, err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit index creation transaction: %w", err)
	}

	return nil
}

// DropIndices drops specified indices from a table
func (p *Postgres) DropIndices(ctx context.Context, table string, indices []infra.DbIndex) error {
	p.println("dropping indices from", table)

	// Begin transaction
	tx := p.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Use deferring with explicit rollback check
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r) // re-throw panic after rollback
		}
	}()

	for _, index := range indices {
		// For concurrent operations, disable statement timeout
		if index.Concurrent {
			if err := tx.Exec("SET LOCAL statement_timeout = 0").Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to disable timeout for concurrent drop: %w", err)
			}
		}

		// Build a DROP INDEX statement
		stmt := fmt.Sprintf("DROP INDEX IF EXISTS %s", index.Name)
		if err := tx.Exec(stmt).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to drop index %s: %w", index.Name, err)
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit index drop transaction: %w", err)
	}

	return nil
}

// Insert inserts one or more documents into the specified table
func (p *Postgres) Insert(ctx context.Context, table string, docs interface{}) error {
	p.println("insert into", table)

	// Handle batch inserts if docs is a slice
	if reflect.TypeOf(docs).Kind() == reflect.Slice {
		result := p.DB.WithContext(ctx).Table(table).CreateInBatches(docs, 100) // Batch size of 100
		if result.Error != nil {
			return fmt.Errorf("failed to insert documents: %w", result.Error)
		}
		return nil
	}

	// Single document insert
	// On conflict does nothing
	result := p.DB.WithContext(ctx).Table(table).Create(docs)
	if result.Error != nil {
		if utils.IsDup(result.Error) {
			return infra.ErrDuplicateKey
		}
		return fmt.Errorf("failed to insert document: %w", result.Error)
	}

	return nil
}

// List finds documents with advanced options
func (p *Postgres) List(ctx context.Context, table string, filter infra.DbQuery, skip, limit int, v interface{}, sorts []string, opts ...*infra.ListOptions) error {
	query := p.DB.WithContext(ctx).Table(table)

	// Apply WHERE conditions
	for field, value := range filter {
		query = query.Where(field, value)
	}

	// Apply sorting
	for _, sort := range sorts {
		query = query.Order(sort)
	}

	// Apply pagination
	if skip > 0 {
		query = query.Offset(skip)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	// Apply additional options
	if len(opts) > 0 {
		opt := opts[0]
		if len(opt.Select) > 0 {
			query = query.Select(opt.Select)
		}
		for _, association := range opt.Preload {
			query = query.Preload(association)
		}
		if opt.GroupBy != "" {
			query = query.Group(opt.GroupBy)
			for field, value := range opt.Having {
				query = query.Having(field, value)
			}
		}
	}

	// Execute query
	result := query.Find(v)
	if result.Error != nil {
		return fmt.Errorf("failed to list documents: %w", result.Error)
	}

	return nil
}

/*
	FindOne

// 1. Single sort field
err: = db.FindOne(ctx, "users",

	DbQuery{"status": "active"},
	&user,
	"created_at DESC")

// 2. Multiple sort fields
err:= db.FindOne(ctx, "users",

	DbQuery{"company_id": 5},
	&user,
	"last_name ASC", "hire_date DESC")
*/
func (p *Postgres) FindOne(ctx context.Context, table string, q infra.DbQuery, preloads []string, v interface{}, sorts ...string) error {
	p.println("find", q, "from", table)

	query := p.DB.WithContext(ctx).Table(table)

	// Apply WHERE conditions
	for field, value := range q {
		query = query.Where(field, value)
	}

	// Apply sorting
	for _, sort := range sorts {
		query = query.Order(sort)
	}

	// Apply preloads dynamically
	for _, preload := range preloads {
		query = query.Preload(preload)
	}

	// Get first result
	result := query.First(v)

	// Handle errors
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return infra.ErrNotFound
		}
		return fmt.Errorf("failed to find document: %w", result.Error)
	}

	return nil
}

// PartialUpdateMany updates multiple rows with partial data
func (p *Postgres) PartialUpdateMany(ctx context.Context, table string, filter infra.DbQuery, data interface{}) error {
	p.println("partial update many in", table, "where", filter)

	// Get the underlying map if data is a struct
	updateData, err := utils.ToMap(data)
	if err != nil {
		return fmt.Errorf("failed to convert update data: %w", err)
	}

	// Execute update
	result := p.DB.WithContext(ctx).
		Table(table).
		Where(filter).
		Updates(updateData)

	if result.Error != nil {
		return fmt.Errorf("failed to update records: %w", result.Error)
	}

	p.println("updated", result.RowsAffected, "rows")
	return nil
}

// Aggregate performs SQL aggregation operations
func (p *Postgres) Aggregate(ctx context.Context, table string, stages []infra.DbQuery, v interface{}) error { // v: Pointer to result slice  stages: Map of SQL clauses
	p.println("aggregate", stages, "from", table)

	query := p.DB.WithContext(ctx).Table(table)

	// Process each aggregation stage
	for _, stage := range stages {
		for op, params := range stage {
			switch op {
			case "select":
				query = query.Select(params)
			case "where":
				query = query.Where(params)
			case "group":
				query = query.Group(params.(string))
			case "having":
				query = query.Having(params)
			case "order":
				query = query.Order(params.(string))
			case "join":
				if join, ok := params.(map[string]interface{}); ok {
					query = query.Joins(
						fmt.Sprintf("%s %s ON %s",
							join["type"],
							join["table"],
							join["on"],
						),
					)
				}
			case "limit":
				query = query.Limit(params.(int))
			case "offset":
				query = query.Offset(params.(int))
			}
		}
	}

	// Execute and scan results
	if err := query.Find(v).Error; err != nil {
		return fmt.Errorf("aggregation failed: %w", err)
	}
	return nil
}

// Distinct returns distinct values with options
func (p *Postgres) Distinct(ctx context.Context, table string, field string, q infra.DbQuery, v interface{}, opts ...*infra.DistinctOptions) error {
	query := p.DB.WithContext(ctx).Table(table)

	// Apply WHERE conditions
	if len(q) > 0 {
		query = query.Where(q)
	}

	// Apply options
	if len(opts) > 0 {
		opt := opts[0]
		if opt.OrderBy != "" {
			query = query.Order(opt.OrderBy)
		}
		if opt.Limit > 0 {
			query = query.Limit(opt.Limit)
		}
	}

	// Execute distinct query
	result := query.Distinct(field).Find(v)
	if result.Error != nil {
		return fmt.Errorf("failed to get distinct values: %w", result.Error)
	}

	return nil
}

//func (p *Postgres) PartialUpdateManyByQuery(ctx context.Context, col string, filter infra.DbQuery, query infra.UnorderedDbQuery) error {
//	_, err := p.database.Collection(col).UpdateMany(ctx, filter, query)
//	if err != nil {
//		return err
//	}
//	return nil
//}

// BulkUpdate executes multiple write operations in a single transaction
func (p *Postgres) BulkUpdate(ctx context.Context, table string, models []infra.BulkWriteModel) error {
	// Begin transaction
	tx := p.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Process each operation
	for _, model := range models {
		switch model.Operation {
		case "insert":
			if err := tx.Table(table).Create(model.Document).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("bulk insert failed: %w", err)
			}

		case "update":
			if model.Update != nil {
				// Handle complex updates
				updateData, err := utils.ToMap(model.Update)
				if err != nil {
					tx.Rollback()
					return fmt.Errorf("invalid update data: %w", err)
				}
				res := tx.Table(table).Where(model.Filter).Updates(updateData)
				if res.Error != nil {
					tx.Rollback()
					return fmt.Errorf("bulk update failed: %w", res.Error)
				}
			} else if model.Document != nil {
				// Full document replacement
				res := tx.Table(table).Where(model.Filter).Updates(model.Document)
				if res.Error != nil {
					tx.Rollback()
					return fmt.Errorf("bulk replace failed: %w", res.Error)
				}
			}

		case "delete":
			res := tx.Table(table).Where(model.Filter).Delete(nil)
			if res.Error != nil {
				tx.Rollback()
				return fmt.Errorf("bulk delete failed: %w", res.Error)
			}

		default:
			tx.Rollback()
			return fmt.Errorf("unsupported bulk operation: %s", model.Operation)
		}
	}

	// Commit transaction
	return tx.Commit().Error
}

// DeleteMany deletes multiple records matching the filter
func (p *Postgres) DeleteMany(ctx context.Context, table string, filter interface{}, opts ...infra.DeleteOptions) error {
	p.println("delete many from", table, "where", filter)

	options := infra.DeleteOptions{
		HardDelete: false,
		BatchSize:  1000,
	}

	if len(opts) > 0 {
		options = opts[0]
	}

	// Begin transaction (recommended for multi-record operations)
	tx := p.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Execute delete
	query := tx.Table(table).Where(filter)
	if options.HardDelete {
		query = query.Unscoped()
	}

	if options.BatchSize > 0 {
		query = query.Limit(options.BatchSize)
	}

	result := query.Delete(nil)
	p.println("deleted", result.RowsAffected, "rows from", table)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	return nil
}

func (p *Postgres) UpdateOne(ctx context.Context, table string, filter interface{}, update interface{}, opts ...infra.UpdateOptions) error {
	// Default optsForUpdate
	optsForUpdate := infra.UpdateOptions{
		ReturnUpdated: false,
		OmitFields:    nil,
	}
	if len(opts) > 0 {
		optsForUpdate = opts[0]
	}

	tx := p.DB.WithContext(ctx).Begin()
	query := tx.Table(table).Where(filter).Limit(1)

	if len(optsForUpdate.OmitFields) > 0 {
		query = query.Omit(optsForUpdate.OmitFields...)
	}

	result := query.Updates(update)
	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("update failed: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		tx.Rollback()
		return infra.ErrNotFound
	}

	p.println("updated 1 row in", table)
	if optsForUpdate.ReturnUpdated {
		//return  TODO: Currently not implementing...
	}

	return tx.Commit().Error
}
