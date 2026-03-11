package vendor

import (
	"testing"
)

func TestVendorStub_CleanPass(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "ACC-001", Amount: 150.00, TransferID: "transfer-1"}
	resp := s.Validate(req, "")
	if resp.Status != "pass" {
		t.Errorf("expected pass, got %s", resp.Status)
	}
	if resp.MICR == nil {
		t.Error("expected MICR data")
	}
	if resp.MICR.Routing != "021000021" {
		t.Errorf("expected routing 021000021, got %s", resp.MICR.Routing)
	}
}

func TestVendorStub_IQABlur(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "ACC-IQA-BLUR", Amount: 100.00, TransferID: "t2"}
	resp := s.Validate(req, "")
	if resp.Status != "fail" {
		t.Errorf("expected fail, got %s", resp.Status)
	}
	if resp.Reason != "blur" {
		t.Errorf("expected reason blur, got %s", resp.Reason)
	}
}

func TestVendorStub_IQAGlare(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "ACC-IQA-GLARE", Amount: 100.00, TransferID: "t3"}
	resp := s.Validate(req, "")
	if resp.Status != "fail" {
		t.Errorf("expected fail, got %s", resp.Status)
	}
	if resp.Reason != "glare" {
		t.Errorf("expected reason glare, got %s", resp.Reason)
	}
}

func TestVendorStub_MICRFail(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "ACC-MICR-FAIL", Amount: 100.00, TransferID: "t4"}
	resp := s.Validate(req, "")
	if resp.Status != "flagged" {
		t.Errorf("expected flagged, got %s", resp.Status)
	}
	if resp.Reason != "micr_fail" {
		t.Errorf("expected reason micr_fail, got %s", resp.Reason)
	}
}

func TestVendorStub_Duplicate(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "ACC-DUP-001", Amount: 100.00, TransferID: "t5"}
	resp := s.Validate(req, "")
	if resp.Status != "reject" {
		t.Errorf("expected reject, got %s", resp.Status)
	}
	if resp.Reason != "duplicate" {
		t.Errorf("expected reason duplicate, got %s", resp.Reason)
	}
}

func TestVendorStub_AmountMismatch(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "ACC-MISMATCH", Amount: 1500.00, TransferID: "t6"}
	resp := s.Validate(req, "")
	if resp.Status != "flagged" {
		t.Errorf("expected flagged, got %s", resp.Status)
	}
	if resp.Reason != "amount_mismatch" {
		t.Errorf("expected reason amount_mismatch, got %s", resp.Reason)
	}
}

func TestVendorStub_ScenarioOverride(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "ACC-001", Amount: 100.00, TransferID: "t7"}
	resp := s.Validate(req, "iqafail_blur")
	if resp.Status != "fail" {
		t.Errorf("expected fail via override, got %s", resp.Status)
	}
}

func TestVendorStub_DefaultScenario(t *testing.T) {
	s := &Stub{config: defaultConfig()}
	req := &VendorRequest{AccountID: "UNKNOWN-ACCT", Amount: 100.00, TransferID: "t8"}
	resp := s.Validate(req, "")
	if resp.Status != "pass" {
		t.Errorf("expected pass for unknown account, got %s", resp.Status)
	}
}
