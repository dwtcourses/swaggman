package openapi3

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	oas3 "github.com/getkin/kin-openapi/openapi3"
	"github.com/grokify/gocharts/data/table"
	"github.com/grokify/simplego/encoding/jsonutil"
	"github.com/grokify/simplego/net/urlutil"
	"github.com/grokify/simplego/text"
	"github.com/grokify/simplego/type/stringsutil"
)

type SpecMore struct {
	Spec *oas3.Swagger
}

func ReadSpecMore(path string, validate bool) (*SpecMore, error) {
	spec, err := ReadFile(path, validate)
	if err != nil {
		return nil, err
	}
	return &SpecMore{Spec: spec}, nil
}

func (sm *SpecMore) SchemasCount() int {
	if sm.Spec == nil {
		return -1
	} else if sm.Spec.Components.Schemas == nil {
		return 0
	}
	return len(sm.Spec.Components.Schemas)
}

func (sm *SpecMore) OperationsTable(columns *text.TextSet) (*table.Table, error) {
	return operationsTable(sm.Spec, columns)
}

func operationsTable(spec *oas3.Swagger, columns *text.TextSet) (*table.Table, error) {
	if columns == nil {
		columns = &text.TextSet{Texts: OpTableColumnsDefault()}
	}
	tbl := table.NewTable()
	tbl.Name = spec.Info.Title
	tbl.Columns = columns.DisplayTexts()

	specMore := SpecMore{Spec: spec}

	//tgs, err := SpecTagGroups(spec)
	tgs, err := specMore.TagGroups()
	if err != nil {
		return nil, err
	}

	VisitOperations(spec, func(path, method string, op *oas3.Operation) {
		row := []string{}

		for _, text := range columns.Texts {
			switch text.Slug {
			case "method":
				row = append(row, method)
			case "path":
				row = append(row, path)
			case "operationId":
				row = append(row, op.OperationID)
			case "summary":
				row = append(row, op.Summary)
			case "tags":
				row = append(row, strings.Join(op.Tags, ", "))
			case "x-tag-groups":
				row = append(row, strings.Join(
					tgs.GetTagGroupNamesForTagNames(op.Tags...), ", "))
			default:
				row = append(row, GetExtensionPropStringOrEmpty(op.ExtensionProps, text.Slug))
			}
		}

		tbl.Records = append(tbl.Records, row)
	})
	return &tbl, nil
}

func OpTableColumnsDefault() []text.Text {
	texts := []text.Text{
		{
			Display: "Method",
			Slug:    "method"},
		{
			Display: "Path",
			Slug:    "path"},
		{
			Display: "OperationID",
			Slug:    "operationId"},
		{
			Display: "Summary",
			Slug:    "summary"},
		{
			Display: "Tags",
			Slug:    "tags"},
	}
	return texts
}

func OpTableColumnsRingCentral() *text.TextSet {
	texts := []text.Text{
		{
			Display: "Method",
			Slug:    "method"},
		{
			Display: "Path",
			Slug:    "path"},
		{
			Display: "OperationID",
			Slug:    "operationId"},
		{
			Display: "Summary",
			Slug:    "summary"},
		{
			Display: "Tags",
			Slug:    "tags"},
		{
			Display: "API Group",
			Slug:    "x-api-group"},
		{
			Display: "Throttling",
			Slug:    "x-throttling-group"},
		{
			Display: "App Permission",
			Slug:    "x-app-permission"},
		{
			Display: "User Permissions",
			Slug:    "x-user-permission"},
	}
	return &text.TextSet{Texts: texts}
}

func (sm *SpecMore) OperationMetas() []OperationMeta {
	ometas := []OperationMeta{}
	if sm.Spec == nil {
		return ometas
	}
	for url, path := range sm.Spec.Paths {
		if path.Connect != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodConnect, path.Connect))
		}
		if path.Delete != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodDelete, path.Delete))
		}
		if path.Get != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodGet, path.Get))
		}
		if path.Head != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodHead, path.Head))
		}
		if path.Options != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodOptions, path.Options))
		}
		if path.Patch != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodPatch, path.Patch))
		}
		if path.Post != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodPost, path.Post))
		}
		if path.Put != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodPut, path.Put))
		}
		if path.Trace != nil {
			ometas = append(ometas, OperationToMeta(url, http.MethodTrace, path.Trace))
		}
	}

	return ometas
}

func (sm *SpecMore) OperationsCount() int {
	if sm.Spec == nil {
		return -1
	}
	return len(sm.OperationMetas())
}

func (sm *SpecMore) SchemaNames() []string {
	schemaNames := []string{}
	for schemaName := range sm.Spec.Components.Schemas {
		schemaNames = append(schemaNames, schemaName)
	}
	return stringsutil.SliceCondenseSpace(schemaNames, true, true)
}

func (sm *SpecMore) SchemaNameExists(schemaName string, includeNil bool) bool {
	for schemaNameTry, schemaRef := range sm.Spec.Components.Schemas {
		if schemaNameTry == schemaName {
			if includeNil {
				return true
			} else if schemaRef == nil {
				return false
			}
			schemaRef.Ref = strings.TrimSpace(schemaRef.Ref)
			if len(schemaRef.Ref) > 0 {
				return true
			}
			if schemaRef.Value == nil {
				return false
			} else {
				return true
			}
		}
	}
	return false
}

// ServerURL returns the OAS3 Spec URL for the index
// specified.
func (sm *SpecMore) ServerURL(index uint) string {
	if int(index)+1 > len(sm.Spec.Servers) {
		return ""
	}
	server := sm.Spec.Servers[index]
	return strings.TrimSpace(server.URL)
}

// ServerURLBasePath extracts the base path from a OAS URL
// which can include variables.
func (sm *SpecMore) ServerURLBasePath(index uint) (string, error) {
	serverURL := sm.ServerURL(index)
	if len(serverURL) == 0 {
		return "", nil
	}
	serverURLParsed, err := urlutil.ParseURLTemplate(serverURL)
	if err != nil {
		return "", err
	}
	return serverURLParsed.Path, nil
}

func (sm *SpecMore) Tags(inclTop, inclOps bool) []string {
	tags := []string{}
	tagsMap := sm.TagsMap(inclTop, inclOps)
	for tag := range tagsMap {
		tags = append(tags, tag)
	}
	return stringsutil.SliceCondenseSpace(tags, true, true)
}

// TagsMap returns a set of tags present in the current
// spec.
func (sm *SpecMore) TagsMap(inclTop, inclOps bool) map[string]int {
	tagsMap := map[string]int{}
	if inclTop {
		for _, tag := range sm.Spec.Tags {
			tagName := strings.TrimSpace(tag.Name)
			if len(tagName) > 0 {
				if _, ok := tagsMap[tagName]; !ok {
					tagsMap[tagName] = 0
				}
				tagsMap[tagName]++
			}
		}
	}
	if inclOps {
		VisitOperations(sm.Spec, func(skipPath, skipMethod string, op *oas3.Operation) {
			for _, tagName := range op.Tags {
				tagName = strings.TrimSpace(tagName)
				if len(tagName) > 0 {
					if _, ok := tagsMap[tagName]; !ok {
						tagsMap[tagName] = 0
					}
					tagsMap[tagName]++
				}
			}
		})
	}
	return tagsMap
}

type SpecStats struct {
	OperationsCount int
	SchemasCount    int
}

func (sm *SpecMore) Stats() SpecStats {
	return SpecStats{
		OperationsCount: sm.OperationsCount(),
		SchemasCount:    sm.SchemasCount(),
	}
}

func (sm *SpecMore) WriteFileJSON(filename string, perm os.FileMode, prefix, indent string) error {
	jsonData, err := sm.Spec.MarshalJSON()
	if err != nil {
		return err
	}
	pretty := false
	if len(prefix) > 0 || len(indent) > 0 {
		pretty = true
	}
	if pretty {
		jsonData = jsonutil.PrettyPrint(jsonData, "", "  ")
	}
	return ioutil.WriteFile(filename, jsonData, perm)
}

func (sm *SpecMore) WriteFileXLSX(filename string) error {
	tbl, err := sm.OperationsTable(nil)
	if err != nil {
		return err
	}
	return table.WriteXLSX(filename, tbl)
}

type TagsMore struct {
	Tags oas3.Tags
}

func (tg *TagsMore) Get(tagName string) *oas3.Tag {
	for _, tag := range tg.Tags {
		if tagName == tag.Name {
			return tag
		}
	}
	return nil
}
