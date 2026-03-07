// Package docs CLAOJ API v2
//
// CLAOJ (Competitive Programming Online Judge) API v2 documentation.
// This API provides access to problems, contests, submissions, users, and administrative functions.
//
//	@title			CLAOJ API v2
//	@version		2.0.0
//	@description	Competitive Programming Online Judge API
//	@termsOfService	http://claoj.edu.vn/terms/

//	@contact.name	CLAOJ Support
//	@contact.email	support@claoj.edu.vn

//	@license.name	MIT
//	@license.url	https://opensource.org/licenses/MIT

//	@host			beta.claoj.edu.vn
//	@BasePath		/api/v2
//	@schemes		https http

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization
//	@description	API Token authentication (64 hex character token) or Bearer JWT token

//	@securityDefinitions.cookie	AccessTokenCookie
//	@in							cookie
//	@name						access_token
//	@description	HTTP-only cookie containing JWT access token

//	@tag.name		Authentication
//	@tag.description	User authentication, registration, and token management

//	@tag.name		Problems
//	@tag.description	Problem listing, details, and submissions

//	@tag.name		Contests
//	@tag.description	Contest listing, details, and participation

//	@tag.name		Submissions
//	@tag.description	Submission listing, details, and results

//	@tag.name		Users
//	@tag.description	User profiles, ratings, and statistics

//	@tag.name		Organizations
//	@tag.description	Organization management and membership

//	@tag.name		Admin
//	@tag.description	Administrative endpoints for user, problem, and contest management

//	@tag.name		Comments
//	@tag.description	Comment system for problems and general discussions

//	@tag.name		Blog
//	@tag.description	Blog posts and feeds

//	@tag.name		Tickets
//	@tag.description	Support tickets and issue tracking

//	@tag.name		Stats
//	@tag.description	Statistics and analytics

//	@externalDocs.description	OpenAPI Specification
//	@externalDocs.url			http://beta.claoj.edu.vn/api/v2/swagger/doc.json
package docs
