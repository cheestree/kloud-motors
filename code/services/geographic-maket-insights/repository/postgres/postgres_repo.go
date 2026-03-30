package postgres

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"geographic-maket-insights/repository"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RelationalRepo struct {
	pool *pgxpool.Pool
	cfg  repository.DBConfig
}

func NewPostgresRepo(ctx context.Context, cfg repository.DBConfig) (*RelationalRepo, error) {
	if cfg.Dsn == "" {
		return nil, fmt.Errorf("missing POSTGRES_DSN")
	}

	if cfg.DefaultLimit <= 0 {
		cfg.DefaultLimit = 20
	}
	if cfg.MaxLimit <= 0 {
		cfg.MaxLimit = 100
	}
	if cfg.Schema == "" {
		cfg.Schema = "public"
	}
	if cfg.Table == "" {
		cfg.Table = "listings"
	}

	pool, err := pgxpool.New(ctx, cfg.Dsn)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	return &RelationalRepo{pool: pool, cfg: cfg}, nil
}

func (db *RelationalRepo) NormalizePage(limitRaw, skipRaw int32) (int, int) {
	limit := int(limitRaw)
	if limit <= 0 {
		limit = db.cfg.DefaultLimit
	}
	if limit > db.cfg.MaxLimit {
		limit = db.cfg.MaxLimit
	}

	skip := int(skipRaw)
	if skip < 0 {
		skip = 0
	}
	return limit, skip
}

func (db *RelationalRepo) FetchAggregates(ctx context.Context, filters repository.Filters, groupCol string, locations []string, limit, skip int) ([]repository.AggregateRow, bool, error) {
	whereSQL, args := buildBaseFilters(filters)
	if len(locations) > 0 {
		args = append(args, locations)
		whereSQL += fmt.Sprintf(" AND %s = ANY($%d)", groupCol, len(args))
	}

	args = append(args, limit+1, skip)
	q := fmt.Sprintf(`
		SELECT
			%s AS location,
			COALESCE(AVG(price)::bigint, 0)::int AS avg_price,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY price)::bigint, 0)::int AS median_price,
			COUNT(*)::int AS count
		FROM %s
		%s
		GROUP BY %s
		ORDER BY %s ASC
		LIMIT $%d OFFSET $%d`,
		groupCol, db.qualifiedTable(), whereSQL, groupCol, groupCol, len(args)-1, len(args))

	rows, err := db.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, false, fmt.Errorf("query aggregates: %w", err)
	}
	defer rows.Close()

	out := make([]repository.AggregateRow, 0, limit+1)
	for rows.Next() {
		var row repository.AggregateRow
		if err := rows.Scan(&row.Location, &row.AvgPrice, &row.MedianPrice, &row.Count); err != nil {
			return nil, false, fmt.Errorf("scan aggregates: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate aggregates: %w", err)
	}

	hasNext := len(out) > limit
	if hasNext {
		out = out[:limit]
	}
	return out, hasNext, nil
}

func (db *RelationalRepo) FetchPriceComparison(
	ctx context.Context,
	filters repository.Filters,
	groupCol, sortCol, order string,
	limit, skip int) ([]repository.ComparisonRow, bool, error) {
	whereSQL, args := buildBaseFilters(filters)
	args = append(args, limit+1, skip)

	q := fmt.Sprintf(`
		SELECT
			%s AS location,
			COALESCE(AVG(price)::bigint, 0)::int AS average_price,
			COUNT(*)::int AS listing_count
		FROM %s
		%s
		GROUP BY %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, groupCol, db.qualifiedTable(), whereSQL, groupCol, sortCol, order, len(args)-1, len(args))

	rows, err := db.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, false, fmt.Errorf("query price comparison: %w", err)
	}
	defer rows.Close()

	out := make([]repository.ComparisonRow, 0, limit+1)
	for rows.Next() {
		var row repository.ComparisonRow
		if err := rows.Scan(&row.Location, &row.AveragePrice, &row.ListingCount); err != nil {
			return nil, false, fmt.Errorf("scan price comparison: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate price comparison: %w", err)
	}

	hasNext := len(out) > limit
	if hasNext {
		out = out[:limit]
	}
	return out, hasNext, nil
}

func (db *RelationalRepo) FetchByLocation(ctx context.Context, filters repository.Filters,
	location *string) (repository.StatsRow, error) {
	whereSQL, args := buildBaseFilters(filters)
	if location != nil && strings.TrimSpace(*location) != "" {
		args = append(args, *location)
		idx := len(args)
		whereSQL += fmt.Sprintf(" AND (district = $%d OR city = $%d OR country = $%d)", idx, idx, idx)
	}

	q := fmt.Sprintf(`
		SELECT
			COALESCE(MIN(price), 0)::int AS min_price,
			COALESCE(MAX(price), 0)::int AS max_price,
			COALESCE(AVG(price)::bigint, 0)::int AS avg_price,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY price)::bigint, 0)::int AS median_price
		FROM %s
		%s`, db.qualifiedTable(), whereSQL)

	var out repository.StatsRow
	if err := db.pool.QueryRow(ctx, q, args...).Scan(&out.MinPrice, &out.MaxPrice, &out.AvgPrice, &out.MedianPrice); err != nil {
		return repository.StatsRow{}, fmt.Errorf("query by location stats: %w", err)
	}
	return out, nil
}

func buildBaseFilters(filters repository.Filters) (string, []any) {
	clauses := []string{"brand = $1", "model = $2"}
	args := []any{filters.Brand, filters.Model}

	if filters.YearFrom != nil {
		args = append(args, *filters.YearFrom)
		clauses = append(clauses, fmt.Sprintf("year >= $%d", len(args)))
	}
	if filters.YearTo != nil {
		args = append(args, *filters.YearTo)
		clauses = append(clauses, fmt.Sprintf("year <= $%d", len(args)))
	}
	if filters.FuelType != nil && strings.TrimSpace(*filters.FuelType) != "" {
		args = append(args, *filters.FuelType)
		clauses = append(clauses, fmt.Sprintf("fuel_type = $%d", len(args)))
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

var identRx = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func safeIdent(v string) string {
	if !identRx.MatchString(v) {
		return ""
	}
	return `"` + v + `"`
}

func (db *RelationalRepo) qualifiedTable() string {
	schema := safeIdent(db.cfg.Schema)
	table := safeIdent(db.cfg.Table)
	if schema == "" || table == "" {
		return `"public"."listings"`
	}
	return schema + "." + table
}

func (db *RelationalRepo) Close() error {
	db.pool.Close()
	return nil
}
