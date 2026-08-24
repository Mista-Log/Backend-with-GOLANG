package main

import (
	"net/url"
	"sort"
	"strconv"
)

// filterProducts applies every recognized query parameter, skipping any
// that weren't provided — the guide's Filtering section, generalized to
// several fields at once.
func filterProducts(products []*Product, q url.Values) []*Product {
	category := q.Get("category")
	minPriceStr := q.Get("minPrice")
	maxPriceStr := q.Get("maxPrice")
	inStockStr := q.Get("inStock")

	var minPrice, maxPrice float64
	hasMin := minPriceStr != ""
	hasMax := maxPriceStr != ""
	if hasMin {
		minPrice, _ = strconv.ParseFloat(minPriceStr, 64)
	}
	if hasMax {
		maxPrice, _ = strconv.ParseFloat(maxPriceStr, 64)
	}

	var filtered []*Product
	for _, p := range products {
		if category != "" && p.Category != category {
			continue
		}
		if hasMin && p.Price < minPrice {
			continue
		}
		if hasMax && p.Price > maxPrice {
			continue
		}
		if inStockStr != "" {
			wantInStock := inStockStr == "true"
			if p.InStock != wantInStock {
				continue
			}
		}
		filtered = append(filtered, p)
	}
	return filtered
}

// allowedSortFields is an ALLOW-LIST — the guide's Sorting section warns
// against passing an unchecked client value into anything resembling a
// query's ORDER BY; here it just guards against an unrecognized field
// silently falling through to the default instead of erroring, but the
// principle is the same one that matters far more with a real database.
var allowedSortFields = map[string]bool{
	"name":  true,
	"price": true,
	"id":    true,
}

func sortProducts(products []*Product, sortField, order string) {
	if !allowedSortFields[sortField] {
		sortField = "id"
	}
	desc := order == "desc"

	sort.Slice(products, func(i, j int) bool {
		var less bool
		switch sortField {
		case "name":
			less = products[i].Name < products[j].Name
		case "price":
			less = products[i].Price < products[j].Price
		default:
			less = products[i].ID < products[j].ID
		}
		if desc {
			return !less
		}
		return less
	})
}

type PagedResult struct {
	Data       []*Product `json:"data"`
	Page       int        `json:"page"`
	Limit      int        `json:"limit"`
	Total      int        `json:"total"`
	TotalPages int        `json:"totalPages"`
}

func paginate(products []*Product, page, limit int) PagedResult {
	total := len(products)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit
	var pageItems []*Product
	if offset < total {
		end := offset + limit
		if end > total {
			end = total
		}
		pageItems = products[offset:end]
	} else {
		pageItems = []*Product{}
	}

	totalPages := (total + limit - 1) / limit
	if totalPages == 0 {
		totalPages = 1
	}

	return PagedResult{Data: pageItems, Page: page, Limit: limit, Total: total, TotalPages: totalPages}
}
