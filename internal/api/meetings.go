package api

import (
	"fmt"
	"net/url"
	"strconv"
)

// SearchMeetings searches the user's meeting history.
// Uses /vc/v1/meeting_list with start_time/end_time filters.
// Note: The exact Lark VC endpoint semantics vary by tenant; fall back to
// `lark api` for more advanced queries.
func (c *Client) SearchMeetings(opts *SearchMeetingsOptions) ([]Meeting, bool, string, error) {
	params := url.Values{}
	if opts != nil {
		if opts.StartTime != "" {
			params.Set("start_time", opts.StartTime)
		}
		if opts.EndTime != "" {
			params.Set("end_time", opts.EndTime)
		}
		if opts.MeetingNo != "" {
			params.Set("meeting_no", opts.MeetingNo)
		}
		if opts.OrganizerID != "" {
			params.Set("user_id", opts.OrganizerID)
		}
		if opts.PageSize > 0 {
			params.Set("page_size", strconv.Itoa(opts.PageSize))
		}
		if opts.PageToken != "" {
			params.Set("page_token", opts.PageToken)
		}
	}
	path := "/vc/v1/meeting_list"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var resp MeetingListResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, false, "", err
	}
	if err := resp.Err(); err != nil {
		return nil, false, "", err
	}
	items := resp.Data.MeetingList
	if len(items) == 0 {
		items = resp.Data.Items
	}
	return items, resp.Data.HasMore, resp.Data.PageToken, nil
}

// GetMeeting returns metadata for a meeting by ID.
func (c *Client) GetMeeting(meetingID string) (*Meeting, error) {
	path := fmt.Sprintf("/vc/v1/meetings/%s", url.PathEscape(meetingID))
	var resp MeetingResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data.Meeting, nil
}

// GetMeetingNotes returns notes attached to a meeting.
func (c *Client) GetMeetingNotes(meetingID string) (*MeetingNotes, error) {
	path := fmt.Sprintf("/vc/v1/meetings/%s/notes", url.PathEscape(meetingID))
	var resp MeetingNotesResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return &MeetingNotes{MeetingID: meetingID}, nil
	}
	resp.Data.MeetingID = meetingID
	return resp.Data, nil
}

// GetMeetingRecording returns recording info for a meeting, including the
// minute_token that can be fed into the existing minutes API.
func (c *Client) GetMeetingRecording(meetingID string) (*MeetingRecording, error) {
	path := fmt.Sprintf("/vc/v1/meetings/%s/recording", url.PathEscape(meetingID))
	var resp MeetingRecordingResponse
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	if resp.Data.Recording == nil {
		return &MeetingRecording{MeetingID: meetingID}, nil
	}
	resp.Data.Recording.MeetingID = meetingID
	return resp.Data.Recording, nil
}
