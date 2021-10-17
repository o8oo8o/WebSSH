package validator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	ut "gossh/lib/gin/validator/translator"
	"net"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// Package validator implements value validations for structs and individual fields // based on tags.
//
// It can also handle Cross-Field and Cross-Struct validation for nested structs
// and has the ability to dive into arrays and maps of any type.
//
// see more examples https://github.com/go-playground/validator/tree/master/_examples
//
// Singleton
//
// Validator is designed to be thread-safe and used as a singleton instance.
// It caches information about your struct and validations,
// in essence only parsing your validation tags once per struct type.
// Using multiple instances neglects the benefit of caching.
// The not thread-safe functions are explicitly marked as such in the documentation.
//
// Validation Functions Return Type error
//
// Doing things this way is actually the way the standard library does, see the
// file.Open method here:
//
// 	https://golang.org/pkg/os/#Open.
//
// The authors return type "error" to avoid the issue discussed in the following,
// where err is always != nil:
//
// 	http://stackoverflow.com/a/29138676/3158232
// 	https://github.com/go-playground/validator/issues/134
//
// Validator only InvalidValidationError for bad validation input, nil or
// ValidationErrors as type error; so, in your code all you need to do is check
// if the error returned is not nil, and if it's not check if error is
// InvalidValidationError ( if necessary, most of the time it isn't ) type cast
// it to type ValidationErrors like so err.(validator.ValidationErrors).
//
// Custom Validation Functions
//
// Custom Validation functions can be added. Example:
//
// 	// Structure
// 	func customFunc(fl validator.FieldLevel) bool {
//
// 		if fl.Field().String() == "invalid" {
// 			return false
// 		}
//
// 		return true
// 	}
//
// 	validate.RegisterValidation("custom tag name", customFunc)
// 	// NOTES: using the same tag name as an existing function
// 	//        will overwrite the existing one
//
// Cross-Field Validation
//
// Cross-Field Validation can be done via the following tags:
// 	- eqfield
// 	- nefield
// 	- gtfield
// 	- gtefield
// 	- ltfield
// 	- ltefield
// 	- eqcsfield
// 	- necsfield
// 	- gtcsfield
// 	- gtecsfield
// 	- ltcsfield
// 	- ltecsfield
//
// If, however, some custom cross-field validation is required, it can be done
// using a custom validation.
//
// Why not just have cross-fields validation tags (i.e. only eqcsfield and not
// eqfield)?
//
// The reason is efficiency. If you want to check a field within the same struct
// "eqfield" only has to find the field on the same struct (1 level). But, if we
// used "eqcsfield" it could be multiple levels down. Example:
//
// 	type Inner struct {
// 		StartDate time.Time
// 	}
//
// 	type Outer struct {
// 		InnerStructField *Inner
// 		CreatedAt time.Time      `validate:"ltecsfield=InnerStructField.StartDate"`
// 	}
//
// 	now := time.Now()
//
// 	inner := &Inner{
// 		StartDate: now,
// 	}
//
// 	outer := &Outer{
// 		InnerStructField: inner,
// 		CreatedAt: now,
// 	}
//
// 	errs := validate.Struct(outer)
//
// 	// NOTE: when calling validate.Struct(val) topStruct will be the top level struct passed
// 	//       into the function
// 	//       when calling validate.VarWithValue(val, field, tag) val will be
// 	//       whatever you pass, struct, field...
// 	//       when calling validate.Field(field, tag) val will be nil
//
// Multiple Validators
//
// Multiple validators on a field will process in the order defined. Example:
//
// 	type Test struct {
// 		Field `validate:"max=10,min=1"`
// 	}
//
// 	// max will be checked then min
//
// Bad Validator definitions are not handled by the library. Example:
//
// 	type Test struct {
// 		Field `validate:"min=10,max=0"`
// 	}
//
// 	// this definition of min max will never succeed
//
// Using Validator Tags
//
// Baked In Cross-Field validation only compares fields on the same struct.
// If Cross-Field + Cross-Struct validation is needed you should implement your
// own custom validator.
//
// Comma (",") is the default separator of validation tags. If you wish to
// have a comma included within the parameter (i.e. excludesall=,) you will need to
// use the UTF-8 hex representation 0x2C, which is replaced in the code as a comma,
// so the above will become excludesall=0x2C.
//
// 	type Test struct {
// 		Field `validate:"excludesall=,"`    // BAD! Do not include a comma.
// 		Field `validate:"excludesall=0x2C"` // GOOD! Use the UTF-8 hex representation.
// 	}
//
// Pipe ("|") is the 'or' validation tags deparator. If you wish to
// have a pipe included within the parameter i.e. excludesall=| you will need to
// use the UTF-8 hex representation 0x7C, which is replaced in the code as a pipe,
// so the above will become excludesall=0x7C
//
// 	type Test struct {
// 		Field `validate:"excludesall=|"`    // BAD! Do not include a a pipe!
// 		Field `validate:"excludesall=0x7C"` // GOOD! Use the UTF-8 hex representation.
// 	}
//
// Baked In Validators and Tags
//
// Here is a list of the current built in validators:
//
// Skip Field
//
// Tells the validation to skip this struct field; this is particularly
// handy in ignoring embedded structs from being validated. (Usage: -)
// 	Usage: -
//
// Or Operator
//
// This is the 'or' operator allowing multiple validators to be used and
// accepted. (Usage: rgb|rgba) <-- this would allow either rgb or rgba
// colors to be accepted. This can also be combined with 'and' for example
// ( Usage: omitempty,rgb|rgba)
//
// 	Usage: |
//
// StructOnly
//
// When a field that is a nested struct is encountered, and contains this flag
// any validation on the nested struct will be run, but none of the nested
// struct fields will be validated. This is useful if inside of your program
// you know the struct will be valid, but need to verify it has been assigned.
// NOTE: only "required" and "omitempty" can be used on a struct itself.
//
// 	Usage: structonly
//
// NoStructLevel
//
// Same as structonly tag except that any struct level validations will not run.
//
// 	Usage: nostructlevel
//
// Omit Empty
//
// Allows conditional validation, for example if a field is not set with
// a value (Determined by the "required" validator) then other validation
// such as min or max won't run, but if a value is set validation will run.
//
// 	Usage: omitempty
//
// Dive
//
// This tells the validator to dive into a slice, array or map and validate that
// level of the slice, array or map with the validation tags that follow.
// Multidimensional nesting is also supported, each level you wish to dive will
// require another dive tag. dive has some sub-tags, 'keys' & 'endkeys', please see
// the Keys & EndKeys section just below.
//
// 	Usage: dive
//
// Example #1
//
// 	[][]string with validation tag "gt=0,dive,len=1,dive,required"
// 	// gt=0 will be applied to []
// 	// len=1 will be applied to []string
// 	// required will be applied to string
//
// Example #2
//
// 	[][]string with validation tag "gt=0,dive,dive,required"
// 	// gt=0 will be applied to []
// 	// []string will be spared validation
// 	// required will be applied to string
//
// Keys & EndKeys
//
// These are to be used together directly after the dive tag and tells the validator
// that anything between 'keys' and 'endkeys' applies to the keys of a map and not the
// values; think of it like the 'dive' tag, but for map keys instead of values.
// Multidimensional nesting is also supported, each level you wish to validate will
// require another 'keys' and 'endkeys' tag. These tags are only valid for maps.
//
// 	Usage: dive,keys,othertagvalidation(s),endkeys,valuevalidationtags
//
// Example #1
//
// 	map[string]string with validation tag "gt=0,dive,keys,eg=1|eq=2,endkeys,required"
// 	// gt=0 will be applied to the map itself
// 	// eg=1|eq=2 will be applied to the map keys
// 	// required will be applied to map values
//
// Example #2
//
// 	map[[2]string]string with validation tag "gt=0,dive,keys,dive,eq=1|eq=2,endkeys,required"
// 	// gt=0 will be applied to the map itself
// 	// eg=1|eq=2 will be applied to each array element in the the map keys
// 	// required will be applied to map values
//
// Required
//
// This validates that the value is not the data types default zero value.
// For numbers ensures value is not zero. For strings ensures value is
// not "". For slices, maps, pointers, interfaces, channels and functions
// ensures the value is not nil.
//
// 	Usage: required
//
// Required If
//
// The field under validation must be present and not empty only if all
// the other specified fields are equal to the value following the specified
// field. For strings ensures value is not "". For slices, maps, pointers,
// interfaces, channels and functions ensures the value is not nil.
//
// 	Usage: required_if
//
// Examples:
//
// 	// require the field if the Field1 is equal to the parameter given:
// 	Usage: required_if=Field1 foobar
//
// 	// require the field if the Field1 and Field2 is equal to the value respectively:
// 	Usage: required_if=Field1 foo Field2 bar
//
// Required Unless
//
// The field under validation must be present and not empty unless all
// the other specified fields are equal to the value following the specified
// field. For strings ensures value is not "". For slices, maps, pointers,
// interfaces, channels and functions ensures the value is not nil.
//
// 	Usage: required_unless
//
// Examples:
//
// 	// require the field unless the Field1 is equal to the parameter given:
// 	Usage: required_unless=Field1 foobar
//
// 	// require the field unless the Field1 and Field2 is equal to the value respectively:
// 	Usage: required_unless=Field1 foo Field2 bar
//
// Required With
//
// The field under validation must be present and not empty only if any
// of the other specified fields are present. For strings ensures value is
// not "". For slices, maps, pointers, interfaces, channels and functions
// ensures the value is not nil.
//
// 	Usage: required_with
//
// Examples:
//
// 	// require the field if the Field1 is present:
// 	Usage: required_with=Field1
//
// 	// require the field if the Field1 or Field2 is present:
// 	Usage: required_with=Field1 Field2
//
// Required With All
//
// The field under validation must be present and not empty only if all
// of the other specified fields are present. For strings ensures value is
// not "". For slices, maps, pointers, interfaces, channels and functions
// ensures the value is not nil.
//
// 	Usage: required_with_all
//
// Example:
//
// 	// require the field if the Field1 and Field2 is present:
// 	Usage: required_with_all=Field1 Field2
//
// Required Without
//
// The field under validation must be present and not empty only when any
// of the other specified fields are not present. For strings ensures value is
// not "". For slices, maps, pointers, interfaces, channels and functions
// ensures the value is not nil.
//
// 	Usage: required_without
//
// Examples:
//
// 	// require the field if the Field1 is not present:
// 	Usage: required_without=Field1
//
// 	// require the field if the Field1 or Field2 is not present:
// 	Usage: required_without=Field1 Field2
//
// Required Without All
//
// The field under validation must be present and not empty only when all
// of the other specified fields are not present. For strings ensures value is
// not "". For slices, maps, pointers, interfaces, channels and functions
// ensures the value is not nil.
//
// 	Usage: required_without_all
//
// Example:
//
// 	// require the field if the Field1 and Field2 is not present:
// 	Usage: required_without_all=Field1 Field2
//
// Is Default
//
// This validates that the value is the default value and is almost the
// opposite of required.
//
// 	Usage: isdefault
//
// Length
//
// For numbers, length will ensure that the value is
// equal to the parameter given. For strings, it checks that
// the string length is exactly that number of characters. For slices,
// arrays, and maps, validates the number of items.
//
// Example #1
//
// 	Usage: len=10
//
// Example #2 (time.Duration)
//
// For time.Duration, len will ensure that the value is equal to the duration given
// in the parameter.
//
// 	Usage: len=1h30m
//
// Maximum
//
// For numbers, max will ensure that the value is
// less than or equal to the parameter given. For strings, it checks
// that the string length is at most that number of characters. For
// slices, arrays, and maps, validates the number of items.
//
// Example #1
//
// 	Usage: max=10
//
// Example #2 (time.Duration)
//
// For time.Duration, max will ensure that the value is less than or equal to the
// duration given in the parameter.
//
// 	Usage: max=1h30m
//
// Minimum
//
// For numbers, min will ensure that the value is
// greater or equal to the parameter given. For strings, it checks that
// the string length is at least that number of characters. For slices,
// arrays, and maps, validates the number of items.
//
// Example #1
//
// 	Usage: min=10
//
// Example #2 (time.Duration)
//
// For time.Duration, min will ensure that the value is greater than or equal to
// the duration given in the parameter.
//
// 	Usage: min=1h30m
//
// Equals
//
// For strings & numbers, eq will ensure that the value is
// equal to the parameter given. For slices, arrays, and maps,
// validates the number of items.
//
// Example #1
//
// 	Usage: eq=10
//
// Example #2 (time.Duration)
//
// For time.Duration, eq will ensure that the value is equal to the duration given
// in the parameter.
//
// 	Usage: eq=1h30m
//
// Not Equal
//
// For strings & numbers, ne will ensure that the value is not
// equal to the parameter given. For slices, arrays, and maps,
// validates the number of items.
//
// Example #1
//
// 	Usage: ne=10
//
// Example #2 (time.Duration)
//
// For time.Duration, ne will ensure that the value is not equal to the duration
// given in the parameter.
//
// 	Usage: ne=1h30m
//
// One Of
//
// For strings, ints, and uints, oneof will ensure that the value
// is one of the values in the parameter.  The parameter should be
// a list of values separated by whitespace. Values may be
// strings or numbers. To match strings with spaces in them, include
// the target string between single quotes.
//
//     Usage: oneof=red green
//            oneof='red green' 'blue yellow'
//            oneof=5 7 9
//
// Greater Than
//
// For numbers, this will ensure that the value is greater than the
// parameter given. For strings, it checks that the string length
// is greater than that number of characters. For slices, arrays
// and maps it validates the number of items.
//
// Example #1
//
// 	Usage: gt=10
//
// Example #2 (time.Time)
//
// For time.Time ensures the time value is greater than time.Now.UTC().
//
// 	Usage: gt
//
// Example #3 (time.Duration)
//
// For time.Duration, gt will ensure that the value is greater than the duration
// given in the parameter.
//
// 	Usage: gt=1h30m
//
// Greater Than or Equal
//
// Same as 'min' above. Kept both to make terminology with 'len' easier.
//
// Example #1
//
// 	Usage: gte=10
//
// Example #2 (time.Time)
//
// For time.Time ensures the time value is greater than or equal to time.Now.UTC().
//
// 	Usage: gte
//
// Example #3 (time.Duration)
//
// For time.Duration, gte will ensure that the value is greater than or equal to
// the duration given in the parameter.
//
// 	Usage: gte=1h30m
//
// Less Than
//
// For numbers, this will ensure that the value is less than the parameter given.
// For strings, it checks that the string length is less than that number of
// characters. For slices, arrays, and maps it validates the number of items.
//
// Example #1
//
// 	Usage: lt=10
//
// Example #2 (time.Time)
//
// For time.Time ensures the time value is less than time.Now.UTC().
//
// 	Usage: lt
//
// Example #3 (time.Duration)
//
// For time.Duration, lt will ensure that the value is less than the duration given
// in the parameter.
//
// 	Usage: lt=1h30m
//
// Less Than or Equal
//
// Same as 'max' above. Kept both to make terminology with 'len' easier.
//
// Example #1
//
// 	Usage: lte=10
//
// Example #2 (time.Time)
//
// For time.Time ensures the time value is less than or equal to time.Now.UTC().
//
// 	Usage: lte
//
// Example #3 (time.Duration)
//
// For time.Duration, lte will ensure that the value is less than or equal to the
// duration given in the parameter.
//
// 	Usage: lte=1h30m
//
// Field Equals Another Field
//
// This will validate the field value against another fields value either within
// a struct or passed in field.
//
// Example #1:
//
// 	// Validation on Password field using:
// 	Usage: eqfield=ConfirmPassword
//
// Example #2:
//
// 	// Validating by field:
// 	validate.VarWithValue(password, confirmpassword, "eqfield")
//
// Field Equals Another Field (relative)
//
// This does the same as eqfield except that it validates the field provided relative
// to the top level struct.
//
// 	Usage: eqcsfield=InnerStructField.Field)
//
// Field Does Not Equal Another Field
//
// This will validate the field value against another fields value either within
// a struct or passed in field.
//
// Examples:
//
// 	// Confirm two colors are not the same:
// 	//
// 	// Validation on Color field:
// 	Usage: nefield=Color2
//
// 	// Validating by field:
// 	validate.VarWithValue(color1, color2, "nefield")
//
// Field Does Not Equal Another Field (relative)
//
// This does the same as nefield except that it validates the field provided
// relative to the top level struct.
//
// 	Usage: necsfield=InnerStructField.Field
//
// Field Greater Than Another Field
//
// Only valid for Numbers, time.Duration and time.Time types, this will validate
// the field value against another fields value either within a struct or passed in
// field. usage examples are for validation of a Start and End date:
//
// Example #1:
//
// 	// Validation on End field using:
// 	validate.Struct Usage(gtfield=Start)
//
// Example #2:
//
// 	// Validating by field:
// 	validate.VarWithValue(start, end, "gtfield")
//
// Field Greater Than Another Relative Field
//
// This does the same as gtfield except that it validates the field provided
// relative to the top level struct.
//
// 	Usage: gtcsfield=InnerStructField.Field
//
// Field Greater Than or Equal To Another Field
//
// Only valid for Numbers, time.Duration and time.Time types, this will validate
// the field value against another fields value either within a struct or passed in
// field. usage examples are for validation of a Start and End date:
//
// Example #1:
//
// 	// Validation on End field using:
// 	validate.Struct Usage(gtefield=Start)
//
// Example #2:
//
// 	// Validating by field:
// 	validate.VarWithValue(start, end, "gtefield")
//
// Field Greater Than or Equal To Another Relative Field
//
// This does the same as gtefield except that it validates the field provided relative
// to the top level struct.
//
// 	Usage: gtecsfield=InnerStructField.Field
//
// Less Than Another Field
//
// Only valid for Numbers, time.Duration and time.Time types, this will validate
// the field value against another fields value either within a struct or passed in
// field. usage examples are for validation of a Start and End date:
//
// Example #1:
//
// 	// Validation on End field using:
// 	validate.Struct Usage(ltfield=Start)
//
// Example #2:
//
// 	// Validating by field:
// 	validate.VarWithValue(start, end, "ltfield")
//
// Less Than Another Relative Field
//
// This does the same as ltfield except that it validates the field provided relative
// to the top level struct.
//
// 	Usage: ltcsfield=InnerStructField.Field
//
// Less Than or Equal To Another Field
//
// Only valid for Numbers, time.Duration and time.Time types, this will validate
// the field value against another fields value either within a struct or passed in
// field. usage examples are for validation of a Start and End date:
//
// Example #1:
//
// 	// Validation on End field using:
// 	validate.Struct Usage(ltefield=Start)
//
// Example #2:
//
// 	// Validating by field:
// 	validate.VarWithValue(start, end, "ltefield")
//
// Less Than or Equal To Another Relative Field
//
// This does the same as ltefield except that it validates the field provided relative
// to the top level struct.
//
// 	Usage: ltecsfield=InnerStructField.Field
//
// Field Contains Another Field
//
// This does the same as contains except for struct fields. It should only be used
// with string types. See the behavior of reflect.Value.String() for behavior on
// other types.
//
// 	Usage: containsfield=InnerStructField.Field
//
// Field Excludes Another Field
//
// This does the same as excludes except for struct fields. It should only be used
// with string types. See the behavior of reflect.Value.String() for behavior on
// other types.
//
// 	Usage: excludesfield=InnerStructField.Field
//
// Unique
//
// For arrays & slices, unique will ensure that there are no duplicates.
// For maps, unique will ensure that there are no duplicate values.
// For slices of struct, unique will ensure that there are no duplicate values
// in a field of the struct specified via a parameter.
//
// 	// For arrays, slices, and maps:
// 	Usage: unique
//
// 	// For slices of struct:
// 	Usage: unique=field
//
// Alpha Only
//
// This validates that a string value contains ASCII alpha characters only
//
// 	Usage: alpha
//
// Alphanumeric
//
// This validates that a string value contains ASCII alphanumeric characters only
//
// 	Usage: alphanum
//
// Alpha Unicode
//
// This validates that a string value contains unicode alpha characters only
//
// 	Usage: alphaunicode
//
// Alphanumeric Unicode
//
// This validates that a string value contains unicode alphanumeric characters only
//
// 	Usage: alphanumunicode
//
// Boolean
//
// This validates that a string value can successfully be parsed into a boolean with strconv.ParseBool
//
// 	Usage: boolean
//
// Number
//
// This validates that a string value contains number values only.
// For integers or float it returns true.
//
// 	Usage: number
//
// Numeric
//
// This validates that a string value contains a basic numeric value.
// basic excludes exponents etc...
// for integers or float it returns true.
//
// 	Usage: numeric
//
// Hexadecimal String
//
// This validates that a string value contains a valid hexadecimal.
//
// 	Usage: hexadecimal
//
// Hexcolor String
//
// This validates that a string value contains a valid hex color including
// hashtag (#)
//
// 		Usage: hexcolor
//
// Lowercase String
//
// This validates that a string value contains only lowercase characters. An empty string is not a valid lowercase string.
//
// 	Usage: lowercase
//
// Uppercase String
//
// This validates that a string value contains only uppercase characters. An empty string is not a valid uppercase string.
//
// 	Usage: uppercase
//
// RGB String
//
// This validates that a string value contains a valid rgb color
//
// 	Usage: rgb
//
// RGBA String
//
// This validates that a string value contains a valid rgba color
//
// 	Usage: rgba
//
// HSL String
//
// This validates that a string value contains a valid hsl color
//
// 	Usage: hsl
//
// HSLA String
//
// This validates that a string value contains a valid hsla color
//
// 	Usage: hsla
//
// E.164 Phone Number String
//
// This validates that a string value contains a valid E.164 Phone number
// https://en.wikipedia.org/wiki/E.164 (ex. +1123456789)
//
// 	Usage: e164
//
// E-mail String
//
// This validates that a string value contains a valid email
// This may not conform to all possibilities of any rfc standard, but neither
// does any email provider accept all possibilities.
//
// 	Usage: email
//
// JSON String
//
// This validates that a string value is valid JSON
//
// 	Usage: json
//
// JWT String
//
// This validates that a string value is a valid JWT
//
// 	Usage: jwt
//
// File path
//
// This validates that a string value contains a valid file path and that
// the file exists on the machine.
// This is done using os.Stat, which is a platform independent function.
//
// 	Usage: file
//
// URL String
//
// This validates that a string value contains a valid url
// This will accept any url the golang request uri accepts but must contain
// a schema for example http:// or rtmp://
//
// 	Usage: url
//
// URI String
//
// This validates that a string value contains a valid uri
// This will accept any uri the golang request uri accepts
//
// 	Usage: uri
//
// Urn RFC 2141 String
//
// This validataes that a string value contains a valid URN
// according to the RFC 2141 spec.
//
// 	Usage: urn_rfc2141
//
// Base64 String
//
// This validates that a string value contains a valid base64 value.
// Although an empty string is valid base64 this will report an empty string
// as an error, if you wish to accept an empty string as valid you can use
// this with the omitempty tag.
//
// 	Usage: base64
//
// Base64URL String
//
// This validates that a string value contains a valid base64 URL safe value
// according the the RFC4648 spec.
// Although an empty string is a valid base64 URL safe value, this will report
// an empty string as an error, if you wish to accept an empty string as valid
// you can use this with the omitempty tag.
//
// 	Usage: base64url
//
// Bitcoin Address
//
// This validates that a string value contains a valid bitcoin address.
// The format of the string is checked to ensure it matches one of the three formats
// P2PKH, P2SH and performs checksum validation.
//
// 	Usage: btc_addr
//
// Bitcoin Bech32 Address (segwit)
//
// This validates that a string value contains a valid bitcoin Bech32 address as defined
// by bip-0173 (https://github.com/bitcoin/bips/blob/master/bip-0173.mediawiki)
// Special thanks to Pieter Wuille for providng reference implementations.
//
// 	Usage: btc_addr_bech32
//
// Ethereum Address
//
// This validates that a string value contains a valid ethereum address.
// The format of the string is checked to ensure it matches the standard Ethereum address format.
//
// 	Usage: eth_addr
//
// Contains
//
// This validates that a string value contains the substring value.
//
// 	Usage: contains=@
//
// Contains Any
//
// This validates that a string value contains any Unicode code points
// in the substring value.
//
// 	Usage: containsany=!@#?
//
// Contains Rune
//
// This validates that a string value contains the supplied rune value.
//
// 	Usage: containsrune=@
//
// Excludes
//
// This validates that a string value does not contain the substring value.
//
// 	Usage: excludes=@
//
// Excludes All
//
// This validates that a string value does not contain any Unicode code
// points in the substring value.
//
// 	Usage: excludesall=!@#?
//
// Excludes Rune
//
// This validates that a string value does not contain the supplied rune value.
//
// 	Usage: excludesrune=@
//
// Starts With
//
// This validates that a string value starts with the supplied string value
//
// 	Usage: startswith=hello
//
// Ends With
//
// This validates that a string value ends with the supplied string value
//
// 	Usage: endswith=goodbye
//
// Does Not Start With
//
// This validates that a string value does not start with the supplied string value
//
// 	Usage: startsnotwith=hello
//
// Does Not End With
//
// This validates that a string value does not end with the supplied string value
//
// 	Usage: endsnotwith=goodbye
//
// International Standard Book Number
//
// This validates that a string value contains a valid isbn10 or isbn13 value.
//
// 	Usage: isbn
//
// International Standard Book Number 10
//
// This validates that a string value contains a valid isbn10 value.
//
// 	Usage: isbn10
//
// International Standard Book Number 13
//
// This validates that a string value contains a valid isbn13 value.
//
// 	Usage: isbn13
//
// Universally Unique Identifier UUID
//
// This validates that a string value contains a valid UUID. Uppercase UUID values will not pass - use `uuid_rfc4122` instead.
//
// 	Usage: uuid
//
// Universally Unique Identifier UUID v3
//
// This validates that a string value contains a valid version 3 UUID.  Uppercase UUID values will not pass - use `uuid3_rfc4122` instead.
//
// 	Usage: uuid3
//
// Universally Unique Identifier UUID v4
//
// This validates that a string value contains a valid version 4 UUID.  Uppercase UUID values will not pass - use `uuid4_rfc4122` instead.
//
// 	Usage: uuid4
//
// Universally Unique Identifier UUID v5
//
// This validates that a string value contains a valid version 5 UUID.  Uppercase UUID values will not pass - use `uuid5_rfc4122` instead.
//
// 	Usage: uuid5
//
// ASCII
//
// This validates that a string value contains only ASCII characters.
// NOTE: if the string is blank, this validates as true.
//
// 	Usage: ascii
//
// Printable ASCII
//
// This validates that a string value contains only printable ASCII characters.
// NOTE: if the string is blank, this validates as true.
//
// 	Usage: printascii
//
// Multi-Byte Characters
//
// This validates that a string value contains one or more multibyte characters.
// NOTE: if the string is blank, this validates as true.
//
// 	Usage: multibyte
//
// Data URL
//
// This validates that a string value contains a valid DataURI.
// NOTE: this will also validate that the data portion is valid base64
//
// 	Usage: datauri
//
// Latitude
//
// This validates that a string value contains a valid latitude.
//
// 	Usage: latitude
//
// Longitude
//
// This validates that a string value contains a valid longitude.
//
// 	Usage: longitude
//
// Social Security Number SSN
//
// This validates that a string value contains a valid U.S. Social Security Number.
//
// 	Usage: ssn
//
// Internet Protocol Address IP
//
// This validates that a string value contains a valid IP Address.
//
// 	Usage: ip
//
// Internet Protocol Address IPv4
//
// This validates that a string value contains a valid v4 IP Address.
//
// 	Usage: ipv4
//
// Internet Protocol Address IPv6
//
// This validates that a string value contains a valid v6 IP Address.
//
// 	Usage: ipv6
//
// Classless Inter-Domain Routing CIDR
//
// This validates that a string value contains a valid CIDR Address.
//
// 	Usage: cidr
//
// Classless Inter-Domain Routing CIDRv4
//
// This validates that a string value contains a valid v4 CIDR Address.
//
// 	Usage: cidrv4
//
// Classless Inter-Domain Routing CIDRv6
//
// This validates that a string value contains a valid v6 CIDR Address.
//
// 	Usage: cidrv6
//
// Transmission Control Protocol Address TCP
//
// This validates that a string value contains a valid resolvable TCP Address.
//
// 	Usage: tcp_addr
//
// Transmission Control Protocol Address TCPv4
//
// This validates that a string value contains a valid resolvable v4 TCP Address.
//
// 	Usage: tcp4_addr
//
// Transmission Control Protocol Address TCPv6
//
// This validates that a string value contains a valid resolvable v6 TCP Address.
//
// 	Usage: tcp6_addr
//
// User Datagram Protocol Address UDP
//
// This validates that a string value contains a valid resolvable UDP Address.
//
// 	Usage: udp_addr
//
// User Datagram Protocol Address UDPv4
//
// This validates that a string value contains a valid resolvable v4 UDP Address.
//
// 	Usage: udp4_addr
//
// User Datagram Protocol Address UDPv6
//
// This validates that a string value contains a valid resolvable v6 UDP Address.
//
// 	Usage: udp6_addr
//
// Internet Protocol Address IP
//
// This validates that a string value contains a valid resolvable IP Address.
//
// 	Usage: ip_addr
//
// Internet Protocol Address IPv4
//
// This validates that a string value contains a valid resolvable v4 IP Address.
//
// 	Usage: ip4_addr
//
// Internet Protocol Address IPv6
//
// This validates that a string value contains a valid resolvable v6 IP Address.
//
// 	Usage: ip6_addr
//
// Unix domain socket end point Address
//
// This validates that a string value contains a valid Unix Address.
//
// 	Usage: unix_addr
//
// Media Access Control Address MAC
//
// This validates that a string value contains a valid MAC Address.
//
// 	Usage: mac
//
// Note: See Go's ParseMAC for accepted formats and types:
//
// 	http://golang.org/src/net/mac.go?s=866:918#L29
//
// Hostname RFC 952
//
// This validates that a string value is a valid Hostname according to RFC 952 https://tools.ietf.org/html/rfc952
//
// 	Usage: hostname
//
// Hostname RFC 1123
//
// This validates that a string value is a valid Hostname according to RFC 1123 https://tools.ietf.org/html/rfc1123
//
// 	Usage: hostname_rfc1123 or if you want to continue to use 'hostname' in your tags, create an alias.
//
// Full Qualified Domain Name (FQDN)
//
// This validates that a string value contains a valid FQDN.
//
// 	Usage: fqdn
//
// HTML Tags
//
// This validates that a string value appears to be an HTML element tag
// including those described at https://developer.mozilla.org/en-US/docs/Web/HTML/Element
//
// 	Usage: html
//
// HTML Encoded
//
// This validates that a string value is a proper character reference in decimal
// or hexadecimal format
//
// 	Usage: html_encoded
//
// URL Encoded
//
// This validates that a string value is percent-encoded (URL encoded) according
// to https://tools.ietf.org/html/rfc3986#section-2.1
//
// 	Usage: url_encoded
//
// Directory
//
// This validates that a string value contains a valid directory and that
// it exists on the machine.
// This is done using os.Stat, which is a platform independent function.
//
// 	Usage: dir
//
// HostPort
//
// This validates that a string value contains a valid DNS hostname and port that
// can be used to valiate fields typically passed to sockets and connections.
//
// 	Usage: hostname_port
//
// Datetime
//
// This validates that a string value is a valid datetime based on the supplied datetime format.
// Supplied format must match the official Go time format layout as documented in https://golang.org/pkg/time/
//
// 	Usage: datetime=2006-01-02
//
// Iso3166-1 alpha-2
//
// This validates that a string value is a valid country code based on iso3166-1 alpha-2 standard.
// see: https://www.iso.org/iso-3166-country-codes.html
//
// 	Usage: iso3166_1_alpha2
//
// Iso3166-1 alpha-3
//
// This validates that a string value is a valid country code based on iso3166-1 alpha-3 standard.
// see: https://www.iso.org/iso-3166-country-codes.html
//
// 	Usage: iso3166_1_alpha3
//
// Iso3166-1 alpha-numeric
//
// This validates that a string value is a valid country code based on iso3166-1 alpha-numeric standard.
// see: https://www.iso.org/iso-3166-country-codes.html
//
// 	Usage: iso3166_1_alpha3
//
// BCP 47 Language Tag
//
// This validates that a string value is a valid BCP 47 language tag, as parsed by language.Parse.
// More information on https://pkg.go.dev/golang.org/x/text/language
//
// 	Usage: bcp47_language_tag
//
// BIC (SWIFT code)
//
// This validates that a string value is a valid Business Identifier Code (SWIFT code), defined in ISO 9362.
// More information on https://www.iso.org/standard/60390.html
//
// 	Usage: bic
//
// TimeZone
//
// This validates that a string value is a valid time zone based on the time zone database present on the system.
// Although empty value and Local value are allowed by time.LoadLocation golang function, they are not allowed by this validator.
// More information on https://golang.org/pkg/time/#LoadLocation
//
// 	Usage: timezone
//
// Alias Validators and Tags
//
// NOTE: When returning an error, the tag returned in "FieldError" will be
// the alias tag unless the dive tag is part of the alias. Everything after the
// dive tag is not reported as the alias tag. Also, the "ActualTag" in the before
// case will be the actual tag within the alias that failed.
//
// Here is a list of the current built in alias tags:
//
// 	"iscolor"
// 		alias is "hexcolor|rgb|rgba|hsl|hsla" (Usage: iscolor)
// 	"country_code"
// 		alias is "iso3166_1_alpha2|iso3166_1_alpha3|iso3166_1_alpha_numeric" (Usage: country_code)
//
// Validator notes:
//
// 	regex
// 		a regex validator won't be added because commas and = signs can be part
// 		of a regex which conflict with the validation definitions. Although
// 		workarounds can be made, they take away from using pure regex's.
// 		Furthermore it's quick and dirty but the regex's become harder to
// 		maintain and are not reusable, so it's as much a programming philosophy
// 		as anything.
//
// 		In place of this new validator functions should be created; a regex can
// 		be used within the validator function and even be precompiled for better
// 		efficiency within regexes.go.
//
// 		And the best reason, you can submit a pull request and we can keep on
// 		adding to the validation library of this package!
//
// Non standard validators
//
// A collection of validation rules that are frequently needed but are more
// complex than the ones found in the baked in validators.
// A non standard validator must be registered manually like you would
// with your own custom validation functions.
//
// Example of registration and use:
//
// 	type Test struct {
// 		TestField string `validate:"yourtag"`
// 	}
//
// 	t := &Test{
// 		TestField: "Test"
// 	}
//
// 	validate := validator.New()
// 	validate.RegisterValidation("yourtag", validators.NotBlank)
//
// Here is a list of the current non standard validators:
//
// 	NotBlank
// 		This validates that the value is not blank or with length zero.
// 		For strings ensures they do not contain only spaces. For channels, maps, slices and arrays
// 		ensures they don't have zero length. For others, a non empty value is required.
//
// 		Usage: notblank
//
// Panics
//
// This package panics when bad input is provided, this is by design, bad code like
// that should not make it to production.
//
// 	type Test struct {
// 		TestField string `validate:"nonexistantfunction=1"`
// 	}
//
// 	t := &Test{
// 		TestField: "Test"
// 	}
//
// 	validate.Struct(t) // this will panic
//

// Func accepts a FieldLevel interface for all validation needs. The return
// value should be true when validation succeeds.
type Func func(fl FieldLevel) bool

// FuncCtx accepts a context.Context and FieldLevel interface for all
// validation needs. The return value should be true when validation succeeds.
type FuncCtx func(ctx context.Context, fl FieldLevel) bool

// wrapFunc wraps noramal Func makes it compatible with FuncCtx
func wrapFunc(fn Func) FuncCtx {
	if fn == nil {
		return nil // be sure not to wrap a bad function.
	}
	return func(ctx context.Context, fl FieldLevel) bool {
		return fn(fl)
	}
}

var (
	restrictedTags = map[string]struct{}{
		diveTag:           {},
		keysTag:           {},
		endKeysTag:        {},
		structOnlyTag:     {},
		omitempty:         {},
		skipValidationTag: {},
		utf8HexComma:      {},
		utf8Pipe:          {},
		noStructLevelTag:  {},
		requiredTag:       {},
		isdefault:         {},
	}

	// bakedInAliases is a default mapping of a single validation tag that
	// defines a common or complex set of validation(s) to simplify
	// adding validation to structs.
	bakedInAliases = map[string]string{
		"iscolor":      "hexcolor|rgb|rgba|hsl|hsla",
		"country_code": "iso3166_1_alpha2|iso3166_1_alpha3|iso3166_1_alpha_numeric",
	}

	// bakedInValidators is the default map of ValidationFunc
	// you can add, remove or even replace items to suite your needs,
	// or even disregard and use your own map if so desired.
	bakedInValidators = map[string]Func{
		"required":                      hasValue,
		"required_if":                   requiredIf,
		"required_unless":               requiredUnless,
		"required_with":                 requiredWith,
		"required_with_all":             requiredWithAll,
		"required_without":              requiredWithout,
		"required_without_all":          requiredWithoutAll,
		"excluded_with":                 excludedWith,
		"excluded_with_all":             excludedWithAll,
		"excluded_without":              excludedWithout,
		"excluded_without_all":          excludedWithoutAll,
		"isdefault":                     isDefault,
		"len":                           hasLengthOf,
		"min":                           hasMinOf,
		"max":                           hasMaxOf,
		"eq":                            isEq,
		"ne":                            isNe,
		"lt":                            isLt,
		"lte":                           isLte,
		"gt":                            isGt,
		"gte":                           isGte,
		"eqfield":                       isEqField,
		"eqcsfield":                     isEqCrossStructField,
		"necsfield":                     isNeCrossStructField,
		"gtcsfield":                     isGtCrossStructField,
		"gtecsfield":                    isGteCrossStructField,
		"ltcsfield":                     isLtCrossStructField,
		"ltecsfield":                    isLteCrossStructField,
		"nefield":                       isNeField,
		"gtefield":                      isGteField,
		"gtfield":                       isGtField,
		"ltefield":                      isLteField,
		"ltfield":                       isLtField,
		"fieldcontains":                 fieldContains,
		"fieldexcludes":                 fieldExcludes,
		"alpha":                         isAlpha,
		"alphanum":                      isAlphanum,
		"alphaunicode":                  isAlphaUnicode,
		"alphanumunicode":               isAlphanumUnicode,
		"boolean":                       isBoolean,
		"numeric":                       isNumeric,
		"number":                        isNumber,
		"hexadecimal":                   isHexadecimal,
		"hexcolor":                      isHEXColor,
		"rgb":                           isRGB,
		"rgba":                          isRGBA,
		"hsl":                           isHSL,
		"hsla":                          isHSLA,
		"e164":                          isE164,
		"email":                         isEmail,
		"url":                           isURL,
		"uri":                           isURI,
		"file":                          isFile,
		"base64":                        isBase64,
		"base64url":                     isBase64URL,
		"contains":                      contains,
		"containsany":                   containsAny,
		"containsrune":                  containsRune,
		"excludes":                      excludes,
		"excludesall":                   excludesAll,
		"excludesrune":                  excludesRune,
		"startswith":                    startsWith,
		"endswith":                      endsWith,
		"startsnotwith":                 startsNotWith,
		"endsnotwith":                   endsNotWith,
		"isbn":                          isISBN,
		"isbn10":                        isISBN10,
		"isbn13":                        isISBN13,
		"eth_addr":                      isEthereumAddress,
		"btc_addr":                      isBitcoinAddress,
		"btc_addr_bech32":               isBitcoinBech32Address,
		"uuid":                          isUUID,
		"uuid3":                         isUUID3,
		"uuid4":                         isUUID4,
		"uuid5":                         isUUID5,
		"uuid_rfc4122":                  isUUIDRFC4122,
		"uuid3_rfc4122":                 isUUID3RFC4122,
		"uuid4_rfc4122":                 isUUID4RFC4122,
		"uuid5_rfc4122":                 isUUID5RFC4122,
		"ascii":                         isASCII,
		"printascii":                    isPrintableASCII,
		"multibyte":                     hasMultiByteCharacter,
		"datauri":                       isDataURI,
		"latitude":                      isLatitude,
		"longitude":                     isLongitude,
		"ssn":                           isSSN,
		"ipv4":                          isIPv4,
		"ipv6":                          isIPv6,
		"ip":                            isIP,
		"cidrv4":                        isCIDRv4,
		"cidrv6":                        isCIDRv6,
		"cidr":                          isCIDR,
		"tcp4_addr":                     isTCP4AddrResolvable,
		"tcp6_addr":                     isTCP6AddrResolvable,
		"tcp_addr":                      isTCPAddrResolvable,
		"udp4_addr":                     isUDP4AddrResolvable,
		"udp6_addr":                     isUDP6AddrResolvable,
		"udp_addr":                      isUDPAddrResolvable,
		"ip4_addr":                      isIP4AddrResolvable,
		"ip6_addr":                      isIP6AddrResolvable,
		"ip_addr":                       isIPAddrResolvable,
		"unix_addr":                     isUnixAddrResolvable,
		"mac":                           isMAC,
		"hostname":                      isHostnameRFC952,  // RFC 952
		"hostname_rfc1123":              isHostnameRFC1123, // RFC 1123
		"fqdn":                          isFQDN,
		"unique":                        isUnique,
		"oneof":                         isOneOf,
		"html":                          isHTML,
		"html_encoded":                  isHTMLEncoded,
		"url_encoded":                   isURLEncoded,
		"dir":                           isDir,
		"json":                          isJSON,
		"jwt":                           isJWT,
		"hostname_port":                 isHostnamePort,
		"lowercase":                     isLowercase,
		"uppercase":                     isUppercase,
		"datetime":                      isDatetime,
		"timezone":                      isTimeZone,
		"iso3166_1_alpha2":              isIso3166Alpha2,
		"iso3166_1_alpha3":              isIso3166Alpha3,
		"iso3166_1_alpha_numeric":       isIso3166AlphaNumeric,
		"iso3166_2":                     isIso31662,
		"iso4217":                       isIso4217,
		"iso4217_numeric":               isIso4217Numeric,
		"postcode_iso3166_alpha2":       isPostcodeByIso3166Alpha2,
		"postcode_iso3166_alpha2_field": isPostcodeByIso3166Alpha2Field,
		"bic":                           isIsoBicFormat,
	}
)

var oneofValsCache = map[string][]string{}

var oneofValsCacheRWLock = sync.RWMutex{}

func parseOneOfParam2(s string) []string {
	oneofValsCacheRWLock.RLock()
	vals, ok := oneofValsCache[s]
	oneofValsCacheRWLock.RUnlock()
	if !ok {
		oneofValsCacheRWLock.Lock()
		vals = splitParamsRegex.FindAllString(s, -1)
		for i := 0; i < len(vals); i++ {
			vals[i] = strings.Replace(vals[i], "'", "", -1)
		}
		oneofValsCache[s] = vals
		oneofValsCacheRWLock.Unlock()
	}
	return vals
}

func isURLEncoded(fl FieldLevel) bool {
	return uRLEncodedRegex.MatchString(fl.Field().String())
}

func isHTMLEncoded(fl FieldLevel) bool {
	return hTMLEncodedRegex.MatchString(fl.Field().String())
}

func isHTML(fl FieldLevel) bool {
	return hTMLRegex.MatchString(fl.Field().String())
}

func isOneOf(fl FieldLevel) bool {
	vals := parseOneOfParam2(fl.Param())

	field := fl.Field()

	var v string
	switch field.Kind() {
	case reflect.String:
		v = field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v = strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v = strconv.FormatUint(field.Uint(), 10)
	default:
		panic(fmt.Sprintf("Bad field type %T", field.Interface()))
	}
	for i := 0; i < len(vals); i++ {
		if vals[i] == v {
			return true
		}
	}
	return false
}

// isUnique is the validation function for validating if each array|slice|map value is unique
func isUnique(fl FieldLevel) bool {

	field := fl.Field()
	param := fl.Param()
	v := reflect.ValueOf(struct{}{})

	switch field.Kind() {
	case reflect.Slice, reflect.Array:
		elem := field.Type().Elem()
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		if param == "" {
			m := reflect.MakeMap(reflect.MapOf(elem, v.Type()))

			for i := 0; i < field.Len(); i++ {
				m.SetMapIndex(reflect.Indirect(field.Index(i)), v)
			}
			return field.Len() == m.Len()
		}

		sf, ok := elem.FieldByName(param)
		if !ok {
			panic(fmt.Sprintf("Bad field name %s", param))
		}

		sfTyp := sf.Type
		if sfTyp.Kind() == reflect.Ptr {
			sfTyp = sfTyp.Elem()
		}

		m := reflect.MakeMap(reflect.MapOf(sfTyp, v.Type()))
		for i := 0; i < field.Len(); i++ {
			m.SetMapIndex(reflect.Indirect(reflect.Indirect(field.Index(i)).FieldByName(param)), v)
		}
		return field.Len() == m.Len()
	case reflect.Map:
		m := reflect.MakeMap(reflect.MapOf(field.Type().Elem(), v.Type()))

		for _, k := range field.MapKeys() {
			m.SetMapIndex(field.MapIndex(k), v)
		}
		return field.Len() == m.Len()
	default:
		panic(fmt.Sprintf("Bad field type %T", field.Interface()))
	}
}

// isMAC is the validation function for validating if the field's value is a valid MAC address.
func isMAC(fl FieldLevel) bool {

	_, err := net.ParseMAC(fl.Field().String())

	return err == nil
}

// isCIDRv4 is the validation function for validating if the field's value is a valid v4 CIDR address.
func isCIDRv4(fl FieldLevel) bool {

	ip, _, err := net.ParseCIDR(fl.Field().String())

	return err == nil && ip.To4() != nil
}

// isCIDRv6 is the validation function for validating if the field's value is a valid v6 CIDR address.
func isCIDRv6(fl FieldLevel) bool {

	ip, _, err := net.ParseCIDR(fl.Field().String())

	return err == nil && ip.To4() == nil
}

// isCIDR is the validation function for validating if the field's value is a valid v4 or v6 CIDR address.
func isCIDR(fl FieldLevel) bool {

	_, _, err := net.ParseCIDR(fl.Field().String())

	return err == nil
}

// isIPv4 is the validation function for validating if a value is a valid v4 IP address.
func isIPv4(fl FieldLevel) bool {

	ip := net.ParseIP(fl.Field().String())

	return ip != nil && ip.To4() != nil
}

// isIPv6 is the validation function for validating if the field's value is a valid v6 IP address.
func isIPv6(fl FieldLevel) bool {

	ip := net.ParseIP(fl.Field().String())

	return ip != nil && ip.To4() == nil
}

// isIP is the validation function for validating if the field's value is a valid v4 or v6 IP address.
func isIP(fl FieldLevel) bool {

	ip := net.ParseIP(fl.Field().String())

	return ip != nil
}

// isSSN is the validation function for validating if the field's value is a valid SSN.
func isSSN(fl FieldLevel) bool {

	field := fl.Field()

	if field.Len() != 11 {
		return false
	}

	return sSNRegex.MatchString(field.String())
}

// isLongitude is the validation function for validating if the field's value is a valid longitude coordinate.
func isLongitude(fl FieldLevel) bool {
	field := fl.Field()

	var v string
	switch field.Kind() {
	case reflect.String:
		v = field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v = strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v = strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32:
		v = strconv.FormatFloat(field.Float(), 'f', -1, 32)
	case reflect.Float64:
		v = strconv.FormatFloat(field.Float(), 'f', -1, 64)
	default:
		panic(fmt.Sprintf("Bad field type %T", field.Interface()))
	}

	return longitudeRegex.MatchString(v)
}

// isLatitude is the validation function for validating if the field's value is a valid latitude coordinate.
func isLatitude(fl FieldLevel) bool {
	field := fl.Field()

	var v string
	switch field.Kind() {
	case reflect.String:
		v = field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v = strconv.FormatInt(field.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v = strconv.FormatUint(field.Uint(), 10)
	case reflect.Float32:
		v = strconv.FormatFloat(field.Float(), 'f', -1, 32)
	case reflect.Float64:
		v = strconv.FormatFloat(field.Float(), 'f', -1, 64)
	default:
		panic(fmt.Sprintf("Bad field type %T", field.Interface()))
	}

	return latitudeRegex.MatchString(v)
}

// isDataURI is the validation function for validating if the field's value is a valid data URI.
func isDataURI(fl FieldLevel) bool {

	uri := strings.SplitN(fl.Field().String(), ",", 2)

	if len(uri) != 2 {
		return false
	}

	if !dataURIRegex.MatchString(uri[0]) {
		return false
	}

	return base64Regex.MatchString(uri[1])
}

// hasMultiByteCharacter is the validation function for validating if the field's value has a multi byte character.
func hasMultiByteCharacter(fl FieldLevel) bool {

	field := fl.Field()

	if field.Len() == 0 {
		return true
	}

	return multibyteRegex.MatchString(field.String())
}

// isPrintableASCII is the validation function for validating if the field's value is a valid printable ASCII character.
func isPrintableASCII(fl FieldLevel) bool {
	return printableASCIIRegex.MatchString(fl.Field().String())
}

// isASCII is the validation function for validating if the field's value is a valid ASCII character.
func isASCII(fl FieldLevel) bool {
	return aSCIIRegex.MatchString(fl.Field().String())
}

// isUUID5 is the validation function for validating if the field's value is a valid v5 UUID.
func isUUID5(fl FieldLevel) bool {
	return uUID5Regex.MatchString(fl.Field().String())
}

// isUUID4 is the validation function for validating if the field's value is a valid v4 UUID.
func isUUID4(fl FieldLevel) bool {
	return uUID4Regex.MatchString(fl.Field().String())
}

// isUUID3 is the validation function for validating if the field's value is a valid v3 UUID.
func isUUID3(fl FieldLevel) bool {
	return uUID3Regex.MatchString(fl.Field().String())
}

// isUUID is the validation function for validating if the field's value is a valid UUID of any version.
func isUUID(fl FieldLevel) bool {
	return uUIDRegex.MatchString(fl.Field().String())
}

// isUUID5RFC4122 is the validation function for validating if the field's value is a valid RFC4122 v5 UUID.
func isUUID5RFC4122(fl FieldLevel) bool {
	return uUID5RFC4122Regex.MatchString(fl.Field().String())
}

// isUUID4RFC4122 is the validation function for validating if the field's value is a valid RFC4122 v4 UUID.
func isUUID4RFC4122(fl FieldLevel) bool {
	return uUID4RFC4122Regex.MatchString(fl.Field().String())
}

// isUUID3RFC4122 is the validation function for validating if the field's value is a valid RFC4122 v3 UUID.
func isUUID3RFC4122(fl FieldLevel) bool {
	return uUID3RFC4122Regex.MatchString(fl.Field().String())
}

// isUUIDRFC4122 is the validation function for validating if the field's value is a valid RFC4122 UUID of any version.
func isUUIDRFC4122(fl FieldLevel) bool {
	return uUIDRFC4122Regex.MatchString(fl.Field().String())
}

// isISBN is the validation function for validating if the field's value is a valid v10 or v13 ISBN.
func isISBN(fl FieldLevel) bool {
	return isISBN10(fl) || isISBN13(fl)
}

// isISBN13 is the validation function for validating if the field's value is a valid v13 ISBN.
func isISBN13(fl FieldLevel) bool {

	s := strings.Replace(strings.Replace(fl.Field().String(), "-", "", 4), " ", "", 4)

	if !iSBN13Regex.MatchString(s) {
		return false
	}

	var checksum int32
	var i int32

	factor := []int32{1, 3}

	for i = 0; i < 12; i++ {
		checksum += factor[i%2] * int32(s[i]-'0')
	}

	return (int32(s[12]-'0'))-((10-(checksum%10))%10) == 0
}

// isISBN10 is the validation function for validating if the field's value is a valid v10 ISBN.
func isISBN10(fl FieldLevel) bool {

	s := strings.Replace(strings.Replace(fl.Field().String(), "-", "", 3), " ", "", 3)

	if !iSBN10Regex.MatchString(s) {
		return false
	}

	var checksum int32
	var i int32

	for i = 0; i < 9; i++ {
		checksum += (i + 1) * int32(s[i]-'0')
	}

	if s[9] == 'X' {
		checksum += 10 * 10
	} else {
		checksum += 10 * int32(s[9]-'0')
	}

	return checksum%11 == 0
}

// isEthereumAddress is the validation function for validating if the field's value is a valid Ethereum address.
func isEthereumAddress(fl FieldLevel) bool {
	address := fl.Field().String()

	if !ethAddressRegex.MatchString(address) {
		return false
	}

	if ethAddressRegexUpper.MatchString(address) || ethAddressRegexLower.MatchString(address) {
		return true
	}

	// Checksum validation. Reference: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-55.md
	address = address[2:] // Skip "0x" prefix.
	h := NewLegacyKeccak256()
	// hash.Hash's io.Writer implementation says it never returns an error. https://golang.org/pkg/hash/#Hash
	_, _ = h.Write([]byte(strings.ToLower(address)))
	hash := hex.EncodeToString(h.Sum(nil))

	for i := 0; i < len(address); i++ {
		if address[i] <= '9' { // Skip 0-9 digits: they don't have upper/lower-case.
			continue
		}
		if hash[i] > '7' && address[i] >= 'a' || hash[i] <= '7' && address[i] <= 'F' {
			return false
		}
	}

	return true
}

// isBitcoinAddress is the validation function for validating if the field's value is a valid btc address
func isBitcoinAddress(fl FieldLevel) bool {
	address := fl.Field().String()

	if !btcAddressRegex.MatchString(address) {
		return false
	}

	alphabet := []byte("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")

	decode := [25]byte{}

	for _, n := range []byte(address) {
		d := bytes.IndexByte(alphabet, n)

		for i := 24; i >= 0; i-- {
			d += 58 * int(decode[i])
			decode[i] = byte(d % 256)
			d /= 256
		}
	}

	h := sha256.New()
	_, _ = h.Write(decode[:21])
	d := h.Sum([]byte{})
	h = sha256.New()
	_, _ = h.Write(d)

	validchecksum := [4]byte{}
	computedchecksum := [4]byte{}

	copy(computedchecksum[:], h.Sum(d[:0]))
	copy(validchecksum[:], decode[21:])

	return validchecksum == computedchecksum
}

// isBitcoinBech32Address is the validation function for validating if the field's value is a valid bech32 btc address
func isBitcoinBech32Address(fl FieldLevel) bool {
	address := fl.Field().String()

	if !btcLowerAddressRegexBech32.MatchString(address) && !btcUpperAddressRegexBech32.MatchString(address) {
		return false
	}

	am := len(address) % 8

	if am == 0 || am == 3 || am == 5 {
		return false
	}

	address = strings.ToLower(address)

	alphabet := "qpzry9x8gf2tvdw0s3jn54khce6mua7l"

	hr := []int{3, 3, 0, 2, 3} // the human readable part will always be bc
	addr := address[3:]
	dp := make([]int, 0, len(addr))

	for _, c := range addr {
		dp = append(dp, strings.IndexRune(alphabet, c))
	}

	ver := dp[0]

	if ver < 0 || ver > 16 {
		return false
	}

	if ver == 0 {
		if len(address) != 42 && len(address) != 62 {
			return false
		}
	}

	values := append(hr, dp...)

	GEN := []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

	p := 1

	for _, v := range values {
		b := p >> 25
		p = (p&0x1ffffff)<<5 ^ v

		for i := 0; i < 5; i++ {
			if (b>>uint(i))&1 == 1 {
				p ^= GEN[i]
			}
		}
	}

	if p != 1 {
		return false
	}

	b := uint(0)
	acc := 0
	mv := (1 << 5) - 1
	var sw []int

	for _, v := range dp[1 : len(dp)-6] {
		acc = (acc << 5) | v
		b += 5
		for b >= 8 {
			b -= 8
			sw = append(sw, (acc>>b)&mv)
		}
	}

	if len(sw) < 2 || len(sw) > 40 {
		return false
	}

	return true
}

// excludesRune is the validation function for validating that the field's value does not contain the rune specified within the param.
func excludesRune(fl FieldLevel) bool {
	return !containsRune(fl)
}

// excludesAll is the validation function for validating that the field's value does not contain any of the characters specified within the param.
func excludesAll(fl FieldLevel) bool {
	return !containsAny(fl)
}

// excludes is the validation function for validating that the field's value does not contain the text specified within the param.
func excludes(fl FieldLevel) bool {
	return !contains(fl)
}

// containsRune is the validation function for validating that the field's value contains the rune specified within the param.
func containsRune(fl FieldLevel) bool {

	r, _ := utf8.DecodeRuneInString(fl.Param())

	return strings.ContainsRune(fl.Field().String(), r)
}

// containsAny is the validation function for validating that the field's value contains any of the characters specified within the param.
func containsAny(fl FieldLevel) bool {
	return strings.ContainsAny(fl.Field().String(), fl.Param())
}

// contains is the validation function for validating that the field's value contains the text specified within the param.
func contains(fl FieldLevel) bool {
	return strings.Contains(fl.Field().String(), fl.Param())
}

// startsWith is the validation function for validating that the field's value starts with the text specified within the param.
func startsWith(fl FieldLevel) bool {
	return strings.HasPrefix(fl.Field().String(), fl.Param())
}

// endsWith is the validation function for validating that the field's value ends with the text specified within the param.
func endsWith(fl FieldLevel) bool {
	return strings.HasSuffix(fl.Field().String(), fl.Param())
}

// startsNotWith is the validation function for validating that the field's value does not start with the text specified within the param.
func startsNotWith(fl FieldLevel) bool {
	return !startsWith(fl)
}

// endsNotWith is the validation function for validating that the field's value does not end with the text specified within the param.
func endsNotWith(fl FieldLevel) bool {
	return !endsWith(fl)
}

// fieldContains is the validation function for validating if the current field's value contains the field specified by the param's value.
func fieldContains(fl FieldLevel) bool {
	field := fl.Field()

	currentField, _, ok := fl.GetStructFieldOK()

	if !ok {
		return false
	}

	return strings.Contains(field.String(), currentField.String())
}

// fieldExcludes is the validation function for validating if the current field's value excludes the field specified by the param's value.
func fieldExcludes(fl FieldLevel) bool {
	field := fl.Field()

	currentField, _, ok := fl.GetStructFieldOK()
	if !ok {
		return true
	}

	return !strings.Contains(field.String(), currentField.String())
}

// isNeField is the validation function for validating if the current field's value is not equal to the field specified by the param's value.
func isNeField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	currentField, currentKind, ok := fl.GetStructFieldOK()

	if !ok || currentKind != kind {
		return true
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() != currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() != currentField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() != currentField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) != int64(currentField.Len())

	case reflect.Bool:
		return field.Bool() != currentField.Bool()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return true
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return !fieldTime.Equal(t)
		}

	}

	// default reflect.String:
	return field.String() != currentField.String()
}

// isNe is the validation function for validating that the field's value does not equal the provided param value.
func isNe(fl FieldLevel) bool {
	return !isEq(fl)
}

// isLteCrossStructField is the validation function for validating if the current field's value is less than or equal to the field, within a separate struct, specified by the param's value.
func isLteCrossStructField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	topField, topKind, ok := fl.GetStructFieldOK()
	if !ok || topKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() <= topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() <= topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() <= topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) <= int64(topField.Len())

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.Before(topTime) || fieldTime.Equal(topTime)
		}
	}

	// default reflect.String:
	return field.String() <= topField.String()
}

// isLtCrossStructField is the validation function for validating if the current field's value is less than the field, within a separate struct, specified by the param's value.
// NOTE: This is exposed for use within your own custom functions and not intended to be called directly.
func isLtCrossStructField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	topField, topKind, ok := fl.GetStructFieldOK()
	if !ok || topKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() < topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() < topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() < topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) < int64(topField.Len())

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.Before(topTime)
		}
	}

	// default reflect.String:
	return field.String() < topField.String()
}

// isGteCrossStructField is the validation function for validating if the current field's value is greater than or equal to the field, within a separate struct, specified by the param's value.
func isGteCrossStructField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	topField, topKind, ok := fl.GetStructFieldOK()
	if !ok || topKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() >= topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() >= topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() >= topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) >= int64(topField.Len())

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.After(topTime) || fieldTime.Equal(topTime)
		}
	}

	// default reflect.String:
	return field.String() >= topField.String()
}

// isGtCrossStructField is the validation function for validating if the current field's value is greater than the field, within a separate struct, specified by the param's value.
func isGtCrossStructField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	topField, topKind, ok := fl.GetStructFieldOK()
	if !ok || topKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() > topField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() > topField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() > topField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) > int64(topField.Len())

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			fieldTime := field.Interface().(time.Time)
			topTime := topField.Interface().(time.Time)

			return fieldTime.After(topTime)
		}
	}

	// default reflect.String:
	return field.String() > topField.String()
}

// isNeCrossStructField is the validation function for validating that the current field's value is not equal to the field, within a separate struct, specified by the param's value.
func isNeCrossStructField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	topField, currentKind, ok := fl.GetStructFieldOK()
	if !ok || currentKind != kind {
		return true
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return topField.Int() != field.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return topField.Uint() != field.Uint()

	case reflect.Float32, reflect.Float64:
		return topField.Float() != field.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(topField.Len()) != int64(field.Len())

	case reflect.Bool:
		return topField.Bool() != field.Bool()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return true
		}

		if fieldType == timeType {

			t := field.Interface().(time.Time)
			fieldTime := topField.Interface().(time.Time)

			return !fieldTime.Equal(t)
		}
	}

	// default reflect.String:
	return topField.String() != field.String()
}

// isEqCrossStructField is the validation function for validating that the current field's value is equal to the field, within a separate struct, specified by the param's value.
func isEqCrossStructField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	topField, topKind, ok := fl.GetStructFieldOK()
	if !ok || topKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return topField.Int() == field.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return topField.Uint() == field.Uint()

	case reflect.Float32, reflect.Float64:
		return topField.Float() == field.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(topField.Len()) == int64(field.Len())

	case reflect.Bool:
		return topField.Bool() == field.Bool()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != topField.Type() {
			return false
		}

		if fieldType == timeType {

			t := field.Interface().(time.Time)
			fieldTime := topField.Interface().(time.Time)

			return fieldTime.Equal(t)
		}
	}

	// default reflect.String:
	return topField.String() == field.String()
}

// isEqField is the validation function for validating if the current field's value is equal to the field specified by the param's value.
func isEqField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	currentField, currentKind, ok := fl.GetStructFieldOK()
	if !ok || currentKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() == currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() == currentField.Uint()

	case reflect.Float32, reflect.Float64:
		return field.Float() == currentField.Float()

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) == int64(currentField.Len())

	case reflect.Bool:
		return field.Bool() == currentField.Bool()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.Equal(t)
		}

	}

	// default reflect.String:
	return field.String() == currentField.String()
}

// isEq is the validation function for validating if the current field's value is equal to the param's value.
func isEq(fl FieldLevel) bool {

	field := fl.Field()
	param := fl.Param()

	switch field.Kind() {

	case reflect.String:
		return field.String() == param

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) == p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asIntFromType(field.Type(), param)

		return field.Int() == p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() == p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() == p

	case reflect.Bool:
		p := asBool(param)

		return field.Bool() == p
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isPostcodeByIso3166Alpha2 validates by value which is country code in iso 3166 alpha 2
// example: `postcode_iso3166_alpha2=US`
func isPostcodeByIso3166Alpha2(fl FieldLevel) bool {
	field := fl.Field()
	param := fl.Param()

	reg, found := postCodeRegexDict[param]
	if !found {
		return false
	}

	return reg.MatchString(field.String())
}

// isPostcodeByIso3166Alpha2 validates by field which represents for a value of country code in iso 3166 alpha 2
// example: `postcode_iso3166_alpha2_field=CountryCode`
func isPostcodeByIso3166Alpha2Field(fl FieldLevel) bool {
	field := fl.Field()
	params := parseOneOfParam2(fl.Param())

	if len(params) != 1 {
		return false
	}

	currentField, kind, _, found := fl.GetStructFieldOKAdvanced2(fl.Parent(), params[0])
	if !found {
		return false
	}

	if kind != reflect.String {
		panic(fmt.Sprintf("Bad field type %T", currentField.Interface()))
	}

	reg, found := postCodeRegexDict[currentField.String()]
	if !found {
		return false
	}

	return reg.MatchString(field.String())
}

// isBase64 is the validation function for validating if the current field's value is a valid base 64.
func isBase64(fl FieldLevel) bool {
	return base64Regex.MatchString(fl.Field().String())
}

// isBase64URL is the validation function for validating if the current field's value is a valid base64 URL safe string.
func isBase64URL(fl FieldLevel) bool {
	return base64URLRegex.MatchString(fl.Field().String())
}

// isURI is the validation function for validating if the current field's value is a valid URI.
func isURI(fl FieldLevel) bool {

	field := fl.Field()

	switch field.Kind() {

	case reflect.String:

		s := field.String()

		// checks needed as of Go 1.6 because of change https://github.com/golang/go/commit/617c93ce740c3c3cc28cdd1a0d712be183d0b328#diff-6c2d018290e298803c0c9419d8739885L195
		// emulate browser and strip the '#' suffix prior to validation. see issue-#237
		if i := strings.Index(s, "#"); i > -1 {
			s = s[:i]
		}

		if len(s) == 0 {
			return false
		}

		_, err := url.ParseRequestURI(s)

		return err == nil
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isURL is the validation function for validating if the current field's value is a valid URL.
func isURL(fl FieldLevel) bool {

	field := fl.Field()

	switch field.Kind() {

	case reflect.String:

		var i int
		s := field.String()

		// checks needed as of Go 1.6 because of change https://github.com/golang/go/commit/617c93ce740c3c3cc28cdd1a0d712be183d0b328#diff-6c2d018290e298803c0c9419d8739885L195
		// emulate browser and strip the '#' suffix prior to validation. see issue-#237
		if i = strings.Index(s, "#"); i > -1 {
			s = s[:i]
		}

		if len(s) == 0 {
			return false
		}

		url, err := url.ParseRequestURI(s)

		if err != nil || url.Scheme == "" {
			return false
		}

		return true
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isUrnRFC2141 is the validation function for validating if the current field's value is a valid URN as per RFC 2141.
//func isUrnRFC2141(fl FieldLevel) bool {
//	field := fl.Field()
//
//	switch field.Kind() {
//
//	case reflect.String:
//
//		str := field.String()
//
//		_, match := urn_Parse([]byte(str))
//
//		return match
//	}
//
//	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
//}

// isFile is the validation function for validating if the current field's value is a valid file path.
func isFile(fl FieldLevel) bool {
	field := fl.Field()

	switch field.Kind() {
	case reflect.String:
		fileInfo, err := os.Stat(field.String())
		if err != nil {
			return false
		}

		return !fileInfo.IsDir()
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isE164 is the validation function for validating if the current field's value is a valid e.164 formatted phone number.
func isE164(fl FieldLevel) bool {
	return e164Regex.MatchString(fl.Field().String())
}

// isEmail is the validation function for validating if the current field's value is a valid email address.
func isEmail(fl FieldLevel) bool {
	return emailRegex.MatchString(fl.Field().String())
}

// isHSLA is the validation function for validating if the current field's value is a valid HSLA color.
func isHSLA(fl FieldLevel) bool {
	return hslaRegex.MatchString(fl.Field().String())
}

// isHSL is the validation function for validating if the current field's value is a valid HSL color.
func isHSL(fl FieldLevel) bool {
	return hslRegex.MatchString(fl.Field().String())
}

// isRGBA is the validation function for validating if the current field's value is a valid RGBA color.
func isRGBA(fl FieldLevel) bool {
	return rgbaRegex.MatchString(fl.Field().String())
}

// isRGB is the validation function for validating if the current field's value is a valid RGB color.
func isRGB(fl FieldLevel) bool {
	return rgbRegex.MatchString(fl.Field().String())
}

// isHEXColor is the validation function for validating if the current field's value is a valid HEX color.
func isHEXColor(fl FieldLevel) bool {
	return hexColorRegex.MatchString(fl.Field().String())
}

// isHexadecimal is the validation function for validating if the current field's value is a valid hexadecimal.
func isHexadecimal(fl FieldLevel) bool {
	return hexadecimalRegex.MatchString(fl.Field().String())
}

// isNumber is the validation function for validating if the current field's value is a valid number.
func isNumber(fl FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return true
	default:
		return numberRegex.MatchString(fl.Field().String())
	}
}

// isNumeric is the validation function for validating if the current field's value is a valid numeric value.
func isNumeric(fl FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		return true
	default:
		return numericRegex.MatchString(fl.Field().String())
	}
}

// isAlphanum is the validation function for validating if the current field's value is a valid alphanumeric value.
func isAlphanum(fl FieldLevel) bool {
	return alphaNumericRegex.MatchString(fl.Field().String())
}

// isAlpha is the validation function for validating if the current field's value is a valid alpha value.
func isAlpha(fl FieldLevel) bool {
	return alphaRegex.MatchString(fl.Field().String())
}

// isAlphanumUnicode is the validation function for validating if the current field's value is a valid alphanumeric unicode value.
func isAlphanumUnicode(fl FieldLevel) bool {
	return alphaUnicodeNumericRegex.MatchString(fl.Field().String())
}

// isAlphaUnicode is the validation function for validating if the current field's value is a valid alpha unicode value.
func isAlphaUnicode(fl FieldLevel) bool {
	return alphaUnicodeRegex.MatchString(fl.Field().String())
}

// isBoolean is the validation function for validating if the current field's value can be safely converted to a boolean.
func isBoolean(fl FieldLevel) bool {
	_, err := strconv.ParseBool(fl.Field().String())
	return err == nil
}

// isDefault is the opposite of required aka hasValue
func isDefault(fl FieldLevel) bool {
	return !hasValue(fl)
}

// hasValue is the validation function for validating if the current field's value is not the default static value.
func hasValue(fl FieldLevel) bool {
	field := fl.Field()
	switch field.Kind() {
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return !field.IsNil()
	default:
		if fl.(*validate).fldIsPointer && field.Interface() != nil {
			return true
		}
		return field.IsValid() && field.Interface() != reflect.Zero(field.Type()).Interface()
	}
}

// requireCheckField is a func for check field kind
func requireCheckFieldKind(fl FieldLevel, param string, defaultNotFoundValue bool) bool {
	field := fl.Field()
	kind := field.Kind()
	var nullable, found bool
	if len(param) > 0 {
		field, kind, nullable, found = fl.GetStructFieldOKAdvanced2(fl.Parent(), param)
		if !found {
			return defaultNotFoundValue
		}
	}
	switch kind {
	case reflect.Invalid:
		return defaultNotFoundValue
	case reflect.Slice, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Chan, reflect.Func:
		return field.IsNil()
	default:
		if nullable && field.Interface() != nil {
			return false
		}
		return field.IsValid() && field.Interface() == reflect.Zero(field.Type()).Interface()
	}
}

// requireCheckFieldValue is a func for check field value
func requireCheckFieldValue(fl FieldLevel, param string, value string, defaultNotFoundValue bool) bool {
	field, kind, _, found := fl.GetStructFieldOKAdvanced2(fl.Parent(), param)
	if !found {
		return defaultNotFoundValue
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() == asInt(value)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return field.Uint() == asUint(value)

	case reflect.Float32, reflect.Float64:
		return field.Float() == asFloat(value)

	case reflect.Slice, reflect.Map, reflect.Array:
		return int64(field.Len()) == asInt(value)

	case reflect.Bool:
		return field.Bool() == asBool(value)
	}

	// default reflect.String:
	return field.String() == value
}

// requiredIf is the validation function
// The field under validation must be present and not empty only if all the other specified fields are equal to the value following with the specified field.
func requiredIf(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	if len(params)%2 != 0 {
		panic(fmt.Sprintf("Bad param number for required_if %s", fl.FieldName()))
	}
	for i := 0; i < len(params); i += 2 {
		if !requireCheckFieldValue(fl, params[i], params[i+1], false) {
			return true
		}
	}
	return hasValue(fl)
}

// requiredUnless is the validation function
// The field under validation must be present and not empty only unless all the other specified fields are equal to the value following with the specified field.
func requiredUnless(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	if len(params)%2 != 0 {
		panic(fmt.Sprintf("Bad param number for required_unless %s", fl.FieldName()))
	}

	for i := 0; i < len(params); i += 2 {
		if requireCheckFieldValue(fl, params[i], params[i+1], false) {
			return true
		}
	}
	return hasValue(fl)
}

// excludedWith is the validation function
// The field under validation must not be present or is empty if any of the other specified fields are present.
func excludedWith(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	for _, param := range params {
		if !requireCheckFieldKind(fl, param, true) {
			return !hasValue(fl)
		}
	}
	return true
}

// requiredWith is the validation function
// The field under validation must be present and not empty only if any of the other specified fields are present.
func requiredWith(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	for _, param := range params {
		if !requireCheckFieldKind(fl, param, true) {
			return hasValue(fl)
		}
	}
	return true
}

// excludedWithAll is the validation function
// The field under validation must not be present or is empty if all of the other specified fields are present.
func excludedWithAll(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	for _, param := range params {
		if requireCheckFieldKind(fl, param, true) {
			return true
		}
	}
	return !hasValue(fl)
}

// requiredWithAll is the validation function
// The field under validation must be present and not empty only if all of the other specified fields are present.
func requiredWithAll(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	for _, param := range params {
		if requireCheckFieldKind(fl, param, true) {
			return true
		}
	}
	return hasValue(fl)
}

// excludedWithout is the validation function
// The field under validation must not be present or is empty when any of the other specified fields are not present.
func excludedWithout(fl FieldLevel) bool {
	if requireCheckFieldKind(fl, strings.TrimSpace(fl.Param()), true) {
		return !hasValue(fl)
	}
	return true
}

// requiredWithout is the validation function
// The field under validation must be present and not empty only when any of the other specified fields are not present.
func requiredWithout(fl FieldLevel) bool {
	if requireCheckFieldKind(fl, strings.TrimSpace(fl.Param()), true) {
		return hasValue(fl)
	}
	return true
}

// excludedWithoutAll is the validation function
// The field under validation must not be present or is empty when all of the other specified fields are not present.
func excludedWithoutAll(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	for _, param := range params {
		if !requireCheckFieldKind(fl, param, true) {
			return true
		}
	}
	return !hasValue(fl)
}

// requiredWithoutAll is the validation function
// The field under validation must be present and not empty only when all of the other specified fields are not present.
func requiredWithoutAll(fl FieldLevel) bool {
	params := parseOneOfParam2(fl.Param())
	for _, param := range params {
		if !requireCheckFieldKind(fl, param, true) {
			return true
		}
	}
	return hasValue(fl)
}

// isGteField is the validation function for validating if the current field's value is greater than or equal to the field specified by the param's value.
func isGteField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	currentField, currentKind, ok := fl.GetStructFieldOK()
	if !ok || currentKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() >= currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() >= currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() >= currentField.Float()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.After(t) || fieldTime.Equal(t)
		}
	}

	// default reflect.String
	return len(field.String()) >= len(currentField.String())
}

// isGtField is the validation function for validating if the current field's value is greater than the field specified by the param's value.
func isGtField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	currentField, currentKind, ok := fl.GetStructFieldOK()
	if !ok || currentKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() > currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() > currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() > currentField.Float()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.After(t)
		}
	}

	// default reflect.String
	return len(field.String()) > len(currentField.String())
}

// isGte is the validation function for validating if the current field's value is greater than or equal to the param's value.
func isGte(fl FieldLevel) bool {

	field := fl.Field()
	param := fl.Param()

	switch field.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) >= p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) >= p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asIntFromType(field.Type(), param)

		return field.Int() >= p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() >= p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() >= p

	case reflect.Struct:

		if field.Type() == timeType {

			now := time.Now().UTC()
			t := field.Interface().(time.Time)

			return t.After(now) || t.Equal(now)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isGt is the validation function for validating if the current field's value is greater than the param's value.
func isGt(fl FieldLevel) bool {

	field := fl.Field()
	param := fl.Param()

	switch field.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) > p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) > p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asIntFromType(field.Type(), param)

		return field.Int() > p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() > p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() > p
	case reflect.Struct:

		if field.Type() == timeType {

			return field.Interface().(time.Time).After(time.Now().UTC())
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// hasLengthOf is the validation function for validating if the current field's value is equal to the param's value.
func hasLengthOf(fl FieldLevel) bool {

	field := fl.Field()
	param := fl.Param()

	switch field.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) == p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) == p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asIntFromType(field.Type(), param)

		return field.Int() == p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() == p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() == p
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// hasMinOf is the validation function for validating if the current field's value is greater than or equal to the param's value.
func hasMinOf(fl FieldLevel) bool {
	return isGte(fl)
}

// isLteField is the validation function for validating if the current field's value is less than or equal to the field specified by the param's value.
func isLteField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	currentField, currentKind, ok := fl.GetStructFieldOK()
	if !ok || currentKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() <= currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() <= currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() <= currentField.Float()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.Before(t) || fieldTime.Equal(t)
		}
	}

	// default reflect.String
	return len(field.String()) <= len(currentField.String())
}

// isLtField is the validation function for validating if the current field's value is less than the field specified by the param's value.
func isLtField(fl FieldLevel) bool {

	field := fl.Field()
	kind := field.Kind()

	currentField, currentKind, ok := fl.GetStructFieldOK()
	if !ok || currentKind != kind {
		return false
	}

	switch kind {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:

		return field.Int() < currentField.Int()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:

		return field.Uint() < currentField.Uint()

	case reflect.Float32, reflect.Float64:

		return field.Float() < currentField.Float()

	case reflect.Struct:

		fieldType := field.Type()

		// Not Same underlying type i.e. struct and time
		if fieldType != currentField.Type() {
			return false
		}

		if fieldType == timeType {

			t := currentField.Interface().(time.Time)
			fieldTime := field.Interface().(time.Time)

			return fieldTime.Before(t)
		}
	}

	// default reflect.String
	return len(field.String()) < len(currentField.String())
}

// isLte is the validation function for validating if the current field's value is less than or equal to the param's value.
func isLte(fl FieldLevel) bool {

	field := fl.Field()
	param := fl.Param()

	switch field.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) <= p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) <= p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asIntFromType(field.Type(), param)

		return field.Int() <= p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() <= p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() <= p

	case reflect.Struct:

		if field.Type() == timeType {

			now := time.Now().UTC()
			t := field.Interface().(time.Time)

			return t.Before(now) || t.Equal(now)
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isLt is the validation function for validating if the current field's value is less than the param's value.
func isLt(fl FieldLevel) bool {

	field := fl.Field()
	param := fl.Param()

	switch field.Kind() {

	case reflect.String:
		p := asInt(param)

		return int64(utf8.RuneCountInString(field.String())) < p

	case reflect.Slice, reflect.Map, reflect.Array:
		p := asInt(param)

		return int64(field.Len()) < p

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p := asIntFromType(field.Type(), param)

		return field.Int() < p

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p := asUint(param)

		return field.Uint() < p

	case reflect.Float32, reflect.Float64:
		p := asFloat(param)

		return field.Float() < p

	case reflect.Struct:

		if field.Type() == timeType {

			return field.Interface().(time.Time).Before(time.Now().UTC())
		}
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// hasMaxOf is the validation function for validating if the current field's value is less than or equal to the param's value.
func hasMaxOf(fl FieldLevel) bool {
	return isLte(fl)
}

// isTCP4AddrResolvable is the validation function for validating if the field's value is a resolvable tcp4 address.
func isTCP4AddrResolvable(fl FieldLevel) bool {

	if !isIP4Addr(fl) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp4", fl.Field().String())
	return err == nil
}

// isTCP6AddrResolvable is the validation function for validating if the field's value is a resolvable tcp6 address.
func isTCP6AddrResolvable(fl FieldLevel) bool {

	if !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp6", fl.Field().String())

	return err == nil
}

// isTCPAddrResolvable is the validation function for validating if the field's value is a resolvable tcp address.
func isTCPAddrResolvable(fl FieldLevel) bool {

	if !isIP4Addr(fl) && !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp", fl.Field().String())

	return err == nil
}

// isUDP4AddrResolvable is the validation function for validating if the field's value is a resolvable udp4 address.
func isUDP4AddrResolvable(fl FieldLevel) bool {

	if !isIP4Addr(fl) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp4", fl.Field().String())

	return err == nil
}

// isUDP6AddrResolvable is the validation function for validating if the field's value is a resolvable udp6 address.
func isUDP6AddrResolvable(fl FieldLevel) bool {

	if !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp6", fl.Field().String())

	return err == nil
}

// isUDPAddrResolvable is the validation function for validating if the field's value is a resolvable udp address.
func isUDPAddrResolvable(fl FieldLevel) bool {

	if !isIP4Addr(fl) && !isIP6Addr(fl) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp", fl.Field().String())

	return err == nil
}

// isIP4AddrResolvable is the validation function for validating if the field's value is a resolvable ip4 address.
func isIP4AddrResolvable(fl FieldLevel) bool {

	if !isIPv4(fl) {
		return false
	}

	_, err := net.ResolveIPAddr("ip4", fl.Field().String())

	return err == nil
}

// isIP6AddrResolvable is the validation function for validating if the field's value is a resolvable ip6 address.
func isIP6AddrResolvable(fl FieldLevel) bool {

	if !isIPv6(fl) {
		return false
	}

	_, err := net.ResolveIPAddr("ip6", fl.Field().String())

	return err == nil
}

// isIPAddrResolvable is the validation function for validating if the field's value is a resolvable ip address.
func isIPAddrResolvable(fl FieldLevel) bool {

	if !isIP(fl) {
		return false
	}

	_, err := net.ResolveIPAddr("ip", fl.Field().String())

	return err == nil
}

// isUnixAddrResolvable is the validation function for validating if the field's value is a resolvable unix address.
func isUnixAddrResolvable(fl FieldLevel) bool {

	_, err := net.ResolveUnixAddr("unix", fl.Field().String())

	return err == nil
}

func isIP4Addr(fl FieldLevel) bool {

	val := fl.Field().String()

	if idx := strings.LastIndex(val, ":"); idx != -1 {
		val = val[0:idx]
	}

	ip := net.ParseIP(val)

	return ip != nil && ip.To4() != nil
}

func isIP6Addr(fl FieldLevel) bool {

	val := fl.Field().String()

	if idx := strings.LastIndex(val, ":"); idx != -1 {
		if idx != 0 && val[idx-1:idx] == "]" {
			val = val[1 : idx-1]
		}
	}

	ip := net.ParseIP(val)

	return ip != nil && ip.To4() == nil
}

func isHostnameRFC952(fl FieldLevel) bool {
	return hostnameRegexRFC952.MatchString(fl.Field().String())
}

func isHostnameRFC1123(fl FieldLevel) bool {
	return hostnameRegexRFC1123.MatchString(fl.Field().String())
}

func isFQDN(fl FieldLevel) bool {
	val := fl.Field().String()

	if val == "" {
		return false
	}

	return fqdnRegexRFC1123.MatchString(val)
}

// isDir is the validation function for validating if the current field's value is a valid directory.
func isDir(fl FieldLevel) bool {
	field := fl.Field()

	if field.Kind() == reflect.String {
		fileInfo, err := os.Stat(field.String())
		if err != nil {
			return false
		}

		return fileInfo.IsDir()
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isJSON is the validation function for validating if the current field's value is a valid json string.
func isJSON(fl FieldLevel) bool {
	field := fl.Field()

	if field.Kind() == reflect.String {
		val := field.String()
		return json.Valid([]byte(val))
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isJWT is the validation function for validating if the current field's value is a valid JWT string.
func isJWT(fl FieldLevel) bool {
	return jWTRegex.MatchString(fl.Field().String())
}

// isHostnamePort validates a <dns>:<port> combination for fields typically used for socket address.
func isHostnamePort(fl FieldLevel) bool {
	val := fl.Field().String()
	host, port, err := net.SplitHostPort(val)
	if err != nil {
		return false
	}
	// Port must be a iny <= 65535.
	if portNum, err := strconv.ParseInt(port, 10, 32); err != nil || portNum > 65535 || portNum < 1 {
		return false
	}

	// If host is specified, it should match a DNS name
	if host != "" {
		return hostnameRegexRFC1123.MatchString(host)
	}
	return true
}

// isLowercase is the validation function for validating if the current field's value is a lowercase string.
func isLowercase(fl FieldLevel) bool {
	field := fl.Field()

	if field.Kind() == reflect.String {
		if field.String() == "" {
			return false
		}
		return field.String() == strings.ToLower(field.String())
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isUppercase is the validation function for validating if the current field's value is an uppercase string.
func isUppercase(fl FieldLevel) bool {
	field := fl.Field()

	if field.Kind() == reflect.String {
		if field.String() == "" {
			return false
		}
		return field.String() == strings.ToUpper(field.String())
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isDatetime is the validation function for validating if the current field's value is a valid datetime string.
func isDatetime(fl FieldLevel) bool {
	field := fl.Field()
	param := fl.Param()

	if field.Kind() == reflect.String {
		_, err := time.Parse(param, field.String())

		return err == nil
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isTimeZone is the validation function for validating if the current field's value is a valid time zone string.
func isTimeZone(fl FieldLevel) bool {
	field := fl.Field()

	if field.Kind() == reflect.String {
		// empty value is converted to UTC by time.LoadLocation but disallow it as it is not a valid time zone name
		if field.String() == "" {
			return false
		}

		// Local value is converted to the current system time zone by time.LoadLocation but disallow it as it is not a valid time zone name
		if strings.ToLower(field.String()) == "local" {
			return false
		}

		_, err := time.LoadLocation(field.String())
		return err == nil
	}

	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
}

// isIso3166Alpha2 is the validation function for validating if the current field's value is a valid iso3166-1 alpha-2 country code.
func isIso3166Alpha2(fl FieldLevel) bool {
	val := fl.Field().String()
	return iso3166_1_alpha2[val]
}

// isIso3166Alpha2 is the validation function for validating if the current field's value is a valid iso3166-1 alpha-3 country code.
func isIso3166Alpha3(fl FieldLevel) bool {
	val := fl.Field().String()
	return iso3166_1_alpha3[val]
}

// isIso3166Alpha2 is the validation function for validating if the current field's value is a valid iso3166-1 alpha-numeric country code.
func isIso3166AlphaNumeric(fl FieldLevel) bool {
	field := fl.Field()

	var code int
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		code = int(field.Int() % 1000)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		code = int(field.Uint() % 1000)
	default:
		panic(fmt.Sprintf("Bad field type %T", field.Interface()))
	}
	return iso3166_1_alpha_numeric[code]
}

// isIso31662 is the validation function for validating if the current field's value is a valid iso3166-2 code.
func isIso31662(fl FieldLevel) bool {
	val := fl.Field().String()
	return iso3166_2[val]
}

// isIso4217 is the validation function for validating if the current field's value is a valid iso4217 currency code.
func isIso4217(fl FieldLevel) bool {
	val := fl.Field().String()
	return iso4217[val]
}

// isIso4217Numeric is the validation function for validating if the current field's value is a valid iso4217 numeric currency code.
func isIso4217Numeric(fl FieldLevel) bool {
	field := fl.Field()

	var code int
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		code = int(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		code = int(field.Uint())
	default:
		panic(fmt.Sprintf("Bad field type %T", field.Interface()))
	}
	return iso4217_numeric[code]
}

// isBCP47LanguageTag is the validation function for validating if the current field's value is a valid BCP 47 language tag, as parsed by language.Parse
//func isBCP47LanguageTag(fl FieldLevel) bool {
//	field := fl.Field()
//
//	if field.Kind() == reflect.String {
//		_, err := language.Parse(field.String())
//		return err == nil
//	}
//
//	panic(fmt.Sprintf("Bad field type %T", field.Interface()))
//}

// isIsoBicFormat is the validation function for validating if the current field's value is a valid Business Identifier Code (SWIFT code), defined in ISO 9362
func isIsoBicFormat(fl FieldLevel) bool {
	bicString := fl.Field().String()

	return bicRegex.MatchString(bicString)
}

type tagType uint8

const (
	typeDefault tagType = iota
	typeOmitEmpty
	typeIsDefault
	typeNoStructLevel
	typeStructOnly
	typeDive
	typeOr
	typeKeys
	typeEndKeys
)

const (
	invalidValidation   = "Invalid validation tag on field '%s'"
	undefinedValidation = "Undefined validation function '%s' on field '%s'"
	keysTagNotDefined   = "'" + endKeysTag + "' tag encountered without a corresponding '" + keysTag + "' tag"
)

type structCache struct {
	lock sync.Mutex
	m    atomic.Value // map[reflect.Type]*cStruct
}

func (sc *structCache) Get(key reflect.Type) (c *cStruct, found bool) {
	c, found = sc.m.Load().(map[reflect.Type]*cStruct)[key]
	return
}

func (sc *structCache) Set(key reflect.Type, value *cStruct) {
	m := sc.m.Load().(map[reflect.Type]*cStruct)
	nm := make(map[reflect.Type]*cStruct, len(m)+1)
	for k, v := range m {
		nm[k] = v
	}
	nm[key] = value
	sc.m.Store(nm)
}

type tagCache struct {
	lock sync.Mutex
	m    atomic.Value // map[string]*cTag
}

func (tc *tagCache) Get(key string) (c *cTag, found bool) {
	c, found = tc.m.Load().(map[string]*cTag)[key]
	return
}

func (tc *tagCache) Set(key string, value *cTag) {
	m := tc.m.Load().(map[string]*cTag)
	nm := make(map[string]*cTag, len(m)+1)
	for k, v := range m {
		nm[k] = v
	}
	nm[key] = value
	tc.m.Store(nm)
}

type cStruct struct {
	name   string
	fields []*cField
	fn     StructLevelFuncCtx
}

type cField struct {
	idx        int
	name       string
	altName    string
	namesEqual bool
	cTags      *cTag
}

type cTag struct {
	tag                  string
	aliasTag             string
	actualAliasTag       string
	param                string
	keys                 *cTag // only populated when using tag's 'keys' and 'endkeys' for map key validation
	next                 *cTag
	fn                   FuncCtx
	typeof               tagType
	hasTag               bool
	hasAlias             bool
	hasParam             bool // true if parameter used eg. eq= where the equal sign has been set
	isBlockEnd           bool // indicates the current tag represents the last validation in the block
	runValidationWhenNil bool
}

func (v *Validate) extractStructCache(current reflect.Value, sName string) *cStruct {
	v.structCache.lock.Lock()
	defer v.structCache.lock.Unlock() // leave as defer! because if inner panics, it will never get unlocked otherwise!

	typ := current.Type()

	// could have been multiple trying to access, but once first is done this ensures struct
	// isn't parsed again.
	cs, ok := v.structCache.Get(typ)
	if ok {
		return cs
	}

	cs = &cStruct{name: sName, fields: make([]*cField, 0), fn: v.structLevelFuncs[typ]}

	numFields := current.NumField()

	var ctag *cTag
	var fld reflect.StructField
	var tag string
	var customName string

	for i := 0; i < numFields; i++ {

		fld = typ.Field(i)

		if !fld.Anonymous && len(fld.PkgPath) > 0 {
			continue
		}

		tag = fld.Tag.Get(v.tagName)

		if tag == skipValidationTag {
			continue
		}

		customName = fld.Name

		if v.hasTagNameFunc {
			name := v.tagNameFunc(fld)
			if len(name) > 0 {
				customName = name
			}
		}

		// NOTE: cannot use shared tag cache, because tags may be equal, but things like alias may be different
		// and so only struct level caching can be used instead of combined with Field tag caching

		if len(tag) > 0 {
			ctag, _ = v.parseFieldTagsRecursive(tag, fld.Name, "", false)
		} else {
			// even if field doesn't have validations need cTag for traversing to potential inner/nested
			// elements of the field.
			ctag = new(cTag)
		}

		cs.fields = append(cs.fields, &cField{
			idx:        i,
			name:       fld.Name,
			altName:    customName,
			cTags:      ctag,
			namesEqual: fld.Name == customName,
		})
	}
	v.structCache.Set(typ, cs)
	return cs
}

func (v *Validate) parseFieldTagsRecursive(tag string, fieldName string, alias string, hasAlias bool) (firstCtag *cTag, current *cTag) {
	var t string
	noAlias := len(alias) == 0
	tags := strings.Split(tag, tagSeparator)

	for i := 0; i < len(tags); i++ {
		t = tags[i]
		if noAlias {
			alias = t
		}

		// check map for alias and process new tags, otherwise process as usual
		if tagsVal, found := v.aliases[t]; found {
			if i == 0 {
				firstCtag, current = v.parseFieldTagsRecursive(tagsVal, fieldName, t, true)
			} else {
				next, curr := v.parseFieldTagsRecursive(tagsVal, fieldName, t, true)
				current.next, current = next, curr

			}
			continue
		}

		var prevTag tagType

		if i == 0 {
			current = &cTag{aliasTag: alias, hasAlias: hasAlias, hasTag: true, typeof: typeDefault}
			firstCtag = current
		} else {
			prevTag = current.typeof
			current.next = &cTag{aliasTag: alias, hasAlias: hasAlias, hasTag: true}
			current = current.next
		}

		switch t {
		case diveTag:
			current.typeof = typeDive
			continue

		case keysTag:
			current.typeof = typeKeys

			if i == 0 || prevTag != typeDive {
				panic(fmt.Sprintf("'%s' tag must be immediately preceded by the '%s' tag", keysTag, diveTag))
			}

			current.typeof = typeKeys

			// need to pass along only keys tag
			// need to increment i to skip over the keys tags
			b := make([]byte, 0, 64)

			i++

			for ; i < len(tags); i++ {

				b = append(b, tags[i]...)
				b = append(b, ',')

				if tags[i] == endKeysTag {
					break
				}
			}

			current.keys, _ = v.parseFieldTagsRecursive(string(b[:len(b)-1]), fieldName, "", false)
			continue

		case endKeysTag:
			current.typeof = typeEndKeys

			// if there are more in tags then there was no keysTag defined
			// and an error should be thrown
			if i != len(tags)-1 {
				panic(keysTagNotDefined)
			}
			return

		case omitempty:
			current.typeof = typeOmitEmpty
			continue

		case structOnlyTag:
			current.typeof = typeStructOnly
			continue

		case noStructLevelTag:
			current.typeof = typeNoStructLevel
			continue

		default:
			if t == isdefault {
				current.typeof = typeIsDefault
			}
			// if a pipe character is needed within the param you must use the utf8Pipe representation "0x7C"
			orVals := strings.Split(t, orSeparator)

			for j := 0; j < len(orVals); j++ {
				vals := strings.SplitN(orVals[j], tagKeySeparator, 2)
				if noAlias {
					alias = vals[0]
					current.aliasTag = alias
				} else {
					current.actualAliasTag = t
				}

				if j > 0 {
					current.next = &cTag{aliasTag: alias, actualAliasTag: current.actualAliasTag, hasAlias: hasAlias, hasTag: true}
					current = current.next
				}
				current.hasParam = len(vals) > 1

				current.tag = vals[0]
				if len(current.tag) == 0 {
					panic(strings.TrimSpace(fmt.Sprintf(invalidValidation, fieldName)))
				}

				if wrapper, ok := v.validations[current.tag]; ok {
					current.fn = wrapper.fn
					current.runValidationWhenNil = wrapper.runValidatinOnNil
				} else {
					panic(strings.TrimSpace(fmt.Sprintf(undefinedValidation, current.tag, fieldName)))
				}

				if len(orVals) > 1 {
					current.typeof = typeOr
				}

				if len(vals) > 1 {
					current.param = strings.Replace(strings.Replace(vals[1], utf8HexComma, ",", -1), utf8Pipe, "|", -1)
				}
			}
			current.isBlockEnd = true
		}
	}
	return
}

func (v *Validate) fetchCacheTag(tag string) *cTag {
	// find cached tag
	ctag, found := v.tagCache.Get(tag)
	if !found {
		v.tagCache.lock.Lock()
		defer v.tagCache.lock.Unlock()

		// could have been multiple trying to access, but once first is done this ensures tag
		// isn't parsed again.
		ctag, found = v.tagCache.Get(tag)
		if !found {
			ctag, _ = v.parseFieldTagsRecursive(tag, "", "", false)
			v.tagCache.Set(tag, ctag)
		}
	}
	return ctag
}

var iso3166_1_alpha2 = map[string]bool{
	// see: https://www.iso.org/iso-3166-country-codes.html
	"AF": true, "AX": true, "AL": true, "DZ": true, "AS": true,
	"AD": true, "AO": true, "AI": true, "AQ": true, "AG": true,
	"AR": true, "AM": true, "AW": true, "AU": true, "AT": true,
	"AZ": true, "BS": true, "BH": true, "BD": true, "BB": true,
	"BY": true, "BE": true, "BZ": true, "BJ": true, "BM": true,
	"BT": true, "BO": true, "BQ": true, "BA": true, "BW": true,
	"BV": true, "BR": true, "IO": true, "BN": true, "BG": true,
	"BF": true, "BI": true, "KH": true, "CM": true, "CA": true,
	"CV": true, "KY": true, "CF": true, "TD": true, "CL": true,
	"CN": true, "CX": true, "CC": true, "CO": true, "KM": true,
	"CG": true, "CD": true, "CK": true, "CR": true, "CI": true,
	"HR": true, "CU": true, "CW": true, "CY": true, "CZ": true,
	"DK": true, "DJ": true, "DM": true, "DO": true, "EC": true,
	"EG": true, "SV": true, "GQ": true, "ER": true, "EE": true,
	"ET": true, "FK": true, "FO": true, "FJ": true, "FI": true,
	"FR": true, "GF": true, "PF": true, "TF": true, "GA": true,
	"GM": true, "GE": true, "DE": true, "GH": true, "GI": true,
	"GR": true, "GL": true, "GD": true, "GP": true, "GU": true,
	"GT": true, "GG": true, "GN": true, "GW": true, "GY": true,
	"HT": true, "HM": true, "VA": true, "HN": true, "HK": true,
	"HU": true, "IS": true, "IN": true, "ID": true, "IR": true,
	"IQ": true, "IE": true, "IM": true, "IL": true, "IT": true,
	"JM": true, "JP": true, "JE": true, "JO": true, "KZ": true,
	"KE": true, "KI": true, "KP": true, "KR": true, "KW": true,
	"KG": true, "LA": true, "LV": true, "LB": true, "LS": true,
	"LR": true, "LY": true, "LI": true, "LT": true, "LU": true,
	"MO": true, "MK": true, "MG": true, "MW": true, "MY": true,
	"MV": true, "ML": true, "MT": true, "MH": true, "MQ": true,
	"MR": true, "MU": true, "YT": true, "MX": true, "FM": true,
	"MD": true, "MC": true, "MN": true, "ME": true, "MS": true,
	"MA": true, "MZ": true, "MM": true, "NA": true, "NR": true,
	"NP": true, "NL": true, "NC": true, "NZ": true, "NI": true,
	"NE": true, "NG": true, "NU": true, "NF": true, "MP": true,
	"NO": true, "OM": true, "PK": true, "PW": true, "PS": true,
	"PA": true, "PG": true, "PY": true, "PE": true, "PH": true,
	"PN": true, "PL": true, "PT": true, "PR": true, "QA": true,
	"RE": true, "RO": true, "RU": true, "RW": true, "BL": true,
	"SH": true, "KN": true, "LC": true, "MF": true, "PM": true,
	"VC": true, "WS": true, "SM": true, "ST": true, "SA": true,
	"SN": true, "RS": true, "SC": true, "SL": true, "SG": true,
	"SX": true, "SK": true, "SI": true, "SB": true, "SO": true,
	"ZA": true, "GS": true, "SS": true, "ES": true, "LK": true,
	"SD": true, "SR": true, "SJ": true, "SZ": true, "SE": true,
	"CH": true, "SY": true, "TW": true, "TJ": true, "TZ": true,
	"TH": true, "TL": true, "TG": true, "TK": true, "TO": true,
	"TT": true, "TN": true, "TR": true, "TM": true, "TC": true,
	"TV": true, "UG": true, "UA": true, "AE": true, "GB": true,
	"US": true, "UM": true, "UY": true, "UZ": true, "VU": true,
	"VE": true, "VN": true, "VG": true, "VI": true, "WF": true,
	"EH": true, "YE": true, "ZM": true, "ZW": true,
}

var iso3166_1_alpha3 = map[string]bool{
	// see: https://www.iso.org/iso-3166-country-codes.html
	"AFG": true, "ALB": true, "DZA": true, "ASM": true, "AND": true,
	"AGO": true, "AIA": true, "ATA": true, "ATG": true, "ARG": true,
	"ARM": true, "ABW": true, "AUS": true, "AUT": true, "AZE": true,
	"BHS": true, "BHR": true, "BGD": true, "BRB": true, "BLR": true,
	"BEL": true, "BLZ": true, "BEN": true, "BMU": true, "BTN": true,
	"BOL": true, "BES": true, "BIH": true, "BWA": true, "BVT": true,
	"BRA": true, "IOT": true, "BRN": true, "BGR": true, "BFA": true,
	"BDI": true, "CPV": true, "KHM": true, "CMR": true, "CAN": true,
	"CYM": true, "CAF": true, "TCD": true, "CHL": true, "CHN": true,
	"CXR": true, "CCK": true, "COL": true, "COM": true, "COD": true,
	"COG": true, "COK": true, "CRI": true, "HRV": true, "CUB": true,
	"CUW": true, "CYP": true, "CZE": true, "CIV": true, "DNK": true,
	"DJI": true, "DMA": true, "DOM": true, "ECU": true, "EGY": true,
	"SLV": true, "GNQ": true, "ERI": true, "EST": true, "SWZ": true,
	"ETH": true, "FLK": true, "FRO": true, "FJI": true, "FIN": true,
	"FRA": true, "GUF": true, "PYF": true, "ATF": true, "GAB": true,
	"GMB": true, "GEO": true, "DEU": true, "GHA": true, "GIB": true,
	"GRC": true, "GRL": true, "GRD": true, "GLP": true, "GUM": true,
	"GTM": true, "GGY": true, "GIN": true, "GNB": true, "GUY": true,
	"HTI": true, "HMD": true, "VAT": true, "HND": true, "HKG": true,
	"HUN": true, "ISL": true, "IND": true, "IDN": true, "IRN": true,
	"IRQ": true, "IRL": true, "IMN": true, "ISR": true, "ITA": true,
	"JAM": true, "JPN": true, "JEY": true, "JOR": true, "KAZ": true,
	"KEN": true, "KIR": true, "PRK": true, "KOR": true, "KWT": true,
	"KGZ": true, "LAO": true, "LVA": true, "LBN": true, "LSO": true,
	"LBR": true, "LBY": true, "LIE": true, "LTU": true, "LUX": true,
	"MAC": true, "MDG": true, "MWI": true, "MYS": true, "MDV": true,
	"MLI": true, "MLT": true, "MHL": true, "MTQ": true, "MRT": true,
	"MUS": true, "MYT": true, "MEX": true, "FSM": true, "MDA": true,
	"MCO": true, "MNG": true, "MNE": true, "MSR": true, "MAR": true,
	"MOZ": true, "MMR": true, "NAM": true, "NRU": true, "NPL": true,
	"NLD": true, "NCL": true, "NZL": true, "NIC": true, "NER": true,
	"NGA": true, "NIU": true, "NFK": true, "MKD": true, "MNP": true,
	"NOR": true, "OMN": true, "PAK": true, "PLW": true, "PSE": true,
	"PAN": true, "PNG": true, "PRY": true, "PER": true, "PHL": true,
	"PCN": true, "POL": true, "PRT": true, "PRI": true, "QAT": true,
	"ROU": true, "RUS": true, "RWA": true, "REU": true, "BLM": true,
	"SHN": true, "KNA": true, "LCA": true, "MAF": true, "SPM": true,
	"VCT": true, "WSM": true, "SMR": true, "STP": true, "SAU": true,
	"SEN": true, "SRB": true, "SYC": true, "SLE": true, "SGP": true,
	"SXM": true, "SVK": true, "SVN": true, "SLB": true, "SOM": true,
	"ZAF": true, "SGS": true, "SSD": true, "ESP": true, "LKA": true,
	"SDN": true, "SUR": true, "SJM": true, "SWE": true, "CHE": true,
	"SYR": true, "TWN": true, "TJK": true, "TZA": true, "THA": true,
	"TLS": true, "TGO": true, "TKL": true, "TON": true, "TTO": true,
	"TUN": true, "TUR": true, "TKM": true, "TCA": true, "TUV": true,
	"UGA": true, "UKR": true, "ARE": true, "GBR": true, "UMI": true,
	"USA": true, "URY": true, "UZB": true, "VUT": true, "VEN": true,
	"VNM": true, "VGB": true, "VIR": true, "WLF": true, "ESH": true,
	"YEM": true, "ZMB": true, "ZWE": true, "ALA": true,
}

var iso3166_1_alpha_numeric = map[int]bool{
	// see: https://www.iso.org/iso-3166-country-codes.html
	4: true, 8: true, 12: true, 16: true, 20: true,
	24: true, 660: true, 10: true, 28: true, 32: true,
	51: true, 533: true, 36: true, 40: true, 31: true,
	44: true, 48: true, 50: true, 52: true, 112: true,
	56: true, 84: true, 204: true, 60: true, 64: true,
	68: true, 535: true, 70: true, 72: true, 74: true,
	76: true, 86: true, 96: true, 100: true, 854: true,
	108: true, 132: true, 116: true, 120: true, 124: true,
	136: true, 140: true, 148: true, 152: true, 156: true,
	162: true, 166: true, 170: true, 174: true, 180: true,
	178: true, 184: true, 188: true, 191: true, 192: true,
	531: true, 196: true, 203: true, 384: true, 208: true,
	262: true, 212: true, 214: true, 218: true, 818: true,
	222: true, 226: true, 232: true, 233: true, 748: true,
	231: true, 238: true, 234: true, 242: true, 246: true,
	250: true, 254: true, 258: true, 260: true, 266: true,
	270: true, 268: true, 276: true, 288: true, 292: true,
	300: true, 304: true, 308: true, 312: true, 316: true,
	320: true, 831: true, 324: true, 624: true, 328: true,
	332: true, 334: true, 336: true, 340: true, 344: true,
	348: true, 352: true, 356: true, 360: true, 364: true,
	368: true, 372: true, 833: true, 376: true, 380: true,
	388: true, 392: true, 832: true, 400: true, 398: true,
	404: true, 296: true, 408: true, 410: true, 414: true,
	417: true, 418: true, 428: true, 422: true, 426: true,
	430: true, 434: true, 438: true, 440: true, 442: true,
	446: true, 450: true, 454: true, 458: true, 462: true,
	466: true, 470: true, 584: true, 474: true, 478: true,
	480: true, 175: true, 484: true, 583: true, 498: true,
	492: true, 496: true, 499: true, 500: true, 504: true,
	508: true, 104: true, 516: true, 520: true, 524: true,
	528: true, 540: true, 554: true, 558: true, 562: true,
	566: true, 570: true, 574: true, 807: true, 580: true,
	578: true, 512: true, 586: true, 585: true, 275: true,
	591: true, 598: true, 600: true, 604: true, 608: true,
	612: true, 616: true, 620: true, 630: true, 634: true,
	642: true, 643: true, 646: true, 638: true, 652: true,
	654: true, 659: true, 662: true, 663: true, 666: true,
	670: true, 882: true, 674: true, 678: true, 682: true,
	686: true, 688: true, 690: true, 694: true, 702: true,
	534: true, 703: true, 705: true, 90: true, 706: true,
	710: true, 239: true, 728: true, 724: true, 144: true,
	729: true, 740: true, 744: true, 752: true, 756: true,
	760: true, 158: true, 762: true, 834: true, 764: true,
	626: true, 768: true, 772: true, 776: true, 780: true,
	788: true, 792: true, 795: true, 796: true, 798: true,
	800: true, 804: true, 784: true, 826: true, 581: true,
	840: true, 858: true, 860: true, 548: true, 862: true,
	704: true, 92: true, 850: true, 876: true, 732: true,
	887: true, 894: true, 716: true, 248: true,
}

var iso3166_2 = map[string]bool{
	"AD-02": true, "AD-03": true, "AD-04": true, "AD-05": true, "AD-06": true,
	"AD-07": true, "AD-08": true, "AE-AJ": true, "AE-AZ": true, "AE-DU": true,
	"AE-FU": true, "AE-RK": true, "AE-SH": true, "AE-UQ": true, "AF-BAL": true,
	"AF-BAM": true, "AF-BDG": true, "AF-BDS": true, "AF-BGL": true, "AF-DAY": true,
	"AF-FRA": true, "AF-FYB": true, "AF-GHA": true, "AF-GHO": true, "AF-HEL": true,
	"AF-HER": true, "AF-JOW": true, "AF-KAB": true, "AF-KAN": true, "AF-KAP": true,
	"AF-KDZ": true, "AF-KHO": true, "AF-KNR": true, "AF-LAG": true, "AF-LOG": true,
	"AF-NAN": true, "AF-NIM": true, "AF-NUR": true, "AF-PAN": true, "AF-PAR": true,
	"AF-PIA": true, "AF-PKA": true, "AF-SAM": true, "AF-SAR": true, "AF-TAK": true,
	"AF-URU": true, "AF-WAR": true, "AF-ZAB": true, "AG-03": true, "AG-04": true,
	"AG-05": true, "AG-06": true, "AG-07": true, "AG-08": true, "AG-10": true,
	"AG-11": true, "AL-01": true, "AL-02": true, "AL-03": true, "AL-04": true,
	"AL-05": true, "AL-06": true, "AL-07": true, "AL-08": true, "AL-09": true,
	"AL-10": true, "AL-11": true, "AL-12": true, "AL-BR": true, "AL-BU": true,
	"AL-DI": true, "AL-DL": true, "AL-DR": true, "AL-DV": true, "AL-EL": true,
	"AL-ER": true, "AL-FR": true, "AL-GJ": true, "AL-GR": true, "AL-HA": true,
	"AL-KA": true, "AL-KB": true, "AL-KC": true, "AL-KO": true, "AL-KR": true,
	"AL-KU": true, "AL-LB": true, "AL-LE": true, "AL-LU": true, "AL-MK": true,
	"AL-MM": true, "AL-MR": true, "AL-MT": true, "AL-PG": true, "AL-PQ": true,
	"AL-PR": true, "AL-PU": true, "AL-SH": true, "AL-SK": true, "AL-SR": true,
	"AL-TE": true, "AL-TP": true, "AL-TR": true, "AL-VL": true, "AM-AG": true,
	"AM-AR": true, "AM-AV": true, "AM-ER": true, "AM-GR": true, "AM-KT": true,
	"AM-LO": true, "AM-SH": true, "AM-SU": true, "AM-TV": true, "AM-VD": true,
	"AO-BGO": true, "AO-BGU": true, "AO-BIE": true, "AO-CAB": true, "AO-CCU": true,
	"AO-CNN": true, "AO-CNO": true, "AO-CUS": true, "AO-HUA": true, "AO-HUI": true,
	"AO-LNO": true, "AO-LSU": true, "AO-LUA": true, "AO-MAL": true, "AO-MOX": true,
	"AO-NAM": true, "AO-UIG": true, "AO-ZAI": true, "AR-A": true, "AR-B": true,
	"AR-C": true, "AR-D": true, "AR-E": true, "AR-G": true, "AR-H": true,
	"AR-J": true, "AR-K": true, "AR-L": true, "AR-M": true, "AR-N": true,
	"AR-P": true, "AR-Q": true, "AR-R": true, "AR-S": true, "AR-T": true,
	"AR-U": true, "AR-V": true, "AR-W": true, "AR-X": true, "AR-Y": true,
	"AR-Z": true, "AT-1": true, "AT-2": true, "AT-3": true, "AT-4": true,
	"AT-5": true, "AT-6": true, "AT-7": true, "AT-8": true, "AT-9": true,
	"AU-ACT": true, "AU-NSW": true, "AU-NT": true, "AU-QLD": true, "AU-SA": true,
	"AU-TAS": true, "AU-VIC": true, "AU-WA": true, "AZ-ABS": true, "AZ-AGA": true,
	"AZ-AGC": true, "AZ-AGM": true, "AZ-AGS": true, "AZ-AGU": true, "AZ-AST": true,
	"AZ-BA": true, "AZ-BAB": true, "AZ-BAL": true, "AZ-BAR": true, "AZ-BEY": true,
	"AZ-BIL": true, "AZ-CAB": true, "AZ-CAL": true, "AZ-CUL": true, "AZ-DAS": true,
	"AZ-FUZ": true, "AZ-GA": true, "AZ-GAD": true, "AZ-GOR": true, "AZ-GOY": true,
	"AZ-GYG": true, "AZ-HAC": true, "AZ-IMI": true, "AZ-ISM": true, "AZ-KAL": true,
	"AZ-KAN": true, "AZ-KUR": true, "AZ-LA": true, "AZ-LAC": true, "AZ-LAN": true,
	"AZ-LER": true, "AZ-MAS": true, "AZ-MI": true, "AZ-NA": true, "AZ-NEF": true,
	"AZ-NV": true, "AZ-NX": true, "AZ-OGU": true, "AZ-ORD": true, "AZ-QAB": true,
	"AZ-QAX": true, "AZ-QAZ": true, "AZ-QBA": true, "AZ-QBI": true, "AZ-QOB": true,
	"AZ-QUS": true, "AZ-SA": true, "AZ-SAB": true, "AZ-SAD": true, "AZ-SAH": true,
	"AZ-SAK": true, "AZ-SAL": true, "AZ-SAR": true, "AZ-SAT": true, "AZ-SBN": true,
	"AZ-SIY": true, "AZ-SKR": true, "AZ-SM": true, "AZ-SMI": true, "AZ-SMX": true,
	"AZ-SR": true, "AZ-SUS": true, "AZ-TAR": true, "AZ-TOV": true, "AZ-UCA": true,
	"AZ-XA": true, "AZ-XAC": true, "AZ-XCI": true, "AZ-XIZ": true, "AZ-XVD": true,
	"AZ-YAR": true, "AZ-YE": true, "AZ-YEV": true, "AZ-ZAN": true, "AZ-ZAQ": true,
	"AZ-ZAR": true, "BA-01": true, "BA-02": true, "BA-03": true, "BA-04": true,
	"BA-05": true, "BA-06": true, "BA-07": true, "BA-08": true, "BA-09": true,
	"BA-10": true, "BA-BIH": true, "BA-BRC": true, "BA-SRP": true, "BB-01": true,
	"BB-02": true, "BB-03": true, "BB-04": true, "BB-05": true, "BB-06": true,
	"BB-07": true, "BB-08": true, "BB-09": true, "BB-10": true, "BB-11": true,
	"BD-01": true, "BD-02": true, "BD-03": true, "BD-04": true, "BD-05": true,
	"BD-06": true, "BD-07": true, "BD-08": true, "BD-09": true, "BD-10": true,
	"BD-11": true, "BD-12": true, "BD-13": true, "BD-14": true, "BD-15": true,
	"BD-16": true, "BD-17": true, "BD-18": true, "BD-19": true, "BD-20": true,
	"BD-21": true, "BD-22": true, "BD-23": true, "BD-24": true, "BD-25": true,
	"BD-26": true, "BD-27": true, "BD-28": true, "BD-29": true, "BD-30": true,
	"BD-31": true, "BD-32": true, "BD-33": true, "BD-34": true, "BD-35": true,
	"BD-36": true, "BD-37": true, "BD-38": true, "BD-39": true, "BD-40": true,
	"BD-41": true, "BD-42": true, "BD-43": true, "BD-44": true, "BD-45": true,
	"BD-46": true, "BD-47": true, "BD-48": true, "BD-49": true, "BD-50": true,
	"BD-51": true, "BD-52": true, "BD-53": true, "BD-54": true, "BD-55": true,
	"BD-56": true, "BD-57": true, "BD-58": true, "BD-59": true, "BD-60": true,
	"BD-61": true, "BD-62": true, "BD-63": true, "BD-64": true, "BD-A": true,
	"BD-B": true, "BD-C": true, "BD-D": true, "BD-E": true, "BD-F": true,
	"BD-G": true, "BE-BRU": true, "BE-VAN": true, "BE-VBR": true, "BE-VLG": true,
	"BE-VLI": true, "BE-VOV": true, "BE-VWV": true, "BE-WAL": true, "BE-WBR": true,
	"BE-WHT": true, "BE-WLG": true, "BE-WLX": true, "BE-WNA": true, "BF-01": true,
	"BF-02": true, "BF-03": true, "BF-04": true, "BF-05": true, "BF-06": true,
	"BF-07": true, "BF-08": true, "BF-09": true, "BF-10": true, "BF-11": true,
	"BF-12": true, "BF-13": true, "BF-BAL": true, "BF-BAM": true, "BF-BAN": true,
	"BF-BAZ": true, "BF-BGR": true, "BF-BLG": true, "BF-BLK": true, "BF-COM": true,
	"BF-GAN": true, "BF-GNA": true, "BF-GOU": true, "BF-HOU": true, "BF-IOB": true,
	"BF-KAD": true, "BF-KEN": true, "BF-KMD": true, "BF-KMP": true, "BF-KOP": true,
	"BF-KOS": true, "BF-KOT": true, "BF-KOW": true, "BF-LER": true, "BF-LOR": true,
	"BF-MOU": true, "BF-NAM": true, "BF-NAO": true, "BF-NAY": true, "BF-NOU": true,
	"BF-OUB": true, "BF-OUD": true, "BF-PAS": true, "BF-PON": true, "BF-SEN": true,
	"BF-SIS": true, "BF-SMT": true, "BF-SNG": true, "BF-SOM": true, "BF-SOR": true,
	"BF-TAP": true, "BF-TUI": true, "BF-YAG": true, "BF-YAT": true, "BF-ZIR": true,
	"BF-ZON": true, "BF-ZOU": true, "BG-01": true, "BG-02": true, "BG-03": true,
	"BG-04": true, "BG-05": true, "BG-06": true, "BG-07": true, "BG-08": true,
	"BG-09": true, "BG-10": true, "BG-11": true, "BG-12": true, "BG-13": true,
	"BG-14": true, "BG-15": true, "BG-16": true, "BG-17": true, "BG-18": true,
	"BG-19": true, "BG-20": true, "BG-21": true, "BG-22": true, "BG-23": true,
	"BG-24": true, "BG-25": true, "BG-26": true, "BG-27": true, "BG-28": true,
	"BH-13": true, "BH-14": true, "BH-15": true, "BH-16": true, "BH-17": true,
	"BI-BB": true, "BI-BL": true, "BI-BM": true, "BI-BR": true, "BI-CA": true,
	"BI-CI": true, "BI-GI": true, "BI-KI": true, "BI-KR": true, "BI-KY": true,
	"BI-MA": true, "BI-MU": true, "BI-MW": true, "BI-NG": true, "BI-RT": true,
	"BI-RY": true, "BJ-AK": true, "BJ-AL": true, "BJ-AQ": true, "BJ-BO": true,
	"BJ-CO": true, "BJ-DO": true, "BJ-KO": true, "BJ-LI": true, "BJ-MO": true,
	"BJ-OU": true, "BJ-PL": true, "BJ-ZO": true, "BN-BE": true, "BN-BM": true,
	"BN-TE": true, "BN-TU": true, "BO-B": true, "BO-C": true, "BO-H": true,
	"BO-L": true, "BO-N": true, "BO-O": true, "BO-P": true, "BO-S": true,
	"BO-T": true, "BQ-BO": true, "BQ-SA": true, "BQ-SE": true, "BR-AC": true,
	"BR-AL": true, "BR-AM": true, "BR-AP": true, "BR-BA": true, "BR-CE": true,
	"BR-DF": true, "BR-ES": true, "BR-FN": true, "BR-GO": true, "BR-MA": true,
	"BR-MG": true, "BR-MS": true, "BR-MT": true, "BR-PA": true, "BR-PB": true,
	"BR-PE": true, "BR-PI": true, "BR-PR": true, "BR-RJ": true, "BR-RN": true,
	"BR-RO": true, "BR-RR": true, "BR-RS": true, "BR-SC": true, "BR-SE": true,
	"BR-SP": true, "BR-TO": true, "BS-AK": true, "BS-BI": true, "BS-BP": true,
	"BS-BY": true, "BS-CE": true, "BS-CI": true, "BS-CK": true, "BS-CO": true,
	"BS-CS": true, "BS-EG": true, "BS-EX": true, "BS-FP": true, "BS-GC": true,
	"BS-HI": true, "BS-HT": true, "BS-IN": true, "BS-LI": true, "BS-MC": true,
	"BS-MG": true, "BS-MI": true, "BS-NE": true, "BS-NO": true, "BS-NS": true,
	"BS-RC": true, "BS-RI": true, "BS-SA": true, "BS-SE": true, "BS-SO": true,
	"BS-SS": true, "BS-SW": true, "BS-WG": true, "BT-11": true, "BT-12": true,
	"BT-13": true, "BT-14": true, "BT-15": true, "BT-21": true, "BT-22": true,
	"BT-23": true, "BT-24": true, "BT-31": true, "BT-32": true, "BT-33": true,
	"BT-34": true, "BT-41": true, "BT-42": true, "BT-43": true, "BT-44": true,
	"BT-45": true, "BT-GA": true, "BT-TY": true, "BW-CE": true, "BW-GH": true,
	"BW-KG": true, "BW-KL": true, "BW-KW": true, "BW-NE": true, "BW-NW": true,
	"BW-SE": true, "BW-SO": true, "BY-BR": true, "BY-HM": true, "BY-HO": true,
	"BY-HR": true, "BY-MA": true, "BY-MI": true, "BY-VI": true, "BZ-BZ": true,
	"BZ-CY": true, "BZ-CZL": true, "BZ-OW": true, "BZ-SC": true, "BZ-TOL": true,
	"CA-AB": true, "CA-BC": true, "CA-MB": true, "CA-NB": true, "CA-NL": true,
	"CA-NS": true, "CA-NT": true, "CA-NU": true, "CA-ON": true, "CA-PE": true,
	"CA-QC": true, "CA-SK": true, "CA-YT": true, "CD-BC": true, "CD-BN": true,
	"CD-EQ": true, "CD-KA": true, "CD-KE": true, "CD-KN": true, "CD-KW": true,
	"CD-MA": true, "CD-NK": true, "CD-OR": true, "CD-SK": true, "CF-AC": true,
	"CF-BB": true, "CF-BGF": true, "CF-BK": true, "CF-HK": true, "CF-HM": true,
	"CF-HS": true, "CF-KB": true, "CF-KG": true, "CF-LB": true, "CF-MB": true,
	"CF-MP": true, "CF-NM": true, "CF-OP": true, "CF-SE": true, "CF-UK": true,
	"CF-VK": true, "CG-11": true, "CG-12": true, "CG-13": true, "CG-14": true,
	"CG-15": true, "CG-2": true, "CG-5": true, "CG-7": true, "CG-8": true,
	"CG-9": true, "CG-BZV": true, "CH-AG": true, "CH-AI": true, "CH-AR": true,
	"CH-BE": true, "CH-BL": true, "CH-BS": true, "CH-FR": true, "CH-GE": true,
	"CH-GL": true, "CH-GR": true, "CH-JU": true, "CH-LU": true, "CH-NE": true,
	"CH-NW": true, "CH-OW": true, "CH-SG": true, "CH-SH": true, "CH-SO": true,
	"CH-SZ": true, "CH-TG": true, "CH-TI": true, "CH-UR": true, "CH-VD": true,
	"CH-VS": true, "CH-ZG": true, "CH-ZH": true, "CI-01": true, "CI-02": true,
	"CI-03": true, "CI-04": true, "CI-05": true, "CI-06": true, "CI-07": true,
	"CI-08": true, "CI-09": true, "CI-10": true, "CI-11": true, "CI-12": true,
	"CI-13": true, "CI-14": true, "CI-15": true, "CI-16": true, "CI-17": true,
	"CI-18": true, "CI-19": true, "CL-AI": true, "CL-AN": true, "CL-AP": true,
	"CL-AR": true, "CL-AT": true, "CL-BI": true, "CL-CO": true, "CL-LI": true,
	"CL-LL": true, "CL-LR": true, "CL-MA": true, "CL-ML": true, "CL-RM": true,
	"CL-TA": true, "CL-VS": true, "CM-AD": true, "CM-CE": true, "CM-EN": true,
	"CM-ES": true, "CM-LT": true, "CM-NO": true, "CM-NW": true, "CM-OU": true,
	"CM-SU": true, "CM-SW": true, "CN-11": true, "CN-12": true, "CN-13": true,
	"CN-14": true, "CN-15": true, "CN-21": true, "CN-22": true, "CN-23": true,
	"CN-31": true, "CN-32": true, "CN-33": true, "CN-34": true, "CN-35": true,
	"CN-36": true, "CN-37": true, "CN-41": true, "CN-42": true, "CN-43": true,
	"CN-44": true, "CN-45": true, "CN-46": true, "CN-50": true, "CN-51": true,
	"CN-52": true, "CN-53": true, "CN-54": true, "CN-61": true, "CN-62": true,
	"CN-63": true, "CN-64": true, "CN-65": true, "CN-71": true, "CN-91": true,
	"CN-92": true, "CO-AMA": true, "CO-ANT": true, "CO-ARA": true, "CO-ATL": true,
	"CO-BOL": true, "CO-BOY": true, "CO-CAL": true, "CO-CAQ": true, "CO-CAS": true,
	"CO-CAU": true, "CO-CES": true, "CO-CHO": true, "CO-COR": true, "CO-CUN": true,
	"CO-DC": true, "CO-GUA": true, "CO-GUV": true, "CO-HUI": true, "CO-LAG": true,
	"CO-MAG": true, "CO-MET": true, "CO-NAR": true, "CO-NSA": true, "CO-PUT": true,
	"CO-QUI": true, "CO-RIS": true, "CO-SAN": true, "CO-SAP": true, "CO-SUC": true,
	"CO-TOL": true, "CO-VAC": true, "CO-VAU": true, "CO-VID": true, "CR-A": true,
	"CR-C": true, "CR-G": true, "CR-H": true, "CR-L": true, "CR-P": true,
	"CR-SJ": true, "CU-01": true, "CU-02": true, "CU-03": true, "CU-04": true,
	"CU-05": true, "CU-06": true, "CU-07": true, "CU-08": true, "CU-09": true,
	"CU-10": true, "CU-11": true, "CU-12": true, "CU-13": true, "CU-14": true,
	"CU-99": true, "CV-B": true, "CV-BR": true, "CV-BV": true, "CV-CA": true,
	"CV-CF": true, "CV-CR": true, "CV-MA": true, "CV-MO": true, "CV-PA": true,
	"CV-PN": true, "CV-PR": true, "CV-RB": true, "CV-RG": true, "CV-RS": true,
	"CV-S": true, "CV-SD": true, "CV-SF": true, "CV-SL": true, "CV-SM": true,
	"CV-SO": true, "CV-SS": true, "CV-SV": true, "CV-TA": true, "CV-TS": true,
	"CY-01": true, "CY-02": true, "CY-03": true, "CY-04": true, "CY-05": true,
	"CY-06": true, "CZ-10": true, "CZ-101": true, "CZ-102": true, "CZ-103": true,
	"CZ-104": true, "CZ-105": true, "CZ-106": true, "CZ-107": true, "CZ-108": true,
	"CZ-109": true, "CZ-110": true, "CZ-111": true, "CZ-112": true, "CZ-113": true,
	"CZ-114": true, "CZ-115": true, "CZ-116": true, "CZ-117": true, "CZ-118": true,
	"CZ-119": true, "CZ-120": true, "CZ-121": true, "CZ-122": true, "CZ-20": true,
	"CZ-201": true, "CZ-202": true, "CZ-203": true, "CZ-204": true, "CZ-205": true,
	"CZ-206": true, "CZ-207": true, "CZ-208": true, "CZ-209": true, "CZ-20A": true,
	"CZ-20B": true, "CZ-20C": true, "CZ-31": true, "CZ-311": true, "CZ-312": true,
	"CZ-313": true, "CZ-314": true, "CZ-315": true, "CZ-316": true, "CZ-317": true,
	"CZ-32": true, "CZ-321": true, "CZ-322": true, "CZ-323": true, "CZ-324": true,
	"CZ-325": true, "CZ-326": true, "CZ-327": true, "CZ-41": true, "CZ-411": true,
	"CZ-412": true, "CZ-413": true, "CZ-42": true, "CZ-421": true, "CZ-422": true,
	"CZ-423": true, "CZ-424": true, "CZ-425": true, "CZ-426": true, "CZ-427": true,
	"CZ-51": true, "CZ-511": true, "CZ-512": true, "CZ-513": true, "CZ-514": true,
	"CZ-52": true, "CZ-521": true, "CZ-522": true, "CZ-523": true, "CZ-524": true,
	"CZ-525": true, "CZ-53": true, "CZ-531": true, "CZ-532": true, "CZ-533": true,
	"CZ-534": true, "CZ-63": true, "CZ-631": true, "CZ-632": true, "CZ-633": true,
	"CZ-634": true, "CZ-635": true, "CZ-64": true, "CZ-641": true, "CZ-642": true,
	"CZ-643": true, "CZ-644": true, "CZ-645": true, "CZ-646": true, "CZ-647": true,
	"CZ-71": true, "CZ-711": true, "CZ-712": true, "CZ-713": true, "CZ-714": true,
	"CZ-715": true, "CZ-72": true, "CZ-721": true, "CZ-722": true, "CZ-723": true,
	"CZ-724": true, "CZ-80": true, "CZ-801": true, "CZ-802": true, "CZ-803": true,
	"CZ-804": true, "CZ-805": true, "CZ-806": true, "DE-BB": true, "DE-BE": true,
	"DE-BW": true, "DE-BY": true, "DE-HB": true, "DE-HE": true, "DE-HH": true,
	"DE-MV": true, "DE-NI": true, "DE-NW": true, "DE-RP": true, "DE-SH": true,
	"DE-SL": true, "DE-SN": true, "DE-ST": true, "DE-TH": true, "DJ-AR": true,
	"DJ-AS": true, "DJ-DI": true, "DJ-DJ": true, "DJ-OB": true, "DJ-TA": true,
	"DK-81": true, "DK-82": true, "DK-83": true, "DK-84": true, "DK-85": true,
	"DM-01": true, "DM-02": true, "DM-03": true, "DM-04": true, "DM-05": true,
	"DM-06": true, "DM-07": true, "DM-08": true, "DM-09": true, "DM-10": true,
	"DO-01": true, "DO-02": true, "DO-03": true, "DO-04": true, "DO-05": true,
	"DO-06": true, "DO-07": true, "DO-08": true, "DO-09": true, "DO-10": true,
	"DO-11": true, "DO-12": true, "DO-13": true, "DO-14": true, "DO-15": true,
	"DO-16": true, "DO-17": true, "DO-18": true, "DO-19": true, "DO-20": true,
	"DO-21": true, "DO-22": true, "DO-23": true, "DO-24": true, "DO-25": true,
	"DO-26": true, "DO-27": true, "DO-28": true, "DO-29": true, "DO-30": true,
	"DZ-01": true, "DZ-02": true, "DZ-03": true, "DZ-04": true, "DZ-05": true,
	"DZ-06": true, "DZ-07": true, "DZ-08": true, "DZ-09": true, "DZ-10": true,
	"DZ-11": true, "DZ-12": true, "DZ-13": true, "DZ-14": true, "DZ-15": true,
	"DZ-16": true, "DZ-17": true, "DZ-18": true, "DZ-19": true, "DZ-20": true,
	"DZ-21": true, "DZ-22": true, "DZ-23": true, "DZ-24": true, "DZ-25": true,
	"DZ-26": true, "DZ-27": true, "DZ-28": true, "DZ-29": true, "DZ-30": true,
	"DZ-31": true, "DZ-32": true, "DZ-33": true, "DZ-34": true, "DZ-35": true,
	"DZ-36": true, "DZ-37": true, "DZ-38": true, "DZ-39": true, "DZ-40": true,
	"DZ-41": true, "DZ-42": true, "DZ-43": true, "DZ-44": true, "DZ-45": true,
	"DZ-46": true, "DZ-47": true, "DZ-48": true, "EC-A": true, "EC-B": true,
	"EC-C": true, "EC-D": true, "EC-E": true, "EC-F": true, "EC-G": true,
	"EC-H": true, "EC-I": true, "EC-L": true, "EC-M": true, "EC-N": true,
	"EC-O": true, "EC-P": true, "EC-R": true, "EC-S": true, "EC-SD": true,
	"EC-SE": true, "EC-T": true, "EC-U": true, "EC-W": true, "EC-X": true,
	"EC-Y": true, "EC-Z": true, "EE-37": true, "EE-39": true, "EE-44": true,
	"EE-49": true, "EE-51": true, "EE-57": true, "EE-59": true, "EE-65": true,
	"EE-67": true, "EE-70": true, "EE-74": true, "EE-78": true, "EE-82": true,
	"EE-84": true, "EE-86": true, "EG-ALX": true, "EG-ASN": true, "EG-AST": true,
	"EG-BA": true, "EG-BH": true, "EG-BNS": true, "EG-C": true, "EG-DK": true,
	"EG-DT": true, "EG-FYM": true, "EG-GH": true, "EG-GZ": true, "EG-HU": true,
	"EG-IS": true, "EG-JS": true, "EG-KB": true, "EG-KFS": true, "EG-KN": true,
	"EG-MN": true, "EG-MNF": true, "EG-MT": true, "EG-PTS": true, "EG-SHG": true,
	"EG-SHR": true, "EG-SIN": true, "EG-SU": true, "EG-SUZ": true, "EG-WAD": true,
	"ER-AN": true, "ER-DK": true, "ER-DU": true, "ER-GB": true, "ER-MA": true,
	"ER-SK": true, "ES-A": true, "ES-AB": true, "ES-AL": true, "ES-AN": true,
	"ES-AR": true, "ES-AS": true, "ES-AV": true, "ES-B": true, "ES-BA": true,
	"ES-BI": true, "ES-BU": true, "ES-C": true, "ES-CA": true, "ES-CB": true,
	"ES-CC": true, "ES-CE": true, "ES-CL": true, "ES-CM": true, "ES-CN": true,
	"ES-CO": true, "ES-CR": true, "ES-CS": true, "ES-CT": true, "ES-CU": true,
	"ES-EX": true, "ES-GA": true, "ES-GC": true, "ES-GI": true, "ES-GR": true,
	"ES-GU": true, "ES-H": true, "ES-HU": true, "ES-IB": true, "ES-J": true,
	"ES-L": true, "ES-LE": true, "ES-LO": true, "ES-LU": true, "ES-M": true,
	"ES-MA": true, "ES-MC": true, "ES-MD": true, "ES-ML": true, "ES-MU": true,
	"ES-NA": true, "ES-NC": true, "ES-O": true, "ES-OR": true, "ES-P": true,
	"ES-PM": true, "ES-PO": true, "ES-PV": true, "ES-RI": true, "ES-S": true,
	"ES-SA": true, "ES-SE": true, "ES-SG": true, "ES-SO": true, "ES-SS": true,
	"ES-T": true, "ES-TE": true, "ES-TF": true, "ES-TO": true, "ES-V": true,
	"ES-VA": true, "ES-VC": true, "ES-VI": true, "ES-Z": true, "ES-ZA": true,
	"ET-AA": true, "ET-AF": true, "ET-AM": true, "ET-BE": true, "ET-DD": true,
	"ET-GA": true, "ET-HA": true, "ET-OR": true, "ET-SN": true, "ET-SO": true,
	"ET-TI": true, "FI-01": true, "FI-02": true, "FI-03": true, "FI-04": true,
	"FI-05": true, "FI-06": true, "FI-07": true, "FI-08": true, "FI-09": true,
	"FI-10": true, "FI-11": true, "FI-12": true, "FI-13": true, "FI-14": true,
	"FI-15": true, "FI-16": true, "FI-17": true, "FI-18": true, "FI-19": true,
	"FJ-C": true, "FJ-E": true, "FJ-N": true, "FJ-R": true, "FJ-W": true,
	"FM-KSA": true, "FM-PNI": true, "FM-TRK": true, "FM-YAP": true, "FR-01": true,
	"FR-02": true, "FR-03": true, "FR-04": true, "FR-05": true, "FR-06": true,
	"FR-07": true, "FR-08": true, "FR-09": true, "FR-10": true, "FR-11": true,
	"FR-12": true, "FR-13": true, "FR-14": true, "FR-15": true, "FR-16": true,
	"FR-17": true, "FR-18": true, "FR-19": true, "FR-21": true, "FR-22": true,
	"FR-23": true, "FR-24": true, "FR-25": true, "FR-26": true, "FR-27": true,
	"FR-28": true, "FR-29": true, "FR-2A": true, "FR-2B": true, "FR-30": true,
	"FR-31": true, "FR-32": true, "FR-33": true, "FR-34": true, "FR-35": true,
	"FR-36": true, "FR-37": true, "FR-38": true, "FR-39": true, "FR-40": true,
	"FR-41": true, "FR-42": true, "FR-43": true, "FR-44": true, "FR-45": true,
	"FR-46": true, "FR-47": true, "FR-48": true, "FR-49": true, "FR-50": true,
	"FR-51": true, "FR-52": true, "FR-53": true, "FR-54": true, "FR-55": true,
	"FR-56": true, "FR-57": true, "FR-58": true, "FR-59": true, "FR-60": true,
	"FR-61": true, "FR-62": true, "FR-63": true, "FR-64": true, "FR-65": true,
	"FR-66": true, "FR-67": true, "FR-68": true, "FR-69": true, "FR-70": true,
	"FR-71": true, "FR-72": true, "FR-73": true, "FR-74": true, "FR-75": true,
	"FR-76": true, "FR-77": true, "FR-78": true, "FR-79": true, "FR-80": true,
	"FR-81": true, "FR-82": true, "FR-83": true, "FR-84": true, "FR-85": true,
	"FR-86": true, "FR-87": true, "FR-88": true, "FR-89": true, "FR-90": true,
	"FR-91": true, "FR-92": true, "FR-93": true, "FR-94": true, "FR-95": true,
	"FR-ARA": true, "FR-BFC": true, "FR-BL": true, "FR-BRE": true, "FR-COR": true,
	"FR-CP": true, "FR-CVL": true, "FR-GES": true, "FR-GF": true, "FR-GP": true,
	"FR-GUA": true, "FR-HDF": true, "FR-IDF": true, "FR-LRE": true, "FR-MAY": true,
	"FR-MF": true, "FR-MQ": true, "FR-NAQ": true, "FR-NC": true, "FR-NOR": true,
	"FR-OCC": true, "FR-PAC": true, "FR-PDL": true, "FR-PF": true, "FR-PM": true,
	"FR-RE": true, "FR-TF": true, "FR-WF": true, "FR-YT": true, "GA-1": true,
	"GA-2": true, "GA-3": true, "GA-4": true, "GA-5": true, "GA-6": true,
	"GA-7": true, "GA-8": true, "GA-9": true, "GB-ABC": true, "GB-ABD": true,
	"GB-ABE": true, "GB-AGB": true, "GB-AGY": true, "GB-AND": true, "GB-ANN": true,
	"GB-ANS": true, "GB-BAS": true, "GB-BBD": true, "GB-BDF": true, "GB-BDG": true,
	"GB-BEN": true, "GB-BEX": true, "GB-BFS": true, "GB-BGE": true, "GB-BGW": true,
	"GB-BIR": true, "GB-BKM": true, "GB-BMH": true, "GB-BNE": true, "GB-BNH": true,
	"GB-BNS": true, "GB-BOL": true, "GB-BPL": true, "GB-BRC": true, "GB-BRD": true,
	"GB-BRY": true, "GB-BST": true, "GB-BUR": true, "GB-CAM": true, "GB-CAY": true,
	"GB-CBF": true, "GB-CCG": true, "GB-CGN": true, "GB-CHE": true, "GB-CHW": true,
	"GB-CLD": true, "GB-CLK": true, "GB-CMA": true, "GB-CMD": true, "GB-CMN": true,
	"GB-CON": true, "GB-COV": true, "GB-CRF": true, "GB-CRY": true, "GB-CWY": true,
	"GB-DAL": true, "GB-DBY": true, "GB-DEN": true, "GB-DER": true, "GB-DEV": true,
	"GB-DGY": true, "GB-DNC": true, "GB-DND": true, "GB-DOR": true, "GB-DRS": true,
	"GB-DUD": true, "GB-DUR": true, "GB-EAL": true, "GB-EAW": true, "GB-EAY": true,
	"GB-EDH": true, "GB-EDU": true, "GB-ELN": true, "GB-ELS": true, "GB-ENF": true,
	"GB-ENG": true, "GB-ERW": true, "GB-ERY": true, "GB-ESS": true, "GB-ESX": true,
	"GB-FAL": true, "GB-FIF": true, "GB-FLN": true, "GB-FMO": true, "GB-GAT": true,
	"GB-GBN": true, "GB-GLG": true, "GB-GLS": true, "GB-GRE": true, "GB-GWN": true,
	"GB-HAL": true, "GB-HAM": true, "GB-HAV": true, "GB-HCK": true, "GB-HEF": true,
	"GB-HIL": true, "GB-HLD": true, "GB-HMF": true, "GB-HNS": true, "GB-HPL": true,
	"GB-HRT": true, "GB-HRW": true, "GB-HRY": true, "GB-IOS": true, "GB-IOW": true,
	"GB-ISL": true, "GB-IVC": true, "GB-KEC": true, "GB-KEN": true, "GB-KHL": true,
	"GB-KIR": true, "GB-KTT": true, "GB-KWL": true, "GB-LAN": true, "GB-LBC": true,
	"GB-LBH": true, "GB-LCE": true, "GB-LDS": true, "GB-LEC": true, "GB-LEW": true,
	"GB-LIN": true, "GB-LIV": true, "GB-LND": true, "GB-LUT": true, "GB-MAN": true,
	"GB-MDB": true, "GB-MDW": true, "GB-MEA": true, "GB-MIK": true, "GD-01": true,
	"GB-MLN": true, "GB-MON": true, "GB-MRT": true, "GB-MRY": true, "GB-MTY": true,
	"GB-MUL": true, "GB-NAY": true, "GB-NBL": true, "GB-NEL": true, "GB-NET": true,
	"GB-NFK": true, "GB-NGM": true, "GB-NIR": true, "GB-NLK": true, "GB-NLN": true,
	"GB-NMD": true, "GB-NSM": true, "GB-NTH": true, "GB-NTL": true, "GB-NTT": true,
	"GB-NTY": true, "GB-NWM": true, "GB-NWP": true, "GB-NYK": true, "GB-OLD": true,
	"GB-ORK": true, "GB-OXF": true, "GB-PEM": true, "GB-PKN": true, "GB-PLY": true,
	"GB-POL": true, "GB-POR": true, "GB-POW": true, "GB-PTE": true, "GB-RCC": true,
	"GB-RCH": true, "GB-RCT": true, "GB-RDB": true, "GB-RDG": true, "GB-RFW": true,
	"GB-RIC": true, "GB-ROT": true, "GB-RUT": true, "GB-SAW": true, "GB-SAY": true,
	"GB-SCB": true, "GB-SCT": true, "GB-SFK": true, "GB-SFT": true, "GB-SGC": true,
	"GB-SHF": true, "GB-SHN": true, "GB-SHR": true, "GB-SKP": true, "GB-SLF": true,
	"GB-SLG": true, "GB-SLK": true, "GB-SND": true, "GB-SOL": true, "GB-SOM": true,
	"GB-SOS": true, "GB-SRY": true, "GB-STE": true, "GB-STG": true, "GB-STH": true,
	"GB-STN": true, "GB-STS": true, "GB-STT": true, "GB-STY": true, "GB-SWA": true,
	"GB-SWD": true, "GB-SWK": true, "GB-TAM": true, "GB-TFW": true, "GB-THR": true,
	"GB-TOB": true, "GB-TOF": true, "GB-TRF": true, "GB-TWH": true, "GB-UKM": true,
	"GB-VGL": true, "GB-WAR": true, "GB-WBK": true, "GB-WDU": true, "GB-WFT": true,
	"GB-WGN": true, "GB-WIL": true, "GB-WKF": true, "GB-WLL": true, "GB-WLN": true,
	"GB-WLS": true, "GB-WLV": true, "GB-WND": true, "GB-WNM": true, "GB-WOK": true,
	"GB-WOR": true, "GB-WRL": true, "GB-WRT": true, "GB-WRX": true, "GB-WSM": true,
	"GB-WSX": true, "GB-YOR": true, "GB-ZET": true, "GD-02": true, "GD-03": true,
	"GD-04": true, "GD-05": true, "GD-06": true, "GD-10": true, "GE-AB": true,
	"GE-AJ": true, "GE-GU": true, "GE-IM": true, "GE-KA": true, "GE-KK": true,
	"GE-MM": true, "GE-RL": true, "GE-SJ": true, "GE-SK": true, "GE-SZ": true,
	"GE-TB": true, "GH-AA": true, "GH-AH": true, "GH-BA": true, "GH-CP": true,
	"GH-EP": true, "GH-NP": true, "GH-TV": true, "GH-UE": true, "GH-UW": true,
	"GH-WP": true, "GL-KU": true, "GL-QA": true, "GL-QE": true, "GL-SM": true,
	"GM-B": true, "GM-L": true, "GM-M": true, "GM-N": true, "GM-U": true,
	"GM-W": true, "GN-B": true, "GN-BE": true, "GN-BF": true, "GN-BK": true,
	"GN-C": true, "GN-CO": true, "GN-D": true, "GN-DB": true, "GN-DI": true,
	"GN-DL": true, "GN-DU": true, "GN-F": true, "GN-FA": true, "GN-FO": true,
	"GN-FR": true, "GN-GA": true, "GN-GU": true, "GN-K": true, "GN-KA": true,
	"GN-KB": true, "GN-KD": true, "GN-KE": true, "GN-KN": true, "GN-KO": true,
	"GN-KS": true, "GN-L": true, "GN-LA": true, "GN-LE": true, "GN-LO": true,
	"GN-M": true, "GN-MC": true, "GN-MD": true, "GN-ML": true, "GN-MM": true,
	"GN-N": true, "GN-NZ": true, "GN-PI": true, "GN-SI": true, "GN-TE": true,
	"GN-TO": true, "GN-YO": true, "GQ-AN": true, "GQ-BN": true, "GQ-BS": true,
	"GQ-C": true, "GQ-CS": true, "GQ-I": true, "GQ-KN": true, "GQ-LI": true,
	"GQ-WN": true, "GR-01": true, "GR-03": true, "GR-04": true, "GR-05": true,
	"GR-06": true, "GR-07": true, "GR-11": true, "GR-12": true, "GR-13": true,
	"GR-14": true, "GR-15": true, "GR-16": true, "GR-17": true, "GR-21": true,
	"GR-22": true, "GR-23": true, "GR-24": true, "GR-31": true, "GR-32": true,
	"GR-33": true, "GR-34": true, "GR-41": true, "GR-42": true, "GR-43": true,
	"GR-44": true, "GR-51": true, "GR-52": true, "GR-53": true, "GR-54": true,
	"GR-55": true, "GR-56": true, "GR-57": true, "GR-58": true, "GR-59": true,
	"GR-61": true, "GR-62": true, "GR-63": true, "GR-64": true, "GR-69": true,
	"GR-71": true, "GR-72": true, "GR-73": true, "GR-81": true, "GR-82": true,
	"GR-83": true, "GR-84": true, "GR-85": true, "GR-91": true, "GR-92": true,
	"GR-93": true, "GR-94": true, "GR-A": true, "GR-A1": true, "GR-B": true,
	"GR-C": true, "GR-D": true, "GR-E": true, "GR-F": true, "GR-G": true,
	"GR-H": true, "GR-I": true, "GR-J": true, "GR-K": true, "GR-L": true,
	"GR-M": true, "GT-AV": true, "GT-BV": true, "GT-CM": true, "GT-CQ": true,
	"GT-ES": true, "GT-GU": true, "GT-HU": true, "GT-IZ": true, "GT-JA": true,
	"GT-JU": true, "GT-PE": true, "GT-PR": true, "GT-QC": true, "GT-QZ": true,
	"GT-RE": true, "GT-SA": true, "GT-SM": true, "GT-SO": true, "GT-SR": true,
	"GT-SU": true, "GT-TO": true, "GT-ZA": true, "GW-BA": true, "GW-BL": true,
	"GW-BM": true, "GW-BS": true, "GW-CA": true, "GW-GA": true, "GW-L": true,
	"GW-N": true, "GW-OI": true, "GW-QU": true, "GW-S": true, "GW-TO": true,
	"GY-BA": true, "GY-CU": true, "GY-DE": true, "GY-EB": true, "GY-ES": true,
	"GY-MA": true, "GY-PM": true, "GY-PT": true, "GY-UD": true, "GY-UT": true,
	"HN-AT": true, "HN-CH": true, "HN-CL": true, "HN-CM": true, "HN-CP": true,
	"HN-CR": true, "HN-EP": true, "HN-FM": true, "HN-GD": true, "HN-IB": true,
	"HN-IN": true, "HN-LE": true, "HN-LP": true, "HN-OC": true, "HN-OL": true,
	"HN-SB": true, "HN-VA": true, "HN-YO": true, "HR-01": true, "HR-02": true,
	"HR-03": true, "HR-04": true, "HR-05": true, "HR-06": true, "HR-07": true,
	"HR-08": true, "HR-09": true, "HR-10": true, "HR-11": true, "HR-12": true,
	"HR-13": true, "HR-14": true, "HR-15": true, "HR-16": true, "HR-17": true,
	"HR-18": true, "HR-19": true, "HR-20": true, "HR-21": true, "HT-AR": true,
	"HT-CE": true, "HT-GA": true, "HT-ND": true, "HT-NE": true, "HT-NO": true,
	"HT-OU": true, "HT-SD": true, "HT-SE": true, "HU-BA": true, "HU-BC": true,
	"HU-BE": true, "HU-BK": true, "HU-BU": true, "HU-BZ": true, "HU-CS": true,
	"HU-DE": true, "HU-DU": true, "HU-EG": true, "HU-ER": true, "HU-FE": true,
	"HU-GS": true, "HU-GY": true, "HU-HB": true, "HU-HE": true, "HU-HV": true,
	"HU-JN": true, "HU-KE": true, "HU-KM": true, "HU-KV": true, "HU-MI": true,
	"HU-NK": true, "HU-NO": true, "HU-NY": true, "HU-PE": true, "HU-PS": true,
	"HU-SD": true, "HU-SF": true, "HU-SH": true, "HU-SK": true, "HU-SN": true,
	"HU-SO": true, "HU-SS": true, "HU-ST": true, "HU-SZ": true, "HU-TB": true,
	"HU-TO": true, "HU-VA": true, "HU-VE": true, "HU-VM": true, "HU-ZA": true,
	"HU-ZE": true, "ID-AC": true, "ID-BA": true, "ID-BB": true, "ID-BE": true,
	"ID-BT": true, "ID-GO": true, "ID-IJ": true, "ID-JA": true, "ID-JB": true,
	"ID-JI": true, "ID-JK": true, "ID-JT": true, "ID-JW": true, "ID-KA": true,
	"ID-KB": true, "ID-KI": true, "ID-KR": true, "ID-KS": true, "ID-KT": true,
	"ID-LA": true, "ID-MA": true, "ID-ML": true, "ID-MU": true, "ID-NB": true,
	"ID-NT": true, "ID-NU": true, "ID-PA": true, "ID-PB": true, "ID-RI": true,
	"ID-SA": true, "ID-SB": true, "ID-SG": true, "ID-SL": true, "ID-SM": true,
	"ID-SN": true, "ID-SR": true, "ID-SS": true, "ID-ST": true, "ID-SU": true,
	"ID-YO": true, "IE-C": true, "IE-CE": true, "IE-CN": true, "IE-CO": true,
	"IE-CW": true, "IE-D": true, "IE-DL": true, "IE-G": true, "IE-KE": true,
	"IE-KK": true, "IE-KY": true, "IE-L": true, "IE-LD": true, "IE-LH": true,
	"IE-LK": true, "IE-LM": true, "IE-LS": true, "IE-M": true, "IE-MH": true,
	"IE-MN": true, "IE-MO": true, "IE-OY": true, "IE-RN": true, "IE-SO": true,
	"IE-TA": true, "IE-U": true, "IE-WD": true, "IE-WH": true, "IE-WW": true,
	"IE-WX": true, "IL-D": true, "IL-HA": true, "IL-JM": true, "IL-M": true,
	"IL-TA": true, "IL-Z": true, "IN-AN": true, "IN-AP": true, "IN-AR": true,
	"IN-AS": true, "IN-BR": true, "IN-CH": true, "IN-CT": true, "IN-DD": true,
	"IN-DL": true, "IN-DN": true, "IN-GA": true, "IN-GJ": true, "IN-HP": true,
	"IN-HR": true, "IN-JH": true, "IN-JK": true, "IN-KA": true, "IN-KL": true,
	"IN-LD": true, "IN-MH": true, "IN-ML": true, "IN-MN": true, "IN-MP": true,
	"IN-MZ": true, "IN-NL": true, "IN-OR": true, "IN-PB": true, "IN-PY": true,
	"IN-RJ": true, "IN-SK": true, "IN-TN": true, "IN-TR": true, "IN-UP": true,
	"IN-UT": true, "IN-WB": true, "IQ-AN": true, "IQ-AR": true, "IQ-BA": true,
	"IQ-BB": true, "IQ-BG": true, "IQ-DA": true, "IQ-DI": true, "IQ-DQ": true,
	"IQ-KA": true, "IQ-MA": true, "IQ-MU": true, "IQ-NA": true, "IQ-NI": true,
	"IQ-QA": true, "IQ-SD": true, "IQ-SW": true, "IQ-TS": true, "IQ-WA": true,
	"IR-01": true, "IR-02": true, "IR-03": true, "IR-04": true, "IR-05": true,
	"IR-06": true, "IR-07": true, "IR-08": true, "IR-10": true, "IR-11": true,
	"IR-12": true, "IR-13": true, "IR-14": true, "IR-15": true, "IR-16": true,
	"IR-17": true, "IR-18": true, "IR-19": true, "IR-20": true, "IR-21": true,
	"IR-22": true, "IR-23": true, "IR-24": true, "IR-25": true, "IR-26": true,
	"IR-27": true, "IR-28": true, "IR-29": true, "IR-30": true, "IR-31": true,
	"IS-0": true, "IS-1": true, "IS-2": true, "IS-3": true, "IS-4": true,
	"IS-5": true, "IS-6": true, "IS-7": true, "IS-8": true, "IT-21": true,
	"IT-23": true, "IT-25": true, "IT-32": true, "IT-34": true, "IT-36": true,
	"IT-42": true, "IT-45": true, "IT-52": true, "IT-55": true, "IT-57": true,
	"IT-62": true, "IT-65": true, "IT-67": true, "IT-72": true, "IT-75": true,
	"IT-77": true, "IT-78": true, "IT-82": true, "IT-88": true, "IT-AG": true,
	"IT-AL": true, "IT-AN": true, "IT-AO": true, "IT-AP": true, "IT-AQ": true,
	"IT-AR": true, "IT-AT": true, "IT-AV": true, "IT-BA": true, "IT-BG": true,
	"IT-BI": true, "IT-BL": true, "IT-BN": true, "IT-BO": true, "IT-BR": true,
	"IT-BS": true, "IT-BT": true, "IT-BZ": true, "IT-CA": true, "IT-CB": true,
	"IT-CE": true, "IT-CH": true, "IT-CI": true, "IT-CL": true, "IT-CN": true,
	"IT-CO": true, "IT-CR": true, "IT-CS": true, "IT-CT": true, "IT-CZ": true,
	"IT-EN": true, "IT-FC": true, "IT-FE": true, "IT-FG": true, "IT-FI": true,
	"IT-FM": true, "IT-FR": true, "IT-GE": true, "IT-GO": true, "IT-GR": true,
	"IT-IM": true, "IT-IS": true, "IT-KR": true, "IT-LC": true, "IT-LE": true,
	"IT-LI": true, "IT-LO": true, "IT-LT": true, "IT-LU": true, "IT-MB": true,
	"IT-MC": true, "IT-ME": true, "IT-MI": true, "IT-MN": true, "IT-MO": true,
	"IT-MS": true, "IT-MT": true, "IT-NA": true, "IT-NO": true, "IT-NU": true,
	"IT-OG": true, "IT-OR": true, "IT-OT": true, "IT-PA": true, "IT-PC": true,
	"IT-PD": true, "IT-PE": true, "IT-PG": true, "IT-PI": true, "IT-PN": true,
	"IT-PO": true, "IT-PR": true, "IT-PT": true, "IT-PU": true, "IT-PV": true,
	"IT-PZ": true, "IT-RA": true, "IT-RC": true, "IT-RE": true, "IT-RG": true,
	"IT-RI": true, "IT-RM": true, "IT-RN": true, "IT-RO": true, "IT-SA": true,
	"IT-SI": true, "IT-SO": true, "IT-SP": true, "IT-SR": true, "IT-SS": true,
	"IT-SV": true, "IT-TA": true, "IT-TE": true, "IT-TN": true, "IT-TO": true,
	"IT-TP": true, "IT-TR": true, "IT-TS": true, "IT-TV": true, "IT-UD": true,
	"IT-VA": true, "IT-VB": true, "IT-VC": true, "IT-VE": true, "IT-VI": true,
	"IT-VR": true, "IT-VS": true, "IT-VT": true, "IT-VV": true, "JM-01": true,
	"JM-02": true, "JM-03": true, "JM-04": true, "JM-05": true, "JM-06": true,
	"JM-07": true, "JM-08": true, "JM-09": true, "JM-10": true, "JM-11": true,
	"JM-12": true, "JM-13": true, "JM-14": true, "JO-AJ": true, "JO-AM": true,
	"JO-AQ": true, "JO-AT": true, "JO-AZ": true, "JO-BA": true, "JO-IR": true,
	"JO-JA": true, "JO-KA": true, "JO-MA": true, "JO-MD": true, "JO-MN": true,
	"JP-01": true, "JP-02": true, "JP-03": true, "JP-04": true, "JP-05": true,
	"JP-06": true, "JP-07": true, "JP-08": true, "JP-09": true, "JP-10": true,
	"JP-11": true, "JP-12": true, "JP-13": true, "JP-14": true, "JP-15": true,
	"JP-16": true, "JP-17": true, "JP-18": true, "JP-19": true, "JP-20": true,
	"JP-21": true, "JP-22": true, "JP-23": true, "JP-24": true, "JP-25": true,
	"JP-26": true, "JP-27": true, "JP-28": true, "JP-29": true, "JP-30": true,
	"JP-31": true, "JP-32": true, "JP-33": true, "JP-34": true, "JP-35": true,
	"JP-36": true, "JP-37": true, "JP-38": true, "JP-39": true, "JP-40": true,
	"JP-41": true, "JP-42": true, "JP-43": true, "JP-44": true, "JP-45": true,
	"JP-46": true, "JP-47": true, "KE-110": true, "KE-200": true, "KE-300": true,
	"KE-400": true, "KE-500": true, "KE-700": true, "KE-800": true, "KG-B": true,
	"KG-C": true, "KG-GB": true, "KG-J": true, "KG-N": true, "KG-O": true,
	"KG-T": true, "KG-Y": true, "KH-1": true, "KH-10": true, "KH-11": true,
	"KH-12": true, "KH-13": true, "KH-14": true, "KH-15": true, "KH-16": true,
	"KH-17": true, "KH-18": true, "KH-19": true, "KH-2": true, "KH-20": true,
	"KH-21": true, "KH-22": true, "KH-23": true, "KH-24": true, "KH-3": true,
	"KH-4": true, "KH-5": true, "KH-6": true, "KH-7": true, "KH-8": true,
	"KH-9": true, "KI-G": true, "KI-L": true, "KI-P": true, "KM-A": true,
	"KM-G": true, "KM-M": true, "KN-01": true, "KN-02": true, "KN-03": true,
	"KN-04": true, "KN-05": true, "KN-06": true, "KN-07": true, "KN-08": true,
	"KN-09": true, "KN-10": true, "KN-11": true, "KN-12": true, "KN-13": true,
	"KN-15": true, "KN-K": true, "KN-N": true, "KP-01": true, "KP-02": true,
	"KP-03": true, "KP-04": true, "KP-05": true, "KP-06": true, "KP-07": true,
	"KP-08": true, "KP-09": true, "KP-10": true, "KP-13": true, "KR-11": true,
	"KR-26": true, "KR-27": true, "KR-28": true, "KR-29": true, "KR-30": true,
	"KR-31": true, "KR-41": true, "KR-42": true, "KR-43": true, "KR-44": true,
	"KR-45": true, "KR-46": true, "KR-47": true, "KR-48": true, "KR-49": true,
	"KW-AH": true, "KW-FA": true, "KW-HA": true, "KW-JA": true, "KW-KU": true,
	"KW-MU": true, "KZ-AKM": true, "KZ-AKT": true, "KZ-ALA": true, "KZ-ALM": true,
	"KZ-AST": true, "KZ-ATY": true, "KZ-KAR": true, "KZ-KUS": true, "KZ-KZY": true,
	"KZ-MAN": true, "KZ-PAV": true, "KZ-SEV": true, "KZ-VOS": true, "KZ-YUZ": true,
	"KZ-ZAP": true, "KZ-ZHA": true, "LA-AT": true, "LA-BK": true, "LA-BL": true,
	"LA-CH": true, "LA-HO": true, "LA-KH": true, "LA-LM": true, "LA-LP": true,
	"LA-OU": true, "LA-PH": true, "LA-SL": true, "LA-SV": true, "LA-VI": true,
	"LA-VT": true, "LA-XA": true, "LA-XE": true, "LA-XI": true, "LA-XS": true,
	"LB-AK": true, "LB-AS": true, "LB-BA": true, "LB-BH": true, "LB-BI": true,
	"LB-JA": true, "LB-JL": true, "LB-NA": true, "LI-01": true, "LI-02": true,
	"LI-03": true, "LI-04": true, "LI-05": true, "LI-06": true, "LI-07": true,
	"LI-08": true, "LI-09": true, "LI-10": true, "LI-11": true, "LK-1": true,
	"LK-11": true, "LK-12": true, "LK-13": true, "LK-2": true, "LK-21": true,
	"LK-22": true, "LK-23": true, "LK-3": true, "LK-31": true, "LK-32": true,
	"LK-33": true, "LK-4": true, "LK-41": true, "LK-42": true, "LK-43": true,
	"LK-44": true, "LK-45": true, "LK-5": true, "LK-51": true, "LK-52": true,
	"LK-53": true, "LK-6": true, "LK-61": true, "LK-62": true, "LK-7": true,
	"LK-71": true, "LK-72": true, "LK-8": true, "LK-81": true, "LK-82": true,
	"LK-9": true, "LK-91": true, "LK-92": true, "LR-BG": true, "LR-BM": true,
	"LR-CM": true, "LR-GB": true, "LR-GG": true, "LR-GK": true, "LR-LO": true,
	"LR-MG": true, "LR-MO": true, "LR-MY": true, "LR-NI": true, "LR-RI": true,
	"LR-SI": true, "LS-A": true, "LS-B": true, "LS-C": true, "LS-D": true,
	"LS-E": true, "LS-F": true, "LS-G": true, "LS-H": true, "LS-J": true,
	"LS-K": true, "LT-AL": true, "LT-KL": true, "LT-KU": true, "LT-MR": true,
	"LT-PN": true, "LT-SA": true, "LT-TA": true, "LT-TE": true, "LT-UT": true,
	"LT-VL": true, "LU-D": true, "LU-G": true, "LU-L": true, "LV-001": true,
	"LV-002": true, "LV-003": true, "LV-004": true, "LV-005": true, "LV-006": true,
	"LV-007": true, "LV-008": true, "LV-009": true, "LV-010": true, "LV-011": true,
	"LV-012": true, "LV-013": true, "LV-014": true, "LV-015": true, "LV-016": true,
	"LV-017": true, "LV-018": true, "LV-019": true, "LV-020": true, "LV-021": true,
	"LV-022": true, "LV-023": true, "LV-024": true, "LV-025": true, "LV-026": true,
	"LV-027": true, "LV-028": true, "LV-029": true, "LV-030": true, "LV-031": true,
	"LV-032": true, "LV-033": true, "LV-034": true, "LV-035": true, "LV-036": true,
	"LV-037": true, "LV-038": true, "LV-039": true, "LV-040": true, "LV-041": true,
	"LV-042": true, "LV-043": true, "LV-044": true, "LV-045": true, "LV-046": true,
	"LV-047": true, "LV-048": true, "LV-049": true, "LV-050": true, "LV-051": true,
	"LV-052": true, "LV-053": true, "LV-054": true, "LV-055": true, "LV-056": true,
	"LV-057": true, "LV-058": true, "LV-059": true, "LV-060": true, "LV-061": true,
	"LV-062": true, "LV-063": true, "LV-064": true, "LV-065": true, "LV-066": true,
	"LV-067": true, "LV-068": true, "LV-069": true, "LV-070": true, "LV-071": true,
	"LV-072": true, "LV-073": true, "LV-074": true, "LV-075": true, "LV-076": true,
	"LV-077": true, "LV-078": true, "LV-079": true, "LV-080": true, "LV-081": true,
	"LV-082": true, "LV-083": true, "LV-084": true, "LV-085": true, "LV-086": true,
	"LV-087": true, "LV-088": true, "LV-089": true, "LV-090": true, "LV-091": true,
	"LV-092": true, "LV-093": true, "LV-094": true, "LV-095": true, "LV-096": true,
	"LV-097": true, "LV-098": true, "LV-099": true, "LV-100": true, "LV-101": true,
	"LV-102": true, "LV-103": true, "LV-104": true, "LV-105": true, "LV-106": true,
	"LV-107": true, "LV-108": true, "LV-109": true, "LV-110": true, "LV-DGV": true,
	"LV-JEL": true, "LV-JKB": true, "LV-JUR": true, "LV-LPX": true, "LV-REZ": true,
	"LV-RIX": true, "LV-VEN": true, "LV-VMR": true, "LY-BA": true, "LY-BU": true,
	"LY-DR": true, "LY-GT": true, "LY-JA": true, "LY-JB": true, "LY-JG": true,
	"LY-JI": true, "LY-JU": true, "LY-KF": true, "LY-MB": true, "LY-MI": true,
	"LY-MJ": true, "LY-MQ": true, "LY-NL": true, "LY-NQ": true, "LY-SB": true,
	"LY-SR": true, "LY-TB": true, "LY-WA": true, "LY-WD": true, "LY-WS": true,
	"LY-ZA": true, "MA-01": true, "MA-02": true, "MA-03": true, "MA-04": true,
	"MA-05": true, "MA-06": true, "MA-07": true, "MA-08": true, "MA-09": true,
	"MA-10": true, "MA-11": true, "MA-12": true, "MA-13": true, "MA-14": true,
	"MA-15": true, "MA-16": true, "MA-AGD": true, "MA-AOU": true, "MA-ASZ": true,
	"MA-AZI": true, "MA-BEM": true, "MA-BER": true, "MA-BES": true, "MA-BOD": true,
	"MA-BOM": true, "MA-CAS": true, "MA-CHE": true, "MA-CHI": true, "MA-CHT": true,
	"MA-ERR": true, "MA-ESI": true, "MA-ESM": true, "MA-FAH": true, "MA-FES": true,
	"MA-FIG": true, "MA-GUE": true, "MA-HAJ": true, "MA-HAO": true, "MA-HOC": true,
	"MA-IFR": true, "MA-INE": true, "MA-JDI": true, "MA-JRA": true, "MA-KEN": true,
	"MA-KES": true, "MA-KHE": true, "MA-KHN": true, "MA-KHO": true, "MA-LAA": true,
	"MA-LAR": true, "MA-MED": true, "MA-MEK": true, "MA-MMD": true, "MA-MMN": true,
	"MA-MOH": true, "MA-MOU": true, "MA-NAD": true, "MA-NOU": true, "MA-OUA": true,
	"MA-OUD": true, "MA-OUJ": true, "MA-RAB": true, "MA-SAF": true, "MA-SAL": true,
	"MA-SEF": true, "MA-SET": true, "MA-SIK": true, "MA-SKH": true, "MA-SYB": true,
	"MA-TAI": true, "MA-TAO": true, "MA-TAR": true, "MA-TAT": true, "MA-TAZ": true,
	"MA-TET": true, "MA-TIZ": true, "MA-TNG": true, "MA-TNT": true, "MA-ZAG": true,
	"MC-CL": true, "MC-CO": true, "MC-FO": true, "MC-GA": true, "MC-JE": true,
	"MC-LA": true, "MC-MA": true, "MC-MC": true, "MC-MG": true, "MC-MO": true,
	"MC-MU": true, "MC-PH": true, "MC-SD": true, "MC-SO": true, "MC-SP": true,
	"MC-SR": true, "MC-VR": true, "MD-AN": true, "MD-BA": true, "MD-BD": true,
	"MD-BR": true, "MD-BS": true, "MD-CA": true, "MD-CL": true, "MD-CM": true,
	"MD-CR": true, "MD-CS": true, "MD-CT": true, "MD-CU": true, "MD-DO": true,
	"MD-DR": true, "MD-DU": true, "MD-ED": true, "MD-FA": true, "MD-FL": true,
	"MD-GA": true, "MD-GL": true, "MD-HI": true, "MD-IA": true, "MD-LE": true,
	"MD-NI": true, "MD-OC": true, "MD-OR": true, "MD-RE": true, "MD-RI": true,
	"MD-SD": true, "MD-SI": true, "MD-SN": true, "MD-SO": true, "MD-ST": true,
	"MD-SV": true, "MD-TA": true, "MD-TE": true, "MD-UN": true, "ME-01": true,
	"ME-02": true, "ME-03": true, "ME-04": true, "ME-05": true, "ME-06": true,
	"ME-07": true, "ME-08": true, "ME-09": true, "ME-10": true, "ME-11": true,
	"ME-12": true, "ME-13": true, "ME-14": true, "ME-15": true, "ME-16": true,
	"ME-17": true, "ME-18": true, "ME-19": true, "ME-20": true, "ME-21": true,
	"MG-A": true, "MG-D": true, "MG-F": true, "MG-M": true, "MG-T": true,
	"MG-U": true, "MH-ALK": true, "MH-ALL": true, "MH-ARN": true, "MH-AUR": true,
	"MH-EBO": true, "MH-ENI": true, "MH-JAB": true, "MH-JAL": true, "MH-KIL": true,
	"MH-KWA": true, "MH-L": true, "MH-LAE": true, "MH-LIB": true, "MH-LIK": true,
	"MH-MAJ": true, "MH-MAL": true, "MH-MEJ": true, "MH-MIL": true, "MH-NMK": true,
	"MH-NMU": true, "MH-RON": true, "MH-T": true, "MH-UJA": true, "MH-UTI": true,
	"MH-WTJ": true, "MH-WTN": true, "MK-01": true, "MK-02": true, "MK-03": true,
	"MK-04": true, "MK-05": true, "MK-06": true, "MK-07": true, "MK-08": true,
	"MK-09": true, "MK-10": true, "MK-11": true, "MK-12": true, "MK-13": true,
	"MK-14": true, "MK-15": true, "MK-16": true, "MK-17": true, "MK-18": true,
	"MK-19": true, "MK-20": true, "MK-21": true, "MK-22": true, "MK-23": true,
	"MK-24": true, "MK-25": true, "MK-26": true, "MK-27": true, "MK-28": true,
	"MK-29": true, "MK-30": true, "MK-31": true, "MK-32": true, "MK-33": true,
	"MK-34": true, "MK-35": true, "MK-36": true, "MK-37": true, "MK-38": true,
	"MK-39": true, "MK-40": true, "MK-41": true, "MK-42": true, "MK-43": true,
	"MK-44": true, "MK-45": true, "MK-46": true, "MK-47": true, "MK-48": true,
	"MK-49": true, "MK-50": true, "MK-51": true, "MK-52": true, "MK-53": true,
	"MK-54": true, "MK-55": true, "MK-56": true, "MK-57": true, "MK-58": true,
	"MK-59": true, "MK-60": true, "MK-61": true, "MK-62": true, "MK-63": true,
	"MK-64": true, "MK-65": true, "MK-66": true, "MK-67": true, "MK-68": true,
	"MK-69": true, "MK-70": true, "MK-71": true, "MK-72": true, "MK-73": true,
	"MK-74": true, "MK-75": true, "MK-76": true, "MK-77": true, "MK-78": true,
	"MK-79": true, "MK-80": true, "MK-81": true, "MK-82": true, "MK-83": true,
	"MK-84": true, "ML-1": true, "ML-2": true, "ML-3": true, "ML-4": true,
	"ML-5": true, "ML-6": true, "ML-7": true, "ML-8": true, "ML-BK0": true,
	"MM-01": true, "MM-02": true, "MM-03": true, "MM-04": true, "MM-05": true,
	"MM-06": true, "MM-07": true, "MM-11": true, "MM-12": true, "MM-13": true,
	"MM-14": true, "MM-15": true, "MM-16": true, "MM-17": true, "MN-035": true,
	"MN-037": true, "MN-039": true, "MN-041": true, "MN-043": true, "MN-046": true,
	"MN-047": true, "MN-049": true, "MN-051": true, "MN-053": true, "MN-055": true,
	"MN-057": true, "MN-059": true, "MN-061": true, "MN-063": true, "MN-064": true,
	"MN-065": true, "MN-067": true, "MN-069": true, "MN-071": true, "MN-073": true,
	"MN-1": true, "MR-01": true, "MR-02": true, "MR-03": true, "MR-04": true,
	"MR-05": true, "MR-06": true, "MR-07": true, "MR-08": true, "MR-09": true,
	"MR-10": true, "MR-11": true, "MR-12": true, "MR-NKC": true, "MT-01": true,
	"MT-02": true, "MT-03": true, "MT-04": true, "MT-05": true, "MT-06": true,
	"MT-07": true, "MT-08": true, "MT-09": true, "MT-10": true, "MT-11": true,
	"MT-12": true, "MT-13": true, "MT-14": true, "MT-15": true, "MT-16": true,
	"MT-17": true, "MT-18": true, "MT-19": true, "MT-20": true, "MT-21": true,
	"MT-22": true, "MT-23": true, "MT-24": true, "MT-25": true, "MT-26": true,
	"MT-27": true, "MT-28": true, "MT-29": true, "MT-30": true, "MT-31": true,
	"MT-32": true, "MT-33": true, "MT-34": true, "MT-35": true, "MT-36": true,
	"MT-37": true, "MT-38": true, "MT-39": true, "MT-40": true, "MT-41": true,
	"MT-42": true, "MT-43": true, "MT-44": true, "MT-45": true, "MT-46": true,
	"MT-47": true, "MT-48": true, "MT-49": true, "MT-50": true, "MT-51": true,
	"MT-52": true, "MT-53": true, "MT-54": true, "MT-55": true, "MT-56": true,
	"MT-57": true, "MT-58": true, "MT-59": true, "MT-60": true, "MT-61": true,
	"MT-62": true, "MT-63": true, "MT-64": true, "MT-65": true, "MT-66": true,
	"MT-67": true, "MT-68": true, "MU-AG": true, "MU-BL": true, "MU-BR": true,
	"MU-CC": true, "MU-CU": true, "MU-FL": true, "MU-GP": true, "MU-MO": true,
	"MU-PA": true, "MU-PL": true, "MU-PU": true, "MU-PW": true, "MU-QB": true,
	"MU-RO": true, "MU-RP": true, "MU-SA": true, "MU-VP": true, "MV-00": true,
	"MV-01": true, "MV-02": true, "MV-03": true, "MV-04": true, "MV-05": true,
	"MV-07": true, "MV-08": true, "MV-12": true, "MV-13": true, "MV-14": true,
	"MV-17": true, "MV-20": true, "MV-23": true, "MV-24": true, "MV-25": true,
	"MV-26": true, "MV-27": true, "MV-28": true, "MV-29": true, "MV-CE": true,
	"MV-MLE": true, "MV-NC": true, "MV-NO": true, "MV-SC": true, "MV-SU": true,
	"MV-UN": true, "MV-US": true, "MW-BA": true, "MW-BL": true, "MW-C": true,
	"MW-CK": true, "MW-CR": true, "MW-CT": true, "MW-DE": true, "MW-DO": true,
	"MW-KR": true, "MW-KS": true, "MW-LI": true, "MW-LK": true, "MW-MC": true,
	"MW-MG": true, "MW-MH": true, "MW-MU": true, "MW-MW": true, "MW-MZ": true,
	"MW-N": true, "MW-NB": true, "MW-NE": true, "MW-NI": true, "MW-NK": true,
	"MW-NS": true, "MW-NU": true, "MW-PH": true, "MW-RU": true, "MW-S": true,
	"MW-SA": true, "MW-TH": true, "MW-ZO": true, "MX-AGU": true, "MX-BCN": true,
	"MX-BCS": true, "MX-CAM": true, "MX-CHH": true, "MX-CHP": true, "MX-COA": true,
	"MX-COL": true, "MX-DIF": true, "MX-DUR": true, "MX-GRO": true, "MX-GUA": true,
	"MX-HID": true, "MX-JAL": true, "MX-MEX": true, "MX-MIC": true, "MX-MOR": true,
	"MX-NAY": true, "MX-NLE": true, "MX-OAX": true, "MX-PUE": true, "MX-QUE": true,
	"MX-ROO": true, "MX-SIN": true, "MX-SLP": true, "MX-SON": true, "MX-TAB": true,
	"MX-TAM": true, "MX-TLA": true, "MX-VER": true, "MX-YUC": true, "MX-ZAC": true,
	"MY-01": true, "MY-02": true, "MY-03": true, "MY-04": true, "MY-05": true,
	"MY-06": true, "MY-07": true, "MY-08": true, "MY-09": true, "MY-10": true,
	"MY-11": true, "MY-12": true, "MY-13": true, "MY-14": true, "MY-15": true,
	"MY-16": true, "MZ-A": true, "MZ-B": true, "MZ-G": true, "MZ-I": true,
	"MZ-L": true, "MZ-MPM": true, "MZ-N": true, "MZ-P": true, "MZ-Q": true,
	"MZ-S": true, "MZ-T": true, "NA-CA": true, "NA-ER": true, "NA-HA": true,
	"NA-KA": true, "NA-KH": true, "NA-KU": true, "NA-OD": true, "NA-OH": true,
	"NA-OK": true, "NA-ON": true, "NA-OS": true, "NA-OT": true, "NA-OW": true,
	"NE-1": true, "NE-2": true, "NE-3": true, "NE-4": true, "NE-5": true,
	"NE-6": true, "NE-7": true, "NE-8": true, "NG-AB": true, "NG-AD": true,
	"NG-AK": true, "NG-AN": true, "NG-BA": true, "NG-BE": true, "NG-BO": true,
	"NG-BY": true, "NG-CR": true, "NG-DE": true, "NG-EB": true, "NG-ED": true,
	"NG-EK": true, "NG-EN": true, "NG-FC": true, "NG-GO": true, "NG-IM": true,
	"NG-JI": true, "NG-KD": true, "NG-KE": true, "NG-KN": true, "NG-KO": true,
	"NG-KT": true, "NG-KW": true, "NG-LA": true, "NG-NA": true, "NG-NI": true,
	"NG-OG": true, "NG-ON": true, "NG-OS": true, "NG-OY": true, "NG-PL": true,
	"NG-RI": true, "NG-SO": true, "NG-TA": true, "NG-YO": true, "NG-ZA": true,
	"NI-AN": true, "NI-AS": true, "NI-BO": true, "NI-CA": true, "NI-CI": true,
	"NI-CO": true, "NI-ES": true, "NI-GR": true, "NI-JI": true, "NI-LE": true,
	"NI-MD": true, "NI-MN": true, "NI-MS": true, "NI-MT": true, "NI-NS": true,
	"NI-RI": true, "NI-SJ": true, "NL-AW": true, "NL-BQ1": true, "NL-BQ2": true,
	"NL-BQ3": true, "NL-CW": true, "NL-DR": true, "NL-FL": true, "NL-FR": true,
	"NL-GE": true, "NL-GR": true, "NL-LI": true, "NL-NB": true, "NL-NH": true,
	"NL-OV": true, "NL-SX": true, "NL-UT": true, "NL-ZE": true, "NL-ZH": true,
	"NO-01": true, "NO-02": true, "NO-03": true, "NO-04": true, "NO-05": true,
	"NO-06": true, "NO-07": true, "NO-08": true, "NO-09": true, "NO-10": true,
	"NO-11": true, "NO-12": true, "NO-14": true, "NO-15": true, "NO-16": true,
	"NO-17": true, "NO-18": true, "NO-19": true, "NO-20": true, "NO-21": true,
	"NO-22": true, "NP-1": true, "NP-2": true, "NP-3": true, "NP-4": true,
	"NP-5": true, "NP-BA": true, "NP-BH": true, "NP-DH": true, "NP-GA": true,
	"NP-JA": true, "NP-KA": true, "NP-KO": true, "NP-LU": true, "NP-MA": true,
	"NP-ME": true, "NP-NA": true, "NP-RA": true, "NP-SA": true, "NP-SE": true,
	"NR-01": true, "NR-02": true, "NR-03": true, "NR-04": true, "NR-05": true,
	"NR-06": true, "NR-07": true, "NR-08": true, "NR-09": true, "NR-10": true,
	"NR-11": true, "NR-12": true, "NR-13": true, "NR-14": true, "NZ-AUK": true,
	"NZ-BOP": true, "NZ-CAN": true, "NZ-CIT": true, "NZ-GIS": true, "NZ-HKB": true,
	"NZ-MBH": true, "NZ-MWT": true, "NZ-N": true, "NZ-NSN": true, "NZ-NTL": true,
	"NZ-OTA": true, "NZ-S": true, "NZ-STL": true, "NZ-TAS": true, "NZ-TKI": true,
	"NZ-WGN": true, "NZ-WKO": true, "NZ-WTC": true, "OM-BA": true, "OM-BU": true,
	"OM-DA": true, "OM-MA": true, "OM-MU": true, "OM-SH": true, "OM-WU": true,
	"OM-ZA": true, "OM-ZU": true, "PA-1": true, "PA-2": true, "PA-3": true,
	"PA-4": true, "PA-5": true, "PA-6": true, "PA-7": true, "PA-8": true,
	"PA-9": true, "PA-EM": true, "PA-KY": true, "PA-NB": true, "PE-AMA": true,
	"PE-ANC": true, "PE-APU": true, "PE-ARE": true, "PE-AYA": true, "PE-CAJ": true,
	"PE-CAL": true, "PE-CUS": true, "PE-HUC": true, "PE-HUV": true, "PE-ICA": true,
	"PE-JUN": true, "PE-LAL": true, "PE-LAM": true, "PE-LIM": true, "PE-LMA": true,
	"PE-LOR": true, "PE-MDD": true, "PE-MOQ": true, "PE-PAS": true, "PE-PIU": true,
	"PE-PUN": true, "PE-SAM": true, "PE-TAC": true, "PE-TUM": true, "PE-UCA": true,
	"PG-CPK": true, "PG-CPM": true, "PG-EBR": true, "PG-EHG": true, "PG-EPW": true,
	"PG-ESW": true, "PG-GPK": true, "PG-MBA": true, "PG-MPL": true, "PG-MPM": true,
	"PG-MRL": true, "PG-NCD": true, "PG-NIK": true, "PG-NPP": true, "PG-NSB": true,
	"PG-SAN": true, "PG-SHM": true, "PG-WBK": true, "PG-WHM": true, "PG-WPD": true,
	"PH-00": true, "PH-01": true, "PH-02": true, "PH-03": true, "PH-05": true,
	"PH-06": true, "PH-07": true, "PH-08": true, "PH-09": true, "PH-10": true,
	"PH-11": true, "PH-12": true, "PH-13": true, "PH-14": true, "PH-15": true,
	"PH-40": true, "PH-41": true, "PH-ABR": true, "PH-AGN": true, "PH-AGS": true,
	"PH-AKL": true, "PH-ALB": true, "PH-ANT": true, "PH-APA": true, "PH-AUR": true,
	"PH-BAN": true, "PH-BAS": true, "PH-BEN": true, "PH-BIL": true, "PH-BOH": true,
	"PH-BTG": true, "PH-BTN": true, "PH-BUK": true, "PH-BUL": true, "PH-CAG": true,
	"PH-CAM": true, "PH-CAN": true, "PH-CAP": true, "PH-CAS": true, "PH-CAT": true,
	"PH-CAV": true, "PH-CEB": true, "PH-COM": true, "PH-DAO": true, "PH-DAS": true,
	"PH-DAV": true, "PH-DIN": true, "PH-EAS": true, "PH-GUI": true, "PH-IFU": true,
	"PH-ILI": true, "PH-ILN": true, "PH-ILS": true, "PH-ISA": true, "PH-KAL": true,
	"PH-LAG": true, "PH-LAN": true, "PH-LAS": true, "PH-LEY": true, "PH-LUN": true,
	"PH-MAD": true, "PH-MAG": true, "PH-MAS": true, "PH-MDC": true, "PH-MDR": true,
	"PH-MOU": true, "PH-MSC": true, "PH-MSR": true, "PH-NCO": true, "PH-NEC": true,
	"PH-NER": true, "PH-NSA": true, "PH-NUE": true, "PH-NUV": true, "PH-PAM": true,
	"PH-PAN": true, "PH-PLW": true, "PH-QUE": true, "PH-QUI": true, "PH-RIZ": true,
	"PH-ROM": true, "PH-SAR": true, "PH-SCO": true, "PH-SIG": true, "PH-SLE": true,
	"PH-SLU": true, "PH-SOR": true, "PH-SUK": true, "PH-SUN": true, "PH-SUR": true,
	"PH-TAR": true, "PH-TAW": true, "PH-WSA": true, "PH-ZAN": true, "PH-ZAS": true,
	"PH-ZMB": true, "PH-ZSI": true, "PK-BA": true, "PK-GB": true, "PK-IS": true,
	"PK-JK": true, "PK-KP": true, "PK-PB": true, "PK-SD": true, "PK-TA": true,
	"PL-DS": true, "PL-KP": true, "PL-LB": true, "PL-LD": true, "PL-LU": true,
	"PL-MA": true, "PL-MZ": true, "PL-OP": true, "PL-PD": true, "PL-PK": true,
	"PL-PM": true, "PL-SK": true, "PL-SL": true, "PL-WN": true, "PL-WP": true,
	"PL-ZP": true, "PS-BTH": true, "PS-DEB": true, "PS-GZA": true, "PS-HBN": true,
	"PS-JEM": true, "PS-JEN": true, "PS-JRH": true, "PS-KYS": true, "PS-NBS": true,
	"PS-NGZ": true, "PS-QQA": true, "PS-RBH": true, "PS-RFH": true, "PS-SLT": true,
	"PS-TBS": true, "PS-TKM": true, "PT-01": true, "PT-02": true, "PT-03": true,
	"PT-04": true, "PT-05": true, "PT-06": true, "PT-07": true, "PT-08": true,
	"PT-09": true, "PT-10": true, "PT-11": true, "PT-12": true, "PT-13": true,
	"PT-14": true, "PT-15": true, "PT-16": true, "PT-17": true, "PT-18": true,
	"PT-20": true, "PT-30": true, "PW-002": true, "PW-004": true, "PW-010": true,
	"PW-050": true, "PW-100": true, "PW-150": true, "PW-212": true, "PW-214": true,
	"PW-218": true, "PW-222": true, "PW-224": true, "PW-226": true, "PW-227": true,
	"PW-228": true, "PW-350": true, "PW-370": true, "PY-1": true, "PY-10": true,
	"PY-11": true, "PY-12": true, "PY-13": true, "PY-14": true, "PY-15": true,
	"PY-16": true, "PY-19": true, "PY-2": true, "PY-3": true, "PY-4": true,
	"PY-5": true, "PY-6": true, "PY-7": true, "PY-8": true, "PY-9": true,
	"PY-ASU": true, "QA-DA": true, "QA-KH": true, "QA-MS": true, "QA-RA": true,
	"QA-US": true, "QA-WA": true, "QA-ZA": true, "RO-AB": true, "RO-AG": true,
	"RO-AR": true, "RO-B": true, "RO-BC": true, "RO-BH": true, "RO-BN": true,
	"RO-BR": true, "RO-BT": true, "RO-BV": true, "RO-BZ": true, "RO-CJ": true,
	"RO-CL": true, "RO-CS": true, "RO-CT": true, "RO-CV": true, "RO-DB": true,
	"RO-DJ": true, "RO-GJ": true, "RO-GL": true, "RO-GR": true, "RO-HD": true,
	"RO-HR": true, "RO-IF": true, "RO-IL": true, "RO-IS": true, "RO-MH": true,
	"RO-MM": true, "RO-MS": true, "RO-NT": true, "RO-OT": true, "RO-PH": true,
	"RO-SB": true, "RO-SJ": true, "RO-SM": true, "RO-SV": true, "RO-TL": true,
	"RO-TM": true, "RO-TR": true, "RO-VL": true, "RO-VN": true, "RO-VS": true,
	"RS-00": true, "RS-01": true, "RS-02": true, "RS-03": true, "RS-04": true,
	"RS-05": true, "RS-06": true, "RS-07": true, "RS-08": true, "RS-09": true,
	"RS-10": true, "RS-11": true, "RS-12": true, "RS-13": true, "RS-14": true,
	"RS-15": true, "RS-16": true, "RS-17": true, "RS-18": true, "RS-19": true,
	"RS-20": true, "RS-21": true, "RS-22": true, "RS-23": true, "RS-24": true,
	"RS-25": true, "RS-26": true, "RS-27": true, "RS-28": true, "RS-29": true,
	"RS-KM": true, "RS-VO": true, "RU-AD": true, "RU-AL": true, "RU-ALT": true,
	"RU-AMU": true, "RU-ARK": true, "RU-AST": true, "RU-BA": true, "RU-BEL": true,
	"RU-BRY": true, "RU-BU": true, "RU-CE": true, "RU-CHE": true, "RU-CHU": true,
	"RU-CU": true, "RU-DA": true, "RU-IN": true, "RU-IRK": true, "RU-IVA": true,
	"RU-KAM": true, "RU-KB": true, "RU-KC": true, "RU-KDA": true, "RU-KEM": true,
	"RU-KGD": true, "RU-KGN": true, "RU-KHA": true, "RU-KHM": true, "RU-KIR": true,
	"RU-KK": true, "RU-KL": true, "RU-KLU": true, "RU-KO": true, "RU-KOS": true,
	"RU-KR": true, "RU-KRS": true, "RU-KYA": true, "RU-LEN": true, "RU-LIP": true,
	"RU-MAG": true, "RU-ME": true, "RU-MO": true, "RU-MOS": true, "RU-MOW": true,
	"RU-MUR": true, "RU-NEN": true, "RU-NGR": true, "RU-NIZ": true, "RU-NVS": true,
	"RU-OMS": true, "RU-ORE": true, "RU-ORL": true, "RU-PER": true, "RU-PNZ": true,
	"RU-PRI": true, "RU-PSK": true, "RU-ROS": true, "RU-RYA": true, "RU-SA": true,
	"RU-SAK": true, "RU-SAM": true, "RU-SAR": true, "RU-SE": true, "RU-SMO": true,
	"RU-SPE": true, "RU-STA": true, "RU-SVE": true, "RU-TA": true, "RU-TAM": true,
	"RU-TOM": true, "RU-TUL": true, "RU-TVE": true, "RU-TY": true, "RU-TYU": true,
	"RU-UD": true, "RU-ULY": true, "RU-VGG": true, "RU-VLA": true, "RU-VLG": true,
	"RU-VOR": true, "RU-YAN": true, "RU-YAR": true, "RU-YEV": true, "RU-ZAB": true,
	"RW-01": true, "RW-02": true, "RW-03": true, "RW-04": true, "RW-05": true,
	"SA-01": true, "SA-02": true, "SA-03": true, "SA-04": true, "SA-05": true,
	"SA-06": true, "SA-07": true, "SA-08": true, "SA-09": true, "SA-10": true,
	"SA-11": true, "SA-12": true, "SA-14": true, "SB-CE": true, "SB-CH": true,
	"SB-CT": true, "SB-GU": true, "SB-IS": true, "SB-MK": true, "SB-ML": true,
	"SB-RB": true, "SB-TE": true, "SB-WE": true, "SC-01": true, "SC-02": true,
	"SC-03": true, "SC-04": true, "SC-05": true, "SC-06": true, "SC-07": true,
	"SC-08": true, "SC-09": true, "SC-10": true, "SC-11": true, "SC-12": true,
	"SC-13": true, "SC-14": true, "SC-15": true, "SC-16": true, "SC-17": true,
	"SC-18": true, "SC-19": true, "SC-20": true, "SC-21": true, "SC-22": true,
	"SC-23": true, "SC-24": true, "SC-25": true, "SD-DC": true, "SD-DE": true,
	"SD-DN": true, "SD-DS": true, "SD-DW": true, "SD-GD": true, "SD-GZ": true,
	"SD-KA": true, "SD-KH": true, "SD-KN": true, "SD-KS": true, "SD-NB": true,
	"SD-NO": true, "SD-NR": true, "SD-NW": true, "SD-RS": true, "SD-SI": true,
	"SE-AB": true, "SE-AC": true, "SE-BD": true, "SE-C": true, "SE-D": true,
	"SE-E": true, "SE-F": true, "SE-G": true, "SE-H": true, "SE-I": true,
	"SE-K": true, "SE-M": true, "SE-N": true, "SE-O": true, "SE-S": true,
	"SE-T": true, "SE-U": true, "SE-W": true, "SE-X": true, "SE-Y": true,
	"SE-Z": true, "SG-01": true, "SG-02": true, "SG-03": true, "SG-04": true,
	"SG-05": true, "SH-AC": true, "SH-HL": true, "SH-TA": true, "SI-001": true,
	"SI-002": true, "SI-003": true, "SI-004": true, "SI-005": true, "SI-006": true,
	"SI-007": true, "SI-008": true, "SI-009": true, "SI-010": true, "SI-011": true,
	"SI-012": true, "SI-013": true, "SI-014": true, "SI-015": true, "SI-016": true,
	"SI-017": true, "SI-018": true, "SI-019": true, "SI-020": true, "SI-021": true,
	"SI-022": true, "SI-023": true, "SI-024": true, "SI-025": true, "SI-026": true,
	"SI-027": true, "SI-028": true, "SI-029": true, "SI-030": true, "SI-031": true,
	"SI-032": true, "SI-033": true, "SI-034": true, "SI-035": true, "SI-036": true,
	"SI-037": true, "SI-038": true, "SI-039": true, "SI-040": true, "SI-041": true,
	"SI-042": true, "SI-043": true, "SI-044": true, "SI-045": true, "SI-046": true,
	"SI-047": true, "SI-048": true, "SI-049": true, "SI-050": true, "SI-051": true,
	"SI-052": true, "SI-053": true, "SI-054": true, "SI-055": true, "SI-056": true,
	"SI-057": true, "SI-058": true, "SI-059": true, "SI-060": true, "SI-061": true,
	"SI-062": true, "SI-063": true, "SI-064": true, "SI-065": true, "SI-066": true,
	"SI-067": true, "SI-068": true, "SI-069": true, "SI-070": true, "SI-071": true,
	"SI-072": true, "SI-073": true, "SI-074": true, "SI-075": true, "SI-076": true,
	"SI-077": true, "SI-078": true, "SI-079": true, "SI-080": true, "SI-081": true,
	"SI-082": true, "SI-083": true, "SI-084": true, "SI-085": true, "SI-086": true,
	"SI-087": true, "SI-088": true, "SI-089": true, "SI-090": true, "SI-091": true,
	"SI-092": true, "SI-093": true, "SI-094": true, "SI-095": true, "SI-096": true,
	"SI-097": true, "SI-098": true, "SI-099": true, "SI-100": true, "SI-101": true,
	"SI-102": true, "SI-103": true, "SI-104": true, "SI-105": true, "SI-106": true,
	"SI-107": true, "SI-108": true, "SI-109": true, "SI-110": true, "SI-111": true,
	"SI-112": true, "SI-113": true, "SI-114": true, "SI-115": true, "SI-116": true,
	"SI-117": true, "SI-118": true, "SI-119": true, "SI-120": true, "SI-121": true,
	"SI-122": true, "SI-123": true, "SI-124": true, "SI-125": true, "SI-126": true,
	"SI-127": true, "SI-128": true, "SI-129": true, "SI-130": true, "SI-131": true,
	"SI-132": true, "SI-133": true, "SI-134": true, "SI-135": true, "SI-136": true,
	"SI-137": true, "SI-138": true, "SI-139": true, "SI-140": true, "SI-141": true,
	"SI-142": true, "SI-143": true, "SI-144": true, "SI-146": true, "SI-147": true,
	"SI-148": true, "SI-149": true, "SI-150": true, "SI-151": true, "SI-152": true,
	"SI-153": true, "SI-154": true, "SI-155": true, "SI-156": true, "SI-157": true,
	"SI-158": true, "SI-159": true, "SI-160": true, "SI-161": true, "SI-162": true,
	"SI-163": true, "SI-164": true, "SI-165": true, "SI-166": true, "SI-167": true,
	"SI-168": true, "SI-169": true, "SI-170": true, "SI-171": true, "SI-172": true,
	"SI-173": true, "SI-174": true, "SI-175": true, "SI-176": true, "SI-177": true,
	"SI-178": true, "SI-179": true, "SI-180": true, "SI-181": true, "SI-182": true,
	"SI-183": true, "SI-184": true, "SI-185": true, "SI-186": true, "SI-187": true,
	"SI-188": true, "SI-189": true, "SI-190": true, "SI-191": true, "SI-192": true,
	"SI-193": true, "SI-194": true, "SI-195": true, "SI-196": true, "SI-197": true,
	"SI-198": true, "SI-199": true, "SI-200": true, "SI-201": true, "SI-202": true,
	"SI-203": true, "SI-204": true, "SI-205": true, "SI-206": true, "SI-207": true,
	"SI-208": true, "SI-209": true, "SI-210": true, "SI-211": true, "SK-BC": true,
	"SK-BL": true, "SK-KI": true, "SK-NI": true, "SK-PV": true, "SK-TA": true,
	"SK-TC": true, "SK-ZI": true, "SL-E": true, "SL-N": true, "SL-S": true,
	"SL-W": true, "SM-01": true, "SM-02": true, "SM-03": true, "SM-04": true,
	"SM-05": true, "SM-06": true, "SM-07": true, "SM-08": true, "SM-09": true,
	"SN-DB": true, "SN-DK": true, "SN-FK": true, "SN-KA": true, "SN-KD": true,
	"SN-KE": true, "SN-KL": true, "SN-LG": true, "SN-MT": true, "SN-SE": true,
	"SN-SL": true, "SN-TC": true, "SN-TH": true, "SN-ZG": true, "SO-AW": true,
	"SO-BK": true, "SO-BN": true, "SO-BR": true, "SO-BY": true, "SO-GA": true,
	"SO-GE": true, "SO-HI": true, "SO-JD": true, "SO-JH": true, "SO-MU": true,
	"SO-NU": true, "SO-SA": true, "SO-SD": true, "SO-SH": true, "SO-SO": true,
	"SO-TO": true, "SO-WO": true, "SR-BR": true, "SR-CM": true, "SR-CR": true,
	"SR-MA": true, "SR-NI": true, "SR-PM": true, "SR-PR": true, "SR-SA": true,
	"SR-SI": true, "SR-WA": true, "SS-BN": true, "SS-BW": true, "SS-EC": true,
	"SS-EE8": true, "SS-EW": true, "SS-JG": true, "SS-LK": true, "SS-NU": true,
	"SS-UY": true, "SS-WR": true, "ST-P": true, "ST-S": true, "SV-AH": true,
	"SV-CA": true, "SV-CH": true, "SV-CU": true, "SV-LI": true, "SV-MO": true,
	"SV-PA": true, "SV-SA": true, "SV-SM": true, "SV-SO": true, "SV-SS": true,
	"SV-SV": true, "SV-UN": true, "SV-US": true, "SY-DI": true, "SY-DR": true,
	"SY-DY": true, "SY-HA": true, "SY-HI": true, "SY-HL": true, "SY-HM": true,
	"SY-ID": true, "SY-LA": true, "SY-QU": true, "SY-RA": true, "SY-RD": true,
	"SY-SU": true, "SY-TA": true, "SZ-HH": true, "SZ-LU": true, "SZ-MA": true,
	"SZ-SH": true, "TD-BA": true, "TD-BG": true, "TD-BO": true, "TD-CB": true,
	"TD-EN": true, "TD-GR": true, "TD-HL": true, "TD-KA": true, "TD-LC": true,
	"TD-LO": true, "TD-LR": true, "TD-MA": true, "TD-MC": true, "TD-ME": true,
	"TD-MO": true, "TD-ND": true, "TD-OD": true, "TD-SA": true, "TD-SI": true,
	"TD-TA": true, "TD-TI": true, "TD-WF": true, "TG-C": true, "TG-K": true,
	"TG-M": true, "TG-P": true, "TG-S": true, "TH-10": true, "TH-11": true,
	"TH-12": true, "TH-13": true, "TH-14": true, "TH-15": true, "TH-16": true,
	"TH-17": true, "TH-18": true, "TH-19": true, "TH-20": true, "TH-21": true,
	"TH-22": true, "TH-23": true, "TH-24": true, "TH-25": true, "TH-26": true,
	"TH-27": true, "TH-30": true, "TH-31": true, "TH-32": true, "TH-33": true,
	"TH-34": true, "TH-35": true, "TH-36": true, "TH-37": true, "TH-39": true,
	"TH-40": true, "TH-41": true, "TH-42": true, "TH-43": true, "TH-44": true,
	"TH-45": true, "TH-46": true, "TH-47": true, "TH-48": true, "TH-49": true,
	"TH-50": true, "TH-51": true, "TH-52": true, "TH-53": true, "TH-54": true,
	"TH-55": true, "TH-56": true, "TH-57": true, "TH-58": true, "TH-60": true,
	"TH-61": true, "TH-62": true, "TH-63": true, "TH-64": true, "TH-65": true,
	"TH-66": true, "TH-67": true, "TH-70": true, "TH-71": true, "TH-72": true,
	"TH-73": true, "TH-74": true, "TH-75": true, "TH-76": true, "TH-77": true,
	"TH-80": true, "TH-81": true, "TH-82": true, "TH-83": true, "TH-84": true,
	"TH-85": true, "TH-86": true, "TH-90": true, "TH-91": true, "TH-92": true,
	"TH-93": true, "TH-94": true, "TH-95": true, "TH-96": true, "TH-S": true,
	"TJ-GB": true, "TJ-KT": true, "TJ-SU": true, "TL-AL": true, "TL-AN": true,
	"TL-BA": true, "TL-BO": true, "TL-CO": true, "TL-DI": true, "TL-ER": true,
	"TL-LA": true, "TL-LI": true, "TL-MF": true, "TL-MT": true, "TL-OE": true,
	"TL-VI": true, "TM-A": true, "TM-B": true, "TM-D": true, "TM-L": true,
	"TM-M": true, "TM-S": true, "TN-11": true, "TN-12": true, "TN-13": true,
	"TN-14": true, "TN-21": true, "TN-22": true, "TN-23": true, "TN-31": true,
	"TN-32": true, "TN-33": true, "TN-34": true, "TN-41": true, "TN-42": true,
	"TN-43": true, "TN-51": true, "TN-52": true, "TN-53": true, "TN-61": true,
	"TN-71": true, "TN-72": true, "TN-73": true, "TN-81": true, "TN-82": true,
	"TN-83": true, "TO-01": true, "TO-02": true, "TO-03": true, "TO-04": true,
	"TO-05": true, "TR-01": true, "TR-02": true, "TR-03": true, "TR-04": true,
	"TR-05": true, "TR-06": true, "TR-07": true, "TR-08": true, "TR-09": true,
	"TR-10": true, "TR-11": true, "TR-12": true, "TR-13": true, "TR-14": true,
	"TR-15": true, "TR-16": true, "TR-17": true, "TR-18": true, "TR-19": true,
	"TR-20": true, "TR-21": true, "TR-22": true, "TR-23": true, "TR-24": true,
	"TR-25": true, "TR-26": true, "TR-27": true, "TR-28": true, "TR-29": true,
	"TR-30": true, "TR-31": true, "TR-32": true, "TR-33": true, "TR-34": true,
	"TR-35": true, "TR-36": true, "TR-37": true, "TR-38": true, "TR-39": true,
	"TR-40": true, "TR-41": true, "TR-42": true, "TR-43": true, "TR-44": true,
	"TR-45": true, "TR-46": true, "TR-47": true, "TR-48": true, "TR-49": true,
	"TR-50": true, "TR-51": true, "TR-52": true, "TR-53": true, "TR-54": true,
	"TR-55": true, "TR-56": true, "TR-57": true, "TR-58": true, "TR-59": true,
	"TR-60": true, "TR-61": true, "TR-62": true, "TR-63": true, "TR-64": true,
	"TR-65": true, "TR-66": true, "TR-67": true, "TR-68": true, "TR-69": true,
	"TR-70": true, "TR-71": true, "TR-72": true, "TR-73": true, "TR-74": true,
	"TR-75": true, "TR-76": true, "TR-77": true, "TR-78": true, "TR-79": true,
	"TR-80": true, "TR-81": true, "TT-ARI": true, "TT-CHA": true, "TT-CTT": true,
	"TT-DMN": true, "TT-ETO": true, "TT-PED": true, "TT-POS": true, "TT-PRT": true,
	"TT-PTF": true, "TT-RCM": true, "TT-SFO": true, "TT-SGE": true, "TT-SIP": true,
	"TT-SJL": true, "TT-TUP": true, "TT-WTO": true, "TV-FUN": true, "TV-NIT": true,
	"TV-NKF": true, "TV-NKL": true, "TV-NMA": true, "TV-NMG": true, "TV-NUI": true,
	"TV-VAI": true, "TW-CHA": true, "TW-CYI": true, "TW-CYQ": true, "TW-HSQ": true,
	"TW-HSZ": true, "TW-HUA": true, "TW-ILA": true, "TW-KEE": true, "TW-KHH": true,
	"TW-KHQ": true, "TW-MIA": true, "TW-NAN": true, "TW-PEN": true, "TW-PIF": true,
	"TW-TAO": true, "TW-TNN": true, "TW-TNQ": true, "TW-TPE": true, "TW-TPQ": true,
	"TW-TTT": true, "TW-TXG": true, "TW-TXQ": true, "TW-YUN": true, "TZ-01": true,
	"TZ-02": true, "TZ-03": true, "TZ-04": true, "TZ-05": true, "TZ-06": true,
	"TZ-07": true, "TZ-08": true, "TZ-09": true, "TZ-10": true, "TZ-11": true,
	"TZ-12": true, "TZ-13": true, "TZ-14": true, "TZ-15": true, "TZ-16": true,
	"TZ-17": true, "TZ-18": true, "TZ-19": true, "TZ-20": true, "TZ-21": true,
	"TZ-22": true, "TZ-23": true, "TZ-24": true, "TZ-25": true, "TZ-26": true,
	"UA-05": true, "UA-07": true, "UA-09": true, "UA-12": true, "UA-14": true,
	"UA-18": true, "UA-21": true, "UA-23": true, "UA-26": true, "UA-30": true,
	"UA-32": true, "UA-35": true, "UA-40": true, "UA-43": true, "UA-46": true,
	"UA-48": true, "UA-51": true, "UA-53": true, "UA-56": true, "UA-59": true,
	"UA-61": true, "UA-63": true, "UA-65": true, "UA-68": true, "UA-71": true,
	"UA-74": true, "UA-77": true, "UG-101": true, "UG-102": true, "UG-103": true,
	"UG-104": true, "UG-105": true, "UG-106": true, "UG-107": true, "UG-108": true,
	"UG-109": true, "UG-110": true, "UG-111": true, "UG-112": true, "UG-113": true,
	"UG-114": true, "UG-115": true, "UG-116": true, "UG-201": true, "UG-202": true,
	"UG-203": true, "UG-204": true, "UG-205": true, "UG-206": true, "UG-207": true,
	"UG-208": true, "UG-209": true, "UG-210": true, "UG-211": true, "UG-212": true,
	"UG-213": true, "UG-214": true, "UG-215": true, "UG-216": true, "UG-217": true,
	"UG-218": true, "UG-219": true, "UG-220": true, "UG-221": true, "UG-222": true,
	"UG-223": true, "UG-224": true, "UG-301": true, "UG-302": true, "UG-303": true,
	"UG-304": true, "UG-305": true, "UG-306": true, "UG-307": true, "UG-308": true,
	"UG-309": true, "UG-310": true, "UG-311": true, "UG-312": true, "UG-313": true,
	"UG-314": true, "UG-315": true, "UG-316": true, "UG-317": true, "UG-318": true,
	"UG-319": true, "UG-320": true, "UG-321": true, "UG-401": true, "UG-402": true,
	"UG-403": true, "UG-404": true, "UG-405": true, "UG-406": true, "UG-407": true,
	"UG-408": true, "UG-409": true, "UG-410": true, "UG-411": true, "UG-412": true,
	"UG-413": true, "UG-414": true, "UG-415": true, "UG-416": true, "UG-417": true,
	"UG-418": true, "UG-419": true, "UG-C": true, "UG-E": true, "UG-N": true,
	"UG-W": true, "UM-67": true, "UM-71": true, "UM-76": true, "UM-79": true,
	"UM-81": true, "UM-84": true, "UM-86": true, "UM-89": true, "UM-95": true,
	"US-AK": true, "US-AL": true, "US-AR": true, "US-AS": true, "US-AZ": true,
	"US-CA": true, "US-CO": true, "US-CT": true, "US-DC": true, "US-DE": true,
	"US-FL": true, "US-GA": true, "US-GU": true, "US-HI": true, "US-IA": true,
	"US-ID": true, "US-IL": true, "US-IN": true, "US-KS": true, "US-KY": true,
	"US-LA": true, "US-MA": true, "US-MD": true, "US-ME": true, "US-MI": true,
	"US-MN": true, "US-MO": true, "US-MP": true, "US-MS": true, "US-MT": true,
	"US-NC": true, "US-ND": true, "US-NE": true, "US-NH": true, "US-NJ": true,
	"US-NM": true, "US-NV": true, "US-NY": true, "US-OH": true, "US-OK": true,
	"US-OR": true, "US-PA": true, "US-PR": true, "US-RI": true, "US-SC": true,
	"US-SD": true, "US-TN": true, "US-TX": true, "US-UM": true, "US-UT": true,
	"US-VA": true, "US-VI": true, "US-VT": true, "US-WA": true, "US-WI": true,
	"US-WV": true, "US-WY": true, "UY-AR": true, "UY-CA": true, "UY-CL": true,
	"UY-CO": true, "UY-DU": true, "UY-FD": true, "UY-FS": true, "UY-LA": true,
	"UY-MA": true, "UY-MO": true, "UY-PA": true, "UY-RN": true, "UY-RO": true,
	"UY-RV": true, "UY-SA": true, "UY-SJ": true, "UY-SO": true, "UY-TA": true,
	"UY-TT": true, "UZ-AN": true, "UZ-BU": true, "UZ-FA": true, "UZ-JI": true,
	"UZ-NG": true, "UZ-NW": true, "UZ-QA": true, "UZ-QR": true, "UZ-SA": true,
	"UZ-SI": true, "UZ-SU": true, "UZ-TK": true, "UZ-TO": true, "UZ-XO": true,
	"VC-01": true, "VC-02": true, "VC-03": true, "VC-04": true, "VC-05": true,
	"VC-06": true, "VE-A": true, "VE-B": true, "VE-C": true, "VE-D": true,
	"VE-E": true, "VE-F": true, "VE-G": true, "VE-H": true, "VE-I": true,
	"VE-J": true, "VE-K": true, "VE-L": true, "VE-M": true, "VE-N": true,
	"VE-O": true, "VE-P": true, "VE-R": true, "VE-S": true, "VE-T": true,
	"VE-U": true, "VE-V": true, "VE-W": true, "VE-X": true, "VE-Y": true,
	"VE-Z": true, "VN-01": true, "VN-02": true, "VN-03": true, "VN-04": true,
	"VN-05": true, "VN-06": true, "VN-07": true, "VN-09": true, "VN-13": true,
	"VN-14": true, "VN-15": true, "VN-18": true, "VN-20": true, "VN-21": true,
	"VN-22": true, "VN-23": true, "VN-24": true, "VN-25": true, "VN-26": true,
	"VN-27": true, "VN-28": true, "VN-29": true, "VN-30": true, "VN-31": true,
	"VN-32": true, "VN-33": true, "VN-34": true, "VN-35": true, "VN-36": true,
	"VN-37": true, "VN-39": true, "VN-40": true, "VN-41": true, "VN-43": true,
	"VN-44": true, "VN-45": true, "VN-46": true, "VN-47": true, "VN-49": true,
	"VN-50": true, "VN-51": true, "VN-52": true, "VN-53": true, "VN-54": true,
	"VN-55": true, "VN-56": true, "VN-57": true, "VN-58": true, "VN-59": true,
	"VN-61": true, "VN-63": true, "VN-66": true, "VN-67": true, "VN-68": true,
	"VN-69": true, "VN-70": true, "VN-71": true, "VN-72": true, "VN-73": true,
	"VN-CT": true, "VN-DN": true, "VN-HN": true, "VN-HP": true, "VN-SG": true,
	"VU-MAP": true, "VU-PAM": true, "VU-SAM": true, "VU-SEE": true, "VU-TAE": true,
	"VU-TOB": true, "WS-AA": true, "WS-AL": true, "WS-AT": true, "WS-FA": true,
	"WS-GE": true, "WS-GI": true, "WS-PA": true, "WS-SA": true, "WS-TU": true,
	"WS-VF": true, "WS-VS": true, "YE-AB": true, "YE-AD": true, "YE-AM": true,
	"YE-BA": true, "YE-DA": true, "YE-DH": true, "YE-HD": true, "YE-HJ": true,
	"YE-IB": true, "YE-JA": true, "YE-LA": true, "YE-MA": true, "YE-MR": true,
	"YE-MU": true, "YE-MW": true, "YE-RA": true, "YE-SD": true, "YE-SH": true,
	"YE-SN": true, "YE-TA": true, "ZA-EC": true, "ZA-FS": true, "ZA-GP": true,
	"ZA-LP": true, "ZA-MP": true, "ZA-NC": true, "ZA-NW": true, "ZA-WC": true,
	"ZA-ZN": true, "ZM-01": true, "ZM-02": true, "ZM-03": true, "ZM-04": true,
	"ZM-05": true, "ZM-06": true, "ZM-07": true, "ZM-08": true, "ZM-09": true,
	"ZW-BU": true, "ZW-HA": true, "ZW-MA": true, "ZW-MC": true, "ZW-ME": true,
	"ZW-MI": true, "ZW-MN": true, "ZW-MS": true, "ZW-MV": true, "ZW-MW": true,
}

var iso4217 = map[string]bool{
	"AFN": true, "EUR": true, "ALL": true, "DZD": true, "USD": true,
	"AOA": true, "XCD": true, "ARS": true, "AMD": true, "AWG": true,
	"AUD": true, "AZN": true, "BSD": true, "BHD": true, "BDT": true,
	"BBD": true, "BYN": true, "BZD": true, "XOF": true, "BMD": true,
	"INR": true, "BTN": true, "BOB": true, "BOV": true, "BAM": true,
	"BWP": true, "NOK": true, "BRL": true, "BND": true, "BGN": true,
	"BIF": true, "CVE": true, "KHR": true, "XAF": true, "CAD": true,
	"KYD": true, "CLP": true, "CLF": true, "CNY": true, "COP": true,
	"COU": true, "KMF": true, "CDF": true, "NZD": true, "CRC": true,
	"HRK": true, "CUP": true, "CUC": true, "ANG": true, "CZK": true,
	"DKK": true, "DJF": true, "DOP": true, "EGP": true, "SVC": true,
	"ERN": true, "SZL": true, "ETB": true, "FKP": true, "FJD": true,
	"XPF": true, "GMD": true, "GEL": true, "GHS": true, "GIP": true,
	"GTQ": true, "GBP": true, "GNF": true, "GYD": true, "HTG": true,
	"HNL": true, "HKD": true, "HUF": true, "ISK": true, "IDR": true,
	"XDR": true, "IRR": true, "IQD": true, "ILS": true, "JMD": true,
	"JPY": true, "JOD": true, "KZT": true, "KES": true, "KPW": true,
	"KRW": true, "KWD": true, "KGS": true, "LAK": true, "LBP": true,
	"LSL": true, "ZAR": true, "LRD": true, "LYD": true, "CHF": true,
	"MOP": true, "MKD": true, "MGA": true, "MWK": true, "MYR": true,
	"MVR": true, "MRU": true, "MUR": true, "XUA": true, "MXN": true,
	"MXV": true, "MDL": true, "MNT": true, "MAD": true, "MZN": true,
	"MMK": true, "NAD": true, "NPR": true, "NIO": true, "NGN": true,
	"OMR": true, "PKR": true, "PAB": true, "PGK": true, "PYG": true,
	"PEN": true, "PHP": true, "PLN": true, "QAR": true, "RON": true,
	"RUB": true, "RWF": true, "SHP": true, "WST": true, "STN": true,
	"SAR": true, "RSD": true, "SCR": true, "SLL": true, "SGD": true,
	"XSU": true, "SBD": true, "SOS": true, "SSP": true, "LKR": true,
	"SDG": true, "SRD": true, "SEK": true, "CHE": true, "CHW": true,
	"SYP": true, "TWD": true, "TJS": true, "TZS": true, "THB": true,
	"TOP": true, "TTD": true, "TND": true, "TRY": true, "TMT": true,
	"UGX": true, "UAH": true, "AED": true, "USN": true, "UYU": true,
	"UYI": true, "UYW": true, "UZS": true, "VUV": true, "VES": true,
	"VND": true, "YER": true, "ZMW": true, "ZWL": true, "XBA": true,
	"XBB": true, "XBC": true, "XBD": true, "XTS": true, "XXX": true,
	"XAU": true, "XPD": true, "XPT": true, "XAG": true,
}

var iso4217_numeric = map[int]bool{
	8: true, 12: true, 32: true, 36: true, 44: true,
	48: true, 50: true, 51: true, 52: true, 60: true,
	64: true, 68: true, 72: true, 84: true, 90: true,
	96: true, 104: true, 108: true, 116: true, 124: true,
	132: true, 136: true, 144: true, 152: true, 156: true,
	170: true, 174: true, 188: true, 191: true, 192: true,
	203: true, 208: true, 214: true, 222: true, 230: true,
	232: true, 238: true, 242: true, 262: true, 270: true,
	292: true, 320: true, 324: true, 328: true, 332: true,
	340: true, 344: true, 348: true, 352: true, 356: true,
	360: true, 364: true, 368: true, 376: true, 388: true,
	392: true, 398: true, 400: true, 404: true, 408: true,
	410: true, 414: true, 417: true, 418: true, 422: true,
	426: true, 430: true, 434: true, 446: true, 454: true,
	458: true, 462: true, 480: true, 484: true, 496: true,
	498: true, 504: true, 512: true, 516: true, 524: true,
	532: true, 533: true, 548: true, 554: true, 558: true,
	566: true, 578: true, 586: true, 590: true, 598: true,
	600: true, 604: true, 608: true, 634: true, 643: true,
	646: true, 654: true, 682: true, 690: true, 694: true,
	702: true, 704: true, 706: true, 710: true, 728: true,
	748: true, 752: true, 756: true, 760: true, 764: true,
	776: true, 780: true, 784: true, 788: true, 800: true,
	807: true, 818: true, 826: true, 834: true, 840: true,
	858: true, 860: true, 882: true, 886: true, 901: true,
	927: true, 928: true, 929: true, 930: true, 931: true,
	932: true, 933: true, 934: true, 936: true, 938: true,
	940: true, 941: true, 943: true, 944: true, 946: true,
	947: true, 948: true, 949: true, 950: true, 951: true,
	952: true, 953: true, 955: true, 956: true, 957: true,
	958: true, 959: true, 960: true, 961: true, 962: true,
	963: true, 964: true, 965: true, 967: true, 968: true,
	969: true, 970: true, 971: true, 972: true, 973: true,
	975: true, 976: true, 977: true, 978: true, 979: true,
	980: true, 981: true, 984: true, 985: true, 986: true,
	990: true, 994: true, 997: true, 999: true,
}

const (
	fieldErrMsg = "Key: '%s' Error:Field validation for '%s' failed on the '%s' tag"
)

// ValidationErrorsTranslations is the translation return type
type ValidationErrorsTranslations map[string]string

// InvalidValidationError describes an invalid argument passed to
// `Struct`, `StructExcept`, StructPartial` or `Field`
type InvalidValidationError struct {
	Type reflect.Type
}

// Error returns InvalidValidationError message
func (e *InvalidValidationError) Error() string {

	if e.Type == nil {
		return "validator: (nil)"
	}

	return "validator: (nil " + e.Type.String() + ")"
}

// ValidationErrors is an array of FieldError's
// for use in custom error messages post validation.
type ValidationErrors []FieldError

// Error is intended for use in development + debugging and not intended to be a production error message.
// It allows ValidationErrors to subscribe to the Error interface.
// All information to create an error message specific to your application is contained within
// the FieldError found within the ValidationErrors array

// Error is intended for use in development + debugging and not intended to be a production error message.
// It allows ValidationErrors to subscribe to the Error interface.
// All information to create an error message specific to your application is contained within
// the FieldError found within the ValidationErrors array
func (ve ValidationErrors) Error() string {

	buff := bytes.NewBufferString("")

	var fe *fieldError

	for i := 0; i < len(ve); i++ {

		fe = ve[i].(*fieldError)
		buff.WriteString(fe.Error())
		buff.WriteString("\n")
	}

	return strings.TrimSpace(buff.String())
}

// Translate translates all of the ValidationErrors
func (ve ValidationErrors) Translate(ut ut.Translator) ValidationErrorsTranslations {

	trans := make(ValidationErrorsTranslations)

	var fe *fieldError

	for i := 0; i < len(ve); i++ {
		fe = ve[i].(*fieldError)

		// // in case an Anonymous struct was used, ensure that the key
		// // would be 'Username' instead of ".Username"
		// if len(fe.ns) > 0 && fe.ns[:1] == "." {
		// 	trans[fe.ns[1:]] = fe.Translate(ut)
		// 	continue
		// }

		trans[fe.ns] = fe.Translate(ut)
	}

	return trans
}

// FieldError contains all functions to get error details
type FieldError interface {

	// Tag returns the validation tag that failed. if the
	// validation was an alias, this will return the
	// alias name and not the underlying tag that failed.
	//
	// eg. alias "iscolor": "hexcolor|rgb|rgba|hsl|hsla"
	// will return "iscolor"
	Tag() string

	// ActualTag returns the validation tag that failed, even if an
	// alias the actual tag within the alias will be returned.
	// If an 'or' validation fails the entire or will be returned.
	//
	// eg. alias "iscolor": "hexcolor|rgb|rgba|hsl|hsla"
	// will return "hexcolor|rgb|rgba|hsl|hsla"
	ActualTag() string

	// Namespace returns the namespace for the field error, with the tag
	// name taking precedence over the field's actual name.
	//
	// eg. JSON name "User.fname"
	//
	// See StructNamespace() for a version that returns actual names.
	//
	// NOTE: this field can be blank when validating a single primitive field
	// using validate.Field(...) as there is no way to extract it's name
	Namespace() string

	// StructNamespace returns the namespace for the field error, with the field's
	// actual name.
	//
	// eq. "User.FirstName" see Namespace for comparison
	//
	// NOTE: this field can be blank when validating a single primitive field
	// using validate.Field(...) as there is no way to extract its name
	StructNamespace() string

	// Field returns the fields name with the tag name taking precedence over the
	// field's actual name.
	//
	// eq. JSON name "fname"
	// see StructField for comparison
	Field() string

	// StructField returns the field's actual name from the struct, when able to determine.
	//
	// eq.  "FirstName"
	// see Field for comparison
	StructField() string

	// Value returns the actual field's value in case needed for creating the error
	// message
	Value() interface{}

	// Param returns the param value, in string form for comparison; this will also
	// help with generating an error message
	Param() string

	// Kind returns the Field's reflect Kind
	//
	// eg. time.Time's kind is a struct
	Kind() reflect.Kind

	// Type returns the Field's reflect Type
	//
	// eg. time.Time's type is time.Time
	Type() reflect.Type

	// Translate returns the FieldError's translated error
	// from the provided 'ut.Translator' and registered 'TranslationFunc'
	//
	// NOTE: if no registered translator can be found it returns the same as
	// calling fe.Error()
	Translate(ut ut.Translator) string

	// Error returns the FieldError's message
	Error() string
}

// compile time interface checks
var _ FieldError = new(fieldError)

var _ error = new(fieldError)

// fieldError contains a single field's validation error along
// with other properties that may be needed for error message creation
// it complies with the FieldError interface
type fieldError struct {
	v              *Validate
	tag            string
	actualTag      string
	ns             string
	structNs       string
	fieldLen       uint8
	structfieldLen uint8
	value          interface{}
	param          string
	kind           reflect.Kind
	typ            reflect.Type
}

// Tag returns the validation tag that failed.
func (fe *fieldError) Tag() string {
	return fe.tag
}

// ActualTag returns the validation tag that failed, even if an
// alias the actual tag within the alias will be returned.
func (fe *fieldError) ActualTag() string {
	return fe.actualTag
}

// Namespace returns the namespace for the field error, with the tag
// name taking precedence over the field's actual name.
func (fe *fieldError) Namespace() string {
	return fe.ns
}

// StructNamespace returns the namespace for the field error, with the field's
// actual name.
func (fe *fieldError) StructNamespace() string {
	return fe.structNs
}

// Field returns the field's name with the tag name taking precedence over the
// field's actual name.
func (fe *fieldError) Field() string {

	return fe.ns[len(fe.ns)-int(fe.fieldLen):]
	// // return fe.field
	// fld := fe.ns[len(fe.ns)-int(fe.fieldLen):]

	// log.Println("FLD:", fld)

	// if len(fld) > 0 && fld[:1] == "." {
	// 	return fld[1:]
	// }

	// return fld
}

// StructField returns the field's actual name from the struct, when able to determine.
func (fe *fieldError) StructField() string {
	// return fe.structField
	return fe.structNs[len(fe.structNs)-int(fe.structfieldLen):]
}

// Value returns the actual field's value in case needed for creating the error
// message
func (fe *fieldError) Value() interface{} {
	return fe.value
}

// Param returns the param value, in string form for comparison; this will
// also help with generating an error message
func (fe *fieldError) Param() string {
	return fe.param
}

// Kind returns the Field's reflect Kind
func (fe *fieldError) Kind() reflect.Kind {
	return fe.kind
}

// Type returns the Field's reflect Type
func (fe *fieldError) Type() reflect.Type {
	return fe.typ
}

// Error returns the fieldError's error message
func (fe *fieldError) Error() string {
	return fmt.Sprintf(fieldErrMsg, fe.ns, fe.Field(), fe.tag)
}

// Translate returns the FieldError's translated error
// from the provided 'ut.Translator' and registered 'TranslationFunc'
//
// NOTE: if no registered translation can be found, it returns the original
// untranslated error message.
func (fe *fieldError) Translate(ut ut.Translator) string {

	m, ok := fe.v.transTagFunc[ut]
	if !ok {
		return fe.Error()
	}

	fn, ok := m[fe.tag]
	if !ok {
		return fe.Error()
	}

	return fn(ut, fe)
}

// FieldLevel contains all the information and helper functions
// to validate a field
type FieldLevel interface {

	// Top returns the top level struct, if any
	Top() reflect.Value

	// Parent returns the current fields parent struct, if any or
	// the comparison value if called 'VarWithValue'
	Parent() reflect.Value

	// Field returns current field for validation
	Field() reflect.Value

	// FieldName returns the field's name with the tag
	// name taking precedence over the fields actual name.
	FieldName() string

	// StructFieldName returns the struct field's name
	StructFieldName() string

	// Param returns param for validation against current field
	Param() string

	// GetTag returns the current validations tag name
	GetTag() string

	// ExtractType gets the actual underlying type of field value.
	// It will dive into pointers, customTypes and return you the
	// underlying value and it's kind.
	ExtractType(field reflect.Value) (value reflect.Value, kind reflect.Kind, nullable bool)

	// GetStructFieldOK traverses the parent struct to retrieve a specific field denoted by the provided namespace
	// in the param and returns the field, field kind and whether is was successful in retrieving
	// the field at all.
	//
	// NOTE: when not successful ok will be false, this can happen when a nested struct is nil and so the field
	// could not be retrieved because it didn't exist.
	//
	//Deprecated: Use GetStructFieldOK2() instead which also return if the value is nullable.
	GetStructFieldOK() (reflect.Value, reflect.Kind, bool)

	// GetStructFieldOKAdvanced is the same as GetStructFieldOK except that it accepts the parent struct to start looking for
	// the field and namespace allowing more extensibility for validators.
	//
	// Deprecated: Use GetStructFieldOKAdvanced2() instead which also return if the value is nullable.
	GetStructFieldOKAdvanced(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool)

	// GetStructFieldOK2 traverses the parent struct to retrieve a specific field denoted by the provided namespace
	// in the param and returns the field, field kind, if it's a nullable type and whether is was successful in retrieving
	// the field at all.
	//
	// NOTE: when not successful ok will be false, this can happen when a nested struct is nil and so the field
	// could not be retrieved because it didn't exist.
	GetStructFieldOK2() (reflect.Value, reflect.Kind, bool, bool)

	// GetStructFieldOKAdvanced2 is the same as GetStructFieldOK except that it accepts the parent struct to start looking for
	// the field and namespace allowing more extensibility for validators.
	GetStructFieldOKAdvanced2(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool, bool)
}

var _ FieldLevel = new(validate)

// Field returns current field for validation
func (v *validate) Field() reflect.Value {
	return v.flField
}

// FieldName returns the field's name with the tag
// name taking precedence over the fields actual name.
func (v *validate) FieldName() string {
	return v.cf.altName
}

// GetTag returns the current validations tag name
func (v *validate) GetTag() string {
	return v.ct.tag
}

// StructFieldName returns the struct field's name
func (v *validate) StructFieldName() string {
	return v.cf.name
}

// Param returns param for validation against current field
func (v *validate) Param() string {
	return v.ct.param
}

// GetStructFieldOK returns Param returns param for validation against current field
//
// Deprecated: Use GetStructFieldOK2() instead which also return if the value is nullable.
func (v *validate) GetStructFieldOK() (reflect.Value, reflect.Kind, bool) {
	current, kind, _, found := v.getStructFieldOKInternal(v.slflParent, v.ct.param)
	return current, kind, found
}

// GetStructFieldOKAdvanced is the same as GetStructFieldOK except that it accepts the parent struct to start looking for
// the field and namespace allowing more extensibility for validators.
//
// Deprecated: Use GetStructFieldOKAdvanced2() instead which also return if the value is nullable.
func (v *validate) GetStructFieldOKAdvanced(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool) {
	current, kind, _, found := v.GetStructFieldOKAdvanced2(val, namespace)
	return current, kind, found
}

// GetStructFieldOK2 returns Param returns param for validation against current field
func (v *validate) GetStructFieldOK2() (reflect.Value, reflect.Kind, bool, bool) {
	return v.getStructFieldOKInternal(v.slflParent, v.ct.param)
}

// GetStructFieldOKAdvanced2 is the same as GetStructFieldOK except that it accepts the parent struct to start looking for
// the field and namespace allowing more extensibility for validators.
func (v *validate) GetStructFieldOKAdvanced2(val reflect.Value, namespace string) (reflect.Value, reflect.Kind, bool, bool) {
	return v.getStructFieldOKInternal(val, namespace)
}

var postCodePatternDict = map[string]string{
	"GB": `^GIR[ ]?0AA|((AB|AL|B|BA|BB|BD|BH|BL|BN|BR|BS|BT|CA|CB|CF|CH|CM|CO|CR|CT|CV|CW|DA|DD|DE|DG|DH|DL|DN|DT|DY|E|EC|EH|EN|EX|FK|FY|G|GL|GY|GU|HA|HD|HG|HP|HR|HS|HU|HX|IG|IM|IP|IV|JE|KA|KT|KW|KY|L|LA|LD|LE|LL|LN|LS|LU|M|ME|MK|ML|N|NE|NG|NN|NP|NR|NW|OL|OX|PA|PE|PH|PL|PO|PR|RG|RH|RM|S|SA|SE|SG|SK|SL|SM|SN|SO|SP|SR|SS|ST|SW|SY|TA|TD|TF|TN|TQ|TR|TS|TW|UB|W|WA|WC|WD|WF|WN|WR|WS|WV|YO|ZE)(\d[\dA-Z]?[ ]?\d[ABD-HJLN-UW-Z]{2}))|BFPO[ ]?\d{1,4}$`,
	"JE": `^JE\d[\dA-Z]?[ ]?\d[ABD-HJLN-UW-Z]{2}$`,
	"GG": `^GY\d[\dA-Z]?[ ]?\d[ABD-HJLN-UW-Z]{2}$`,
	"IM": `^IM\d[\dA-Z]?[ ]?\d[ABD-HJLN-UW-Z]{2}$`,
	"US": `^\d{5}([ \-]\d{4})?$`,
	"CA": `^[ABCEGHJKLMNPRSTVXY]\d[ABCEGHJ-NPRSTV-Z][ ]?\d[ABCEGHJ-NPRSTV-Z]\d$`,
	"DE": `^\d{5}$`,
	"JP": `^\d{3}-\d{4}$`,
	"FR": `^\d{2}[ ]?\d{3}$`,
	"AU": `^\d{4}$`,
	"IT": `^\d{5}$`,
	"CH": `^\d{4}$`,
	"AT": `^\d{4}$`,
	"ES": `^\d{5}$`,
	"NL": `^\d{4}[ ]?[A-Z]{2}$`,
	"BE": `^\d{4}$`,
	"DK": `^\d{4}$`,
	"SE": `^\d{3}[ ]?\d{2}$`,
	"NO": `^\d{4}$`,
	"BR": `^\d{5}[\-]?\d{3}$`,
	"PT": `^\d{4}([\-]\d{3})?$`,
	"FI": `^\d{5}$`,
	"AX": `^22\d{3}$`,
	"KR": `^\d{3}[\-]\d{3}$`,
	"CN": `^\d{6}$`,
	"TW": `^\d{3}(\d{2})?$`,
	"SG": `^\d{6}$`,
	"DZ": `^\d{5}$`,
	"AD": `^AD\d{3}$`,
	"AR": `^([A-HJ-NP-Z])?\d{4}([A-Z]{3})?$`,
	"AM": `^(37)?\d{4}$`,
	"AZ": `^\d{4}$`,
	"BH": `^((1[0-2]|[2-9])\d{2})?$`,
	"BD": `^\d{4}$`,
	"BB": `^(BB\d{5})?$`,
	"BY": `^\d{6}$`,
	"BM": `^[A-Z]{2}[ ]?[A-Z0-9]{2}$`,
	"BA": `^\d{5}$`,
	"IO": `^BBND 1ZZ$`,
	"BN": `^[A-Z]{2}[ ]?\d{4}$`,
	"BG": `^\d{4}$`,
	"KH": `^\d{5}$`,
	"CV": `^\d{4}$`,
	"CL": `^\d{7}$`,
	"CR": `^\d{4,5}|\d{3}-\d{4}$`,
	"HR": `^\d{5}$`,
	"CY": `^\d{4}$`,
	"CZ": `^\d{3}[ ]?\d{2}$`,
	"DO": `^\d{5}$`,
	"EC": `^([A-Z]\d{4}[A-Z]|(?:[A-Z]{2})?\d{6})?$`,
	"EG": `^\d{5}$`,
	"EE": `^\d{5}$`,
	"FO": `^\d{3}$`,
	"GE": `^\d{4}$`,
	"GR": `^\d{3}[ ]?\d{2}$`,
	"GL": `^39\d{2}$`,
	"GT": `^\d{5}$`,
	"HT": `^\d{4}$`,
	"HN": `^(?:\d{5})?$`,
	"HU": `^\d{4}$`,
	"IS": `^\d{3}$`,
	"IN": `^\d{6}$`,
	"ID": `^\d{5}$`,
	"IL": `^\d{5}$`,
	"JO": `^\d{5}$`,
	"KZ": `^\d{6}$`,
	"KE": `^\d{5}$`,
	"KW": `^\d{5}$`,
	"LA": `^\d{5}$`,
	"LV": `^\d{4}$`,
	"LB": `^(\d{4}([ ]?\d{4})?)?$`,
	"LI": `^(948[5-9])|(949[0-7])$`,
	"LT": `^\d{5}$`,
	"LU": `^\d{4}$`,
	"MK": `^\d{4}$`,
	"MY": `^\d{5}$`,
	"MV": `^\d{5}$`,
	"MT": `^[A-Z]{3}[ ]?\d{2,4}$`,
	"MU": `^(\d{3}[A-Z]{2}\d{3})?$`,
	"MX": `^\d{5}$`,
	"MD": `^\d{4}$`,
	"MC": `^980\d{2}$`,
	"MA": `^\d{5}$`,
	"NP": `^\d{5}$`,
	"NZ": `^\d{4}$`,
	"NI": `^((\d{4}-)?\d{3}-\d{3}(-\d{1})?)?$`,
	"NG": `^(\d{6})?$`,
	"OM": `^(PC )?\d{3}$`,
	"PK": `^\d{5}$`,
	"PY": `^\d{4}$`,
	"PH": `^\d{4}$`,
	"PL": `^\d{2}-\d{3}$`,
	"PR": `^00[679]\d{2}([ \-]\d{4})?$`,
	"RO": `^\d{6}$`,
	"RU": `^\d{6}$`,
	"SM": `^4789\d$`,
	"SA": `^\d{5}$`,
	"SN": `^\d{5}$`,
	"SK": `^\d{3}[ ]?\d{2}$`,
	"SI": `^\d{4}$`,
	"ZA": `^\d{4}$`,
	"LK": `^\d{5}$`,
	"TJ": `^\d{6}$`,
	"TH": `^\d{5}$`,
	"TN": `^\d{4}$`,
	"TR": `^\d{5}$`,
	"TM": `^\d{6}$`,
	"UA": `^\d{5}$`,
	"UY": `^\d{5}$`,
	"UZ": `^\d{6}$`,
	"VA": `^00120$`,
	"VE": `^\d{4}$`,
	"ZM": `^\d{5}$`,
	"AS": `^96799$`,
	"CC": `^6799$`,
	"CK": `^\d{4}$`,
	"RS": `^\d{6}$`,
	"ME": `^8\d{4}$`,
	"CS": `^\d{5}$`,
	"YU": `^\d{5}$`,
	"CX": `^6798$`,
	"ET": `^\d{4}$`,
	"FK": `^FIQQ 1ZZ$`,
	"NF": `^2899$`,
	"FM": `^(9694[1-4])([ \-]\d{4})?$`,
	"GF": `^9[78]3\d{2}$`,
	"GN": `^\d{3}$`,
	"GP": `^9[78][01]\d{2}$`,
	"GS": `^SIQQ 1ZZ$`,
	"GU": `^969[123]\d([ \-]\d{4})?$`,
	"GW": `^\d{4}$`,
	"HM": `^\d{4}$`,
	"IQ": `^\d{5}$`,
	"KG": `^\d{6}$`,
	"LR": `^\d{4}$`,
	"LS": `^\d{3}$`,
	"MG": `^\d{3}$`,
	"MH": `^969[67]\d([ \-]\d{4})?$`,
	"MN": `^\d{6}$`,
	"MP": `^9695[012]([ \-]\d{4})?$`,
	"MQ": `^9[78]2\d{2}$`,
	"NC": `^988\d{2}$`,
	"NE": `^\d{4}$`,
	"VI": `^008(([0-4]\d)|(5[01]))([ \-]\d{4})?$`,
	"VN": `^[0-9]{1,6}$`,
	"PF": `^987\d{2}$`,
	"PG": `^\d{3}$`,
	"PM": `^9[78]5\d{2}$`,
	"PN": `^PCRN 1ZZ$`,
	"PW": `^96940$`,
	"RE": `^9[78]4\d{2}$`,
	"SH": `^(ASCN|STHL) 1ZZ$`,
	"SJ": `^\d{4}$`,
	"SO": `^\d{5}$`,
	"SZ": `^[HLMS]\d{3}$`,
	"TC": `^TKCA 1ZZ$`,
	"WF": `^986\d{2}$`,
	"XK": `^\d{5}$`,
	"YT": `^976\d{2}$`,
}

var postCodeRegexDict = map[string]*regexp.Regexp{}

func init() {
	for countryCode, pattern := range postCodePatternDict {
		postCodeRegexDict[countryCode] = regexp.MustCompile(pattern)
	}
}

const (
	alphaRegexString                 = "^[a-zA-Z]+$"
	alphaNumericRegexString          = "^[a-zA-Z0-9]+$"
	alphaUnicodeRegexString          = "^[\\p{L}]+$"
	alphaUnicodeNumericRegexString   = "^[\\p{L}\\p{N}]+$"
	numericRegexString               = "^[-+]?[0-9]+(?:\\.[0-9]+)?$"
	numberRegexString                = "^[0-9]+$"
	hexadecimalRegexString           = "^(0[xX])?[0-9a-fA-F]+$"
	hexColorRegexString              = "^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$"
	rgbRegexString                   = "^rgb\\(\\s*(?:(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])|(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])%\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])%\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])%)\\s*\\)$"
	rgbaRegexString                  = "^rgba\\(\\s*(?:(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])|(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])%\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])%\\s*,\\s*(?:0|[1-9]\\d?|1\\d\\d?|2[0-4]\\d|25[0-5])%)\\s*,\\s*(?:(?:0.[1-9]*)|[01])\\s*\\)$"
	hslRegexString                   = "^hsl\\(\\s*(?:0|[1-9]\\d?|[12]\\d\\d|3[0-5]\\d|360)\\s*,\\s*(?:(?:0|[1-9]\\d?|100)%)\\s*,\\s*(?:(?:0|[1-9]\\d?|100)%)\\s*\\)$"
	hslaRegexString                  = "^hsla\\(\\s*(?:0|[1-9]\\d?|[12]\\d\\d|3[0-5]\\d|360)\\s*,\\s*(?:(?:0|[1-9]\\d?|100)%)\\s*,\\s*(?:(?:0|[1-9]\\d?|100)%)\\s*,\\s*(?:(?:0.[1-9]*)|[01])\\s*\\)$"
	emailRegexString                 = "^(?:(?:(?:(?:[a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(?:\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|(?:(?:\\x22)(?:(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(?:\\x20|\\x09)+)?(?:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(\\x20|\\x09)+)?(?:\\x22))))@(?:(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$"
	e164RegexString                  = "^\\+[1-9]?[0-9]{7,14}$"
	base64RegexString                = "^(?:[A-Za-z0-9+\\/]{4})*(?:[A-Za-z0-9+\\/]{2}==|[A-Za-z0-9+\\/]{3}=|[A-Za-z0-9+\\/]{4})$"
	base64URLRegexString             = "^(?:[A-Za-z0-9-_]{4})*(?:[A-Za-z0-9-_]{2}==|[A-Za-z0-9-_]{3}=|[A-Za-z0-9-_]{4})$"
	iSBN10RegexString                = "^(?:[0-9]{9}X|[0-9]{10})$"
	iSBN13RegexString                = "^(?:(?:97(?:8|9))[0-9]{10})$"
	uUID3RegexString                 = "^[0-9a-f]{8}-[0-9a-f]{4}-3[0-9a-f]{3}-[0-9a-f]{4}-[0-9a-f]{12}$"
	uUID4RegexString                 = "^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
	uUID5RegexString                 = "^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
	uUIDRegexString                  = "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
	uUID3RFC4122RegexString          = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-3[0-9a-fA-F]{3}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	uUID4RFC4122RegexString          = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$"
	uUID5RFC4122RegexString          = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-5[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$"
	uUIDRFC4122RegexString           = "^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
	aSCIIRegexString                 = "^[\x00-\x7F]*$"
	printableASCIIRegexString        = "^[\x20-\x7E]*$"
	multibyteRegexString             = "[^\x00-\x7F]"
	dataURIRegexString               = `^data:((?:\w+\/(?:([^;]|;[^;]).)+)?)`
	latitudeRegexString              = "^[-+]?([1-8]?\\d(\\.\\d+)?|90(\\.0+)?)$"
	longitudeRegexString             = "^[-+]?(180(\\.0+)?|((1[0-7]\\d)|([1-9]?\\d))(\\.\\d+)?)$"
	sSNRegexString                   = `^[0-9]{3}[ -]?(0[1-9]|[1-9][0-9])[ -]?([1-9][0-9]{3}|[0-9][1-9][0-9]{2}|[0-9]{2}[1-9][0-9]|[0-9]{3}[1-9])$`
	hostnameRegexStringRFC952        = `^[a-zA-Z]([a-zA-Z0-9\-]+[\.]?)*[a-zA-Z0-9]$`                                                                      // https://tools.ietf.org/html/rfc952
	hostnameRegexStringRFC1123       = `^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*?$`                                 // accepts hostname starting with a digit https://tools.ietf.org/html/rfc1123
	fqdnRegexStringRFC1123           = `^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{0,62})(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*?(\.[a-zA-Z]{1}[a-zA-Z0-9]{0,62})\.?$` // same as hostnameRegexStringRFC1123 but must contain a non numerical TLD (possibly ending with '.')
	btcAddressRegexString            = `^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$`                                                                                // bitcoin address
	btcAddressUpperRegexStringBech32 = `^BC1[02-9AC-HJ-NP-Z]{7,76}$`                                                                                      // bitcoin bech32 address https://en.bitcoin.it/wiki/Bech32
	btcAddressLowerRegexStringBech32 = `^bc1[02-9ac-hj-np-z]{7,76}$`                                                                                      // bitcoin bech32 address https://en.bitcoin.it/wiki/Bech32
	ethAddressRegexString            = `^0x[0-9a-fA-F]{40}$`
	ethAddressUpperRegexString       = `^0x[0-9A-F]{40}$`
	ethAddressLowerRegexString       = `^0x[0-9a-f]{40}$`
	uRLEncodedRegexString            = `^(?:[^%]|%[0-9A-Fa-f]{2})*$`
	hTMLEncodedRegexString           = `&#[x]?([0-9a-fA-F]{2})|(&gt)|(&lt)|(&quot)|(&amp)+[;]?`
	hTMLRegexString                  = `<[/]?([a-zA-Z]+).*?>`
	jWTRegexString                   = "^[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]+\\.[A-Za-z0-9-_]*$"
	splitParamsRegexString           = `'[^']*'|\S+`
	bicRegexString                   = `^[A-Za-z]{6}[A-Za-z0-9]{2}([A-Za-z0-9]{3})?$`
)

var (
	alphaRegex                 = regexp.MustCompile(alphaRegexString)
	alphaNumericRegex          = regexp.MustCompile(alphaNumericRegexString)
	alphaUnicodeRegex          = regexp.MustCompile(alphaUnicodeRegexString)
	alphaUnicodeNumericRegex   = regexp.MustCompile(alphaUnicodeNumericRegexString)
	numericRegex               = regexp.MustCompile(numericRegexString)
	numberRegex                = regexp.MustCompile(numberRegexString)
	hexadecimalRegex           = regexp.MustCompile(hexadecimalRegexString)
	hexColorRegex              = regexp.MustCompile(hexColorRegexString)
	rgbRegex                   = regexp.MustCompile(rgbRegexString)
	rgbaRegex                  = regexp.MustCompile(rgbaRegexString)
	hslRegex                   = regexp.MustCompile(hslRegexString)
	hslaRegex                  = regexp.MustCompile(hslaRegexString)
	e164Regex                  = regexp.MustCompile(e164RegexString)
	emailRegex                 = regexp.MustCompile(emailRegexString)
	base64Regex                = regexp.MustCompile(base64RegexString)
	base64URLRegex             = regexp.MustCompile(base64URLRegexString)
	iSBN10Regex                = regexp.MustCompile(iSBN10RegexString)
	iSBN13Regex                = regexp.MustCompile(iSBN13RegexString)
	uUID3Regex                 = regexp.MustCompile(uUID3RegexString)
	uUID4Regex                 = regexp.MustCompile(uUID4RegexString)
	uUID5Regex                 = regexp.MustCompile(uUID5RegexString)
	uUIDRegex                  = regexp.MustCompile(uUIDRegexString)
	uUID3RFC4122Regex          = regexp.MustCompile(uUID3RFC4122RegexString)
	uUID4RFC4122Regex          = regexp.MustCompile(uUID4RFC4122RegexString)
	uUID5RFC4122Regex          = regexp.MustCompile(uUID5RFC4122RegexString)
	uUIDRFC4122Regex           = regexp.MustCompile(uUIDRFC4122RegexString)
	aSCIIRegex                 = regexp.MustCompile(aSCIIRegexString)
	printableASCIIRegex        = regexp.MustCompile(printableASCIIRegexString)
	multibyteRegex             = regexp.MustCompile(multibyteRegexString)
	dataURIRegex               = regexp.MustCompile(dataURIRegexString)
	latitudeRegex              = regexp.MustCompile(latitudeRegexString)
	longitudeRegex             = regexp.MustCompile(longitudeRegexString)
	sSNRegex                   = regexp.MustCompile(sSNRegexString)
	hostnameRegexRFC952        = regexp.MustCompile(hostnameRegexStringRFC952)
	hostnameRegexRFC1123       = regexp.MustCompile(hostnameRegexStringRFC1123)
	fqdnRegexRFC1123           = regexp.MustCompile(fqdnRegexStringRFC1123)
	btcAddressRegex            = regexp.MustCompile(btcAddressRegexString)
	btcUpperAddressRegexBech32 = regexp.MustCompile(btcAddressUpperRegexStringBech32)
	btcLowerAddressRegexBech32 = regexp.MustCompile(btcAddressLowerRegexStringBech32)
	ethAddressRegex            = regexp.MustCompile(ethAddressRegexString)
	ethAddressRegexUpper       = regexp.MustCompile(ethAddressUpperRegexString)
	ethAddressRegexLower       = regexp.MustCompile(ethAddressLowerRegexString)
	uRLEncodedRegex            = regexp.MustCompile(uRLEncodedRegexString)
	hTMLEncodedRegex           = regexp.MustCompile(hTMLEncodedRegexString)
	hTMLRegex                  = regexp.MustCompile(hTMLRegexString)
	jWTRegex                   = regexp.MustCompile(jWTRegexString)
	splitParamsRegex           = regexp.MustCompile(splitParamsRegexString)
	bicRegex                   = regexp.MustCompile(bicRegexString)
)

// StructLevelFunc accepts all values needed for struct level validation
type StructLevelFunc func(sl StructLevel)

// StructLevelFuncCtx accepts all values needed for struct level validation
// but also allows passing of contextual validation information via context.Context.
type StructLevelFuncCtx func(ctx context.Context, sl StructLevel)

// wrapStructLevelFunc wraps normal StructLevelFunc makes it compatible with StructLevelFuncCtx
func wrapStructLevelFunc(fn StructLevelFunc) StructLevelFuncCtx {
	return func(ctx context.Context, sl StructLevel) {
		fn(sl)
	}
}

// StructLevel contains all the information and helper functions
// to validate a struct
type StructLevel interface {

	// Validator returns the main validation object, in case one wants to call validations internally.
	// this is so you don't have to use anonymous functions to get access to the validate
	// instance.
	Validator() *Validate

	// Top returns the top level struct, if any
	Top() reflect.Value

	// Parent returns the current fields parent struct, if any
	Parent() reflect.Value

	// Current returns the current struct.
	Current() reflect.Value

	// ExtractType gets the actual underlying type of field value.
	// It will dive into pointers, customTypes and return you the
	// underlying value and its kind.
	ExtractType(field reflect.Value) (value reflect.Value, kind reflect.Kind, nullable bool)

	// ReportError reports an error just by passing the field and tag information
	//
	// NOTES:
	//
	// fieldName and altName get appended to the existing namespace that
	// validator is on. e.g. pass 'FirstName' or 'Names[0]' depending
	// on the nesting
	//
	// tag can be an existing validation tag or just something you make up
	// and process on the flip side it's up to you.
	ReportError(field interface{}, fieldName, structFieldName string, tag, param string)

	// ReportValidationErrors reports an error just by passing ValidationErrors
	//
	// NOTES:
	//
	// relativeNamespace and relativeActualNamespace get appended to the
	// existing namespace that validator is on.
	// e.g. pass 'User.FirstName' or 'Users[0].FirstName' depending
	// on the nesting. most of the time they will be blank, unless you validate
	// at a level lower the the current field depth
	ReportValidationErrors(relativeNamespace, relativeActualNamespace string, errs ValidationErrors)
}

var _ StructLevel = new(validate)

// Top returns the top level struct
//
// NOTE: this can be the same as the current struct being validated
// if not is a nested struct.
//
// this is only called when within Struct and Field Level validation and
// should not be relied upon for an acurate value otherwise.
func (v *validate) Top() reflect.Value {
	return v.top
}

// Parent returns the current structs parent
//
// NOTE: this can be the same as the current struct being validated
// if not is a nested struct.
//
// this is only called when within Struct and Field Level validation and
// should not be relied upon for an acurate value otherwise.
func (v *validate) Parent() reflect.Value {
	return v.slflParent
}

// Current returns the current struct.
func (v *validate) Current() reflect.Value {
	return v.slCurrent
}

// Validator returns the main validation object, in case one want to call validations internally.
func (v *validate) Validator() *Validate {
	return v.v
}

// ExtractType gets the actual underlying type of field value.
func (v *validate) ExtractType(field reflect.Value) (reflect.Value, reflect.Kind, bool) {
	return v.extractTypeInternal(field, false)
}

// ReportError reports an error just by passing the field and tag information
func (v *validate) ReportError(field interface{}, fieldName, structFieldName, tag, param string) {

	fv, kind, _ := v.extractTypeInternal(reflect.ValueOf(field), false)

	if len(structFieldName) == 0 {
		structFieldName = fieldName
	}

	v.str1 = string(append(v.ns, fieldName...))

	if v.v.hasTagNameFunc || fieldName != structFieldName {
		v.str2 = string(append(v.actualNs, structFieldName...))
	} else {
		v.str2 = v.str1
	}

	if kind == reflect.Invalid {

		v.errs = append(v.errs,
			&fieldError{
				v:              v.v,
				tag:            tag,
				actualTag:      tag,
				ns:             v.str1,
				structNs:       v.str2,
				fieldLen:       uint8(len(fieldName)),
				structfieldLen: uint8(len(structFieldName)),
				param:          param,
				kind:           kind,
			},
		)
		return
	}

	v.errs = append(v.errs,
		&fieldError{
			v:              v.v,
			tag:            tag,
			actualTag:      tag,
			ns:             v.str1,
			structNs:       v.str2,
			fieldLen:       uint8(len(fieldName)),
			structfieldLen: uint8(len(structFieldName)),
			value:          fv.Interface(),
			param:          param,
			kind:           kind,
			typ:            fv.Type(),
		},
	)
}

// ReportValidationErrors reports ValidationErrors obtained from running validations within the Struct Level validation.
//
// NOTE: this function prepends the current namespace to the relative ones.
func (v *validate) ReportValidationErrors(relativeNamespace, relativeStructNamespace string, errs ValidationErrors) {

	var err *fieldError

	for i := 0; i < len(errs); i++ {

		err = errs[i].(*fieldError)
		err.ns = string(append(append(v.ns, relativeNamespace...), err.ns...))
		err.structNs = string(append(append(v.actualNs, relativeStructNamespace...), err.structNs...))

		v.errs = append(v.errs, err)
	}
}

// TranslationFunc is the function type used to register or override
// custom translations
type TranslationFunc func(ut ut.Translator, fe FieldError) string

// RegisterTranslationsFunc allows for registering of translations
// for a 'ut.Translator' for use within the 'TranslationFunc'
type RegisterTranslationsFunc func(ut ut.Translator) error

// extractTypeInternal gets the actual underlying type of field value.
// It will dive into pointers, customTypes and return you the
// underlying value and it's kind.
func (v *validate) extractTypeInternal(current reflect.Value, nullable bool) (reflect.Value, reflect.Kind, bool) {

BEGIN:
	switch current.Kind() {
	case reflect.Ptr:

		nullable = true

		if current.IsNil() {
			return current, reflect.Ptr, nullable
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Interface:

		nullable = true

		if current.IsNil() {
			return current, reflect.Interface, nullable
		}

		current = current.Elem()
		goto BEGIN

	case reflect.Invalid:
		return current, reflect.Invalid, nullable

	default:

		if v.v.hasCustomFuncs {

			if fn, ok := v.v.customFuncs[current.Type()]; ok {
				current = reflect.ValueOf(fn(current))
				goto BEGIN
			}
		}

		return current, current.Kind(), nullable
	}
}

// getStructFieldOKInternal traverses a struct to retrieve a specific field denoted by the provided namespace and
// returns the field, field kind and whether is was successful in retrieving the field at all.
//
// NOTE: when not successful ok will be false, this can happen when a nested struct is nil and so the field
// could not be retrieved because it didn't exist.
func (v *validate) getStructFieldOKInternal(val reflect.Value, namespace string) (current reflect.Value, kind reflect.Kind, nullable bool, found bool) {

BEGIN:
	current, kind, nullable = v.ExtractType(val)
	if kind == reflect.Invalid {
		return
	}

	if namespace == "" {
		found = true
		return
	}

	switch kind {

	case reflect.Ptr, reflect.Interface:
		return

	case reflect.Struct:

		typ := current.Type()
		fld := namespace
		var ns string

		if typ != timeType {

			idx := strings.Index(namespace, namespaceSeparator)

			if idx != -1 {
				fld = namespace[:idx]
				ns = namespace[idx+1:]
			} else {
				ns = ""
			}

			bracketIdx := strings.Index(fld, leftBracket)
			if bracketIdx != -1 {
				fld = fld[:bracketIdx]

				ns = namespace[bracketIdx:]
			}

			val = current.FieldByName(fld)
			namespace = ns
			goto BEGIN
		}

	case reflect.Array, reflect.Slice:
		idx := strings.Index(namespace, leftBracket)
		idx2 := strings.Index(namespace, rightBracket)

		arrIdx, _ := strconv.Atoi(namespace[idx+1 : idx2])

		if arrIdx >= current.Len() {
			return
		}

		startIdx := idx2 + 1

		if startIdx < len(namespace) {
			if namespace[startIdx:startIdx+1] == namespaceSeparator {
				startIdx++
			}
		}

		val = current.Index(arrIdx)
		namespace = namespace[startIdx:]
		goto BEGIN

	case reflect.Map:
		idx := strings.Index(namespace, leftBracket) + 1
		idx2 := strings.Index(namespace, rightBracket)

		endIdx := idx2

		if endIdx+1 < len(namespace) {
			if namespace[endIdx+1:endIdx+2] == namespaceSeparator {
				endIdx++
			}
		}

		key := namespace[idx:idx2]

		switch current.Type().Key().Kind() {
		case reflect.Int:
			i, _ := strconv.Atoi(key)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Int8:
			i, _ := strconv.ParseInt(key, 10, 8)
			val = current.MapIndex(reflect.ValueOf(int8(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int16:
			i, _ := strconv.ParseInt(key, 10, 16)
			val = current.MapIndex(reflect.ValueOf(int16(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int32:
			i, _ := strconv.ParseInt(key, 10, 32)
			val = current.MapIndex(reflect.ValueOf(int32(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Int64:
			i, _ := strconv.ParseInt(key, 10, 64)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Uint:
			i, _ := strconv.ParseUint(key, 10, 0)
			val = current.MapIndex(reflect.ValueOf(uint(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint8:
			i, _ := strconv.ParseUint(key, 10, 8)
			val = current.MapIndex(reflect.ValueOf(uint8(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint16:
			i, _ := strconv.ParseUint(key, 10, 16)
			val = current.MapIndex(reflect.ValueOf(uint16(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint32:
			i, _ := strconv.ParseUint(key, 10, 32)
			val = current.MapIndex(reflect.ValueOf(uint32(i)))
			namespace = namespace[endIdx+1:]

		case reflect.Uint64:
			i, _ := strconv.ParseUint(key, 10, 64)
			val = current.MapIndex(reflect.ValueOf(i))
			namespace = namespace[endIdx+1:]

		case reflect.Float32:
			f, _ := strconv.ParseFloat(key, 32)
			val = current.MapIndex(reflect.ValueOf(float32(f)))
			namespace = namespace[endIdx+1:]

		case reflect.Float64:
			f, _ := strconv.ParseFloat(key, 64)
			val = current.MapIndex(reflect.ValueOf(f))
			namespace = namespace[endIdx+1:]

		case reflect.Bool:
			b, _ := strconv.ParseBool(key)
			val = current.MapIndex(reflect.ValueOf(b))
			namespace = namespace[endIdx+1:]

		// reflect.Type = string
		default:
			val = current.MapIndex(reflect.ValueOf(key))
			namespace = namespace[endIdx+1:]
		}

		goto BEGIN
	}

	// if got here there was more namespace, cannot go any deeper
	panic("Invalid field namespace")
}

// asInt returns the parameter as a int64
// or panics if it can't convert
func asInt(param string) int64 {
	i, err := strconv.ParseInt(param, 0, 64)
	panicIf(err)

	return i
}

// asIntFromTimeDuration parses param as time.Duration and returns it as int64
// or panics on error.
func asIntFromTimeDuration(param string) int64 {
	d, err := time.ParseDuration(param)
	if err != nil {
		// attempt parsing as an an integer assuming nanosecond precision
		return asInt(param)
	}
	return int64(d)
}

// asIntFromType calls the proper function to parse param as int64,
// given a field's Type t.
func asIntFromType(t reflect.Type, param string) int64 {
	switch t {
	case timeDurationType:
		return asIntFromTimeDuration(param)
	default:
		return asInt(param)
	}
}

// asUint returns the parameter as a uint64
// or panics if it can't convert
func asUint(param string) uint64 {

	i, err := strconv.ParseUint(param, 0, 64)
	panicIf(err)

	return i
}

// asFloat returns the parameter as a float64
// or panics if it can't convert
func asFloat(param string) float64 {

	i, err := strconv.ParseFloat(param, 64)
	panicIf(err)

	return i
}

// asBool returns the parameter as a bool
// or panics if it can't convert
func asBool(param string) bool {

	i, err := strconv.ParseBool(param)
	panicIf(err)

	return i
}

func panicIf(err error) {
	if err != nil {
		panic(err.Error())
	}
}

// per validate construct
type validate struct {
	v              *Validate
	top            reflect.Value
	ns             []byte
	actualNs       []byte
	errs           ValidationErrors
	includeExclude map[string]struct{} // reset only if StructPartial or StructExcept are called, no need otherwise
	ffn            FilterFunc
	slflParent     reflect.Value // StructLevel & FieldLevel
	slCurrent      reflect.Value // StructLevel & FieldLevel
	flField        reflect.Value // StructLevel & FieldLevel
	cf             *cField       // StructLevel & FieldLevel
	ct             *cTag         // StructLevel & FieldLevel
	misc           []byte        // misc reusable
	str1           string        // misc reusable
	str2           string        // misc reusable
	fldIsPointer   bool          // StructLevel & FieldLevel
	isPartial      bool
	hasExcludes    bool
}

// parent and current will be the same the first run of validateStruct
func (v *validate) validateStruct(ctx context.Context, parent reflect.Value, current reflect.Value, typ reflect.Type, ns []byte, structNs []byte, ct *cTag) {

	cs, ok := v.v.structCache.Get(typ)
	if !ok {
		cs = v.v.extractStructCache(current, typ.Name())
	}

	if len(ns) == 0 && len(cs.name) != 0 {

		ns = append(ns, cs.name...)
		ns = append(ns, '.')

		structNs = append(structNs, cs.name...)
		structNs = append(structNs, '.')
	}

	// ct is nil on top level struct, and structs as fields that have no tag info
	// so if nil or if not nil and the structonly tag isn't present
	if ct == nil || ct.typeof != typeStructOnly {

		var f *cField

		for i := 0; i < len(cs.fields); i++ {

			f = cs.fields[i]

			if v.isPartial {

				if v.ffn != nil {
					// used with StructFiltered
					if v.ffn(append(structNs, f.name...)) {
						continue
					}

				} else {
					// used with StructPartial & StructExcept
					_, ok = v.includeExclude[string(append(structNs, f.name...))]

					if (ok && v.hasExcludes) || (!ok && !v.hasExcludes) {
						continue
					}
				}
			}

			v.traverseField(ctx, current, current.Field(f.idx), ns, structNs, f, f.cTags)
		}
	}

	// check if any struct level validations, after all field validations already checked.
	// first iteration will have no info about nostructlevel tag, and is checked prior to
	// calling the next iteration of validateStruct called from traverseField.
	if cs.fn != nil {

		v.slflParent = parent
		v.slCurrent = current
		v.ns = ns
		v.actualNs = structNs

		cs.fn(ctx, v)
	}
}

// traverseField validates any field, be it a struct or single field, ensures it's validity and passes it along to be validated via it's tag options
func (v *validate) traverseField(ctx context.Context, parent reflect.Value, current reflect.Value, ns []byte, structNs []byte, cf *cField, ct *cTag) {
	var typ reflect.Type
	var kind reflect.Kind

	current, kind, v.fldIsPointer = v.extractTypeInternal(current, false)

	switch kind {
	case reflect.Ptr, reflect.Interface, reflect.Invalid:

		if ct == nil {
			return
		}

		if ct.typeof == typeOmitEmpty || ct.typeof == typeIsDefault {
			return
		}

		if ct.hasTag {
			if kind == reflect.Invalid {
				v.str1 = string(append(ns, cf.altName...))
				if v.v.hasTagNameFunc {
					v.str2 = string(append(structNs, cf.name...))
				} else {
					v.str2 = v.str1
				}
				v.errs = append(v.errs,
					&fieldError{
						v:              v.v,
						tag:            ct.aliasTag,
						actualTag:      ct.tag,
						ns:             v.str1,
						structNs:       v.str2,
						fieldLen:       uint8(len(cf.altName)),
						structfieldLen: uint8(len(cf.name)),
						param:          ct.param,
						kind:           kind,
					},
				)
				return
			}

			v.str1 = string(append(ns, cf.altName...))
			if v.v.hasTagNameFunc {
				v.str2 = string(append(structNs, cf.name...))
			} else {
				v.str2 = v.str1
			}
			if !ct.runValidationWhenNil {
				v.errs = append(v.errs,
					&fieldError{
						v:              v.v,
						tag:            ct.aliasTag,
						actualTag:      ct.tag,
						ns:             v.str1,
						structNs:       v.str2,
						fieldLen:       uint8(len(cf.altName)),
						structfieldLen: uint8(len(cf.name)),
						value:          current.Interface(),
						param:          ct.param,
						kind:           kind,
						typ:            current.Type(),
					},
				)
				return
			}
		}

	case reflect.Struct:

		typ = current.Type()

		if typ != timeType {

			if ct != nil {

				if ct.typeof == typeStructOnly {
					goto CONTINUE
				} else if ct.typeof == typeIsDefault {
					// set Field Level fields
					v.slflParent = parent
					v.flField = current
					v.cf = cf
					v.ct = ct

					if !ct.fn(ctx, v) {
						v.str1 = string(append(ns, cf.altName...))

						if v.v.hasTagNameFunc {
							v.str2 = string(append(structNs, cf.name...))
						} else {
							v.str2 = v.str1
						}

						v.errs = append(v.errs,
							&fieldError{
								v:              v.v,
								tag:            ct.aliasTag,
								actualTag:      ct.tag,
								ns:             v.str1,
								structNs:       v.str2,
								fieldLen:       uint8(len(cf.altName)),
								structfieldLen: uint8(len(cf.name)),
								value:          current.Interface(),
								param:          ct.param,
								kind:           kind,
								typ:            typ,
							},
						)
						return
					}
				}

				ct = ct.next
			}

			if ct != nil && ct.typeof == typeNoStructLevel {
				return
			}

		CONTINUE:
			// if len == 0 then validating using 'Var' or 'VarWithValue'
			// Var - doesn't make much sense to do it that way, should call 'Struct', but no harm...
			// VarWithField - this allows for validating against each field within the struct against a specific value
			//                pretty handy in certain situations
			if len(cf.name) > 0 {
				ns = append(append(ns, cf.altName...), '.')
				structNs = append(append(structNs, cf.name...), '.')
			}

			v.validateStruct(ctx, parent, current, typ, ns, structNs, ct)
			return
		}
	}

	if ct == nil || !ct.hasTag {
		return
	}

	typ = current.Type()

OUTER:
	for {
		if ct == nil {
			return
		}

		switch ct.typeof {

		case typeOmitEmpty:

			// set Field Level fields
			v.slflParent = parent
			v.flField = current
			v.cf = cf
			v.ct = ct

			if !hasValue(v) {
				return
			}

			ct = ct.next
			continue

		case typeEndKeys:
			return

		case typeDive:

			ct = ct.next

			// traverse slice or map here
			// or panic ;)
			switch kind {
			case reflect.Slice, reflect.Array:

				var i64 int64
				reusableCF := &cField{}

				for i := 0; i < current.Len(); i++ {

					i64 = int64(i)

					v.misc = append(v.misc[0:0], cf.name...)
					v.misc = append(v.misc, '[')
					v.misc = strconv.AppendInt(v.misc, i64, 10)
					v.misc = append(v.misc, ']')

					reusableCF.name = string(v.misc)

					if cf.namesEqual {
						reusableCF.altName = reusableCF.name
					} else {

						v.misc = append(v.misc[0:0], cf.altName...)
						v.misc = append(v.misc, '[')
						v.misc = strconv.AppendInt(v.misc, i64, 10)
						v.misc = append(v.misc, ']')

						reusableCF.altName = string(v.misc)
					}
					v.traverseField(ctx, parent, current.Index(i), ns, structNs, reusableCF, ct)
				}

			case reflect.Map:

				var pv string
				reusableCF := &cField{}

				for _, key := range current.MapKeys() {

					pv = fmt.Sprintf("%v", key.Interface())

					v.misc = append(v.misc[0:0], cf.name...)
					v.misc = append(v.misc, '[')
					v.misc = append(v.misc, pv...)
					v.misc = append(v.misc, ']')

					reusableCF.name = string(v.misc)

					if cf.namesEqual {
						reusableCF.altName = reusableCF.name
					} else {
						v.misc = append(v.misc[0:0], cf.altName...)
						v.misc = append(v.misc, '[')
						v.misc = append(v.misc, pv...)
						v.misc = append(v.misc, ']')

						reusableCF.altName = string(v.misc)
					}

					if ct != nil && ct.typeof == typeKeys && ct.keys != nil {
						v.traverseField(ctx, parent, key, ns, structNs, reusableCF, ct.keys)
						// can be nil when just keys being validated
						if ct.next != nil {
							v.traverseField(ctx, parent, current.MapIndex(key), ns, structNs, reusableCF, ct.next)
						}
					} else {
						v.traverseField(ctx, parent, current.MapIndex(key), ns, structNs, reusableCF, ct)
					}
				}

			default:
				// throw error, if not a slice or map then should not have gotten here
				// bad dive tag
				panic("dive error! can't dive on a non slice or map")
			}

			return

		case typeOr:

			v.misc = v.misc[0:0]

			for {

				// set Field Level fields
				v.slflParent = parent
				v.flField = current
				v.cf = cf
				v.ct = ct

				if ct.fn(ctx, v) {

					// drain rest of the 'or' values, then continue or leave
					for {

						ct = ct.next

						if ct == nil {
							return
						}

						if ct.typeof != typeOr {
							continue OUTER
						}
					}
				}

				v.misc = append(v.misc, '|')
				v.misc = append(v.misc, ct.tag...)

				if ct.hasParam {
					v.misc = append(v.misc, '=')
					v.misc = append(v.misc, ct.param...)
				}

				if ct.isBlockEnd || ct.next == nil {
					// if we get here, no valid 'or' value and no more tags
					v.str1 = string(append(ns, cf.altName...))

					if v.v.hasTagNameFunc {
						v.str2 = string(append(structNs, cf.name...))
					} else {
						v.str2 = v.str1
					}

					if ct.hasAlias {

						v.errs = append(v.errs,
							&fieldError{
								v:              v.v,
								tag:            ct.aliasTag,
								actualTag:      ct.actualAliasTag,
								ns:             v.str1,
								structNs:       v.str2,
								fieldLen:       uint8(len(cf.altName)),
								structfieldLen: uint8(len(cf.name)),
								value:          current.Interface(),
								param:          ct.param,
								kind:           kind,
								typ:            typ,
							},
						)

					} else {

						tVal := string(v.misc)[1:]

						v.errs = append(v.errs,
							&fieldError{
								v:              v.v,
								tag:            tVal,
								actualTag:      tVal,
								ns:             v.str1,
								structNs:       v.str2,
								fieldLen:       uint8(len(cf.altName)),
								structfieldLen: uint8(len(cf.name)),
								value:          current.Interface(),
								param:          ct.param,
								kind:           kind,
								typ:            typ,
							},
						)
					}

					return
				}

				ct = ct.next
			}

		default:

			// set Field Level fields
			v.slflParent = parent
			v.flField = current
			v.cf = cf
			v.ct = ct

			if !ct.fn(ctx, v) {

				v.str1 = string(append(ns, cf.altName...))

				if v.v.hasTagNameFunc {
					v.str2 = string(append(structNs, cf.name...))
				} else {
					v.str2 = v.str1
				}

				v.errs = append(v.errs,
					&fieldError{
						v:              v.v,
						tag:            ct.aliasTag,
						actualTag:      ct.tag,
						ns:             v.str1,
						structNs:       v.str2,
						fieldLen:       uint8(len(cf.altName)),
						structfieldLen: uint8(len(cf.name)),
						value:          current.Interface(),
						param:          ct.param,
						kind:           kind,
						typ:            typ,
					},
				)

				return
			}
			ct = ct.next
		}
	}

}

const (
	defaultTagName        = "validate"
	utf8HexComma          = "0x2C"
	utf8Pipe              = "0x7C"
	tagSeparator          = ","
	orSeparator           = "|"
	tagKeySeparator       = "="
	structOnlyTag         = "structonly"
	noStructLevelTag      = "nostructlevel"
	omitempty             = "omitempty"
	isdefault             = "isdefault"
	requiredWithoutAllTag = "required_without_all"
	requiredWithoutTag    = "required_without"
	requiredWithTag       = "required_with"
	requiredWithAllTag    = "required_with_all"
	requiredIfTag         = "required_if"
	requiredUnlessTag     = "required_unless"
	excludedWithoutAllTag = "excluded_without_all"
	excludedWithoutTag    = "excluded_without"
	excludedWithTag       = "excluded_with"
	excludedWithAllTag    = "excluded_with_all"
	skipValidationTag     = "-"
	diveTag               = "dive"
	keysTag               = "keys"
	endKeysTag            = "endkeys"
	requiredTag           = "required"
	namespaceSeparator    = "."
	leftBracket           = "["
	rightBracket          = "]"
	restrictedTagChars    = ".[],|=+()`~!@#$%^&*\\\"/?<>{}"
	restrictedAliasErr    = "Alias '%s' either contains restricted characters or is the same as a restricted tag needed for normal operation"
	restrictedTagErr      = "Tag '%s' either contains restricted characters or is the same as a restricted tag needed for normal operation"
)

var (
	timeDurationType = reflect.TypeOf(time.Duration(0))
	timeType         = reflect.TypeOf(time.Time{})

	defaultCField = &cField{namesEqual: true}
)

// FilterFunc is the type used to filter fields using
// StructFiltered(...) function.
// returning true results in the field being filtered/skiped from
// validation
type FilterFunc func(ns []byte) bool

// CustomTypeFunc allows for overriding or adding custom field type handler functions
// field = field value of the type to return a value to be validated
// example Valuer from sql drive see https://golang.org/src/database/sql/driver/types.go?s=1210:1293#L29
type CustomTypeFunc func(field reflect.Value) interface{}

// TagNameFunc allows for adding of a custom tag name parser
type TagNameFunc func(field reflect.StructField) string

type internalValidationFuncWrapper struct {
	fn                FuncCtx
	runValidatinOnNil bool
}

// Validate contains the validator settings and cache
type Validate struct {
	tagName          string
	pool             *sync.Pool
	hasCustomFuncs   bool
	hasTagNameFunc   bool
	tagNameFunc      TagNameFunc
	structLevelFuncs map[reflect.Type]StructLevelFuncCtx
	customFuncs      map[reflect.Type]CustomTypeFunc
	aliases          map[string]string
	validations      map[string]internalValidationFuncWrapper
	transTagFunc     map[ut.Translator]map[string]TranslationFunc // map[<locale>]map[<tag>]TranslationFunc
	tagCache         *tagCache
	structCache      *structCache
}

// New returns a new instance of 'validate' with sane defaults.
// Validate is designed to be thread-safe and used as a singleton instance.
// It caches information about your struct and validations,
// in essence only parsing your validation tags once per struct type.
// Using multiple instances neglects the benefit of caching.
func New() *Validate {

	tc := new(tagCache)
	tc.m.Store(make(map[string]*cTag))

	sc := new(structCache)
	sc.m.Store(make(map[reflect.Type]*cStruct))

	v := &Validate{
		tagName:     defaultTagName,
		aliases:     make(map[string]string, len(bakedInAliases)),
		validations: make(map[string]internalValidationFuncWrapper, len(bakedInValidators)),
		tagCache:    tc,
		structCache: sc,
	}

	// must copy alias validators for separate validations to be used in each validator instance
	for k, val := range bakedInAliases {
		v.RegisterAlias(k, val)
	}

	// must copy validators for separate validations to be used in each instance
	for k, val := range bakedInValidators {

		switch k {
		// these require that even if the value is nil that the validation should run, omitempty still overrides this behaviour
		case requiredIfTag, requiredUnlessTag, requiredWithTag, requiredWithAllTag, requiredWithoutTag, requiredWithoutAllTag,
			excludedWithTag, excludedWithAllTag, excludedWithoutTag, excludedWithoutAllTag:
			_ = v.registerValidation(k, wrapFunc(val), true, true)
		default:
			// no need to error check here, baked in will always be valid
			_ = v.registerValidation(k, wrapFunc(val), true, false)
		}
	}

	v.pool = &sync.Pool{
		New: func() interface{} {
			return &validate{
				v:        v,
				ns:       make([]byte, 0, 64),
				actualNs: make([]byte, 0, 64),
				misc:     make([]byte, 32),
			}
		},
	}

	return v
}

// SetTagName allows for changing of the default tag name of 'validate'
func (v *Validate) SetTagName(name string) {
	v.tagName = name
}

// ValidateMapCtx validates a map using a map of validation rules and allows passing of contextual
// validation validation information via context.Context.
func (v Validate) ValidateMapCtx(ctx context.Context, data map[string]interface{}, rules map[string]interface{}) map[string]interface{} {
	errs := make(map[string]interface{})
	for field, rule := range rules {
		if reflect.ValueOf(rule).Kind() == reflect.Map && reflect.ValueOf(data[field]).Kind() == reflect.Map {
			err := v.ValidateMapCtx(ctx, data[field].(map[string]interface{}), rule.(map[string]interface{}))
			if len(err) > 0 {
				errs[field] = err
			}
		} else if reflect.ValueOf(rule).Kind() == reflect.Map {
			errs[field] = errors.New("The field: '" + field + "' is not a map to dive")
		} else {
			err := v.VarCtx(ctx, data[field], rule.(string))
			if err != nil {
				errs[field] = err
			}
		}
	}
	return errs
}

// ValidateMap validates map data form a map of tags
func (v *Validate) ValidateMap(data map[string]interface{}, rules map[string]interface{}) map[string]interface{} {
	return v.ValidateMapCtx(context.Background(), data, rules)
}

// RegisterTagNameFunc registers a function to get alternate names for StructFields.
//
// eg. to use the names which have been specified for JSON representations of structs, rather than normal Go field names:
//
//    validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
//        name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
//        if name == "-" {
//            return ""
//        }
//        return name
//    })
func (v *Validate) RegisterTagNameFunc(fn TagNameFunc) {
	v.tagNameFunc = fn
	v.hasTagNameFunc = true
}

// RegisterValidation adds a validation with the given tag
//
// NOTES:
// - if the key already exists, the previous validation function will be replaced.
// - this method is not thread-safe it is intended that these all be registered prior to any validation
func (v *Validate) RegisterValidation(tag string, fn Func, callValidationEvenIfNull ...bool) error {
	return v.RegisterValidationCtx(tag, wrapFunc(fn), callValidationEvenIfNull...)
}

// RegisterValidationCtx does the same as RegisterValidation on accepts a FuncCtx validation
// allowing context.Context validation support.
func (v *Validate) RegisterValidationCtx(tag string, fn FuncCtx, callValidationEvenIfNull ...bool) error {
	var nilCheckable bool
	if len(callValidationEvenIfNull) > 0 {
		nilCheckable = callValidationEvenIfNull[0]
	}
	return v.registerValidation(tag, fn, false, nilCheckable)
}

func (v *Validate) registerValidation(tag string, fn FuncCtx, bakedIn bool, nilCheckable bool) error {
	if len(tag) == 0 {
		return errors.New("function Key cannot be empty")
	}

	if fn == nil {
		return errors.New("function cannot be empty")
	}

	_, ok := restrictedTags[tag]
	if !bakedIn && (ok || strings.ContainsAny(tag, restrictedTagChars)) {
		panic(fmt.Sprintf(restrictedTagErr, tag))
	}
	v.validations[tag] = internalValidationFuncWrapper{fn: fn, runValidatinOnNil: nilCheckable}
	return nil
}

// RegisterAlias registers a mapping of a single validation tag that
// defines a common or complex set of validation(s) to simplify adding validation
// to structs.
//
// NOTE: this function is not thread-safe it is intended that these all be registered prior to any validation
func (v *Validate) RegisterAlias(alias, tags string) {

	_, ok := restrictedTags[alias]

	if ok || strings.ContainsAny(alias, restrictedTagChars) {
		panic(fmt.Sprintf(restrictedAliasErr, alias))
	}

	v.aliases[alias] = tags
}

// RegisterStructValidation registers a StructLevelFunc against a number of types.
//
// NOTE:
// - this method is not thread-safe it is intended that these all be registered prior to any validation
func (v *Validate) RegisterStructValidation(fn StructLevelFunc, types ...interface{}) {
	v.RegisterStructValidationCtx(wrapStructLevelFunc(fn), types...)
}

// RegisterStructValidationCtx registers a StructLevelFuncCtx against a number of types and allows passing
// of contextual validation information via context.Context.
//
// NOTE:
// - this method is not thread-safe it is intended that these all be registered prior to any validation
func (v *Validate) RegisterStructValidationCtx(fn StructLevelFuncCtx, types ...interface{}) {

	if v.structLevelFuncs == nil {
		v.structLevelFuncs = make(map[reflect.Type]StructLevelFuncCtx)
	}

	for _, t := range types {
		tv := reflect.ValueOf(t)
		if tv.Kind() == reflect.Ptr {
			t = reflect.Indirect(tv).Interface()
		}

		v.structLevelFuncs[reflect.TypeOf(t)] = fn
	}
}

// RegisterCustomTypeFunc registers a CustomTypeFunc against a number of types
//
// NOTE: this method is not thread-safe it is intended that these all be registered prior to any validation
func (v *Validate) RegisterCustomTypeFunc(fn CustomTypeFunc, types ...interface{}) {

	if v.customFuncs == nil {
		v.customFuncs = make(map[reflect.Type]CustomTypeFunc)
	}

	for _, t := range types {
		v.customFuncs[reflect.TypeOf(t)] = fn
	}

	v.hasCustomFuncs = true
}

// RegisterTranslation registers translations against the provided tag.
func (v *Validate) RegisterTranslation(tag string, trans ut.Translator, registerFn RegisterTranslationsFunc, translationFn TranslationFunc) (err error) {

	if v.transTagFunc == nil {
		v.transTagFunc = make(map[ut.Translator]map[string]TranslationFunc)
	}

	if err = registerFn(trans); err != nil {
		return
	}

	m, ok := v.transTagFunc[trans]
	if !ok {
		m = make(map[string]TranslationFunc)
		v.transTagFunc[trans] = m
	}

	m[tag] = translationFn

	return
}

// Struct validates a structs exposed fields, and automatically validates nested structs, unless otherwise specified.
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) Struct(s interface{}) error {
	return v.StructCtx(context.Background(), s)
}

// StructCtx validates a structs exposed fields, and automatically validates nested structs, unless otherwise specified
// and also allows passing of context.Context for contextual validation information.
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) StructCtx(ctx context.Context, s interface{}) (err error) {

	val := reflect.ValueOf(s)
	top := val

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct || val.Type() == timeType {
		return &InvalidValidationError{Type: reflect.TypeOf(s)}
	}

	// good to validate
	vd := v.pool.Get().(*validate)
	vd.top = top
	vd.isPartial = false
	// vd.hasExcludes = false // only need to reset in StructPartial and StructExcept

	vd.validateStruct(ctx, top, val, val.Type(), vd.ns[0:0], vd.actualNs[0:0], nil)

	if len(vd.errs) > 0 {
		err = vd.errs
		vd.errs = nil
	}

	v.pool.Put(vd)

	return
}

// StructFiltered validates a structs exposed fields, that pass the FilterFunc check and automatically validates
// nested structs, unless otherwise specified.
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) StructFiltered(s interface{}, fn FilterFunc) error {
	return v.StructFilteredCtx(context.Background(), s, fn)
}

// StructFilteredCtx validates a structs exposed fields, that pass the FilterFunc check and automatically validates
// nested structs, unless otherwise specified and also allows passing of contextual validation information via
// context.Context
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) StructFilteredCtx(ctx context.Context, s interface{}, fn FilterFunc) (err error) {
	val := reflect.ValueOf(s)
	top := val

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct || val.Type() == timeType {
		return &InvalidValidationError{Type: reflect.TypeOf(s)}
	}

	// good to validate
	vd := v.pool.Get().(*validate)
	vd.top = top
	vd.isPartial = true
	vd.ffn = fn
	// vd.hasExcludes = false // only need to reset in StructPartial and StructExcept

	vd.validateStruct(ctx, top, val, val.Type(), vd.ns[0:0], vd.actualNs[0:0], nil)

	if len(vd.errs) > 0 {
		err = vd.errs
		vd.errs = nil
	}

	v.pool.Put(vd)

	return
}

// StructPartial validates the fields passed in only, ignoring all others.
// Fields may be provided in a namespaced fashion relative to the  struct provided
// eg. NestedStruct.Field or NestedArrayField[0].Struct.Name
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) StructPartial(s interface{}, fields ...string) error {
	return v.StructPartialCtx(context.Background(), s, fields...)
}

// StructPartialCtx validates the fields passed in only, ignoring all others and allows passing of contextual
// validation validation information via context.Context
// Fields may be provided in a namespaced fashion relative to the  struct provided
// eg. NestedStruct.Field or NestedArrayField[0].Struct.Name
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) StructPartialCtx(ctx context.Context, s interface{}, fields ...string) (err error) {
	val := reflect.ValueOf(s)
	top := val

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct || val.Type() == timeType {
		return &InvalidValidationError{Type: reflect.TypeOf(s)}
	}

	// good to validate
	vd := v.pool.Get().(*validate)
	vd.top = top
	vd.isPartial = true
	vd.ffn = nil
	vd.hasExcludes = false
	vd.includeExclude = make(map[string]struct{})

	typ := val.Type()
	name := typ.Name()

	for _, k := range fields {

		flds := strings.Split(k, namespaceSeparator)
		if len(flds) > 0 {

			vd.misc = append(vd.misc[0:0], name...)
			// Don't append empty name for unnamed structs
			if len(vd.misc) != 0 {
				vd.misc = append(vd.misc, '.')
			}

			for _, s := range flds {

				idx := strings.Index(s, leftBracket)

				if idx != -1 {
					for idx != -1 {
						vd.misc = append(vd.misc, s[:idx]...)
						vd.includeExclude[string(vd.misc)] = struct{}{}

						idx2 := strings.Index(s, rightBracket)
						idx2++
						vd.misc = append(vd.misc, s[idx:idx2]...)
						vd.includeExclude[string(vd.misc)] = struct{}{}
						s = s[idx2:]
						idx = strings.Index(s, leftBracket)
					}
				} else {

					vd.misc = append(vd.misc, s...)
					vd.includeExclude[string(vd.misc)] = struct{}{}
				}

				vd.misc = append(vd.misc, '.')
			}
		}
	}

	vd.validateStruct(ctx, top, val, typ, vd.ns[0:0], vd.actualNs[0:0], nil)

	if len(vd.errs) > 0 {
		err = vd.errs
		vd.errs = nil
	}

	v.pool.Put(vd)

	return
}

// StructExcept validates all fields except the ones passed in.
// Fields may be provided in a namespaced fashion relative to the  struct provided
// i.e. NestedStruct.Field or NestedArrayField[0].Struct.Name
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) StructExcept(s interface{}, fields ...string) error {
	return v.StructExceptCtx(context.Background(), s, fields...)
}

// StructExceptCtx validates all fields except the ones passed in and allows passing of contextual
// validation validation information via context.Context
// Fields may be provided in a namespaced fashion relative to the  struct provided
// i.e. NestedStruct.Field or NestedArrayField[0].Struct.Name
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
func (v *Validate) StructExceptCtx(ctx context.Context, s interface{}, fields ...string) (err error) {
	val := reflect.ValueOf(s)
	top := val

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct || val.Type() == timeType {
		return &InvalidValidationError{Type: reflect.TypeOf(s)}
	}

	// good to validate
	vd := v.pool.Get().(*validate)
	vd.top = top
	vd.isPartial = true
	vd.ffn = nil
	vd.hasExcludes = true
	vd.includeExclude = make(map[string]struct{})

	typ := val.Type()
	name := typ.Name()

	for _, key := range fields {

		vd.misc = vd.misc[0:0]

		if len(name) > 0 {
			vd.misc = append(vd.misc, name...)
			vd.misc = append(vd.misc, '.')
		}

		vd.misc = append(vd.misc, key...)
		vd.includeExclude[string(vd.misc)] = struct{}{}
	}

	vd.validateStruct(ctx, top, val, typ, vd.ns[0:0], vd.actualNs[0:0], nil)

	if len(vd.errs) > 0 {
		err = vd.errs
		vd.errs = nil
	}

	v.pool.Put(vd)

	return
}

// Var validates a single variable using tag style validation.
// eg.
// var i int
// validate.Var(i, "gt=1,lt=10")
//
// WARNING: a struct can be passed for validation eg. time.Time is a struct or
// if you have a custom type and have registered a custom type handler, so must
// allow it; however unforeseen validations will occur if trying to validate a
// struct that is meant to be passed to 'validate.Struct'
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
// validate Array, Slice and maps fields which may contain more than one error
func (v *Validate) Var(field interface{}, tag string) error {
	return v.VarCtx(context.Background(), field, tag)
}

// VarCtx validates a single variable using tag style validation and allows passing of contextual
// validation validation information via context.Context.
// eg.
// var i int
// validate.Var(i, "gt=1,lt=10")
//
// WARNING: a struct can be passed for validation eg. time.Time is a struct or
// if you have a custom type and have registered a custom type handler, so must
// allow it; however unforeseen validations will occur if trying to validate a
// struct that is meant to be passed to 'validate.Struct'
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
// validate Array, Slice and maps fields which may contain more than one error
func (v *Validate) VarCtx(ctx context.Context, field interface{}, tag string) (err error) {
	if len(tag) == 0 || tag == skipValidationTag {
		return nil
	}

	ctag := v.fetchCacheTag(tag)
	val := reflect.ValueOf(field)
	vd := v.pool.Get().(*validate)
	vd.top = val
	vd.isPartial = false
	vd.traverseField(ctx, val, val, vd.ns[0:0], vd.actualNs[0:0], defaultCField, ctag)

	if len(vd.errs) > 0 {
		err = vd.errs
		vd.errs = nil
	}
	v.pool.Put(vd)
	return
}

// VarWithValue validates a single variable, against another variable/field's value using tag style validation
// eg.
// s1 := "abcd"
// s2 := "abcd"
// validate.VarWithValue(s1, s2, "eqcsfield") // returns true
//
// WARNING: a struct can be passed for validation eg. time.Time is a struct or
// if you have a custom type and have registered a custom type handler, so must
// allow it; however unforeseen validations will occur if trying to validate a
// struct that is meant to be passed to 'validate.Struct'
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
// validate Array, Slice and maps fields which may contain more than one error
func (v *Validate) VarWithValue(field interface{}, other interface{}, tag string) error {
	return v.VarWithValueCtx(context.Background(), field, other, tag)
}

// VarWithValueCtx validates a single variable, against another variable/field's value using tag style validation and
// allows passing of contextual validation validation information via context.Context.
// eg.
// s1 := "abcd"
// s2 := "abcd"
// validate.VarWithValue(s1, s2, "eqcsfield") // returns true
//
// WARNING: a struct can be passed for validation eg. time.Time is a struct or
// if you have a custom type and have registered a custom type handler, so must
// allow it; however unforeseen validations will occur if trying to validate a
// struct that is meant to be passed to 'validate.Struct'
//
// It returns InvalidValidationError for bad values passed in and nil or ValidationErrors as error otherwise.
// You will need to assert the error if it's not nil eg. err.(validator.ValidationErrors) to access the array of errors.
// validate Array, Slice and maps fields which may contain more than one error
func (v *Validate) VarWithValueCtx(ctx context.Context, field interface{}, other interface{}, tag string) (err error) {
	if len(tag) == 0 || tag == skipValidationTag {
		return nil
	}
	ctag := v.fetchCacheTag(tag)
	otherVal := reflect.ValueOf(other)
	vd := v.pool.Get().(*validate)
	vd.top = otherVal
	vd.isPartial = false
	vd.traverseField(ctx, otherVal, reflect.ValueOf(field), vd.ns[0:0], vd.actualNs[0:0], defaultCField, ctag)

	if len(vd.errs) > 0 {
		err = vd.errs
		vd.errs = nil
	}
	v.pool.Put(vd)
	return
}
