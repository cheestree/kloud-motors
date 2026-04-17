package postgres

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"services/geographic-maket-insights/repository"

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
		cfg.Table = "automotive_data"
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
	groupExpr, err := locationExpr(groupCol)
	if err != nil {
		return nil, false, err
	}

	whereSQL, args := buildBaseFilters(filters)
	if len(locations) > 0 {
		args = append(args, locations)
		whereSQL += fmt.Sprintf(" AND %s = ANY($%d)", groupExpr, len(args))
	}

	args = append(args, limit+1, skip)
	q := fmt.Sprintf(`
		SELECT
			COALESCE(%s, '') AS location,
			COALESCE(AVG(f.ask_price)::bigint, 0)::int AS avg_price,
			COALESCE(
				(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY f.ask_price))::bigint,
				0
			)::int AS median_price,
			COUNT(*)::int AS count
		%s
		%s
		GROUP BY %s
		ORDER BY %s ASC
		LIMIT $%d OFFSET $%d`,
		groupExpr, db.baseFromSQL(), whereSQL, groupExpr, groupExpr, len(args)-1, len(args))

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
	groupExpr, err := locationExpr(groupCol)
	if err != nil {
		return nil, false, err
	}

	orderBy, err := comparisonSortExpr(sortCol)
	if err != nil {
		return nil, false, err
	}

	whereSQL, args := buildBaseFilters(filters)
	args = append(args, limit+1, skip)

	q := fmt.Sprintf(`
		SELECT
			COALESCE(%s, '') AS location,
			COALESCE(AVG(f.ask_price)::bigint, 0)::int AS average_price,
			COUNT(*)::int AS listing_count
		%s
		%s
		GROUP BY %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d`, groupExpr, db.baseFromSQL(), whereSQL, groupExpr, orderBy, safeOrder(order), len(args)-1, len(args))

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
		args = append(args, strings.TrimSpace(*location))
		idx := len(args)
		whereSQL += fmt.Sprintf(" AND (f.district = $%d OR f.city = $%d OR f.country = $%d OR f.state = $%d)", idx, idx, idx, idx)
	}

	q := fmt.Sprintf(`
		SELECT
			COALESCE(MIN(f.ask_price), 0)::int AS min_price,
			COALESCE(MAX(f.ask_price), 0)::int AS max_price,
			COALESCE(AVG(f.ask_price)::bigint, 0)::int AS avg_price,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY f.ask_price)::bigint, 0)::int AS median_price
		%s
		%s`, db.baseFromSQL(), whereSQL)

	var out repository.StatsRow
	if err := db.pool.QueryRow(ctx, q, args...).Scan(&out.MinPrice, &out.MaxPrice, &out.AvgPrice, &out.MedianPrice); err != nil {
		return repository.StatsRow{}, fmt.Errorf("query by location stats: %w", err)
	}
	return out, nil
}

func buildBaseFilters(filters repository.Filters) (string, []any) {
	clauses := []string{"b.name = $1", "m.name = $2", "f.ask_price IS NOT NULL"}
	args := []any{strings.TrimSpace(filters.Brand), strings.TrimSpace(filters.Model)}

	if filters.YearFrom != nil {
		args = append(args, *filters.YearFrom)
		clauses = append(clauses, fmt.Sprintf("f.model_year >= $%d", len(args)))
	}
	if filters.YearTo != nil {
		args = append(args, *filters.YearTo)
		clauses = append(clauses, fmt.Sprintf("f.model_year <= $%d", len(args)))
	}
	if filters.FuelType != nil && strings.TrimSpace(*filters.FuelType) != "" {
		args = append(args, strings.TrimSpace(*filters.FuelType))
		clauses = append(clauses, fmt.Sprintf("ft.name = $%d", len(args)))
	}

	return "WHERE " + strings.Join(clauses, " AND "), args
}

func locationExpr(groupCol string) (string, error) {
	switch groupCol {
	case "district":
		return "f.district", nil
	case "city":
		return "f.city", nil
	case "country":
		return "f.country", nil
	default:
		return "", fmt.Errorf("invalid group column: %s", groupCol)
	}
}

func comparisonSortExpr(sortCol string) (string, error) {
	switch sortCol {
	case "average_price", "listing_count":
		return sortCol, nil
	default:
		return "", fmt.Errorf("invalid sort column: %s", sortCol)
	}
}

func safeOrder(order string) string {
	if strings.EqualFold(order, "DESC") {
		return "DESC"
	}
	return "ASC"
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
		return `"public"."automotive_data"`
	}
	return schema + "." + table
}

func (db *RelationalRepo) qualifiedDimensionTable(table string) string {
	schema := safeIdent(db.cfg.Schema)
	dimension := safeIdent(table)
	if schema == "" || dimension == "" {
		return `"public"."` + table + `"`
	}
	return schema + "." + dimension
}

func (db *RelationalRepo) baseFromSQL() string {
	return fmt.Sprintf(`
		FROM %s f
		JOIN %s b ON b.id = f.brand_id
		JOIN %s m ON m.id = f.model_id AND m.brand_id = b.id
		LEFT JOIN %s ft ON ft.id = f.fuel_type_id`,
		db.qualifiedTable(),
		db.qualifiedDimensionTable("brand"),
		db.qualifiedDimensionTable("model"),
		db.qualifiedDimensionTable("fuel_type"),
	)
}

func (db *RelationalRepo) Close() error {
	db.pool.Close()
	return nil
}
