package main

import (
	"bytes"
	"net"
	"testing"

	"github.com/miekg/dns"
)

type MockAddr struct{}

func (m *MockAddr) Network() string {
	return "tcp"
}

func (m *MockAddr) String() string {
	return "127.0.0.1"
}

type MockRR string

func (m MockRR) Header() *dns.RR_Header {
	return &dns.RR_Header{}
}

func (m MockRR) String() string {
	return string(m)
}

type MockResponseWriter struct {
	m dns.Msg
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{}
}

func (m *MockResponseWriter) GetM() *dns.Msg {
	return &m.m
}

func (m *MockResponseWriter) LocalAddr() net.Addr {
	return &MockAddr{}
}

func (m *MockResponseWriter) RemoteAddr() net.Addr {
	return &MockAddr{}
}

func (m *MockResponseWriter) WriteMsg(msg *dns.Msg) error {
	m.m = *msg
	return nil
}

func (m *MockResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (m *MockResponseWriter) Close() error {
	return nil
}

func (m *MockResponseWriter) TsigStatus() error {
	return nil
}

func (m *MockResponseWriter) TsigTimersOnly(bool) {}

func (m *MockResponseWriter) Hijack() {}

func Test_dnsHandler_ServeDNS(t *testing.T) {
	type fields struct {
		zone       string
		svcMap     map[string]net.IP
		shutdownCh chan struct{}
	}
	type args struct {
		w dns.ResponseWriter
		r *dns.Msg
	}
	type ans struct {
		m              dns.Msg
		expectedAnswer string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		answer ans
	}{
		{
			"send record for existing A record",
			fields{
				zone: "foo",
				svcMap: map[string]net.IP{
					"bar.foo.": net.ParseIP("127.0.0.1"),
				},
			},
			args{
				w: NewMockResponseWriter(),
				r: &dns.Msg{
					Question: []dns.Question{
						dns.Question{
							Qtype: dns.TypeA,
							Name:  "bar.foo.",
						},
					},
				},
			},
			ans{
				m: dns.Msg{
					MsgHdr: dns.MsgHdr{
						Rcode: 0,
					},
				},
				expectedAnswer: "127.0.0.1",
			},
		},
		{
			"send record for non-existing A record",
			fields{
				zone: "foo",
				svcMap: map[string]net.IP{
					"bar.foo.": net.ParseIP("127.0.0.1"),
				},
			},
			args{
				w: NewMockResponseWriter(),
				r: &dns.Msg{
					Question: []dns.Question{
						dns.Question{
							Qtype: dns.TypeA,
							Name:  "nope.foo.",
						},
					},
				},
			},
			ans{
				m: dns.Msg{
					MsgHdr: dns.MsgHdr{
						Rcode: 3,
					},
				},
				expectedAnswer: "",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &dnsHandler{
				zone:       tt.fields.zone,
				svcMap:     tt.fields.svcMap,
				shutdownCh: tt.fields.shutdownCh,
			}
			h.ServeDNS(tt.args.w, tt.args.r)

			rCode := tt.answer.m.MsgHdr.Rcode
			w := tt.args.w.(*MockResponseWriter)
			if tt.answer.m.MsgHdr.Rcode != w.GetM().Rcode {
				t.Errorf("ServeDNS() Rcode = %v, want %v", rCode, w.GetM().Rcode)
			}

			if tt.answer.expectedAnswer == "" && w.GetM().Answer != nil {
				t.Errorf("ServeDNS() expected no answer, got %v", w.GetM().Answer)
			}

			if tt.answer.expectedAnswer != "" {
				if w.GetM().Answer == nil {
					t.Errorf("ServeDNS() expected answer, got nil")
				}

				a := w.GetM().Answer[0].(*dns.A).A
				if bytes.Compare(a, net.ParseIP(MockRR(tt.answer.expectedAnswer).String())) != 0 {
					t.Errorf("got %v, expected %v", a, MockRR(tt.answer.expectedAnswer).String())
				}
			}
		})
	}
}
