package jmespath

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	trunc "github.com/aquilax/truncate"
	"github.com/blang/semver/v4"
	gojmespath "github.com/jmespath/go-jmespath"
	"github.com/minio/pkg/wildcard"
)

var (
	JpObject      = gojmespath.JpObject
	JpString      = gojmespath.JpString
	JpNumber      = gojmespath.JpNumber
	JpArray       = gojmespath.JpArray
	JpArrayString = gojmespath.JpArrayString
	JpAny         = gojmespath.JpAny
)

type (
	JpType  = gojmespath.JpType
	ArgSpec = gojmespath.ArgSpec
)

// function names
var (
	compare                = "compare"
	equalFold              = "equal_fold"
	replace                = "replace"
	replaceAll             = "replace_all"
	toUpper                = "to_upper"
	toLower                = "to_lower"
	trim                   = "trim"
	split                  = "split"
	regexReplaceAll        = "regex_replace_all"
	regexReplaceAllLiteral = "regex_replace_all_literal"
	regexMatch             = "regex_match"
	patternMatch           = "pattern_match"
	labelMatch             = "label_match"
	add                    = "add"
	subtract               = "subtract"
	multiply               = "multiply"
	divide                 = "divide"
	modulo                 = "modulo"
	base64Decode           = "base64_decode"
	base64Encode           = "base64_encode"
	timeSince              = "time_since"
	pathCanonicalize       = "path_canonicalize"
	truncate               = "truncate"
	semverCompare          = "semver_compare"
	parseJson              = "parse_json"
)

const errorPrefix = "JMESPath function '%s': "
const invalidArgumentTypeError = errorPrefix + "%d argument is expected of %s type"
const genericError = errorPrefix + "%s"
const zeroDivisionError = errorPrefix + "Zero divisor passed"
const undefinedQuoError = errorPrefix + "Undefined quotient"
const nonIntModuloError = errorPrefix + "Non-integer argument(s) passed for modulo"

func getFunctions() []*gojmespath.FunctionEntry {
	return []*gojmespath.FunctionEntry{
		{
			Name: compare,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
			},
			Handler: jpfCompare,
		},
		{
			Name: equalFold,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
			},
			Handler: jpfEqualFold,
		},
		{
			Name: replace,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
				{Types: []JpType{JpNumber}},
			},
			Handler: jpfReplace,
		},
		{
			Name: replaceAll,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
			},
			Handler: jpfReplaceAll,
		},
		{
			Name: toUpper,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
			},
			Handler: jpfToUpper,
		},
		{
			Name: toLower,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
			},
			Handler: jpfToLower,
		},
		{
			Name: trim,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
			},
			Handler: jpfTrim,
		},
		{
			Name: split,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
			},
			Handler: jpfSplit,
		},
		{
			Name: regexReplaceAll,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString, JpNumber}},
				{Types: []JpType{JpString, JpNumber}},
			},
			Handler: jpRegexReplaceAll,
		},
		{
			Name: regexReplaceAllLiteral,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString, JpNumber}},
				{Types: []JpType{JpString, JpNumber}},
			},
			Handler: jpRegexReplaceAllLiteral,
		},
		{
			Name: regexMatch,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString, JpNumber}},
			},
			Handler: jpRegexMatch,
		},
		{
			Name: patternMatch,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString, JpNumber}},
			},
			Handler: jpPatternMatch,
		},
		{
			// Validates if label (param1) would match pod/host/etc labels (param2)
			Name: labelMatch,
			Arguments: []ArgSpec{
				{Types: []JpType{JpObject}},
				{Types: []JpType{JpObject}},
			},
			Handler: jpLabelMatch,
		},
		{
			Name: add,
			Arguments: []ArgSpec{
				{Types: []JpType{JpAny}},
				{Types: []JpType{JpAny}},
			},
			Handler: jpAdd,
		},
		{
			Name: subtract,
			Arguments: []ArgSpec{
				{Types: []JpType{JpAny}},
				{Types: []JpType{JpAny}},
			},
			Handler: jpSubtract,
		},
		{
			Name: multiply,
			Arguments: []ArgSpec{
				{Types: []JpType{JpAny}},
				{Types: []JpType{JpAny}},
			},
			Handler: jpMultiply,
		},
		{
			Name: divide,
			Arguments: []ArgSpec{
				{Types: []JpType{JpAny}},
				{Types: []JpType{JpAny}},
			},
			Handler: jpDivide,
		},
		{
			Name: modulo,
			Arguments: []ArgSpec{
				{Types: []JpType{JpAny}},
				{Types: []JpType{JpAny}},
			},
			Handler: jpModulo,
		},
		{
			Name: base64Decode,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
			},
			Handler: jpBase64Decode,
		},
		{
			Name: base64Encode,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
			},
			Handler: jpBase64Encode,
		},
		{
			Name: timeSince,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
			},
			Handler: jpTimeSince,
		},
		{
			Name: pathCanonicalize,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
			},
			Handler: jpPathCanonicalize,
		},
		{
			Name: truncate,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpNumber}},
			},
			Handler: jpTruncate,
		},
		{
			Name: semverCompare,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
				{Types: []JpType{JpString}},
			},
			Handler: jpSemverCompare,
		},
		{
			Name: parseJson,
			Arguments: []ArgSpec{
				{Types: []JpType{JpString}},
			},
			Handler: jpParseJson,
		},
	}

}

func jpfCompare(arguments []interface{}) (interface{}, error) {
	var err error
	a, err := validateArg(compare, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	b, err := validateArg(compare, arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	return strings.Compare(a.String(), b.String()), nil
}

func jpfEqualFold(arguments []interface{}) (interface{}, error) {
	var err error
	a, err := validateArg(equalFold, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	b, err := validateArg(equalFold, arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	return strings.EqualFold(a.String(), b.String()), nil
}

func jpfReplace(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg(replace, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	old, err := validateArg(replace, arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	new, err := validateArg(replace, arguments, 2, reflect.String)
	if err != nil {
		return nil, err
	}

	n, err := validateArg(replace, arguments, 3, reflect.Float64)
	if err != nil {
		return nil, err
	}

	return strings.Replace(str.String(), old.String(), new.String(), int(n.Float())), nil
}

func jpfReplaceAll(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg(replaceAll, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	old, err := validateArg(replaceAll, arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	new, err := validateArg(replaceAll, arguments, 2, reflect.String)
	if err != nil {
		return nil, err
	}

	return strings.ReplaceAll(str.String(), old.String(), new.String()), nil
}

func jpfToUpper(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg(toUpper, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	return strings.ToUpper(str.String()), nil
}

func jpfToLower(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg(toLower, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	return strings.ToLower(str.String()), nil
}

func jpfTrim(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg(trim, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	cutset, err := validateArg(trim, arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	return strings.Trim(str.String(), cutset.String()), nil
}

func jpfSplit(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg(split, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	sep, err := validateArg(split, arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	split := strings.Split(str.String(), sep.String())
	arr := make([]interface{}, len(split))

	for i, v := range split {
		arr[i] = v
	}

	return arr, nil
}

func jpRegexReplaceAll(arguments []interface{}) (interface{}, error) {
	var err error
	regex, err := validateArg(regexReplaceAll, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	src, err := ifaceToString(arguments[1])
	if err != nil {
		return nil, fmt.Errorf(invalidArgumentTypeError, regexReplaceAll, 2, "String or Real")
	}

	repl, err := ifaceToString(arguments[2])
	if err != nil {
		return nil, fmt.Errorf(invalidArgumentTypeError, regexReplaceAll, 3, "String or Real")
	}

	reg, err := regexp.Compile(regex.String())
	if err != nil {
		return nil, fmt.Errorf(genericError, regexReplaceAll, err.Error())
	}
	return string(reg.ReplaceAll([]byte(src), []byte(repl))), nil
}

func jpRegexReplaceAllLiteral(arguments []interface{}) (interface{}, error) {
	var err error
	regex, err := validateArg(regexReplaceAllLiteral, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	src, err := ifaceToString(arguments[1])
	if err != nil {
		return nil, fmt.Errorf(invalidArgumentTypeError, regexReplaceAllLiteral, 2, "String or Real")
	}

	repl, err := ifaceToString(arguments[2])
	if err != nil {
		return nil, fmt.Errorf(invalidArgumentTypeError, regexReplaceAllLiteral, 3, "String or Real")
	}

	reg, err := regexp.Compile(regex.String())
	if err != nil {
		return nil, fmt.Errorf(genericError, regexReplaceAllLiteral, err.Error())
	}
	return string(reg.ReplaceAllLiteral([]byte(src), []byte(repl))), nil
}

func jpRegexMatch(arguments []interface{}) (interface{}, error) {
	var err error
	regex, err := validateArg(regexMatch, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	src, err := ifaceToString(arguments[1])
	if err != nil {
		return nil, fmt.Errorf(invalidArgumentTypeError, regexMatch, 2, "String or Real")
	}

	return regexp.Match(regex.String(), []byte(src))
}

func jpPatternMatch(arguments []interface{}) (interface{}, error) {
	pattern, err := validateArg(regexMatch, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	src, err := ifaceToString(arguments[1])
	if err != nil {
		return nil, fmt.Errorf(invalidArgumentTypeError, regexMatch, 2, "String or Real")
	}

	return wildcard.Match(pattern.String(), src), nil
}

func jpLabelMatch(arguments []interface{}) (interface{}, error) {
	labelMap, ok := arguments[0].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf(invalidArgumentTypeError, labelMatch, 0, "Object")
	}

	matchMap, ok := arguments[1].(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf(invalidArgumentTypeError, labelMatch, 1, "Object")
	}

	for key, value := range labelMap {
		if val, ok := matchMap[key]; !ok || val != value {
			return false, nil
		}
	}

	return true, nil
}

func jpAdd(arguments []interface{}) (interface{}, error) {
	op1, op2, err := ParseArithemticOperands(arguments, add)
	if err != nil {
		return nil, err
	}

	return op1.Add(op2)
}

func jpSubtract(arguments []interface{}) (interface{}, error) {
	op1, op2, err := ParseArithemticOperands(arguments, subtract)
	if err != nil {
		return nil, err
	}

	return op1.Subtract(op2)
}

func jpMultiply(arguments []interface{}) (interface{}, error) {
	op1, op2, err := ParseArithemticOperands(arguments, multiply)
	if err != nil {
		return nil, err
	}

	return op1.Multiply(op2)
}

func jpDivide(arguments []interface{}) (interface{}, error) {
	op1, op2, err := ParseArithemticOperands(arguments, divide)
	if err != nil {
		return nil, err
	}

	return op1.Divide(op2)
}

func jpModulo(arguments []interface{}) (interface{}, error) {
	op1, op2, err := ParseArithemticOperands(arguments, modulo)
	if err != nil {
		return nil, err
	}

	return op1.Modulo(op2)
}

func jpBase64Decode(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg("", arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	decodedStr, err := base64.StdEncoding.DecodeString(str.String())
	if err != nil {
		return nil, err
	}

	return string(decodedStr), nil
}

func jpBase64Encode(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg("", arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.EncodeToString([]byte(str.String())), nil
}

func jpTimeSince(arguments []interface{}) (interface{}, error) {
	var err error
	layout, err := validateArg("", arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	ts1, err := validateArg("", arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	ts2, err := validateArg("", arguments, 2, reflect.String)
	if err != nil {
		return nil, err
	}

	var t1, t2 time.Time
	if layout.String() != "" {
		t1, err = time.Parse(layout.String(), ts1.String())
	} else {
		t1, err = time.Parse(time.RFC3339, ts1.String())
	}
	if err != nil {
		return nil, err
	}

	t2 = time.Now()
	if ts2.String() != "" {
		if layout.String() != "" {
			t2, err = time.Parse(layout.String(), ts2.String())
		} else {
			t2, err = time.Parse(time.RFC3339, ts2.String())
		}

		if err != nil {
			return nil, err
		}
	}

	return t2.Sub(t1).String(), nil
}

func jpPathCanonicalize(arguments []interface{}) (interface{}, error) {
	var err error
	str, err := validateArg(pathCanonicalize, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	return filepath.Join(str.String()), nil
}

func jpTruncate(arguments []interface{}) (interface{}, error) {
	var err error
	var normalizedLength float64
	str, err := validateArg(truncate, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}
	length, err := validateArg(truncate, arguments, 1, reflect.Float64)
	if err != nil {
		return nil, err
	}

	if length.Float() < 0 {
		normalizedLength = float64(0)
	} else {
		normalizedLength = length.Float()
	}

	return trunc.Truncator(str.String(), int(normalizedLength), trunc.CutStrategy{}), nil
}

func jpSemverCompare(arguments []interface{}) (interface{}, error) {
	var err error
	v, err := validateArg(semverCompare, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}

	r, err := validateArg(semverCompare, arguments, 1, reflect.String)
	if err != nil {
		return nil, err
	}

	version, _ := semver.Parse(v.String())
	expectedRange, err := semver.ParseRange(r.String())
	if err != nil {
		return nil, err
	}

	if expectedRange(version) {
		return true, nil
	}
	return false, nil
}

func jpParseJson(arguments []interface{}) (interface{}, error) {
	input, err := validateArg(parseJson, arguments, 0, reflect.String)
	if err != nil {
		return nil, err
	}
	var output interface{}
	err = json.Unmarshal([]byte(input.String()), &output)
	return output, err
}

// InterfaceToString casts an interface to a string type
func ifaceToString(iface interface{}) (string, error) {
	switch i := iface.(type) {
	case int:
		return strconv.Itoa(i), nil
	case float64:
		return strconv.FormatFloat(i, 'f', -1, 32), nil
	case float32:
		return strconv.FormatFloat(float64(i), 'f', -1, 32), nil
	case string:
		return i, nil
	case bool:
		return strconv.FormatBool(i), nil
	default:
		return "", errors.New("error, undefined type cast")
	}
}

func validateArg(f string, arguments []interface{}, index int, expectedType reflect.Kind) (reflect.Value, error) {
	arg := reflect.ValueOf(arguments[index])
	if arg.Type().Kind() != expectedType {
		return reflect.Value{}, fmt.Errorf(invalidArgumentTypeError, f, index+1, expectedType.String())
	}

	return arg, nil
}
