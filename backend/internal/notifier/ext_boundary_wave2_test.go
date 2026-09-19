// ext_boundary_wave2_test.go — EXT-BOUNDARY-WAVE2 S3 (2026-09-19).
//
// SMTP auth failure is an explicit failure (margin-call / kill-switch alerts
// must not be silently dropped on an unauthenticated path). A relay that
// does not advertise AUTH is a legitimate no-auth relay and skips auth
// silently.
//
// Adversarial (M5): restore the warn-continue → the auth sub-case REDs
// (Send succeeds with bad credentials).
package notifier

import (
	"net"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// startTestSMTP starts a minimal SMTP server speaking just enough of the
// protocol. advertiseAuth controls whether the AUTH extension is offered;
// rejectAuth makes the AUTH handshake fail.
func startTestSMTP(t *testing.T, advertiseAuth, rejectAuth bool) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		write := func(line string) { _, _ = conn.Write([]byte(line + "\r\n")) }
		read := func() string {
			buf := make([]byte, 512)
			n, _ := conn.Read(buf)
			return string(buf[:n])
		}
		write("220 test ESMTP")
		for {
			upper := strings.ToUpper(read())
			switch {
			case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
				write("250-test")
				if advertiseAuth {
					write("250 AUTH PLAIN")
				} else {
					write("250 OK")
				}
			case strings.HasPrefix(upper, "AUTH"):
				if rejectAuth {
					write("535 authentication credentials invalid")
				} else {
					write("235 authentication succeeded")
				}
			case strings.HasPrefix(upper, "MAIL FROM"):
				write("250 OK")
			case strings.HasPrefix(upper, "RCPT TO"):
				write("250 OK")
			case strings.HasPrefix(upper, "DATA"):
				write("354 go ahead")
				for {
					if strings.HasSuffix(read(), "\r\n.\r\n") {
						break
					}
				}
				write("250 OK queued")
			case strings.HasPrefix(upper, "QUIT"):
				write("221 bye")
				return
			default:
				write("250 OK")
			}
		}
	}()
	return l
}

func portOf(l net.Listener) string {
	_, port, _ := net.SplitHostPort(l.Addr().String())
	return port
}

// TestSMTP_AuthRejectedFailsClosed — the server advertises AUTH but rejects
// the credentials → Send returns an explicit smtp auth error.
//
// Adversarial (M5): restore the warn-continue → Send succeeds → RED.
func TestSMTP_AuthRejectedFailsClosed(t *testing.T) {
	l := startTestSMTP(t, true, true)
	defer l.Close()

	n := NewEmailNotifier(EmailConfig{
		Host: hostOf(l), Port: portOf(l), User: "user", Password: "wrong",
		From: "alerts@test", To: []string{"ops@test"},
	}, zap.NewNop())

	err := n.Send("margin call", "body")
	if err == nil || !strings.Contains(err.Error(), "smtp auth") {
		t.Fatalf("err = %v, want it to contain 'smtp auth' (credential rot must not silently send unauthenticated)", err)
	}
}

// TestSMTP_NoAuthAdvertisedSendsSilently — a relay that does not advertise
// AUTH is a legitimate no-auth relay → send succeeds without auth.
func TestSMTP_NoAuthAdvertisedSendsSilently(t *testing.T) {
	l := startTestSMTP(t, false, false)
	defer l.Close()

	n := NewEmailNotifier(EmailConfig{
		Host: hostOf(l), Port: portOf(l), User: "user", Password: "pass",
		From: "alerts@test", To: []string{"ops@test"},
	}, zap.NewNop())

	if err := n.Send("kill switch", "body"); err != nil {
		t.Fatalf("Send on no-AUTH relay err = %v, want nil", err)
	}
}

func hostOf(l net.Listener) string {
	host, _, _ := net.SplitHostPort(l.Addr().String())
	return host
}
