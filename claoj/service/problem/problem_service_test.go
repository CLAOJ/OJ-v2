package problem

import (
	"testing"
	"time"

	"github.com/CLAOJ/claoj/db"
	"github.com/CLAOJ/claoj/models"
	"github.com/CLAOJ/claoj/service/relations"
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
	sqlDB, err := database.DB()
	s.Require().NoError(err)
	sqlDB.SetMaxOpenConns(1)

	s.database = database
	db.DB = database

	// Migrate schema
	err = database.AutoMigrate(
		&models.AuthUser{},
		&models.Problem{},
		&models.ProblemGroup{},
		&models.ProblemType{},
		&models.Profile{},
		&models.Language{},
		&models.Organization{},
		&models.License{},
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
		Code:     code,
		Name:     name,
		GroupID:  groupID,
		Points:   100,
		IsPublic: true,
		Date:     func() *time.Time { t := time.Now(); return &t }(),
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
		Name:       name,
		Key:        key,
		CommonName: name,
	}
	err := s.database.Create(&lang).Error
	s.Require().NoError(err)
	return lang
}

func TestProblemServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ProblemServiceTestSuite))
}

func (s *ProblemServiceTestSuite) createTestOrganization(name string) models.Organization {
	org := models.Organization{
		Name: name, Slug: name, ShortName: name, About: "", CreationDate: time.Now(),
	}
	err := s.database.Omit("Admins", "Members").Create(&org).Error
	s.Require().NoError(err)
	return org
}

func (s *ProblemServiceTestSuite) createTestLicense(key string) models.License {
	license := models.License{Key: key, Link: "https://example.com/" + key, Name: key, Text: ""}
	err := s.database.Create(&license).Error
	s.Require().NoError(err)
	return license
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
		ProblemCode:  "UPDATE001",
		Name:         &newName,
		Points:       &newPoints,
		AddTypeIDs:   []uint{problemType.ID},
		AddAuthorIDs: []uint{author.ID},
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
		ProblemCode: "NONEXISTENT",
		Name:        &newName,
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

func (s *ProblemServiceTestSuite) TestUpdateProblem_FullListsReplace() {
	group := s.createTestGroup("g1")
	p := s.createTestProblem("FULL1", "Full", group.ID)
	a1 := s.createTestProfile("alice")
	a2 := s.createTestProfile("bob")
	ty := s.createTestType("dp")
	lang := s.createTestLanguage("C++17", "CPP17")

	authors := []uint{a1.ID, a2.ID}
	types := []uint{ty.ID}
	langs := []uint{lang.ID}
	curators := []uint{a2.ID}
	_, err := s.service.UpdateProblem(UpdateProblemRequest{
		ProblemCode: p.Code, AuthorIDs: &authors, TypeIDs: &types, AllowedLangIDs: &langs, CuratorIDs: &curators,
	})
	s.Require().NoError(err)

	got, err := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Require().NoError(err)
	s.ElementsMatch(authors, got.AuthorIDs)
	s.Equal(types, got.TypeIDs)
	s.Equal(langs, got.AllowedLangIDs)
	s.Equal(curators, got.CuratorIDs)
	s.ElementsMatch([]string{"alice", "bob"}, []string{got.Authors[0].Username, got.Authors[1].Username})

	authors = []uint{a2.ID}
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, AuthorIDs: &authors})
	s.Require().NoError(err)
	got, _ = s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Equal([]uint{a2.ID}, got.AuthorIDs)
	s.Equal(types, got.TypeIDs, "omitted type_ids must stay unchanged")
	s.Equal(langs, got.AllowedLangIDs, "omitted allowed_lang_ids must stay unchanged")
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_EmptyListClears() {
	group := s.createTestGroup("g2")
	p := s.createTestProblem("CLR1", "Clear", group.ID)
	a := s.createTestProfile("carol")
	authors := []uint{a.ID}
	_, err := s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, AuthorIDs: &authors})
	s.Require().NoError(err)

	none := []uint{}
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, AuthorIDs: &none})
	s.Require().NoError(err)
	got, _ := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Empty(got.AuthorIDs)
	s.NotNil(got.AuthorIDs)
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_UnknownIDRollsBackEverything() {
	group := s.createTestGroup("g3")
	p := s.createTestProblem("RB1", "Before", group.ID)
	name := "After"
	bad := []uint{424242}
	_, err := s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, Name: &name, AuthorIDs: &bad})
	s.Require().Error(err)

	got, _ := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Equal("Before", got.Name, "a failed relation must roll back the scalar update too")
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_GroupID() {
	g1 := s.createTestGroup("g4")
	g2 := s.createTestGroup("g5")
	p := s.createTestProblem("GRP1", "Group", g1.ID)

	_, err := s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, GroupID: &g2.ID})
	s.Require().NoError(err)
	got, _ := s.service.GetProblem(GetProblemRequest{ProblemCode: p.Code})
	s.Equal(g2.ID, got.GroupID)

	missing := uint(98765)
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: p.Code, GroupID: &missing})
	s.ErrorIs(err, ErrInvalidGroup)
}

func (s *ProblemServiceTestSuite) TestListProblems_FiltersAndCountsFilteredTotal() {
	g1 := s.createTestGroup("ga")
	g2 := s.createTestGroup("gb")
	s.createTestProblem("dijkstra", "Shortest Path", g1.ID)
	s.createTestProblem("knapsack", "Knapsack", g2.ID)
	priv := s.createTestProblem("secret", "Hidden Dijkstra", g1.ID)
	s.Require().NoError(s.database.Model(&priv).Update("is_public", false).Error)

	resp, err := s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, Search: "dijkstra"})
	s.Require().NoError(err)
	s.Equal(int64(2), resp.Total, "matches code or name, case-insensitively")

	public := true
	resp, _ = s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, Search: "dijkstra", IsPublic: &public})
	s.Equal(int64(1), resp.Total)
	s.Equal("dijkstra", resp.Problems[0].Code)

	resp, _ = s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, GroupID: g2.ID})
	s.Equal(int64(1), resp.Total)
	s.Equal("knapsack", resp.Problems[0].Code)
}

func (s *ProblemServiceTestSuite) TestListProblems_SearchIsLiteral() {
	g := s.createTestGroup("gl")
	s.createTestProblem("a_b", "Underscore", g.ID)
	s.createTestProblem("axb", "Other", g.ID)

	resp, err := s.service.ListProblems(ListProblemsRequest{Page: 1, PageSize: 10, Search: "a_b"})
	s.Require().NoError(err)
	s.Equal(int64(1), resp.Total)
	s.Equal("a_b", resp.Problems[0].Code)
}

func (s *ProblemServiceTestSuite) TestCreateProblem_V1FieldsRoundTrip() {
	group := s.createTestGroup("gall")
	license := s.createTestLicense("cc-by")
	author := s.createTestProfile("author")
	curator := s.createTestProfile("curator")
	tester := s.createTestProfile("tester")
	banned := s.createTestProfile("banned")
	suggester := s.createTestProfile("suggester")
	org := s.createTestOrganization("org")
	date := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	_, err := s.service.CreateProblem(CreateProblemRequest{
		Code: "ALL1", Name: "All Fields", Description: "body",
		Points: 100, TimeLimit: 1.5, MemoryLimit: 262144, GroupID: group.ID,
		Source: "VOI 2026", Date: &date, LicenseID: &license.ID,
		OgImage: "https://example.com/og.png", Summary: "short summary",
		IsFullMarkup:                   true,
		SubmissionSourceVisibilityMode: models.SubmissionSourceAccessSolved,
		TestcaseVisibilityMode:         models.ProblemTestcaseAccessAuthorOnly,
		IsOrganizationPrivate:          true,
		OrganizationIDs:                []uint{org.ID},
		BannedUserIDs:                  []uint{banned.ID},
		SuggesterID:                    &suggester.ID,
		AuthorIDs:                      []uint{author.ID},
		CuratorIDs:                     []uint{curator.ID},
		TesterIDs:                      []uint{tester.ID},
	})
	s.Require().NoError(err)

	got, err := s.service.GetProblem(GetProblemRequest{ProblemCode: "ALL1"})
	s.Require().NoError(err)
	s.Equal("VOI 2026", got.Source)
	s.Require().NotNil(got.Date)
	s.True(date.Equal(*got.Date))
	s.Require().NotNil(got.LicenseID)
	s.Equal(license.ID, *got.LicenseID)
	s.Equal("https://example.com/og.png", got.OgImage)
	s.Equal("short summary", got.Summary)
	s.True(got.ShortCircuit)
	s.True(got.IsFullMarkup)
	s.Equal(models.SubmissionSourceAccessSolved, got.SubmissionSourceVisibilityMode)
	s.Equal(models.ProblemTestcaseAccessAuthorOnly, got.TestcaseVisibilityMode)
	s.True(got.IsOrganizationPrivate)
	s.Equal([]uint{org.ID}, got.OrganizationIDs)
	s.Equal([]uint{banned.ID}, got.BannedUserIDs)
	s.Equal([]string{"banned"}, []string{got.BannedUsers[0].Username})
	s.Require().NotNil(got.SuggesterID)
	s.Equal(suggester.ID, *got.SuggesterID)
	s.Equal([]uint{author.ID}, got.AuthorIDs)
	s.Equal([]uint{curator.ID}, got.CuratorIDs)
	s.Equal([]uint{tester.ID}, got.TesterIDs)
}

func (s *ProblemServiceTestSuite) TestCreateProblem_EmptyVisibilityModesUseDefaults() {
	group := s.createTestGroup("gdef")

	got, err := s.service.CreateProblem(CreateProblemRequest{
		Code: "DEF1", Name: "Defaults", Description: "body",
		Points: 100, TimeLimit: 1, MemoryLimit: 262144, GroupID: group.ID,
	})
	s.Require().NoError(err)
	s.Equal(models.SubmissionSourceAccessFollow, got.SubmissionSourceVisibilityMode)
	s.Equal(models.ProblemTestcaseAccessOutContest, got.TestcaseVisibilityMode)
}

func (s *ProblemServiceTestSuite) TestVisibilityModes_UnknownRejected() {
	group := s.createTestGroup("gbad")
	base := CreateProblemRequest{
		Code: "BAD1", Name: "Bad", Description: "body",
		Points: 100, TimeLimit: 1, MemoryLimit: 262144, GroupID: group.ID,
	}

	bad := base
	bad.SubmissionSourceVisibilityMode = "X"
	_, err := s.service.CreateProblem(bad)
	s.ErrorIs(err, ErrInvalidVisibilityMode)

	bad = base
	bad.TestcaseVisibilityMode = "B"
	_, err = s.service.CreateProblem(bad)
	s.ErrorIs(err, ErrInvalidVisibilityMode)

	var count int64
	s.database.Model(&models.Problem{}).Count(&count)
	s.Equal(int64(0), count, "an invalid mode must not create the problem")

	_, err = s.service.CreateProblem(base)
	s.Require().NoError(err)

	badMode := "Z"
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: "BAD1", TestcaseVisibilityMode: &badMode})
	s.ErrorIs(err, ErrInvalidVisibilityMode)

	got, err := s.service.GetProblem(GetProblemRequest{ProblemCode: "BAD1"})
	s.Require().NoError(err)
	s.Equal(models.ProblemTestcaseAccessOutContest, got.TestcaseVisibilityMode)
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_V1FieldsAndOrgBannedSync() {
	group := s.createTestGroup("gupd")
	s.createTestProblem("UPD1", "Before", group.ID)
	license := s.createTestLicense("mit")
	suggester := s.createTestProfile("suggester")
	banned := s.createTestProfile("troll")
	org1 := s.createTestOrganization("org1")
	org2 := s.createTestOrganization("org2")
	date := time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)

	source, og, summary := "ICPC 2025", "https://example.com/og.png", "one line"
	partialTrue, fullMarkup, orgPrivate := true, true, true
	srcMode, tcMode := models.SubmissionSourceAccessAlways, models.ProblemTestcaseAccessAlways
	orgIDs := []uint{org1.ID}
	bannedIDs := []uint{banned.ID}

	_, err := s.service.UpdateProblem(UpdateProblemRequest{
		ProblemCode: "UPD1",
		Source:      &source, Date: &date, LicenseID: &license.ID, OgImage: &og, Summary: &summary,
		Partial: &partialTrue, IsFullMarkup: &fullMarkup,
		SubmissionSourceVisibilityMode: &srcMode, TestcaseVisibilityMode: &tcMode,
		IsOrganizationPrivate: &orgPrivate, OrganizationIDs: &orgIDs, BannedUserIDs: &bannedIDs,
		SuggesterID: &suggester.ID,
	})
	s.Require().NoError(err)

	got, err := s.service.GetProblem(GetProblemRequest{ProblemCode: "UPD1"})
	s.Require().NoError(err)
	s.Equal("ICPC 2025", got.Source)
	s.Require().NotNil(got.Date)
	s.True(date.Equal(*got.Date))
	s.Require().NotNil(got.LicenseID)
	s.Equal(license.ID, *got.LicenseID)
	s.Equal("https://example.com/og.png", got.OgImage)
	s.Equal("one line", got.Summary)
	s.False(got.ShortCircuit, "derived: partial was set to true")
	s.True(got.IsFullMarkup)
	s.Equal(models.SubmissionSourceAccessAlways, got.SubmissionSourceVisibilityMode)
	s.Equal(models.ProblemTestcaseAccessAlways, got.TestcaseVisibilityMode)
	s.True(got.IsOrganizationPrivate)
	s.Equal([]uint{org1.ID}, got.OrganizationIDs)
	s.Equal([]uint{banned.ID}, got.BannedUserIDs)
	s.Require().NotNil(got.SuggesterID)
	s.Equal(suggester.ID, *got.SuggesterID)

	// Relations are full replacement lists.
	orgIDs = []uint{org2.ID}
	bannedIDs = []uint{}
	_, err = s.service.UpdateProblem(UpdateProblemRequest{
		ProblemCode: "UPD1", OrganizationIDs: &orgIDs, BannedUserIDs: &bannedIDs,
	})
	s.Require().NoError(err)
	got, _ = s.service.GetProblem(GetProblemRequest{ProblemCode: "UPD1"})
	s.Equal([]uint{org2.ID}, got.OrganizationIDs)
	s.Empty(got.BannedUserIDs)
	s.NotNil(got.BannedUserIDs)
	s.Equal("ICPC 2025", got.Source, "omitted scalars must stay unchanged")
}

func (s *ProblemServiceTestSuite) TestPartialFlipsShortCircuit() {
	group := s.createTestGroup("gsc")
	created, err := s.service.CreateProblem(CreateProblemRequest{
		Code: "SC1", Name: "Stops", Description: "body", GroupID: group.ID,
		Points: 100, TimeLimit: 1, MemoryLimit: 262144, Partial: false,
	})
	s.Require().NoError(err)
	s.True(created.ShortCircuit, "short_circuit = not partial")

	partial := true
	got, err := s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: "SC1", Partial: &partial})
	s.Require().NoError(err)
	s.False(got.ShortCircuit)

	partial = false
	got, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: "SC1", Partial: &partial})
	s.Require().NoError(err)
	s.True(got.ShortCircuit)
}

func (s *ProblemServiceTestSuite) TestCreateProblem_UnknownRelationRollsBack() {
	group := s.createTestGroup("groll")
	_, err := s.service.CreateProblem(CreateProblemRequest{
		Code: "ROLL1", Name: "Rollback", Description: "body", GroupID: group.ID,
		Points: 100, TimeLimit: 1, MemoryLimit: 262144,
		OrganizationIDs: []uint{424242},
	})
	s.Require().ErrorIs(err, relations.ErrUnknownID)

	var count int64
	s.database.Model(&models.Problem{}).Where("code = ?", "ROLL1").Count(&count)
	s.Equal(int64(0), count, "a rejected relation must not leave an orphan problem row")
}

func (s *ProblemServiceTestSuite) TestCreateProblem_UnknownLicenseRejected() {
	group := s.createTestGroup("glic")
	missing := uint(98765)
	_, err := s.service.CreateProblem(CreateProblemRequest{
		Code: "LIC1", Name: "License", Description: "body", GroupID: group.ID,
		Points: 100, TimeLimit: 1, MemoryLimit: 262144, LicenseID: &missing,
	})
	s.Require().ErrorIs(err, ErrInvalidLicense)

	var count int64
	s.database.Model(&models.Problem{}).Where("code = ?", "LIC1").Count(&count)
	s.Equal(int64(0), count)

	s.Require().NoError(s.database.Create(&models.Problem{
		Code: "LIC2", Name: "License", GroupID: group.ID,
	}).Error)
	_, err = s.service.UpdateProblem(UpdateProblemRequest{ProblemCode: "LIC2", LicenseID: &missing})
	s.ErrorIs(err, ErrInvalidLicense)
}

func (s *ProblemServiceTestSuite) TestUpdateProblem_ClearNullableFields() {
	group := s.createTestGroup("gclear")
	license := s.createTestLicense("apache")
	suggester := s.createTestProfile("suggester")
	date := time.Date(2026, 7, 8, 9, 10, 11, 0, time.UTC)

	s.Require().NoError(s.database.Create(&models.Problem{
		Code: "CLR9", Name: "Clear", GroupID: group.ID, Date: &date,
		LicenseID: &license.ID, SuggesterID: &suggester.ID,
	}).Error)

	got, err := s.service.UpdateProblem(UpdateProblemRequest{
		ProblemCode: "CLR9", ClearDate: true, ClearLicenseID: true, ClearSuggesterID: true,
	})
	s.Require().NoError(err)
	s.Nil(got.Date)
	s.Nil(got.LicenseID)
	s.Nil(got.SuggesterID)
}
