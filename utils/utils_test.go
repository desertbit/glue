/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 * 
 *  SPDX-License-Identifier: MIT
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
