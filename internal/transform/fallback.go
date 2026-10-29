package transform

import (
	"fmt"
	"sort"
	"strings"
)

// transformUnknownFields rewrites paths only in fields absent from the selected ACP schema.
// JSON-RPC routing fields always remain opaque.
func transformUnknownFields(message map[string]any, pending map[string]string, catalog *schemaCatalog, direction Direction) (bool, []Decision) {
	method, _ := message["method"].(string)
	if method != "" {
		return transformPayloadField(message, "params", catalog.params[method], catalog, direction)
	}
	if method = pending[requestID(message["id"])]; method != "" {
		return transformPayloadField(message, "result", catalog.results[method], catalog, direction)
	}
	return false, nil
}

func transformPayloadField(message map[string]any, field string, schemas []map[string]any, catalog *schemaCatalog, direction Direction) (bool, []Decision) {
	value, ok := message[field]
	if !ok {
		return false, nil
	}
	converted, decisions := transformUnknownValue(value, schemas, catalog, direction, field)
	if len(decisions) == 0 {
		return false, nil
	}
	message[field] = converted
	return true, decisions
}

func transformUnknownValue(value any, schemas []map[string]any, catalog *schemaCatalog, direction Direction, field string) (any, []Decision) {
	schemas = catalog.flatten(schemas)
	switch value := value.(type) {
	case map[string]any:
		if len(schemas) == 0 {
			return transformEveryValue(value, catalog, direction, field)
		}
		var decisions []Decision
		for _, key := range sortedKeys(value) {
			propertySchemas := catalog.propertySchemas(schemas, key)
			if len(propertySchemas) == 0 && catalog.allowsAdditionalProperties(schemas) {
				continue
			}
			converted, itemDecisions := transformUnknownValue(value[key], propertySchemas, catalog, direction, field+"."+key)
			value[key] = converted
			decisions = append(decisions, itemDecisions...)
		}
		return value, decisions
	case []any:
		if len(schemas) == 0 {
			return transformEveryValue(value, catalog, direction, field)
		}
		itemSchemas, hasItems := catalog.itemSchemas(schemas)
		if !hasItems {
			return value, nil
		}
		var decisions []Decision
		for index, item := range value {
			converted, itemDecisions := transformUnknownValue(item, itemSchemas, catalog, direction, fmt.Sprintf("%s[%d]", field, index))
			value[index] = converted
			decisions = append(decisions, itemDecisions...)
		}
		return value, decisions
	case string:
		if len(schemas) == 0 {
			return transformWholePath(value, direction, field)
		}
	}
	return value, nil
}

func transformEveryValue(value any, catalog *schemaCatalog, direction Direction, field string) (any, []Decision) {
	switch value := value.(type) {
	case map[string]any:
		var decisions []Decision
		for _, key := range sortedKeys(value) {
			converted, itemDecisions := transformEveryValue(value[key], catalog, direction, field+"."+key)
			value[key] = converted
			decisions = append(decisions, itemDecisions...)
		}
		return value, decisions
	case []any:
		var decisions []Decision
		for index, item := range value {
			converted, itemDecisions := transformEveryValue(item, catalog, direction, fmt.Sprintf("%s[%d]", field, index))
			value[index] = converted
			decisions = append(decisions, itemDecisions...)
		}
		return value, decisions
	case string:
		return transformWholePath(value, direction, field)
	}
	return value, nil
}

func transformWholePath(value string, direction Direction, field string) (string, []Decision) {
	if strings.ContainsAny(value, "\x00\r\n") ||
		(direction == ToLinux && !driveAbsolute.MatchString(value)) ||
		(direction == ToWindows && !mountPath.MatchString(value)) {
		return value, nil
	}
	converted, decision := convertPath(value, direction)
	if converted == value {
		return value, nil
	}
	decision.Field = field
	decision.Kind = "unknown-field-" + decision.Kind
	return converted, []Decision{decision}
}

func (catalog *schemaCatalog) flatten(schemas []map[string]any) []map[string]any {
	var flattened []map[string]any
	for _, schema := range schemas {
		flattened = append(flattened, catalog.resolve(schema)...)
	}
	return flattened
}

func (catalog *schemaCatalog) propertySchemas(schemas []map[string]any, name string) []map[string]any {
	var properties []map[string]any
	for _, schema := range schemas {
		object, _ := schema["properties"].(map[string]any)
		property, ok := object[name].(map[string]any)
		if ok {
			properties = append(properties, property)
		}
	}
	return properties
}

func (catalog *schemaCatalog) allowsAdditionalProperties(schemas []map[string]any) bool {
	for _, schema := range schemas {
		value, exists := schema["additionalProperties"]
		if !exists || value == true {
			return true
		}
	}
	return false
}

func (catalog *schemaCatalog) itemSchemas(schemas []map[string]any) ([]map[string]any, bool) {
	var items []map[string]any
	found := false
	for _, schema := range schemas {
		item, ok := schema["items"].(map[string]any)
		if ok {
			items = append(items, item)
			found = true
		}
	}
	return items, found
}

func sortedKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
