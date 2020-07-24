package main

import (
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	type fields struct {
		Bind     string
		PromBind string
		Zone     string
		Services []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"valid config",
			fields{
				Bind:     ":5300",
				PromBind: ":5301",
				Zone:     "foo",
				Services: []string{
					"bar",
				},
			},
			false,
		},
		{
			"missing Bind",
			fields{
				PromBind: ":5301",
				Zone:     "foo",
				Services: []string{
					"bar",
				},
			},
			true,
		},
		{
			"missing PromBind",
			fields{
				Bind: ":5300",
				Zone: "foo",
				Services: []string{
					"bar",
				},
			},
			true,
		},
		{
			"missing Zone",
			fields{
				Bind:     ":5300",
				PromBind: ":5301",
				Services: []string{
					"bar",
				},
			},
			true,
		},
		{
			"missing Services",
			fields{
				Bind:     ":5300",
				PromBind: ":5301",
				Zone:     "foo",
			},
			true,
		},
		{
			"valid Config with multiple services",
			fields{
				Bind:     ":5300",
				PromBind: ":5301",
				Zone:     "foo",
				Services: []string{
					"foo",
					"bar",
				},
			},
			false,
		},
		{
			"valid Config with duplicate services",
			fields{
				Bind:     ":5300",
				PromBind: ":5301",
				Zone:     "foo",
				Services: []string{
					"bar",
					"bar",
				},
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Bind:     tt.fields.Bind,
				PromBind: tt.fields.PromBind,
				Zone:     tt.fields.Zone,
				Services: tt.fields.Services,
			}
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
