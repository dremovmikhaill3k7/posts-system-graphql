package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIssueParseAndCookie(t *testing.T) {
	j, err := NewJWT("test-secret-key-32bytes-long!!", 24*time.Hour, false, "")
	if err != nil {
		t.Fatal(err)
	}

	token, err := j.Issue(42)
	if err != nil {
		t.Fatal(err)
	}

	id, err := j.Parse(token)
	if err != nil || id != 42 {
		t.Fatalf("parse: id=%d err=%v", id, err)
	}

	rr := httptest.NewRecorder()
	if err := j.SetAuthCookie(rr, 42); err != nil {
		t.Fatal(err)
	}
	cookies := rr.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != CookieName {
		t.Fatal("auth cookie not set")
	}
	if !cookies[0].HttpOnly {
		t.Fatal("cookie must be HttpOnly")
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/query", nil)
	req.AddCookie(cookies[0])

	id, ok := j.UserIDFromRequest(req)
	if !ok || id != 42 {
		t.Fatalf("from request: id=%d ok=%v", id, ok)
	}
}
