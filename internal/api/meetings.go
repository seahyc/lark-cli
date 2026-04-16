package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// SearchMeetings searches the user's meeting history via POST /vc/v1/meetings/search.
// Defaults to the last 30 days when no time range is specified.
func (c *Client) SearchMeetings(opts *SearchMeetingsOptions) ([]Meeting, bool, string, error) {
	body := make(map[string]interface{})

	// Default to last 30 days if no time range
	if opts == nil || (opts.StartTime == "" && opts.EndTime == "") {
		now := strconv.FormatInt(time.Now().Unix(), 10)
		monthAgo := strconv.FormatInt(time.Now().AddDate(0, 0, -30).Unix(), 10)
		body["start_time"] = monthAgo
		body["end_time"] = now
	} else {
		if opts.StartTime != "" {
			body["start_time"] = opts.StartTime
		}
		if opts.EndTime != "" {
			body["end_time"] = opts.EndTime
		}
	}

	if opts != nil {
		if opts.MeetingNo != "" {
			body["meeting_no"] = opts.MeetingNo
		}
		if opts.OrganizerID != "" {
			body["user_id"] = opts.OrganizerID
		}
	}

	params := url.Values{}
	if opts != nil {
		if opts.PageSize > 0 {
			params.Set("page_size", strconv.Itoa(opts.PageSize))
		}
		if opts.PageToken != "" {
			params.Set("page_token", opts.PageToken)
		}
	}
	path := "/vc/v1/meetings/search"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}

	var resp MeetingListResponse
	if err := c.Post(path, body, &resp); err != nil {
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
