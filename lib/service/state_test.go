/*
 * Teleport
 * Copyright (C) 2023  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcessStateGetState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		desc   string
		states map[string]*componentState
		want   componentStateEnum
	}{
		{
			desc:   "no components",
			states: map[string]*componentState{},
			want:   stateStarting,
		},
		{
			desc: "one component in stateOK",
			states: map[string]*componentState{
				"one": {state: stateOK},
			},
			want: stateOK,
		},
		{
			desc: "multiple components in stateOK",
			states: map[string]*componentState{
				"one":   {state: stateOK},
				"two":   {state: stateOK},
				"three": {state: stateOK},
			},
			want: stateOK,
		},
		{
			desc: "multiple components, one is degraded",
			states: map[string]*componentState{
				"one":   {state: stateRecovering},
				"two":   {state: stateDegraded},
				"three": {state: stateOK},
			},
			want: stateDegraded,
		},
		{
			desc: "multiple components, one is recovering",
			states: map[string]*componentState{
				"one":   {state: stateOK},
				"two":   {state: stateRecovering},
				"three": {state: stateOK},
			},
			want: stateRecovering,
		},
		{
			desc: "multiple components, one is starting",
			states: map[string]*componentState{
				"one":   {state: stateOK},
				"two":   {state: stateStarting},
				"three": {state: stateOK},
			},
			want: stateStarting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			ps := &processState{states: tt.states}
			got := ps.getState()
			require.Equal(t, tt.want, got)
		})
	}
}
