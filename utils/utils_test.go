/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright DesertBit
 *  Author: Roland Singer
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package utils

import (
	"testing"
)

func TestUnmarshalValues(t *testing.T) {
	first, second, err := UnmarshalValues(MarshalValues("1", "2"))
	if err != nil {
		t.Error(err.Error())
	} else if first != "1" || second != "2" {
		t.Fail()
	}

	first, second, err = UnmarshalValues(MarshalValues("1s"+delimiter+"jsd", "efsf2"+delimiter+"9as"))
	if err != nil {
		t.Error(err.Error())
	} else if first != "1s"+delimiter+"jsd" || second != "efsf2"+delimiter+"9as" {
		t.Fail()
	}

	first, second, err = UnmarshalValues("11" + delimiter + "firstsecond")
	if err != nil {
		t.Error(err.Error())
	} else if first != "firstsecond" || second != "" {
		t.Fail()
	}

	first, second, err = UnmarshalValues("12" + delimiter + "firstsecond")
	if err == nil {
		t.Fail()
	}
}
