/*
Copyright 2023 zncdatadev.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1_test

import (
	"testing"

	"github.com/zncdatadev/operator-go/pkg/testutil"
)

// Defaults inside a role or role-group config block prevent role-level values from being inherited
// whenever the role group sets any sibling field. Product defaults belong at consumption time.
func TestGeneratedCRDsHaveNoInheritedConfigDefaults(t *testing.T) {
	t.Parallel()

	const crdGlob = "../../config/crd/bases/*.yaml"
	matcher := testutil.HaveNoInheritedConfigDefaults()
	ok, err := matcher.Match(crdGlob)
	if err != nil {
		t.Fatalf("inspect generated CRDs: %v", err)
	}
	if !ok {
		t.Fatal(matcher.FailureMessage(crdGlob))
	}
}
