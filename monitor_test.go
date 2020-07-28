package main

import "testing"

type MockFetcher struct{}

func NewMockFetcher(service string) (Fetcher, error) {
	return &MockFetcher{}, nil
}

func (m *MockFetcher) Fetch(service string) []string {
	return []string{}
}

func TestMonitor_Run(t *testing.T) {
	type fields struct {
		Fetcher    func(string) (Fetcher, error)
		services   []string
		shutdownCh chan struct{}
	}
	type args struct {
		notify chan<- *RecordEntry
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"empty Fetcher",
			fields{
				shutdownCh: make(chan struct{}),
			},
			args{
				notify: make(chan *RecordEntry),
			},
			true,
		},
		{
			"valid Fetcher",
			fields{
				Fetcher:    NewMockFetcher,
				shutdownCh: make(chan struct{}),
			},
			args{
				notify: make(chan *RecordEntry),
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Monitor{
				Fetcher:    tt.fields.Fetcher,
				services:   tt.fields.services,
				shutdownCh: tt.fields.shutdownCh,
			}
			if err := m.Run(tt.args.notify); (err != nil) != tt.wantErr {
				t.Errorf("Monitor.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
