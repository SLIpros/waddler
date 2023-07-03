package value

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSetSliceString1(t *testing.T) {
	type args struct {
		field     *reflect.Value
		arr       []string
		equalFunc func()
	}

	tests := []struct {
		name    string
		args    func() args
		wantErr bool
	}{
		{
			name: "String",
			args: func() args {
				var testStruct struct {
					S string
				}

				var field reflect.Value
				v := reflect.Indirect(reflect.ValueOf(&testStruct))

				for i := 0; i < v.NumField(); i++ {
					field = v.Field(i)
				}

				sl := []string{str, str}

				return args{
					field: &field,
					arr:   sl,
					equalFunc: func() {
						require.Equal(t, strings.Join(sl, ","), field.String())
					},
				}
			},
		},
		{
			name: "[]string",
			args: func() args {
				sl := []string{str, str}

				var testStruct struct {
					SL []string
				}

				var field reflect.Value
				v := reflect.Indirect(reflect.ValueOf(&testStruct))
				for i := 0; i < v.NumField(); i++ {
					field = v.Field(i)
				}

				return args{
					field: &field,
					arr:   sl,
					equalFunc: func() {
						require.Equal(t, sl, testStruct.SL)
					},
				}
			},
		},
		{
			name: "[]any",
			args: func() args {
				var testStruct struct {
					SL []any
				}

				var field reflect.Value
				v := reflect.Indirect(reflect.ValueOf(&testStruct))
				for i := 0; i < v.NumField(); i++ {
					field = v.Field(i)
				}

				sl := []string{str, str}
				return args{
					field: &field,
					arr:   sl,
					equalFunc: func() {
						require.Equal(t, []any{str, str}, testStruct.SL)
					},
				}
			},
		},
		{
			name: "[]string in any",
			args: func() args {
				var testStruct struct {
					SL any
				}

				testStruct.SL = make([]string, 0)

				var field reflect.Value
				v := reflect.Indirect(reflect.ValueOf(&testStruct))
				for i := 0; i < v.NumField(); i++ {
					field = v.Field(i)
				}

				sl := []string{str, str}
				return args{
					field: &field,
					arr:   sl,
					equalFunc: func() {
						require.Equal(t, []string{str, str}, testStruct.SL)
					},
				}
			},
		},
		{
			name:    "[]error",
			wantErr: true,
			args: func() args {
				var testStruct struct {
					SL []error
				}

				var field reflect.Value
				v := reflect.Indirect(reflect.ValueOf(&testStruct))
				for i := 0; i < v.NumField(); i++ {
					field = v.Field(i)
				}

				sl := []string{str, str}
				return args{
					field: &field,
					arr:   sl,
				}
			},
		},
		{
			name:    "error",
			wantErr: true,
			args: func() args {
				var testStruct struct {
					Err error
				}

				testStruct.Err = errors.New("")

				var field reflect.Value
				v := reflect.Indirect(reflect.ValueOf(&testStruct))
				for i := 0; i < v.NumField(); i++ {
					field = v.Field(i)
				}
				sl := []string{str, str}

				return args{
					field: &field,
					arr:   sl,
				}
			},
		},
		{
			name:    "map[string]string",
			wantErr: true,
			args: func() args {
				sl := []string{str, str}

				var testStruct struct {
					M map[string]string
				}

				testStruct.M = make(map[string]string)

				var field reflect.Value
				v := reflect.Indirect(reflect.ValueOf(&testStruct))
				for i := 0; i < v.NumField(); i++ {
					field = v.Field(i)
				}

				return args{
					field: &field,
					arr:   sl,
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.args()
			err := SetSliceString(args.field, args.arr)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetSliceString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if args.equalFunc != nil {
				args.equalFunc()
			}
			if tt.wantErr {
				require.Error(t, err)
			}
		})
	}
}
