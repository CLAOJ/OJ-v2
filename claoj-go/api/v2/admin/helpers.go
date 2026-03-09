package admin

import (
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
)

// GetProblemByCode looks up a problem by its code
func GetProblemByCode(code string) (models.Problem, error) {
	var problem models.Problem
	err := db.DB.Where("code = ?", code).First(&problem).Error
	return problem, err
}
