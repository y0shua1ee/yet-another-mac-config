package artifact

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

type ErrorCode string

const (
	CodeJSONInvalid          ErrorCode = "ARTIFACT_JSON_INVALID"
	CodeJSONInvalidUTF8      ErrorCode = "ARTIFACT_JSON_INVALID_UTF8"
	CodeJSONDuplicateKey     ErrorCode = "ARTIFACT_JSON_DUPLICATE_KEY"
	CodeJSONInvalidNumber    ErrorCode = "ARTIFACT_JSON_INVALID_NUMBER"
	CodeJSONTrailingValue    ErrorCode = "ARTIFACT_JSON_TRAILING_VALUE"
	CodeEnvelopeRejected     ErrorCode = "ARTIFACT_ENVELOPE_REJECTED"
	CodeKindRejected         ErrorCode = "ARTIFACT_KIND_REJECTED"
	CodeExpectedKindMismatch ErrorCode = "ARTIFACT_EXPECTED_KIND_MISMATCH"
	CodeSchemaRejected       ErrorCode = "ARTIFACT_SCHEMA_REJECTED"
	CodeProvenanceRejected   ErrorCode = "ARTIFACT_PROVENANCE_REJECTED"
	CodePayloadRejected      ErrorCode = "ARTIFACT_PAYLOAD_REJECTED"
	CodeLifecycleRejected    ErrorCode = "ARTIFACT_LIFECYCLE_REJECTED"
	CodeStorageRejected      ErrorCode = "ARTIFACT_STORAGE_REJECTED"
	CodeStorageReadOnly      ErrorCode = "ARTIFACT_STORAGE_READ_ONLY"
	CodeDigestRejected       ErrorCode = "ARTIFACT_DIGEST_REJECTED"
	CodeCanonicalRejected    ErrorCode = "ARTIFACT_CANONICAL_REJECTED"
)

type ContractError struct {
	Code    ErrorCode
	Pointer string
}

func (err *ContractError) Error() string {
	return fmt.Sprintf("%s:%s", err.Code, err.Pointer)
}

func contractError(code ErrorCode, pointer string) error {
	if pointer == "" {
		pointer = "/"
	}
	return &ContractError{Code: code, Pointer: pointer}
}

func ErrorDetails(err error) (ErrorCode, string, bool) {
	var contractErr *ContractError
	if !errors.As(err, &contractErr) {
		return "", "", false
	}
	return contractErr.Code, contractErr.Pointer, true
}

var restrictedInteger = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

func Canonicalize(data []byte) ([]byte, error) {
	if !utf8.Valid(data) {
		return nil, contractError(CodeJSONInvalidUTF8, "/")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := readRestrictedValue(decoder, "")
	if err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, contractError(CodeJSONTrailingValue, "/")
		}
		return nil, contractError(CodeJSONInvalid, "/")
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, contractError(CodeCanonicalRejected, "/")
	}
	return canonical, nil
}

func readRestrictedValue(decoder *json.Decoder, pointer string) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, contractError(CodeJSONInvalid, pointer)
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			result := make(map[string]any)
			seen := make(map[string]struct{})
			for decoder.More() {
				keyToken, keyErr := decoder.Token()
				if keyErr != nil {
					return nil, contractError(CodeJSONInvalid, pointer)
				}
				key, ok := keyToken.(string)
				if !ok {
					return nil, contractError(CodeJSONInvalid, pointer)
				}
				childPointer := joinPointer(pointer, key)
				if _, exists := seen[key]; exists {
					return nil, contractError(CodeJSONDuplicateKey, childPointer)
				}
				seen[key] = struct{}{}
				child, childErr := readRestrictedValue(decoder, childPointer)
				if childErr != nil {
					return nil, childErr
				}
				result[key] = child
			}
			closing, closingErr := decoder.Token()
			if closingErr != nil || closing != json.Delim('}') {
				return nil, contractError(CodeJSONInvalid, pointer)
			}
			return result, nil
		case '[':
			result := make([]any, 0)
			for index := 0; decoder.More(); index++ {
				child, childErr := readRestrictedValue(decoder, joinPointer(pointer, fmt.Sprintf("%d", index)))
				if childErr != nil {
					return nil, childErr
				}
				result = append(result, child)
			}
			closing, closingErr := decoder.Token()
			if closingErr != nil || closing != json.Delim(']') {
				return nil, contractError(CodeJSONInvalid, pointer)
			}
			return result, nil
		default:
			return nil, contractError(CodeJSONInvalid, pointer)
		}
	case json.Number:
		if !restrictedInteger.MatchString(value.String()) {
			return nil, contractError(CodeJSONInvalidNumber, pointer)
		}
		return value, nil
	case string, bool, nil:
		return value, nil
	default:
		return nil, contractError(CodeJSONInvalid, pointer)
	}
}

func joinPointer(parent, child string) string {
	escaped := strings.ReplaceAll(strings.ReplaceAll(child, "~", "~0"), "/", "~1")
	if parent == "" {
		return "/" + escaped
	}
	return parent + "/" + escaped
}

func decodeClosed(data []byte, target any, pointer string) error {
	if _, err := Canonicalize(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return contractError(CodePayloadRejected, pointer)
	}
	if err := requireEOF(decoder); err != nil {
		return err
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return contractError(CodeJSONTrailingValue, "/")
	}
	return nil
}
