package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestVideoContentRouteSupportsGetAndHeadWithoutAuthentication(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	require.NoError(t, db.Create(&model.Task{
		TaskID: "task_public_preview",
		UserId: 42,
		Status: model.TaskStatusFailure,
	}).Error)

	previousDB := model.DB
	model.DB = db
	t.Cleanup(func() {
		model.DB = previousDB
	})

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	SetVideoRouter(engine)

	for _, method := range []string{http.MethodGet, http.MethodHead} {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/v1/videos/task_public_preview/content", nil)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			if method == http.MethodGet {
				assert.Contains(t, response.Body.String(), "current status: FAILURE")
			}
		})
	}
}
