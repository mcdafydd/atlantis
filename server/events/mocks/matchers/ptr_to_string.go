// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	"reflect"
	"github.com/petergtz/pegomock"
	
)

func AnyPtrToString() *string {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(*string))(nil)).Elem()))
	var nullValue *string
	return nullValue
}

func EqPtrToString(value *string) *string {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue *string
	return nullValue
}