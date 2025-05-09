// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"math"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestQueryArrayCompatSize(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"float64": {
			filter: bson.D{{"v", bson.D{{"$size", float64(2)}}}},
		},
		"int32": {
			filter: bson.D{{"v", bson.D{{"$size", int32(2)}}}},
		},
		"int64": {
			filter: bson.D{{"v", bson.D{{"$size", int64(2)}}}},
		},
		"Infinity": {
			filter:           bson.D{{"v", bson.D{{"$size", math.Inf(+1)}}}},
			resultType:       EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/245",
		},
		"InvalidUse": {
			filter:     bson.D{{"$size", 2}},
			resultType: EmptyResult,
		},
		"InvalidType": {
			filter:     bson.D{{"v", bson.D{{"$size", bson.D{{"$gt", 1}}}}}},
			resultType: EmptyResult,
		},
		"Negative": {
			filter:     bson.D{{"v", bson.D{{"$size", -1}}}},
			resultType: EmptyResult,
		},
		"NotFound": {
			filter:     bson.D{{"v", bson.D{{"$size", 4}}}},
			resultType: EmptyResult,
		},
		"NotWhole": {
			filter:     bson.D{{"v", bson.D{{"$size", 2.1}}}},
			resultType: EmptyResult,
		},
		"Zero": {
			filter: bson.D{{"v", bson.D{{"$size", 0}}}},
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryArrayCompatDotNotation(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"PositionIndexGreaterThanArrayLength": {
			filter:     bson.D{{"v.5", bson.D{{"$type", "double"}}}},
			resultType: EmptyResult,
		},
		"PositionIndexAtTheEndOfArray": {
			filter: bson.D{{"v.1", bson.D{{"$type", "string"}}}},
		},
		"PositionTypeNull": {
			filter: bson.D{{"v.0", bson.D{{"$type", "null"}}}},
		},
		"PositionRegex": {
			filter: bson.D{{"v.1", primitive.Regex{Pattern: "foo"}}},
		},
		"NoSuchFieldPosition": {
			filter:     bson.D{{"v.some.0", bson.A{42}}},
			resultType: EmptyResult,
		},
		"Field": {
			filter: bson.D{{"v.array", int32(42)}},
		},
		"FieldPosition": {
			filter: bson.D{{"v.array.0", int32(42)}},
		},
		"FieldPositionQuery": {
			filter: bson.D{{"v.array.0", bson.D{{"$gte", int32(42)}}}},
		},
		"FieldPositionQueryNonArray": {
			filter:     bson.D{{"v.document.0", bson.D{{"$lt", int32(42)}}}},
			resultType: EmptyResult,
		},
		"DocumentDotNotationArrayDocument": {
			filter: bson.D{{"v.0.foo.0.bar", "hello"}},
		},
		"DocumentDotNotationArrayDocumentNoIndexNin": {
			filter: bson.D{
				{"v.foo.bar", bson.D{{"$nin", bson.A{"baz"}}}},
			},
		},
		"DocumentDotNotationArrayDocumentNoIndex": {
			filter: bson.D{{"v.foo.bar", "hello"}},
		},
		"FieldArrayIndex": {
			filter: bson.D{{"v.foo[0]", int32(42)}},
		},
		"FieldArrayAsterix": {
			filter: bson.D{{"v.foo[*]", int32(42)}},
		},
		"FieldAsterix": {
			filter: bson.D{{"v.*", int32(42)}},
		},
		"FieldAt": {
			filter: bson.D{{"v.@", int32(42)}},
		},
		"FieldComma": {
			filter: bson.D{{"v.f,oo", int32(42)}},
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryArrayCompatElemMatch(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"DoubleTarget": {
			filter: bson.D{
				{"_id", "double"},
				{"v", bson.D{{"$elemMatch", bson.D{{"$gt", int32(0)}}}}},
			},
			resultType: EmptyResult,
		},
		"GtZero": {
			filter: bson.D{{"v", bson.D{{"$elemMatch", bson.D{{"$gt", int32(0)}}}}}},
		},
		"GtZeroWithTypeArray": {
			filter: bson.D{
				{"v", bson.D{
					{"$elemMatch", bson.D{
						{"$gt", int32(0)},
					}},
					{"$type", "array"},
				}},
			},
		},
		"GtZeroWithTypeString": {
			// A document like {"v":[42, "foo"]} matches this filter (there is an elem >0 and an elem of type string)
			filter: bson.D{
				{"v", bson.D{
					{"$elemMatch", bson.D{
						{"$gt", int32(0)},
					}},
					{"$type", "string"},
				}},
			},
		},
		"GtLt": {
			filter: bson.D{
				{"v", bson.D{
					{"$elemMatch", bson.D{
						{"$gt", int32(0)},
						{"$lt", int32(43)},
					}},
				}},
			},
		},
		"UnexpectedFilterString": {
			filter:     bson.D{{"v", bson.D{{"$elemMatch", "foo"}}}},
			resultType: EmptyResult,
		},
		"WhereInsideElemMatch": {
			filter:     bson.D{{"v", bson.D{{"$elemMatch", bson.D{{"$where", "123"}}}}}},
			resultType: EmptyResult,
		},
		"TextInsideElemMatch": {
			filter:     bson.D{{"v", bson.D{{"$elemMatch", bson.D{{"$text", "123"}}}}}},
			resultType: EmptyResult,
		},
		"GtField": {
			filter: bson.D{{"v", bson.D{
				{
					"$elemMatch",
					bson.D{
						{"$gt", int32(0)},
						{"foo", int32(42)},
					},
				},
			}}},
			resultType:       EmptyResult,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/785",
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryArrayCompatEquality(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"One": {
			filter: bson.D{{"v", bson.A{int32(42)}}},
		},
		"Two": {
			filter:     bson.D{{"v", bson.A{42, "foo"}}},
			resultType: EmptyResult,
		},
		"Three": {
			filter: bson.D{{"v", bson.A{int32(42), "foo", nil}}},
		},
		"Three-reverse": {
			filter: bson.D{{"v", bson.A{nil, "foo", int32(42)}}},
		},
		"Empty": {
			filter: bson.D{{"v", bson.A{}}},
		},
		"Null": {
			filter: bson.D{{"v", bson.A{nil}}},
		},
	}

	testQueryCompat(t, testCases)
}

func TestQueryArrayCompatAll(t *testing.T) {
	t.Parallel()

	testCases := map[string]queryCompatTestCase{
		"String": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{"foo"}}}}},
		},
		"StringRepeated": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{"foo", "foo", "foo"}}}}},
		},
		"StringEmpty": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{""}}}}},
		},
		"Whole": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{int32(42)}}}}},
		},
		"WholeNotFound": {
			filter:     bson.D{{"v", bson.D{{"$all", bson.A{int32(46)}}}}},
			resultType: EmptyResult,
		},
		"Zero": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{0}}}}},
		},
		"Double": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{42.13}}}}},
		},
		"DoubleMax": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{math.MaxFloat64}}}}},
		},
		"DoubleMin": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{math.SmallestNonzeroFloat64}}}}},
		},
		"MultiAll": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{"foo", 42}}}}},
		},
		"MultiAllWithNil": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{"foo", nil}}}}},
		},
		"Empty": {
			filter:     bson.D{{"v", bson.D{{"$all", bson.A{}}}}},
			resultType: EmptyResult,
		},
		"NotFound": {
			filter:     bson.D{{"v", bson.D{{"$all", bson.A{"hello"}}}}},
			resultType: EmptyResult,
		},
		"$allNeedsAnArrayInt": {
			filter:     bson.D{{"v", bson.D{{"$all", 1}}}},
			resultType: EmptyResult,
		},
		"$allNeedsAnArrayNil": {
			filter:     bson.D{{"v", bson.D{{"$all", nil}}}},
			resultType: EmptyResult,
		},
		"WholeInTheMiddle": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{int32(43)}}}}},
		},
		"WholeTwoRepeated": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{int32(42), int32(43), int32(43), int32(42)}}}}},
		},
		"Nil": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{nil}}}}},
		},
		"NilRepeated": {
			filter: bson.D{{"v", bson.D{{"$all", bson.A{nil, nil, nil}}}}},
		},
	}

	testQueryCompat(t, testCases)
}
