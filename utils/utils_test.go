/*
 *  Glue - Robust Go and Javascript Socket Library
 *  Copyright (C) 2015  Roland Singer <roland.singer[at]desertbit.com>
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU General Public License as published by
 *  the Free Software Foundation, either version 3 of the License, or
 *  (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU General Public License for more details.
 *
 *  You should have received a copy of the GNU General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
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
