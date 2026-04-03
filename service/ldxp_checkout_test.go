package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractLdxpAutoSubmitFormParsesActionAndInputs(t *testing.T) {
	html := `<form id='alipaysubmit' name='alipaysubmit' action='https://openapi.alipay.com/gateway.do?charset=UTF-8' method='POST'><input type='hidden' name='method' value='alipay.trade.page.pay'/><input type='hidden' name='biz_content' value='{"subject":"demo"}'/></form>`

	form, ok := extractLdxpAutoSubmitForm(html)
	if !ok {
		t.Fatal("expected form to be parsed")
	}
	if form.Action != "https://openapi.alipay.com/gateway.do?charset=UTF-8" {
		t.Fatalf("unexpected action %q", form.Action)
	}
	if form.Method != http.MethodPost {
		t.Fatalf("expected POST method, got %q", form.Method)
	}
	if form.Fields.Get("method") != "alipay.trade.page.pay" {
		t.Fatalf("expected method field to be preserved, got %q", form.Fields.Get("method"))
	}
}

func TestExtractAlipayCheckoutDataReadsQRCodeFields(t *testing.T) {
	html := `
<div id="hidden-input-area" class="fn-hide">
    <input name="qrCode" type="hidden" value="https://qr.alipay.com/demo-code" id="J_qrCode" />
    <input name="qrImgUrl" type="hidden" value="https://mobilecodec.alipay.com/show.htm?code=demo-code" id="J_qrImgUrl" />
</div>`

	data := extractAlipayCheckoutData(html)
	if data.QRCode != "https://qr.alipay.com/demo-code" {
		t.Fatalf("expected qr code, got %q", data.QRCode)
	}
	if data.QRImageURL != "https://mobilecodec.alipay.com/show.htm?code=demo-code" {
		t.Fatalf("expected qr image url, got %q", data.QRImageURL)
	}
}

func TestFetchLdxpCheckoutDataFollowsCheckoutForm(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/shopApi/Pay/payment":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<form id='alipaysubmit' name='alipaysubmit' action='` + server.URL + `/gateway.do?charset=UTF-8' method='POST'><input type='hidden' name='method' value='alipay.trade.page.pay'/><input type='hidden' name='biz_content' value='{"subject":"demo"}'/></form>`))
		case "/gateway.do":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("method") != "alipay.trade.page.pay" {
				t.Fatalf("expected gateway form payload, got %q", r.Form.Get("method"))
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`
<div id="hidden-input-area" class="fn-hide">
    <input name="qrCode" type="hidden" value="https://qr.alipay.com/demo-code" id="J_qrCode" />
    <input name="qrImgUrl" type="hidden" value="https://mobilecodec.alipay.com/show.htm?code=demo-code" id="J_qrImgUrl" />
</div>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := server.Client()
	data, err := fetchLdxpCheckoutData(
		context.Background(),
		server.URL+"/shopApi/Pay/payment?trade_no=demo",
		client,
		"Mozilla/5.0 Chrome/136.0.0.0 Safari/537.36",
	)
	if err != nil {
		t.Fatalf("expected checkout data, got error %v", err)
	}
	if data.QRCode != "https://qr.alipay.com/demo-code" {
		t.Fatalf("expected qr code, got %q", data.QRCode)
	}
	if !strings.Contains(data.QRImageURL, "mobilecodec.alipay.com/show.htm?code=demo-code") {
		t.Fatalf("expected qr image url, got %q", data.QRImageURL)
	}
}
