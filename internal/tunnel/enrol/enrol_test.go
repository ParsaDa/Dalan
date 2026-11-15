package enrol

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newOffer(t *testing.T, panelURL string) (Offer, Link) {
	t.Helper()
	offer, err := NewOffer(panelURL)
	if err != nil {
		t.Fatalf("NewOffer: %v", err)
	}
	link, err := ParseLink(offer.Link)
	if err != nil {
		t.Fatalf("ParseLink(%q): %v", offer.Link, err)
	}
	return offer, link
}

func TestANodeAndThePanelExchangeSecretsThatNeverAppearOnTheWire(t *testing.T) {
	offer, link := newOffer(t, "http://203.0.113.10:2053/panel/")
	info := NodeInfo{Token: link.Token, Port: 2053, BasePath: "/secret-path/", Scheme: "http", APIToken: "node-api-token-value"}

	body, openReply, err := link.Seal(info)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	for _, secret := range []string{link.Token, info.APIToken, "secret-path"} {
		if bytes.Contains(body, []byte(secret)) {
			t.Fatalf("request body carries %q in the clear: %s", secret, body)
		}
	}
	id, err := RequestID(body)
	if err != nil || id != offer.ID {
		t.Fatalf("RequestID = %q, %v; want %q", id, err, offer.ID)
	}
	got, sealReply, err := OpenRequest(offer.PrivateKey, body)
	if err != nil {
		t.Fatalf("OpenRequest: %v", err)
	}
	if got != info {
		t.Fatalf("panel read %+v, want %+v", got, info)
	}

	reply, err := sealReply(Grant{Tunnel: "dalan://tunnel/the-bridge-end"})
	if err != nil {
		t.Fatalf("seal reply: %v", err)
	}
	if bytes.Contains(reply, []byte("the-bridge-end")) {
		t.Fatalf("reply carries the tunnel link in the clear: %s", reply)
	}
	grant, err := openReply(reply)
	if err != nil || grant.Tunnel != "dalan://tunnel/the-bridge-end" {
		t.Fatalf("node read %+v, %v", grant, err)
	}
}

func TestARequestSealedForAnotherPanelKeyIsRefused(t *testing.T) {
	_, link := newOffer(t, "http://203.0.113.10:2053/")
	other, _ := newOffer(t, "http://203.0.113.10:2053/")
	body, _, err := link.Seal(NodeInfo{Token: link.Token, Port: 2053, Scheme: "http", APIToken: "x"})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, _, err := OpenRequest(other.PrivateKey, body); !errors.Is(err, ErrSealed) {
		t.Fatalf("OpenRequest with another offer's key: err = %v, want ErrSealed", err)
	}
}

func TestAReplyCannotBeReplayedAsARequest(t *testing.T) {
	offer, link := newOffer(t, "http://203.0.113.10:2053/")
	body, _, err := link.Seal(NodeInfo{Token: link.Token, Port: 2053, Scheme: "http", APIToken: "x"})
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	_, sealReply, err := OpenRequest(offer.PrivateKey, body)
	if err != nil {
		t.Fatalf("OpenRequest: %v", err)
	}
	reply, _ := sealReply(Grant{Tunnel: "dalan://tunnel/x"})
	var req, rep map[string]string
	_ = json.Unmarshal(body, &req)
	_ = json.Unmarshal(reply, &rep)
	req["box"] = rep["box"]
	forged, _ := json.Marshal(req)
	if _, _, err := OpenRequest(offer.PrivateKey, forged); !errors.Is(err, ErrSealed) {
		t.Fatalf("a reply box presented as a request: err = %v, want ErrSealed", err)
	}
}

func TestParseLinkRefusesWhatIsNotAnEnrolmentLink(t *testing.T) {
	for _, link := range []string{"", "dalan://tunnel/abc", "dalan://enrol/!!!", "dalan://enrol/e30"} {
		if _, err := ParseLink(link); err == nil {
			t.Errorf("ParseLink(%q) succeeded", link)
		}
	}
}

func TestRedeemPostsToThePanelAndReportsARefusalInThePanelsWords(t *testing.T) {
	var offer Offer
	panel := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/base/"+RedeemPath {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		info, seal, err := OpenRequest(offer.PrivateKey, body)
		if err != nil {
			http.Error(w, "bad box", http.StatusBadRequest)
			return
		}
		if info.APIToken == "refuse-me" {
			http.Error(w, "this enrolment link has expired", http.StatusGone)
			return
		}
		reply, _ := seal(Grant{Tunnel: "dalan://tunnel/for-" + info.APIToken})
		_, _ = w.Write(reply)
	}))
	t.Cleanup(panel.Close)
	var link Link
	offer, link = newOffer(t, panel.URL+"/base/")

	grant, err := Redeem(context.Background(), panel.Client(), link, NodeInfo{Token: link.Token, Port: 1, Scheme: "http", APIToken: "tok"})
	if err != nil || grant.Tunnel != "dalan://tunnel/for-tok" {
		t.Fatalf("Redeem = %+v, %v", grant, err)
	}
	_, err = Redeem(context.Background(), panel.Client(), link, NodeInfo{Token: link.Token, Port: 1, Scheme: "http", APIToken: "refuse-me"})
	if err == nil || !strings.Contains(err.Error(), "this enrolment link has expired") {
		t.Fatalf("Redeem on a refusal: err = %v, want the panel's message", err)
	}
}
