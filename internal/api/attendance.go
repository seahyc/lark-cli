package api

// AttendanceUserTask represents a single attendance clock-in record for a user on a given day.
type AttendanceUserTask struct {
	ResultID     string `json:"result_id,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	EmployeeName string `json:"employee_name,omitempty"`
	Day          int    `json:"day,omitempty"`          // YYYYMMDD
	GroupID      string `json:"group_id,omitempty"`
	ShiftID      string `json:"shift_id,omitempty"`
	RecordID     string `json:"record_id,omitempty"`
	CheckInTime  string `json:"check_in_time,omitempty"`
	CheckOutTime string `json:"check_out_time,omitempty"`
	CheckInRes   string `json:"check_in_result,omitempty"`
	CheckOutRes  string `json:"check_out_result,omitempty"`
	Status       string `json:"status,omitempty"`
}

// ListAttendanceUserTasks queries attendance clock-in records for the given users
// and date range. checkDateFrom/checkDateTo are YYYYMMDD integers.
// Lark's endpoint is POST /attendance/v1/user_tasks/query; it requires tenant token.
func (c *Client) ListAttendanceUserTasks(userIDs []string, checkDateFrom, checkDateTo int, needOvertime bool) ([]AttendanceUserTask, error) {
	body := map[string]interface{}{
		"user_ids":        userIDs,
		"check_date_from": checkDateFrom,
		"check_date_to":   checkDateTo,
	}
	if needOvertime {
		body["need_overtime_result"] = true
	}

	var resp struct {
		BaseResponse
		Data struct {
			UserTaskResults []AttendanceUserTask `json:"user_task_results"`
			InvalidUserIDs  []string             `json:"invalid_user_ids,omitempty"`
			UnauthedUserIDs []string             `json:"unauthed_user_ids,omitempty"`
		} `json:"data"`
	}
	if err := c.PostWithTenantToken("/attendance/v1/user_tasks/query?employee_type=open_id&ignore_invalid_users=true&include_terminated_user=false", body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.UserTaskResults, nil
}
