/*
 * Harbor API
 *
 * These APIs provide services for manipulating Harbor project.
 *
 * API version: 2.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package swagger

type Task struct {
	// The ID of task
	Id int32 `json:"id,omitempty"`
	// The ID of task execution
	ExecutionId int32 `json:"execution_id,omitempty"`
	// The status of task
	Status string `json:"status,omitempty"`
	// The status message of task
	StatusMessage string `json:"status_message,omitempty"`
	// The count of task run
	RunCount int32 `json:"run_count,omitempty"`
	ExtraAttrs *ExtraAttrs `json:"extra_attrs,omitempty"`
	// The creation time of task
	CreationTime string `json:"creation_time,omitempty"`
	// The update time of task
	UpdateTime string `json:"update_time,omitempty"`
	// The start time of task
	StartTime string `json:"start_time,omitempty"`
	// The end time of task
	EndTime string `json:"end_time,omitempty"`
}
