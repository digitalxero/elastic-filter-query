package es_filter_query

import (
	"encoding/xml"
	"gopkg.in/olivere/elastic.v5"
)

type FilterGroup struct {
	XMLName xml.Name `json:"-" yaml:"-" xml:"Filter"`
	Label   string   `json:"label" yaml:"label" xml:",attr"`
	Class   string   `json:"class" yaml:"class" xml:",attr"`
	Filters []Filter `json:"filters" yaml:"filters" xml:">Filter"`
}

type Filter struct {
	XMLName   xml.Name `json:"-" yaml:"-" xml:"Filter"`
	ID        string   `json:"id" yaml:"id" xml:",attr"`
	Label     string   `json:"label" yaml:"label" xml:",attr"`
	Selection string   `json:"selection" yaml:"selection" xml:",attr"`
	Logic     string   `json:"logic" yaml:"logic" xml:",attr"`
	Field     string   `json:"field" yaml:"field" xml:",attr"`
	Static    bool     `json:"static" yaml:"static" xml:",attr"`
	Facets    []Facet  `json:"facets" yaml:"facets" xml:">Facet"`
	Format    string   `json:"format" yaml:"format" xml:"format"`
}

type Facet struct {
	XMLName xml.Name `json:"-" yaml:"-" xml:"Facet"`
	ID      string   `json:"id" yaml:"id" xml:",attr"`
	Label   string   `json:"label" yaml:"label" xml:",attr"`
	Query   string   `json:"query" yaml:"query" xml:",chardata"`
	Count   int64    `json:"count" yaml:"count" xml:",attr"`
}

type FilterMap map[string]Filter

type FilterAggs map[string]elastic.Aggregation

func (f *FilterGroup)GetFilter(fieldName string) (filter Filter) {
	for _, filter = range f.Filters {
		if filter.Field == fieldName {
			return
		}
	}

	return Filter{}
}

func (f *FilterGroup)ReplaceFilter(filter Filter) {

	for idx, fil := range f.Filters {
		if fil.Field == filter.Field {
			f.Filters[idx] = filter
		}
	}
}

func (f *Filter)GetFacet(label string) (facet Facet) {
	for _, facet = range f.Facets {
		if facet.Label == label {
			return
		}
	}

	return Facet{}
}
