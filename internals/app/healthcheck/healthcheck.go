package healthcheck

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var db *gorm.DB

type HealthCheckResponse struct {
	Status   string   `json:"status"`
	Services []string `json:"services"`
}

func checkPostgres() string {
	if err := db.Raw("SELECT 1").Error; err != nil {
		return "Postgres is not healthy"
	}
	return "Postgres is healthy"
}

func HealthCheckHandler(c *gin.Context) {
	services := []string{
		checkPostgres(),
	}

	response := HealthCheckResponse{
		Status:   "Client Service is healthy!",
		Services: services,
	}

	c.JSON(http.StatusOK, response)
}
