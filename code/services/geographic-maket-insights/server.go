package main

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"services/geographic-maket-insights/proto"
	"services/geographic-maket-insights/repository"
)

type geoServer struct {
	proto.UnimplementedGeoMarketInsightsServiceServer
	repo repository.InsightsRepo
}

func NewGeoServer(repo repository.InsightsRepo) proto.GeoMarketInsightsServiceServer {
	return &geoServer{repo: repo}
}

func (s *geoServer) Aggregates(ctx context.Context, req *proto.AggregatesRequest) (*proto.AggregatesResponse, error) {
	if strings.TrimSpace(req.GetBrand()) == "" || strings.TrimSpace(req.GetModel()) == "" {
		return nil, status.Error(codes.InvalidArgument, "brand and model are required")
	}

	groupCol, err := mapGroupBy(req.GetGroupBy())
	if err != nil {
		return nil, err
	}

	limit, skip := s.repo.NormalizePage(req.GetLimit(), req.GetSkip())
	locations := []string{}
	if req.GetLocations() != nil {
		locations = req.GetLocations().GetLocation()
	}

	items, hasNext, err := s.repo.FetchAggregates(
		ctx,
		repository.Filters{
			Brand:    req.GetBrand(),
			Model:    req.GetModel(),
			YearFrom: req.YearFrom,
			YearTo:   req.YearTo,
			FuelType: req.FuelType,
		},
		groupCol,
		locations,
		limit,
		skip,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	metricMask := requestedMetrics(req.GetMetrics())
	respItems := make([]*proto.Aggregates, 0, len(items))
	for _, item := range items {
		row := &proto.Aggregates{
			Location:    item.Location,
			AvgPrice:    item.AvgPrice,
			MedianPrice: item.MedianPrice,
			Count:       item.Count,
		}
		if !metricMask[proto.MetricType_METRIC_TYPE_AVG_PRICE] {
			row.AvgPrice = 0
		}
		if !metricMask[proto.MetricType_METRIC_TYPE_MEDIAN_PRICE] {
			row.MedianPrice = 0
		}
		if !metricMask[proto.MetricType_METRIC_TYPE_COUNT] {
			row.Count = 0
		}
		respItems = append(respItems, row)
	}

	return &proto.AggregatesResponse{
		Aggregates: respItems,
		Pagination: pagination(int32(limit), int32(skip), hasNext),
	}, nil
}

func (s *geoServer) PriceComparison(ctx context.Context, req *proto.PriceComparisonRequest) (*proto.PriceComparisonResponse, error) {
	if strings.TrimSpace(req.GetBrand()) == "" || strings.TrimSpace(req.GetModel()) == "" {
		return nil, status.Error(codes.InvalidArgument, "brand and model are required")
	}

	groupCol, err := mapGroupBy(req.GetGroupBy())
	if err != nil {
		return nil, err
	}

	limit, skip := s.repo.NormalizePage(req.GetLimit(), req.GetSkip())
	items, hasNext, err := s.repo.FetchPriceComparison(
		ctx,
		repository.Filters{Brand: req.GetBrand(), Model: req.GetModel()},
		groupCol,
		mapSort(req.GetSortBy()),
		mapOrder(req.GetOrder()),
		limit,
		skip,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	respItems := make([]*proto.Comparisons, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, &proto.Comparisons{
			Location:     item.Location,
			AveragePrice: item.AveragePrice,
			ListingCount: item.ListingCount,
		})
	}

	return &proto.PriceComparisonResponse{
		Comparisons: respItems,
		Pagination:  pagination(int32(limit), int32(skip), hasNext),
	}, nil
}

func (s *geoServer) ByLocation(ctx context.Context, req *proto.ByLocationRequest) (*proto.ByLocationResponse, error) {
	if strings.TrimSpace(req.GetBrand()) == "" || strings.TrimSpace(req.GetModel()) == "" {
		return nil, status.Error(codes.InvalidArgument, "brand and model are required")
	}

	stats, err := s.repo.FetchByLocation(
		ctx,
		repository.Filters{
			Brand:    req.GetBrand(),
			Model:    req.GetModel(),
			YearFrom: req.YearFrom,
			YearTo:   req.YearTo,
			FuelType: req.FuelType,
		},
		req.Location,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}

	return &proto.ByLocationResponse{Stats: &proto.Stats{
		MinPrice:    stats.MinPrice,
		MaxPrice:    stats.MaxPrice,
		AvgPrice:    stats.AvgPrice,
		MedianPrice: stats.MedianPrice,
	}}, nil
}

func requestedMetrics(metrics []proto.MetricType) map[proto.MetricType]bool {
	all := map[proto.MetricType]bool{
		proto.MetricType_METRIC_TYPE_AVG_PRICE:    true,
		proto.MetricType_METRIC_TYPE_MEDIAN_PRICE: true,
		proto.MetricType_METRIC_TYPE_COUNT:        true,
	}
	if len(metrics) == 0 {
		return all
	}

	mask := map[proto.MetricType]bool{}
	for _, m := range metrics {
		mask[m] = true
	}
	return mask
}

func mapGroupBy(g proto.GroupBy) (string, error) {
	switch g {
	case proto.GroupBy_GROUP_BY_DISTRICT:
		return "district", nil
	case proto.GroupBy_GROUP_BY_CITY:
		return "city", nil
	case proto.GroupBy_GROUP_BY_COUNTRY:
		return "country", nil
	default:
		return "", status.Error(codes.InvalidArgument, "invalid group_by")
	}
}

func mapSort(s proto.SortBy) string {
	switch s {
	case proto.SortBy_SORT_BY_COUNT:
		return "listing_count"
	default:
		return "average_price"
	}
}

func mapOrder(o proto.Order) string {
	switch o {
	case proto.Order_ORDER_DESC:
		return "DESC"
	default:
		return "ASC"
	}
}

func pagination(limit, skip int32, hasNext bool) *proto.Pagination {
	p := &proto.Pagination{Limit: limit, Skip: skip, HasNext: hasNext}
	if hasNext {
		next := skip + limit
		p.NextSkip = &next
	}
	return p
}
