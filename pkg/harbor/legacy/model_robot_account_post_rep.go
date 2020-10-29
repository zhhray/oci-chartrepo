/*
 * Harbor API
 *
 * These APIs provide services for manipulating Harbor project.
 *
 * API version: 2.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package swagger

type RobotAccountPostRep struct {
	// the name of robot account
	Name string `json:"name,omitempty"`
	// the token of robot account
	Token string `json:"token,omitempty"`
}
