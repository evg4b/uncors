package config

import (
	"encoding"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type knownFieldsChecker interface {
	checkKnownFields(field string, node *yaml.Node) error
}

var (
	knownFieldsCheckerType = reflect.TypeFor[knownFieldsChecker]()
	textUnmarshalerType    = reflect.TypeFor[encoding.TextUnmarshaler]()
	yamlUnmarshalerType    = reflect.TypeFor[yaml.Unmarshaler]()
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
		checker, _ := reflect.New(typ).Interface().(knownFieldsChecker)

		return checker.checkKnownFields(field, node)
	}

	if ptr.Implements(textUnmarshalerType) || ptr.Implements(yamlUnmarshalerType) {
		return nil
	}

	kind := typ.Kind()
	if kind == reflect.Struct {
		return checkStructFields(field, node, typ)
	}

	if kind == reflect.Slice || kind == reflect.Array {
		return checkSequenceItems(field, node, typ.Elem())
	}

	if kind == reflect.Map {
		return checkMapValues(field, node, typ.Elem())
	}

	return nil
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
