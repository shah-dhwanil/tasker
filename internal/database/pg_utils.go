package database

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shah-dhwanil/tasker/internal/errors"
)

// StructToNamedArgs maps an arbitrary struct to pgx.NamedArgs.
// It uses the `db` struct tag for column naming and supports the `omitempty` option.
//
// The struct argument must be a struct (not a pointer to a struct).
//
// Example usage:
//
//	type User struct {
//	    ID    int64  `db:"id,omitempty"`
//	    Name  string `db:"user_name"`
//	    Email string `db:"email,omitempty"`
//	}
//
// user := User{Name: "Alice"}
// args, err := StructToNamedArgs(user) // args will be pgx.NamedArgs{"user_name": "Alice"}
func StructToNamedArgs(s any) (pgx.NamedArgs, error) {
	v := reflect.ValueOf(s)

	// Ensure the input is a struct
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, fmt.Errorf("StructToNamedArgs received a nil pointer")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("StructToNamedArgs expected a struct, got %s", v.Kind())
	}

	t := v.Type()
	args := make(pgx.NamedArgs)

	// Iterate over all fields of the struct
	for i := 0; i < t.NumField(); i++ {
		fieldT := t.Field(i)
		fieldV := v.Field(i)

		// Only process exported fields (names start with an uppercase letter)
		if !fieldV.CanInterface() {
			continue
		}

		// 1. Get the 'db' tag value
		tag, ok := fieldT.Tag.Lookup("db")
		if !ok {
			// Skip fields without a 'db' tag
			continue
		}

		// 2. Parse the tag for name and options (e.g., "column_name,omitempty")
		parts := strings.Split(tag, ",")
		columnName := parts[0]

		// Ignore fields explicitly tagged with `db:"-"`
		if columnName == "-" {
			continue
		}

		// Check for omitempty option
		isOmitEmpty := false
		if len(parts) > 1 && strings.Contains(parts[1], "omitempty") {
			isOmitEmpty = true
		}

		// 3. Check for zero value if omitempty is set
		if isOmitEmpty {
			// reflect.Value.IsZero() is the canonical way to check for a zero value in Go 1.13+
			if fieldV.IsZero() {
				continue // Skip the field if it's zero value and omitempty is present
			}
		}

		// 4. Add to NamedArgs
		args[columnName] = fieldV.Interface()
	}

	return args, nil
}

// --- Example Usage ---

/*
// To make this file runnable, you'd typically include a main function and necessary imports.
// For demonstration, here's how the example struct would look and how the function is used.

import (
    "time"
    "fmt"
)

// A sample struct representing a row in a database table.
type UpdateModel struct {
	ID        int64      `db:"id"`
	Title     string     `db:"title,omitempty"`
	Content   string     `db:"content,omitempty"`
	Views     int        `db:"view_count,omitempty"`
	UpdatedAt *time.Time `db:"updated_at,omitempty"`
	// A field we want to ignore
	IgnoreMe string `json:"ignore"`
}

func main() {
	now := time.Now()

	// Case 1: All fields populated (no omitempty is triggered)
	model1 := UpdateModel{
		ID: 101,
		Title: "New Title",
		Content: "Some content",
		Views: 5,
		UpdatedAt: &now,
	}
	args1, _ := StructToNamedArgs(model1)
	fmt.Println("Case 1 (All fields):", args1)
	// Output: pgx.NamedArgs{"id":101, "title":"New Title", "content":"Some content", "view_count":5, "updated_at":&time.Time{...}}

	// Case 2: Zero values (Title="", Content="", Views=0, UpdatedAt=nil) are omitted
	model2 := UpdateModel{
		ID: 202,
	}
	args2, _ := StructToNamedArgs(model2)
	fmt.Println("Case 2 (Omitted fields):", args2)
	// Output: pgx.NamedArgs{"id":202} // Title, Content, Views, UpdatedAt are omitted

	// Case 3: Zero value on a non-omitempty field (ID is int64's zero value but no omitempty)
	// This shows ID will still be included as 0, as only omitempty fields are skipped.
	model3 := UpdateModel{
		ID: 0, // Zero value
		Title: "Important Update",
	}
	args3, _ := StructToNamedArgs(model3)
	fmt.Println("Case 3 (ID=0 not omitted):", args3)
	// Output: pgx.NamedArgs{"id":0, "title":"Important Update"}
}
*/

func ExtractOrderParam(orderBy string) (string,string) {
	if strings.HasPrefix(orderBy, "-") {
		return orderBy[1:], "DESC"
	}
	if strings.HasPrefix(orderBy, "+") {
		return orderBy[1:], "ASC"
	}
	return orderBy, "ASC"
}

func ConstructWhereClause(conditions []string) string {
	if len(conditions) == 0 {
		return "1=1" // No conditions, so we return a tautology
	}
	return strings.Join(conditions, " AND ")
}

func ConstructOrderByClause(orderBy []string) string {
	if len(orderBy) == 0 {
		return "created_at DESC" // Default order by
	}
	return strings.Join(orderBy, ", ")
}

func ConstructSetClause(fields []string) string {
	return strings.Join(fields, ", ")
}

func QueryInTransaction[T any](ctx context.Context, executor DBTX, fn func(pgx.Tx)(T,error)) (T,error) {
	var zero T
	txn,err:=executor.Begin(ctx)
	if err!=nil {
		dbError,ok :=errors.ConvertPgError(err)
		if ok {
			return zero,dbError
		}
		return zero,errors.NewUnknownError(err,"Database Error","Unknown Error while starting transaction",nil)
	}
	rows,err:=fn(txn)
	if err!=nil {
		if rbErr:=txn.Rollback(ctx);rbErr!=nil {
			return zero,fmt.Errorf("transaction failed: %v, rollback failed: %v", err, rbErr)
		}
		dbError,ok :=errors.ConvertPgError(err)
		if ok {
			return zero,dbError
		}
		return zero,errors.NewUnknownError(err,"Database Error","Unknown Error while executing transaction",nil)
	}
	if err = txn.Commit(ctx); err != nil {
		dbError,ok :=errors.ConvertPgError(err)
		if ok {
			return zero,dbError
		}
		return zero,errors.NewUnknownError(err,"Database Error","Unknown Error while committing transaction",nil)
	}
	return rows,nil
}

var ExecuteInTransaction func(ctx context.Context, executor DBTX, fn func(pgx.Tx) (pgconn.CommandTag,error)) (pgconn.CommandTag,error) = QueryInTransaction