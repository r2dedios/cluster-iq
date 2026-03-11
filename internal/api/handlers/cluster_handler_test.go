package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RHEcosystemAppEng/cluster-iq/internal/inventory"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/models"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/models/db"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/models/dto"
	"github.com/RHEcosystemAppEng/cluster-iq/internal/repositories"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockClusterService implements the ClusterService interface for testing
type mockClusterService struct {
	powerOnFn       func(ctx context.Context, clusterID string, requester string, description *string) error
	powerOffFn      func(ctx context.Context, clusterID string, requester string, description *string) error
	getFn           func(ctx context.Context, clusterID string) (*db.ClusterDBResponse, error)
	listFn          func(ctx context.Context, opts models.ListOptions) ([]db.ClusterDBResponse, int, error)
	deleteFn        func(ctx context.Context, clusterID string) error
	getTagsFn       func(ctx context.Context, clusterID string) ([]db.TagDBResponse, error)
	getInstancesFn  func(ctx context.Context, clusterID string) ([]db.InstanceDBResponse, error)
	getSummaryFn    func(ctx context.Context) (inventory.ClustersSummary, error)
	createFn        func(ctx context.Context, clusters []inventory.Cluster) error
	updateFn        func(ctx context.Context, cluster dto.ClusterDTORequest) error
}

func (m *mockClusterService) PowerOn(ctx context.Context, clusterID string, requester string, description *string) error {
	if m.powerOnFn != nil {
		return m.powerOnFn(ctx, clusterID, requester, description)
	}
	return nil
}

func (m *mockClusterService) PowerOff(ctx context.Context, clusterID string, requester string, description *string) error {
	if m.powerOffFn != nil {
		return m.powerOffFn(ctx, clusterID, requester, description)
	}
	return nil
}

func (m *mockClusterService) Get(ctx context.Context, clusterID string) (*db.ClusterDBResponse, error) {
	if m.getFn != nil {
		return m.getFn(ctx, clusterID)
	}
	return nil, nil
}

func (m *mockClusterService) List(ctx context.Context, opts models.ListOptions) ([]db.ClusterDBResponse, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, opts)
	}
	return nil, 0, nil
}

func (m *mockClusterService) Delete(ctx context.Context, clusterID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, clusterID)
	}
	return nil
}

func (m *mockClusterService) GetTags(ctx context.Context, clusterID string) ([]db.TagDBResponse, error) {
	if m.getTagsFn != nil {
		return m.getTagsFn(ctx, clusterID)
	}
	return nil, nil
}

func (m *mockClusterService) GetInstances(ctx context.Context, clusterID string) ([]db.InstanceDBResponse, error) {
	if m.getInstancesFn != nil {
		return m.getInstancesFn(ctx, clusterID)
	}
	return nil, nil
}

func (m *mockClusterService) GetSummary(ctx context.Context) (inventory.ClustersSummary, error) {
	if m.getSummaryFn != nil {
		return m.getSummaryFn(ctx)
	}
	return inventory.ClustersSummary{}, nil
}

func (m *mockClusterService) Create(ctx context.Context, clusters []inventory.Cluster) error {
	if m.createFn != nil {
		return m.createFn(ctx, clusters)
	}
	return nil
}

func (m *mockClusterService) Update(ctx context.Context, cluster dto.ClusterDTORequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, cluster)
	}
	return nil
}

func TestClusterHandler_PowerOn(t *testing.T) {
	t.Run("Success", func(t *testing.T) { testClusterHandler_PowerOn_Success(t) })
	t.Run("Invalid JSON body", func(t *testing.T) { testClusterHandler_PowerOn_InvalidJSON(t) })
	t.Run("Missing requester field", func(t *testing.T) { testClusterHandler_PowerOn_MissingRequester(t) })
	t.Run("Requester exceeds max length", func(t *testing.T) { testClusterHandler_PowerOn_RequesterTooLong(t) })
	t.Run("Description exceeds max length", func(t *testing.T) { testClusterHandler_PowerOn_DescriptionTooLong(t) })
	t.Run("Cluster not found", func(t *testing.T) { testClusterHandler_PowerOn_ClusterNotFound(t) })
	t.Run("Service error", func(t *testing.T) { testClusterHandler_PowerOn_ServiceError(t) })
	t.Run("Empty description", func(t *testing.T) { testClusterHandler_PowerOn_EmptyDescription(t) })
}

func testClusterHandler_PowerOn_Success(t *testing.T) {
	mockService := &mockClusterService{
		powerOnFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			assert.Equal(t, "test-cluster", clusterID)
			assert.Equal(t, "test-user", requester)
			assert.NotNil(t, description)
			assert.Equal(t, "test description", *description)
			return nil
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	body := `{"requester":"test-user","description":"test description"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "power on request accepted", response["message"])
}

func testClusterHandler_PowerOn_InvalidJSON(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	body := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOn_MissingRequester(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	body := `{"description":"test description"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOn_RequesterTooLong(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	// Create a requester string longer than 100 characters
	longRequester := strings.Repeat("a", 101)
	body := `{"requester":"` + longRequester + `","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOn_DescriptionTooLong(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	// Create a description string longer than 500 characters
	longDescription := strings.Repeat("a", 501)
	body := `{"requester":"test-user","description":"` + longDescription + `"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOn_ClusterNotFound(t *testing.T) {
	mockService := &mockClusterService{
		powerOnFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			return repositories.ErrNotFound
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	body := `{"requester":"test-user","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Cluster not found", response["message"])
}

func testClusterHandler_PowerOn_ServiceError(t *testing.T) {
	mockService := &mockClusterService{
		powerOnFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			return errors.New("service error")
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	body := `{"requester":"test-user","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Failed to power on cluster")
}

func testClusterHandler_PowerOn_EmptyDescription(t *testing.T) {
	mockService := &mockClusterService{
		powerOnFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			assert.Equal(t, "test-cluster", clusterID)
			assert.Equal(t, "test-user", requester)
			assert.NotNil(t, description)
			assert.Equal(t, "", *description)
			return nil
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_on", handler.PowerOn)

	body := `{"requester":"test-user","description":""}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_on", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "power on request accepted", response["message"])
}

func TestClusterHandler_PowerOff(t *testing.T) {
	t.Run("Success", func(t *testing.T) { testClusterHandler_PowerOff_Success(t) })
	t.Run("Invalid JSON body", func(t *testing.T) { testClusterHandler_PowerOff_InvalidJSON(t) })
	t.Run("Missing requester field", func(t *testing.T) { testClusterHandler_PowerOff_MissingRequester(t) })
	t.Run("Requester exceeds max length", func(t *testing.T) { testClusterHandler_PowerOff_RequesterTooLong(t) })
	t.Run("Description exceeds max length", func(t *testing.T) { testClusterHandler_PowerOff_DescriptionTooLong(t) })
	t.Run("Cluster not found", func(t *testing.T) { testClusterHandler_PowerOff_ClusterNotFound(t) })
	t.Run("Service error", func(t *testing.T) { testClusterHandler_PowerOff_ServiceError(t) })
	t.Run("Empty description", func(t *testing.T) { testClusterHandler_PowerOff_EmptyDescription(t) })
}

func testClusterHandler_PowerOff_Success(t *testing.T) {
	mockService := &mockClusterService{
		powerOffFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			assert.Equal(t, "test-cluster", clusterID)
			assert.Equal(t, "test-user", requester)
			assert.NotNil(t, description)
			assert.Equal(t, "test description", *description)
			return nil
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	body := `{"requester":"test-user","description":"test description"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "power off request accepted", response["message"])
}

func testClusterHandler_PowerOff_InvalidJSON(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	body := `{"invalid json`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOff_MissingRequester(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	body := `{"description":"test description"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOff_RequesterTooLong(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	// Create a requester string longer than 100 characters
	longRequester := strings.Repeat("a", 101)
	body := `{"requester":"` + longRequester + `","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOff_DescriptionTooLong(t *testing.T) {
	mockService := &mockClusterService{}
	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	// Create a description string longer than 500 characters
	longDescription := strings.Repeat("a", 501)
	body := `{"requester":"test-user","description":"` + longDescription + `"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Invalid request body")
}

func testClusterHandler_PowerOff_ClusterNotFound(t *testing.T) {
	mockService := &mockClusterService{
		powerOffFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			return repositories.ErrNotFound
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	body := `{"requester":"test-user","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Cluster not found", response["message"])
}

func testClusterHandler_PowerOff_ServiceError(t *testing.T) {
	mockService := &mockClusterService{
		powerOffFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			return errors.New("service error")
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	body := `{"requester":"test-user","description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Failed to power off cluster")
}

func testClusterHandler_PowerOff_EmptyDescription(t *testing.T) {
	mockService := &mockClusterService{
		powerOffFn: func(ctx context.Context, clusterID string, requester string, description *string) error {
			assert.Equal(t, "test-cluster", clusterID)
			assert.Equal(t, "test-user", requester)
			assert.NotNil(t, description)
			assert.Equal(t, "", *description)
			return nil
		},
	}

	handler := NewClusterHandler(mockService, zap.NewNop())
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/clusters/:id/power_off", handler.PowerOff)

	body := `{"requester":"test-user","description":""}`
	req := httptest.NewRequest(http.MethodPost, "/clusters/test-cluster/power_off", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "power off request accepted", response["message"])
}
