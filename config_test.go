package main

import "testing"

func Test_validateConfig(t *testing.T) {
	type args struct {
		c *config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"valid config",
			args{
				c: &config{
					Bind: ":5300",
					Zone: "foo",
					Services: []string{
						"bar",
					},
				},
			},
			false,
		},
		{
			"missing Bind",
			args{
				c: &config{
					Zone: "foo",
					Services: []string{
						"bar",
					},
				},
			},
			true,
		},
		{
			"missing Zone",
			args{
				c: &config{
					Bind: ":5300",
					Services: []string{
						"bar",
					},
				},
			},
			true,
		},
		{
			"missing Services",
			args{
				c: &config{
					Bind: ":5300",
					Zone: "foo",
				},
			},
			true,
		},
		{
			"valid config with multiple services",
			args{
				c: &config{
					Bind: ":5300",
					Zone: "foo",
					Services: []string{
						"foo",
						"bar",
					},
				},
			},
			false,
		},
		{
			"valid config with duplicate services",
			args{
				c: &config{
					Bind: ":5300",
					Zone: "foo",
					Services: []string{
						"bar",
						"bar",
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateConfig(tt.args.c); (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
