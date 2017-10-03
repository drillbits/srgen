//    Copyright 2017 drillbits
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package testdata

import "testing"

func TestServiceRegistry_Validate(t *testing.T) {
	type fields struct {
		valid      bool
		BarService BarService
		FooService FooService
		ZService   ZService
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "valid services",
			fields: fields{
				valid:      false,
				BarService: struct{}{},
				FooService: struct{}{},
				ZService:   struct{}{},
			},
			wantErr: false,
		},
		{
			name: "invalid services",
			fields: fields{
				valid:      false,
				BarService: struct{}{},
				FooService: nil,
				ZService:   struct{}{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reg := &ServiceRegistry{
				valid:      tt.fields.valid,
				BarService: tt.fields.BarService,
				FooService: tt.fields.FooService,
				ZService:   tt.fields.ZService,
			}
			if err := reg.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ServiceRegistry.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
