/*
 * Harbor API
 *
 * These APIs provide services for manipulating Harbor project.
 *
 * API version: 2.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package swagger

// The metadata info of the scanner adapter
type ScannerAdapterMetadata struct {
	Name *Scanner `json:"name,omitempty"`
	Capabilities []ScannerCapability `json:"capabilities,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}
