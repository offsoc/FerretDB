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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/FerretDB/FerretDB/v2/integration/setup"
	"github.com/FerretDB/FerretDB/v2/integration/shareddata"
)

// countCompatTestCase describes count compatibility test case.
type countCompatTestCase struct {
	filter bson.D // required, filter for the query

	// TODO https://github.com/FerretDB/FerretDB/issues/2255
	// those two probably should be of the same type
	optSkip          any   // optional, skip option for the query, defaults to nil
	limit            int64 // optional, limit option for the query, defaults to 0
	failsForFerretDB string

	resultType CompatTestCaseResultType // defaults to NonEmptyResult
}

// testCountCompat tests count compatibility test cases.
func testCountCompat(t *testing.T, testCases map[string]countCompatTestCase) {
	t.Helper()

	// Use shared setup because find queries can't modify data.
	//
	// Use read-only user.
	// TODO https://github.com/FerretDB/FerretDB/issues/1025
	s := setup.SetupCompatWithOpts(t, &setup.SetupCompatOpts{
		Providers:                shareddata.AllProviders(),
		AddNonExistentCollection: true,
	})
	ctx, targetCollections, compatCollections := s.Ctx, s.TargetCollections, s.CompatCollections

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Helper()

			t.Parallel()

			filter := tc.filter
			require.NotNil(t, filter, "filter should be set")

			var nonEmptyResults bool
			for i := range targetCollections {
				targetCollection := targetCollections[i]
				compatCollection := compatCollections[i]

				t.Run(targetCollection.Name(), func(tt *testing.T) {
					tt.Helper()

					var t testing.TB = tt

					if tc.failsForFerretDB != "" {
						t = setup.FailsForFerretDB(tt, tc.failsForFerretDB)
					}

					// RunCommand must be used to test the count command.
					// It's not possible to use CountDocuments because it calls aggregation.
					var targetRes, compatRes bson.D
					targetErr := targetCollection.Database().RunCommand(ctx, bson.D{
						{"count", targetCollection.Name()},
						{"query", filter},
						{"skip", tc.optSkip},
						{"limit", tc.limit},
					}).Decode(&targetRes)
					compatErr := compatCollection.Database().RunCommand(ctx, bson.D{
						{"count", compatCollection.Name()},
						{"query", filter},
						{"skip", tc.optSkip},
						{"limit", tc.limit},
					}).Decode(&compatRes)

					if targetErr != nil {
						t.Logf("Target error: %v", targetErr)
						t.Logf("Compat error: %v", compatErr)

						// error messages are intentionally not compared
						AssertMatchesCommandError(t, compatErr, targetErr)

						return
					}
					require.NoError(t, compatErr, "compat error; target returned no error")

					t.Logf("Compat (expected) result: %v", compatRes)
					t.Logf("Target (actual)   result: %v", targetRes)

					AssertEqualDocuments(t, compatRes, targetRes)

					if targetRes != nil || compatRes != nil {
						nonEmptyResults = true
					}
				})
			}

			switch tc.resultType {
			case NonEmptyResult:
				if tc.failsForFerretDB != "" {
					return
				}

				assert.True(t, nonEmptyResults, "expected non-empty results")
			case EmptyResult:
				assert.False(t, nonEmptyResults, "expected empty results")
			default:
				t.Fatalf("unknown result type %v", tc.resultType)
			}
		})
	}
}

func TestCountCompat(t *testing.T) {
	t.Parallel()

	testCases := map[string]countCompatTestCase{
		"Empty": {
			filter:  bson.D{},
			optSkip: 0,
		},
		"IDString": {
			filter:  bson.D{{"_id", "string"}},
			optSkip: 0,
		},
		"IDObjectID": {
			filter:  bson.D{{"_id", primitive.NilObjectID}},
			optSkip: 0,
		},
		"IDNotExists": {
			filter:  bson.D{{"_id", "count-id-not-exists"}},
			optSkip: 0,
		},
		"IDBool": {
			filter:  bson.D{{"_id", "bool-true"}},
			optSkip: 0,
		},
		"FieldTrue": {
			filter:  bson.D{{"v", true}},
			optSkip: 0,
		},
		"FieldTypeArrays": {
			filter:  bson.D{{"v", bson.D{{"$type", "array"}}}},
			optSkip: 0,
		},

		"LimitAlmostAll": {
			filter: bson.D{},
			limit:  int64(len(shareddata.Strings.Docs()) - 1),
		},
		"LimitAll": {
			filter: bson.D{},
			limit:  int64(len(shareddata.Strings.Docs())),
		},
		"LimitMore": {
			filter: bson.D{},
			limit:  int64(len(shareddata.Strings.Docs()) + 1),
		},

		"SkipSimple": {
			filter:  bson.D{},
			optSkip: 1,
		},
		"SkipAlmostAll": {
			filter:  bson.D{},
			optSkip: len(shareddata.Strings.Docs()) - 1,
		},
		"SkipAll": {
			filter:  bson.D{},
			optSkip: len(shareddata.Strings.Docs()),
		},
		"SkipMore": {
			filter:  bson.D{},
			optSkip: len(shareddata.Strings.Docs()) + 1,
		},
		"SkipBig": {
			filter:  bson.D{},
			optSkip: 1000,
		},
		"SkipDouble": {
			filter:           bson.D{},
			optSkip:          1.111,
			failsForFerretDB: "https://github.com/FerretDB/FerretDB-DocumentDB/issues/405",
		},
		"SkipNegative": {
			filter:     bson.D{},
			optSkip:    -1,
			resultType: EmptyResult,
		},
		"SkipNegativeDouble": {
			filter:     bson.D{},
			optSkip:    -1.111,
			resultType: EmptyResult,
		},
		"SkipNegativeDoubleCeil": {
			filter:     bson.D{},
			optSkip:    -1.888,
			resultType: EmptyResult,
		},
		"SkipMinFloat": {
			filter:     bson.D{},
			optSkip:    -math.MaxFloat64,
			resultType: EmptyResult,
		},
		"SkipNull": {
			filter: bson.D{},
		},
		"SkipString": {
			filter:     bson.D{},
			optSkip:    "foo",
			resultType: EmptyResult,
		},
	}

	testCountCompat(t, testCases)
}
