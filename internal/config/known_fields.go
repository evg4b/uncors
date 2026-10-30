package config

import (
	"encoding"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type knownFieldsChecker interface {
	checkKnownFields(field string, node *yaml.Node) error
}

var (
	knownFieldsCheckerType = reflect.TypeFor[knownFieldsChecker]()
	textUnmarshalerType    = reflect.TypeFor[encoding.TextUnmarshaler]()
	yamlUnmarshalerType    = reflect.TypeFor[yaml.Unmarshaler]()
	durationType           = reflect.TypeFor[time.Duration]()
	wholeNumberKinds       = []reflect.Kind{
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
	}
)

func checkKnownFields(field string, node *yaml.Node, typ reflect.Type) error {
	node = resolveNode(node)
	if node == nil {
		return nil
	}

	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	ptr := reflect.PointerTo(typ)
	if ptr.Implements(knownFieldsCheckerType) {
		checker, _ := reflect.TypeAssert[knownFieldsChecker](reflect.New(typ))

		return checker.checkKnownFields(field, node)
	}

	if ptr.Implements(yamlUnmarshalerType) {
		return nil
	}

	if ptr.Implements(textUnmarshalerType) {
		return checkValue(field, node, typ)
	}

	kind := typ.Kind()
	if kind == reflect.Struct {
		return expectKind(field, node, yaml.MappingNode, checkStructFields(field, node, typ))
	}

	if kind == reflect.Slice || kind == reflect.Array {
		return expectKind(field, node, yaml.SequenceNode, checkSequenceItems(field, node, typ.Elem()))
	}

	if kind == reflect.Map {
		return expectKind(field, node, yaml.MappingNode, checkMapValues(field, node, typ.Elem()))
	}

	return checkValue(field, node, typ)
}

func expectKind(field string, node *yaml.Node, kind yaml.Kind, nested error) error {
	if node.Kind == kind || node.Tag == "!!null" {
		return nested
	}

	return &ValidationError{fmt.Sprintf("%s must be %s, got %s (line %d)",
		field, describeKind(kind), describeNode(node), node.Line)}
}

func checkValue(field string, node *yaml.Node, typ reflect.Type) error {
	if node.Tag == "!!null" {
		return nil
	}

	err := node.Decode(reflect.New(typ).Interface())
	if err == nil {
		return nil
	}

	if node.Kind == yaml.ScalarNode && reflect.PointerTo(typ).Implements(textUnmarshalerType) {
		return &ValidationError{fmt.Sprintf("%s: %v (line %d)", field, err, node.Line)}
	}

	return &ValidationError{fmt.Sprintf("%s must be %s, got %s (line %d)",
		field, describeType(typ), describeNode(node), node.Line)}
}

func describeType(typ reflect.Type) string {
	if typ == durationType {
		return "a duration without spaces, such as 500ms or 1m30s"
	}

	if typ.Kind() == reflect.Bool {
		return "true or false"
	}

	if slices.Contains(wholeNumberKinds, typ.Kind()) {
		return "a whole number"
	}

	return "a single value"
}

func describeKind(kind yaml.Kind) string {
	if kind == yaml.SequenceNode {
		return "a list"
	}

	return "a map of keys to values"
}

func describeNode(node *yaml.Node) string {
	if node.Kind == yaml.SequenceNode {
		return "a list"
	}

	if node.Kind == yaml.MappingNode {
		return "a map"
	}

	return fmt.Sprintf("%q", node.Value)
}

func resolveNode(node *yaml.Node) *yaml.Node {
	for node != nil {
		if node.Kind == yaml.AliasNode {
			node = node.Alias

			continue
		}

		if node.Kind != yaml.DocumentNode {
			return node
		}

		if len(node.Content) == 0 {
			return nil
		}

		node = node.Content[0]
	}

	return nil
}

func checkStructFields(field string, node *yaml.Node, typ reflect.Type) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	fields := yamlFields(typ)

	var errs []error

	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]

		if key.Tag == "!!merge" {
			errs = append(errs, checkMergedFields(field, value, typ))

			continue
		}

		fieldType, ok := fields[key.Value]
		if !ok {
			errs = append(errs, unknownFieldError(joinPath(field, key.Value), key.Line))

			continue
		}

		errs = append(errs, checkKnownFields(joinPath(field, key.Value), value, fieldType))
	}

	return errors.Join(errs...)
}

func checkMergedFields(field string, node *yaml.Node, typ reflect.Type) error {
	node = resolveNode(node)
	if node == nil || node.Kind != yaml.SequenceNode {
		return checkKnownFields(field, node, typ)
	}

	var errs []error
	for _, item := range node.Content {
		errs = append(errs, checkKnownFields(field, item, typ))
	}

	return errors.Join(errs...)
}

func checkSequenceItems(field string, node *yaml.Node, elem reflect.Type) error {
	if node.Kind != yaml.SequenceNode {
		return nil
	}

	var errs []error
	for i, item := range node.Content {
		errs = append(errs, checkKnownFields(joinPath(field, index(i)), item, elem))
	}

	return errors.Join(errs...)
}

func checkMapValues(field string, node *yaml.Node, elem reflect.Type) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}

	var errs []error
	for i := 0; i+1 < len(node.Content); i += 2 {
		errs = append(errs, checkKnownFields(joinPath(field, node.Content[i].Value), node.Content[i+1], elem))
	}

	return errors.Join(errs...)
}

func yamlFields(typ reflect.Type) map[string]reflect.Type {
	fields := make(map[string]reflect.Type, typ.NumField())

	for structField := range typ.Fields() {
		if !structField.IsExported() {
			continue
		}

		name, options, _ := strings.Cut(structField.Tag.Get("yaml"), ",")
		if name == "-" {
			continue
		}

		if strings.Contains(options, "inline") {
			maps.Copy(fields, yamlFields(structField.Type))

			continue
		}

		if name == "" {
			name = strings.ToLower(structField.Name)
		}

		fields[name] = structField.Type
	}

	return fields
}

func unknownFieldError(field string, line int) error {
	return &ValidationError{fmt.Sprintf("%s: unknown field (line %d)", field, line)}
}
