package admin

import (
	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
)

// GetProblemByCode looks up a problem by its code
func GetProblemByCode(code string) (models.Problem, error) {
	var problem models.Problem
	err := db.DB.Where("code = ?", code).First(&problem).Error
	return problem, err
}
