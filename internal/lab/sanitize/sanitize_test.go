package sanitize

import (
	"strings"
	"testing"
)

func TestApply_SessionToken_RedirectScript(t *testing.T) {
	in := `window.parent.location.href = "http://192.168.1.1/ABCDEFGHIJKLMNOP/userRpm/Index.htm";`
	out := Apply(in, Default())
	if strings.Contains(out, "ABCDEFGHIJKLMNOP") {
		t.Fatalf("token not redacted: %q", out)
	}
	if !strings.Contains(out, PlaceholderSessionToken) {
		t.Fatalf("placeholder missing: %q", out)
	}
}

func TestApply_SessionToken_PathAnywhere(t *testing.T) {
	in := `GET /ABCDEFGHIJKLMNOP/userRpm/StatusRpm.htm`
	out := Apply(in, Default())
	if strings.Contains(out, "ABCDEFGHIJKLMNOP") {
		t.Fatalf("token not redacted: %q", out)
	}
	if !strings.Contains(out, "/"+PlaceholderSessionToken+"/userRpm/") {
		t.Fatalf("placeholder not in expected position: %q", out)
	}
}

func TestApply_Bare16Char_NotRedactedInPreserveMode(t *testing.T) {
	// Hardware strings can contain 16-char hex-like sequences that we
	// must NOT redact.
	in := `Hardware: WR841N v8 00000000`
	out := Apply(in, Default())
	if out != in {
		t.Fatalf("fingerprint altered: %q", out)
	}
}

func TestApply_Authorization_BasicValue(t *testing.T) {
	in := `Authorization: Basic YWRtaW46MTIzNDU2Nzg5MA==`
	out := Apply(in, Default())
	if strings.Contains(out, "YWRtaW46MTIzNDU2Nzg5MA==") {
		t.Fatalf("base64 not redacted: %q", out)
	}
	if !strings.Contains(out, PlaceholderAuthSecret) {
		t.Fatalf("placeholder missing: %q", out)
	}
}

func TestApply_MAC_DefaultClient(t *testing.T) {
	in := `client mac=AA:BB:CC:DD:EE:01`
	out := Apply(in, Default())
	if strings.Contains(out, "AA:BB:CC:DD:EE:01") {
		t.Fatalf("client MAC not redacted: %q", out)
	}
	if !strings.Contains(out, PlaceholderClientMAC) {
		t.Fatalf("placeholder missing: %q", out)
	}
}

func TestApply_MAC_RouterOnly(t *testing.T) {
	in := `wanMac=AA:BB:CC:DD:EE:02`
	out := Apply(in, Options{RouterMACOnly: true})
	if strings.Contains(out, "AA:BB:CC:DD:EE:02") {
		t.Fatalf("router MAC not redacted: %q", out)
	}
	if !strings.Contains(out, PlaceholderRouterMAC) {
		t.Fatalf("placeholder missing: %q", out)
	}
}

func TestApply_PasswordField_Form(t *testing.T) {
	in := `password=hunter2&username=admin`
	out := Apply(in, Default())
	if strings.Contains(out, "hunter2") {
		t.Fatalf("password not redacted: %q", out)
	}
	if !strings.Contains(out, "password="+PlaceholderSecret) {
		t.Fatalf("placeholder missing: %q", out)
	}
	if !strings.Contains(out, "username=admin") {
		t.Fatalf("non-secret field redacted by mistake: %q", out)
	}
}

func TestApply_PasswordField_JSON(t *testing.T) {
	in := `{"username":"admin","password":"hunter2"}`
	out := Apply(in, Default())
	if strings.Contains(out, "hunter2") {
		t.Fatalf("password not redacted: %q", out)
	}
}

func TestApply_SSID(t *testing.T) {
	in := `var ssid="HomeNetwork-5G"`
	out := Apply(in, Default())
	if strings.Contains(out, "HomeNetwork-5G") {
		t.Fatalf("SSID not redacted: %q", out)
	}
	if !strings.Contains(out, PlaceholderSSID) {
		t.Fatalf("placeholder missing: %q", out)
	}
}

func TestApply_WiFiPSK(t *testing.T) {
	in := `psk=mysecretwifi&other=foo`
	out := Apply(in, Default())
	if strings.Contains(out, "mysecretwifi") {
		t.Fatalf("PSK not redacted: %q", out)
	}
	if !strings.Contains(out, PlaceholderWiFiPassword) {
		t.Fatalf("placeholder missing: %q", out)
	}
	if !strings.Contains(out, "other=foo") {
		t.Fatalf("non-PSK field redacted: %q", out)
	}
}

func TestApply_FingerprintPreserved(t *testing.T) {
	cases := []string{
		"TP-Link TL-WR841N/ND",
		"Hardware: WR841N v8 00000000",
		"Firmware: 3.13.33 Build 130506 Rel.48660n",
	}
	for _, in := range cases {
		out := Apply(in, Default())
		if out != in {
			t.Errorf("fingerprint altered: in=%q out=%q", in, out)
		}
	}
}

func TestApply_NoMatchUnchanged(t *testing.T) {
	in := `Reachable: true`
	out := Apply(in, Default())
	if out != in {
		t.Fatalf("unexpected change: %q", out)
	}
}

func TestApply_EmptyUnchanged(t *testing.T) {
	if out := Apply("", Default()); out != "" {
		t.Fatalf("empty input produced %q", out)
	}
}

func TestExtractSessionToken_FromRedirect(t *testing.T) {
	in := `window.parent.location.href = "http://192.168.1.1/ABCDEFGHIJKLMNOP/userRpm/Index.htm";`
	got := ExtractSessionToken(in)
	if got != "ABCDEFGHIJKLMNOP" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSessionToken_FromPath(t *testing.T) {
	in := `GET /QWERTYUIOPASDFGH/userRpm/StatusRpm.htm`
	got := ExtractSessionToken(in)
	if got != "QWERTYUIOPASDFGH" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractSessionToken_NoMatch(t *testing.T) {
	in := `random content without token`
	if got := ExtractSessionToken(in); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestIsSessionTokenMatch(t *testing.T) {
	if !IsSessionTokenMatch(`http://192.168.1.1/ABCDEFGHIJKLMNOP/userRpm/Index.htm`) {
		t.Fatalf("expected match")
	}
	if IsSessionTokenMatch(`no token here`) {
		t.Fatalf("unexpected match")
	}
}

func TestApply_BareTokenOutsideFingerprint_StillRedacted(t *testing.T) {
	// 16-char alphanumeric that is NOT a fingerprint prefix/suffix and
	// NOT inside the structural redirect should still be redacted when
	// PreserveFingerprint is disabled.
	in := `transient: ABCDEFGHIJKLMNOP`
	out := Apply(in, Options{})
	if strings.Contains(out, "ABCDEFGHIJKLMNOP") {
		t.Fatalf("bare token not redacted: %q", out)
	}
}

func TestApply_FailClosed_UnclassifiableCookieValue(t *testing.T) {
	// Authorization header with an unrecognizable but secret-looking
	// token. The sanitizer must err on the side of redaction.
	in := `X-Custom-Auth: thisIsASecretValue123`
	out := Apply(in, Default())
	// 20+ char base64-ish blobs are matched by basicAuthValue. If the
	// heuristic matches, this turns into AUTH_SECRET. Either way, the
	// raw value must not survive.
	if strings.Contains(out, "thisIsASecretValue123") {
		t.Fatalf("secret not redacted: %q", out)
	}
}
func TestApply_QuotedSessionToken_Redacted(t *testing.T) {
	in := `debug="ABCDEFGHIJKLMNOP"`
	out := Apply(in, Default())
	if strings.Contains(out, "ABCDEFGHIJKLMNOP") {
		t.Fatalf("quoted token not redacted: %q", out)
	}
	if !strings.Contains(out, `"`+PlaceholderSessionToken+`"`) {
		t.Fatalf("placeholder missing: %q", out)
	}
}

func TestApply_SingleQuotedSessionToken_Redacted(t *testing.T) {
	in := `debug='ABCDEFGHIJKLMNOP'`
	out := Apply(in, Default())
	if strings.Contains(out, "ABCDEFGHIJKLMNOP") {
		t.Fatalf("single-quoted token not redacted: %q", out)
	}
	if !strings.Contains(out, `'`+PlaceholderSessionToken+`'`) {
		t.Fatalf("placeholder missing: %q", out)
	}
}
