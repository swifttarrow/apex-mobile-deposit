package vendor

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// scenariosConfig is the structure loaded from config/scenarios.json.
type scenariosConfig struct {
	Scenarios       map[string]string `json:"scenarios"`
	DefaultScenario string            `json:"default_scenario"`
}

// Stub is the in-process vendor service stub.
type Stub struct {
	configPath string
	config     *scenariosConfig
}

// NewStub creates a new vendor stub loading config from the given path.
func NewStub(configPath string) *Stub {
	s := &Stub{configPath: configPath}
	if err := s.loadConfig(); err != nil {
		log.Printf("vendor stub: failed to load config %s: %v (using defaults)", configPath, err)
		s.config = defaultConfig()
	}
	return s
}

func defaultConfig() *scenariosConfig {
	return &scenariosConfig{
		Scenarios: map[string]string{
			"ACC-IQA-BLUR":  "iqafail_blur",
			"ACC-IQA-GLARE": "iqafail_glare",
			"ACC-MICR-FAIL": "micr_fail",
			"ACC-DUP-001":   "duplicate",
			"ACC-MISMATCH":  "amount_mismatch",
			"ACC-001":       "clean_pass",
			"ACC-OVER-LIMIT": "clean_pass",
			"ACC-RETIRE-001": "clean_pass",
		},
		DefaultScenario: "clean_pass",
	}
}

func (s *Stub) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	var cfg scenariosConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	s.config = &cfg
	return nil
}

// resolveScenario determines which scenario to use for the given request.
func (s *Stub) resolveScenario(req *VendorRequest, scenarioOverride string) string {
	if scenarioOverride != "" {
		return scenarioOverride
	}
	// Exact match first
	if scenario, ok := s.config.Scenarios[req.AccountID]; ok {
		return scenario
	}
	// Prefix match
	for prefix, scenario := range s.config.Scenarios {
		if strings.HasPrefix(req.AccountID, prefix) {
			return scenario
		}
	}
	return s.config.DefaultScenario
}

// buildResponse constructs the deterministic vendor response for a scenario.
func buildResponse(scenario string, req *VendorRequest) *VendorResponse {
	switch scenario {
	case "iqafail_blur":
		return &VendorResponse{
			Status:  "fail",
			Reason:  "blur",
			Message: "Image too blurry",
		}
	case "iqafail_glare":
		return &VendorResponse{
			Status:  "fail",
			Reason:  "glare",
			Message: "Glare detected",
		}
	case "micr_fail":
		return &VendorResponse{
			Status:         "flagged",
			Reason:         "micr_fail",
			IQScore:        0.85,
			MICRConfidence: 0.0,
		}
	case "duplicate":
		return &VendorResponse{
			Status: "reject",
			Reason: "duplicate",
		}
	case "amount_mismatch":
		return &VendorResponse{
			Status:         "flagged",
			Reason:         "amount_mismatch",
			IQScore:        0.90,
			MICRConfidence: 0.5,
			OCRAmount:      150.00,
			EnteredAmount:  1500.00,
		}
	case "iqapass":
		return &VendorResponse{
			Status:         "pass",
			IQScore:        0.95,
			MICRConfidence: 0.95,
		}
	default: // clean_pass
		return &VendorResponse{
			Status: "pass",
			IQScore:        0.95,
			MICRConfidence: 0.98,
			MICR: &MICRData{
				Routing:     "021000021",
				Account:     "1234567890",
				CheckNumber: "1001",
			},
			Amount:        req.Amount,
			TransactionID: fmt.Sprintf("TXN-%s", req.TransferID),
		}
	}
}

// Validate performs vendor validation in-process (no HTTP round-trip).
func (s *Stub) Validate(req *VendorRequest, scenarioOverride string) *VendorResponse {
	scenario := s.resolveScenario(req, scenarioOverride)
	return buildResponse(scenario, req)
}

// HandleValidate is an HTTP handler that exposes the vendor stub via HTTP.
func (s *Stub) HandleValidate(w http.ResponseWriter, r *http.Request) {
	var req VendorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	scenarioOverride := r.Header.Get("X-Test-Scenario")
	resp := s.Validate(&req, scenarioOverride)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
