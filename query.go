package es_filter_query

import (
	"fmt"
	"gopkg.in/olivere/elastic.v5"
	"net/url"
	"strings"
	"time"
)

/* BuildFilterQuery Creates a Elastic query from url.Values and a custom search syntax
any field name ending in _date is parsed as a date in the format of 2006-01-02
any field ending in _datetime is parsed as a date in the format of 01/02/2006 15:04:05 MST
!field_name=value is a NOT query
+field_name=value is an AND query
field_name>=value is a Greater-than or equal to search (any hour within that day)
field_name<=value is a Less-than or equal to search (any hour within that day)
field_name~=fo[o]{1}.%2Bis.%2Bsome is a RegEx search (the field MUST NOT be an analyized field)
     https://www.elastic.co/guide/en/elasticsearch/reference/2.0/query-dsl-regexp-query.html
field_name?=value is a Wildcard search
     https://www.elastic.co/guide/en/elasticsearch/reference/2.0/query-dsl-wildcard-query.html
field_name=value is an exact term search
     https://www.elastic.co/guide/en/elasticsearch/reference/2.0/query-dsl-term-query.html
Default is an exact Term Search
Examples:
    This: current_workflow=run&!status=success&!status=canceled&end_date<=2017-01-27&!context.remediated=true
    Maps to
        Term search: current_workflow = run
        AND
        Term search: NOT status = success
        AND
        Term search: NOT status = canceled
        AND
        Date Parsed Less-than or equal to search: end_date <= 2017-01-27
        AND
        Term search: NOT context.remediated = true
*/
func BuildFilterQuery(searchFilter *elastic.BoolQuery, filters url.Values, filterMap FilterMap) (err error) {
	var query elastic.Query
	var value interface{}

	for name, values := range filters {
		if len(values) == 0 {
			continue
		}
		// Logic to handle != from a query string
		invert := false
		if strings.HasPrefix(name, "!") {
			invert = true
			name = strings.TrimLeft(name, "!")
		}

		// Lookup filter from know filters
		filter, ok := filterMap[name]
		if !ok {
			// Unknown filter (eg not one set in the config)
			filter = buildCustomQuery(name)
		}

		filterQuery := elastic.NewBoolQuery()
		filterQuery.QueryName(name)

		for _, val := range values {
			val = strings.Trim(val, " ")
			if val == "" {
				continue
			}
			value = val

			if filter.Format != "" {
				value, _ = time.Parse(filter.Format, val)
				value.(time.Time).UTC()
			}

			switch filter.Selection {
			case "multi":
				query = elastic.NewTermQuery(filter.Field, value)
			case "date":
				query = createDateQuery(filter.Field, value.(time.Time))
			case "text":
				query = elastic.NewTermQuery(filter.Field, value)
			case "wildcard":
				query = elastic.NewWildcardQuery(filter.Field, fmt.Sprintf("*%s*", value))
			case "regex":
				query = elastic.NewRegexpQuery(filter.Field, fmt.Sprintf("%s", value))
			case "lte":
				query = elastic.NewRangeQuery(filter.Field)
				query.(*elastic.RangeQuery).Lte(value)
			case "gte":
				query = elastic.NewRangeQuery(filter.Field)
				query.(*elastic.RangeQuery).Gte(value)
			default:
				return fmt.Errorf("Unknown Selection: %s", filter.Selection)
			}

			applyQueryLogic(filter, filterQuery, query, invert)
		}
		searchFilter.Filter(filterQuery)
	}

	return
}

func createDateQuery(field string, date time.Time) (query elastic.Query) {
	query = elastic.NewRangeQuery(field)
	query.(*elastic.RangeQuery).
		From(date).
		To(date.Add(24 * time.Hour))

	return
}

func BuildAggregationQuery(fg []FilterGroup) (aggs FilterAggs, err error) {
	ag := make(map[string]elastic.Aggregation, 0)
	for _, group := range fg {
		for _, filter := range group.Filters {
			if filter.Static || (filter.Selection != "multi" && filter.Selection != "single") {
				continue
			}
			ag[filter.Field] = elastic.NewTermsAggregation().Field(filter.Field).Size(0)
		}
	}

	aggs = FilterAggs(ag)
	return
}

func buildCustomQuery(name string) (filter Filter) {
	// default to a multi select or query
	filter = Filter{
		Logic:     "or",
		Selection: "multi",
	}
	// Should we switch to an and query
	if strings.HasPrefix(name, "+") {
		filter.Logic = "and"
		name = strings.TrimLeft(name, "+")
	}
	if strings.HasSuffix(name, "<") {
		filter.Selection = "lte"
		name = strings.TrimRight(name, "<")
	}
	if strings.HasSuffix(name, ">") {
		filter.Selection = "gte"
		name = strings.TrimRight(name, ">")
	}
	if strings.HasSuffix(name, "~") {
		filter.Selection = "regex"
		name = strings.TrimRight(name, "~")
	}
	if strings.HasSuffix(name, "?") {
		filter.Selection = "wildcard"
		name = strings.TrimRight(name, "?")
	}

	// Should we treat this as a date range query?
	if strings.HasSuffix(name, "_date") {
		filter.Format = "2006-01-02"
	}
	if strings.HasSuffix(name, "_datetime") {
		filter.Format = "01/02/2006 15:04:05 MST"
	}

	filter.Field = name

	return
}

func applyQueryLogic(filter Filter, filterQuery *elastic.BoolQuery, query elastic.Query, invert bool) {
	if !invert {
		switch filter.Logic {
		case "or":
			filterQuery.Should(query)
		case "and":
			filterQuery.Must(query)
		default:
			filterQuery.Must(query)
		}
	} else {
		filterQuery.MustNot(query)
	}
}
