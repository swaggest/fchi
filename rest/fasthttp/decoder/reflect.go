package decoder

import "reflect"

// HasFieldTags checks if the structure has fields with tag name.
func hasFieldTags(t reflect.Type, tagname string) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if tag := field.Tag.Get(tagname); tag != "" && tag != "-" {
			return true
		}

		if field.Anonymous {
			if hasFieldTags(field.Type, tagname) {
				return true
			}
		}
	}

	return false
}

func isSliceOrMap(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() == reflect.Slice || t.Kind() == reflect.Map {
		return true
	}

	return false
}

// HasFileFields checks if the structure has fields to receive uploaded files.
func hasFileFields(t reflect.Type, tagname string) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag := field.Tag.Get(tagname); tag != "" && tag != "-" {
			typeName := field.Type.String()
			if typeName == "multipart.File" {
				return true
			}

			if typeName == "*multipart.FileHeader" {
				return true
			}
		}

		if field.Anonymous {
			if hasFileFields(field.Type, tagname) {
				return true
			}
		}
	}

	return false
}
