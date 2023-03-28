package exporter

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestLoadUserFile(t *testing.T) {
	for _, tst := range []struct {
		name     string
		userFile string
		ok       bool
	}{
		{
			name:     "load-user-file-success",
			userFile: "../contrib/sample-user-file.json",
			ok:       true,
		},
		{
			name:     "load-user-file-missing",
			userFile: "non-existent.json",
			ok:       false,
		},
		{
			name:     "load-user-file-malformed",
			userFile: "../contrib/sample-user-file.json-malformed",
			ok:       false,
		},
	} {
		t.Run(tst.name, func(t *testing.T) {
			_, err := LoadUserFile(tst.userFile)
			if err == nil && !tst.ok {
				t.Fatalf("Test Failed, result is not what we want")
			}
			if err != nil && tst.ok {
				t.Fatalf("Test Failed, result is not what we want")
			}
		})
	}
}

func TestUserMap(t *testing.T) {
	userFile := "../contrib/sample-user-file.json"
	userMap, err := LoadUserFile(userFile)
	if err != nil {
		t.Fatalf("Test Failed, error: %v", err)
	}

	if len(userMap) == 0 {
		t.Fatalf("User map is empty -skipping")
	}

	for _, tst := range []struct {
		name string
		addr string
		want string
	}{
		{name: "user-hit", addr: "redis://pwd-redis6:6390", want: "exporter"},
		{name: "user-missed", addr: "Non-existent-redis-host", want: ""},
	} {
		t.Run(tst.name, func(t *testing.T) {
			user := userMap[tst.addr]
			if !strings.Contains(user, tst.want) {
				t.Errorf("redis host: %s    user is not what we want", tst.addr)
			}
		})
	}
}

func TestHTTPScrapeWithUserFile(t *testing.T) {
	userFile := "../contrib/sample-user-file.json"
	userMap, err := LoadUserFile(userFile)
	if err != nil {
		t.Fatalf("Test Failed, error: %v", err)
	}

	passwordMap := map[string]string{
		"redis://pwd-redis6:6390": "exporter-password",
	}

	if len(userMap) == 0 {
		t.Fatalf("User map is empty!")
	}
	for _, tst := range []struct {
		name           string
		addr           string
		wants          []string
		useWrongUser   bool
		wantStatusCode int
	}{
		{name: "scrape-user-file", addr: "redis://pwd-redis6:6390", wants: []string{
			"uptime_in_seconds",
			"test_up 1",
		}},
		{name: "scrape-user-file-wrong-user", addr: "redis://pwd-redis6:6390", useWrongUser: true, wants: []string{
			"test_up 0",
		}},
	} {
		if tst.useWrongUser {
			userMap[tst.addr] = "wrong-user"
		}
		options := Options{
			Namespace:   "test",
			UserMap:     userMap,
			PasswordMap: passwordMap,
			PingOnConnect: true,
			Registry: prometheus.NewRegistry(),
		}
		t.Run(tst.name, func(t *testing.T) {
			e, _ := NewRedisExporter(tst.addr, options)
			ts := httptest.NewServer(e)

			u := ts.URL
			u += "/scrape"
			v := url.Values{}
			v.Add("target", tst.addr)

			up, _ := url.Parse(u)
			up.RawQuery = v.Encode()
			u = up.String()

			wantStatusCode := http.StatusOK
			if tst.wantStatusCode != 0 {
				wantStatusCode = tst.wantStatusCode
			}

			gotStatusCode, body := downloadURLWithStatusCode(t, u)

			if gotStatusCode != wantStatusCode {
				t.Fatalf("got status code: %d   wanted: %d", gotStatusCode, wantStatusCode)
				return
			}

			// we can stop here if we expected a non-200 response
			if wantStatusCode != http.StatusOK {
				return
			}

			for _, want := range tst.wants {
				if !strings.Contains(body, want) {
					t.Errorf("url: %s    want metrics to include %q, have:\n%s", u, want, body)
					break
				}
			}
			ts.Close()
		})

	}

}
