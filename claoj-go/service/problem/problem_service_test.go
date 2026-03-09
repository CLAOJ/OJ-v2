package problem

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ProblemServiceTestSuite struct {
	suite.Suite
	database *gorm.DB
	service  *ProblemService
}

func (s *ProblemServiceTestSuite) SetupTest() {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	s.Require().NoError(err)

	s.database = database
	db.DB = database

	// Migrate schema
	err = database.AutoMigrate(
		&models.Problem{},
		&models.ProblemGroup{},
		&models.ProblemType{},
		&models.Profile{},
		&models.Language{},
		&models.Organization{},
		&models.ProblemClarification{},
	)
	s.Require().NoError(err)

	s.service = NewProblemService()
}

func (s *ProblemServiceTestSuite) createTestGroup(name string) models.ProblemGroup {
	group := models.ProblemGroup{Name: name}
	err := s.database.Create(&group).Error
	s.Require().NoError(err)
	return group
}

func (s *ProblemServiceTestSuite) createTestProblem(code, name string, groupID uint) models.Problem {
	problem := models.Problem{
		Code:      code,
		Name:      name,
		GroupID:   groupID,
		Points:    100,
		IsPublic:  true,
		Date:      func() *time.Time { t := time.Now(); return &t }(),
	}
	err := s.database.Create(&problem).Error
	s.Require().NoError(err)
	return problem
}

func (s *ProblemServiceTestSuite) createTestType(name string) models.ProblemType {
	problemType := models.ProblemType{Name: name}
	err := s.database.Create(&problemType).Error
	s.Require().NoError(err)
	return problemType
}

func (s *ProblemServiceTestSuite) createTestProfile(username string) models.Profile {
	user := models.AuthUser{
		Username:   username,
		Email:      username + "@example.com",
		Password:   "hashedpassword",
		IsActive:   true,
		DateJoined: time.Now(),
	}
	err := s.database.Create(&user).Error
	s.Require().NoError(err)

	profile := models.Profile{
		UserID:     user.ID,
		Timezone:   "UTC",
		LastAccess: time.Now(),
	}
	err = s.database.Create(&profile).Error
	s.Require().NoError(err)
	return profile
}

func (s *ProblemServiceTestSuite) createTestLanguage(name, key string) models.Language {
	lang := models.Language{
		Name:         name,
		Key:          key,
		CommonName:   name,
	}
	err := s.database.Create(&lang).Error
	s.Require().NoError(err)
	return lang
}

func TestProblemServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProblemServiceTestSuite))
}

func (s *ProblemServiceTestSuite) TestListProblems() {
	// Create test data
	group := s.createTestGroup("Test Group")
	s.createTestProblem("PROB1", "Problem 1", group.ID)
	s.createTestProblem("PROB2", "Problem 2", group.ID)
	s.createTestProblem("PROB3", "Problem 3", group.ID)

	// Test default pagination
	req := ListProblemsRequest{Page: 1, PageSize: 10}
	resp, err := s.service.ListProblems(req)
	s.NoError(err)
	s.Equal(int64(3), resp.Total)
	s.Len(resp.Problems, 3)

	// Test page size limit
	req = ListProblemsRequest{Page: 1, PageSize: 2}
	resp, err = s.service.ListProblems(req)
	s.NoError(err)
	s.Len(resp.Problems, 2)

	// Test max page size enforcement
	req = ListProblemsRequest{Page: 1, PageSize: 200}
	resp, err = s.service.ListProblems(req)
	s.NoError(err)
	s.Equal(100, resp.PageSize)

	// Test default page
	req = ListProblemsRequest{Page: 0, PageSize: 10}
	resp, err = s.service.ListProblems(req)
	s.NoError(err)
	s.Equal(1, resp.Page)
}

func (s *ProblemServiceTestSuite) TestGetProblem() {
	group := s.createTestGroup("Test Group")
	problem := s.createTestProblem("TEST001", "Test Problem", group.ID)

	// Test successful retrieval
	req := GetProblemRequest{ProblemCode: "TEST001"}
	profile, err := s.service.GetProblem(req)
	s.NoError(err)
	s.NotNil(profile)
	s.Equal(problem.Code, profile.Code)
	s.Equal(problem.Name, profile.Name)
	s.Equal(group.Name, profile.GroupName)

	// Test not found
	req = GetProblemRequest{ProblemCode: "NONEXISTENT"}
	profile, err = s.service.GetProblem(req)
	s.ErrorIs(err, ErrProblemNotFound)
	s.Nil(profile)
}

func (s *ProblemServiceTestSuite) TestCreateProblem() {
	group := s.createTestGroup("Test Group")
	problemType := s.createTestType("Algorithm")
	author := s.createTestProfile("author")
	lang := s.createTestLanguage("Python", "python3")

	req := CreateProblemRequest{
		Code:           "NEW001",
		Name:           "New Problem",
		Description:    "This is a test problem",
		Points:         150,
		Partial:        true,
		IsPublic:       true,
		TimeLimit:      2.0,
		MemoryLimit:    256,
		GroupID:        group.ID,
		TypeIDs:        []uint{problemType.ID},
		AuthorIDs:      []uint{author.ID},
		AllowedLangIDs: []uint{lang.ID},
	}

	profile, err := s.service.CreateProblem(req)
	s.NoError(err)
	s.NotNil(profile)
	s.Equal("NEW001", profile.Code)
	s.Equal("New Problem", profile.Name)
	s.Equal(group.Name, profile.GroupName)

	// Verify it was created in database
	var created models.Problem
	err = s.database.Where("code = ?", "NEW001").First(&created).Error
	s.NoError(err)
	s.Equal("New Problem", created.Name)
}

func (s *ProblemServiceTestSuite) TestUpdateProblem() {
	group := s.createTestGroup("Test Group")
	problem := s.createTestProblem("UPDATE001", "Original Name", group.ID)
	problemType := s.createTestType("Math")
	author := s.createTestProfile("author")

	newName := "Updated Name"
	newPoints := float64(200)
	req := UpdateProblemRequest{
		ProblemCode:   "UPDATE001",
		Name:          &newName,
		Points:        &newPoints,
		AddTypeIDs:    []uint{problemType.ID},
		AddAuthorIDs:  []uint{author.ID},
	}

	profile, err := s.service.UpdateProblem(req)
	s.NoError(err)
	s.NotNil(profile)
	s.Equal("Updated Name", profile.Name)
	s.Equal(200.0, profile.Points)

	// Verify update in database
	var updated models.Problem
	err = s.database.Preload("Types").Preload("Authors").First(&updated, problem.ID).Error
	s.NoError(err)
	s.Equal("Updated Name", updated.Name)
	s.Equal(200.0, updated.Points)
	s.Len(updated.Types, 1)
	s.Len(updated.Authors, 1)
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_NotFound() {
	newName := "Updated Name"
	req := UpdateProblemRequest{
		ProblemCode:   "NONEXISTENT",
		Name:          &newName,
	}

	profile, err := s.service.UpdateProblem(req)
	s.ErrorIs(err, ErrProblemNotFound)
	s.Nil(profile)
}

func (s *ProblemServiceTestSuite) TestDeleteProblem() {
	group := s.createTestGroup("Test Group")
	s.createTestProblem("DELETE001", "To Delete", group.ID)

	req := DeleteProblemRequest{ProblemCode: "DELETE001"}
	err := s.service.DeleteProblem(req)
	s.NoError(err)

	// Verify it was soft deleted (marked as not public)
	var problem models.Problem
	err = s.database.Where("code = ?", "DELETE001").First(&problem).Error
	s.NoError(err)
	s.False(problem.IsPublic)
}

func (s *ProblemServiceTestSuite) TestDeleteProblem_NotFound() {
	req := DeleteProblemRequest{ProblemCode: "NONEXISTENT"}
	err := s.service.DeleteProblem(req)
	s.ErrorIs(err, ErrProblemNotFound)
}

func (s *ProblemServiceTestSuite) TestCloneProblem() {
	group := s.createTestGroup("Test Group")
	source := s.createTestProblem("SOURCE001", "Source Problem", group.ID)
	author := s.createTestProfile("author")
	s.database.Model(&source).Association("Authors").Append(&author)

	req := CloneProblemRequest{
		SourceCode: "SOURCE001",
		NewCode:    "CLONE001",
		NewName:    "Cloned Problem",
	}

	profile, err := s.service.CloneProblem(req)
	s.NoError(err)
	s.NotNil(profile)
	s.Equal("CLONE001", profile.Code)
	s.Equal("Cloned Problem", profile.Name)
	s.False(profile.IsPublic) // Should start as hidden

	// Verify clone exists
	var cloned models.Problem
	err = s.database.Where("code = ?", "CLONE001").First(&cloned).Error
	s.NoError(err)
}

func (s *ProblemServiceTestSuite) TestCloneProblem_CodeExists() {
	group := s.createTestGroup("Test Group")
	s.createTestProblem("EXISTING001", "Existing Problem", group.ID)
	s.createTestProblem("SOURCE002", "Source Problem", group.ID)

	req := CloneProblemRequest{
		SourceCode: "SOURCE002",
		NewCode:    "EXISTING001",
		NewName:    "Clone Attempt",
	}

	profile, err := s.service.CloneProblem(req)
	s.ErrorIs(err, ErrProblemCodeExists)
	s.Nil(profile)
}

func (s *ProblemServiceTestSuite) TestCloneProblem_SourceNotFound() {
	req := CloneProblemRequest{
		SourceCode: "NONEXISTENT",
		NewCode:    "NEW001",
		NewName:    "Clone Attempt",
	}

	profile, err := s.service.CloneProblem(req)
	s.ErrorIs(err, ErrProblemNotFound)
	s.Nil(profile)
}

func (s *ProblemServiceTestSuite) TestCreateClarification() {
	group := s.createTestGroup("Test Group")
	s.createTestProblem("CLAR001", "Problem with Clarification", group.ID)

	err := s.service.CreateClarification("CLAR001", "What is the time limit?")
	s.NoError(err)

	// Verify clarification was created
	var clarifications []models.ProblemClarification
	err = s.database.Find(&clarifications).Error
	s.NoError(err)
	s.Len(clarifications, 1)
	s.Equal("What is the time limit?", clarifications[0].Description)
}

func (s *ProblemServiceTestSuite) TestCreateClarification_ProblemNotFound() {
	err := s.service.CreateClarification("NONEXISTENT", "Question")
	s.ErrorIs(err, ErrProblemNotFound)
}

func (s *ProblemServiceTestSuite) TestDeleteClarification() {
	group := s.createTestGroup("Test Group")
	problem := s.createTestProblem("DELCLAR001", "Problem", group.ID)
	clarification := models.ProblemClarification{
		ProblemID:   problem.ID,
		Description: "To be deleted",
		Date:        time.Now(),
	}
	err := s.database.Create(&clarification).Error
	s.Require().NoError(err)

	err = s.service.DeleteClarification(clarification.ID)
	s.NoError(err)

	// Verify deletion
	var count int64
	s.database.Model(&models.ProblemClarification{}).Count(&count)
	s.Equal(int64(0), count)
}

func (s *ProblemServiceTestSuite) TestDeleteClarification_NotFound() {
	err := s.service.DeleteClarification(99999)
	s.ErrorIs(err, ErrClarificationNotFound)
}

func (s *ProblemServiceTestSuite) TestUpdatePdfURL() {
	group := s.createTestGroup("Test Group")
	s.createTestProblem("PDF001", "Problem with PDF", group.ID)

	err := s.service.UpdatePdfURL("PDF001", "https://example.com/problem.pdf")
	s.NoError(err)

	// Verify update
	var problem models.Problem
	err = s.database.Where("code = ?", "PDF001").First(&problem).Error
	s.NoError(err)
	s.Equal("https://example.com/problem.pdf", problem.PdfURL)
}

func (s *ProblemServiceTestSuite) TestUpdatePdfURL_NotFound() {
	err := s.service.UpdatePdfURL("NONEXISTENT", "https://example.com/test.pdf")
	s.ErrorIs(err, ErrProblemNotFound)
}

func (s *ProblemServiceTestSuite) TestClearPdfURL() {
	group := s.createTestGroup("Test Group")
	problem := s.createTestProblem("CLEARPDF001", "Problem", group.ID)
	problem.PdfURL = "https://example.com/problem.pdf"
	s.database.Save(&problem)

	err := s.service.ClearPdfURL("CLEARPDF001")
	s.NoError(err)

	// Verify cleared
	s.database.First(&problem, problem.ID)
	s.Equal("", problem.PdfURL)
}

func TestListProblemsRequest_Validation(t *testing.T) {
	service := NewProblemService()

	// Test default values
	req := ListProblemsRequest{}
	resp, err := service.ListProblems(req)
	assert.NoError(t, err)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 20, resp.PageSize)

	// Test max page size cap
	req = ListProblemsRequest{PageSize: 200}
	resp, err = service.ListProblems(req)
	assert.NoError(t, err)
	assert.Equal(t, 100, resp.PageSize)
}
