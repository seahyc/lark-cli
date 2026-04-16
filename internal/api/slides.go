package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"

	"github.com/yjwong/lark-cli/internal/auth"
)

// SlidesPresentation is a lightweight view of a Lark slide deck.
type SlidesPresentation struct {
	PresentationID string `json:"presentation_id,omitempty"`
	Title          string `json:"title,omitempty"`
	FolderToken    string `json:"folder_token,omitempty"`
	URL            string `json:"url,omitempty"`
	CreateTime     string `json:"create_time,omitempty"`
	UpdateTime     string `json:"update_time,omitempty"`
	OwnerID        string `json:"owner_id,omitempty"`
}

// CreateSlidesPresentation creates a new empty presentation.
// Uses user token (slides:presentation:create).
func (c *Client) CreateSlidesPresentation(title, folderToken string) (*SlidesPresentation, error) {
	body := map[string]interface{}{"title": title}
	if folderToken != "" {
		body["folder_token"] = folderToken
	}
	var resp struct {
		BaseResponse
		Data struct {
			Presentation SlidesPresentation `json:"presentation"`
		} `json:"data"`
	}
	if err := c.Post("/slides/v1/presentations", body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return &resp.Data.Presentation, nil
}

// GetSlidesPresentation fetches an XML presentation's metadata.
func (c *Client) GetSlidesPresentation(presentationID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/slides/v1/xml_presentations/%s", presentationID)
	var resp struct {
		BaseResponse
		Data map[string]interface{} `json:"data"`
	}
	if err := c.Get(path, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// CreateSlidesSlide appends a new slide to an existing presentation using XML content.
func (c *Client) CreateSlidesSlide(presentationID, xmlContent string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/slides/v1/xml_presentations/%s/slides", presentationID)
	body := map[string]interface{}{"xml_content": xmlContent}
	var resp struct {
		BaseResponse
		Data map[string]interface{} `json:"data"`
	}
	if err := c.Post(path, body, &resp); err != nil {
		return nil, err
	}
	if err := resp.Err(); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// DeleteSlidesSlide removes a slide from a presentation by slide_id.
func (c *Client) DeleteSlidesSlide(presentationID, slideID string) error {
	path := fmt.Sprintf("/slides/v1/xml_presentations/%s/slides/%s", presentationID, slideID)
	var resp BaseResponse
	if err := c.Delete(path, &resp); err != nil {
		return err
	}
	return resp.Err()
}

// UploadMedia uploads a file to Lark Drive and returns a file_token that can be
// referenced in slides/docs content. Uses /open-apis/drive/v1/medias/upload_all.
//
// parentType should be "slides_image" or "docx_image" etc. Defaults to "slides_image".
func (c *Client) UploadMedia(filePath, parentType, parentNode string) (string, error) {
	if err := auth.EnsureValidToken(); err != nil {
		return "", err
	}
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}
	if parentType == "" {
		parentType = "slides_image"
	}

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("file_name", filepath.Base(filePath))
	_ = mw.WriteField("parent_type", parentType)
	_ = mw.WriteField("size", fmt.Sprintf("%d", fi.Size()))
	if parentNode != "" {
		_ = mw.WriteField("parent_node", parentNode)
	}
	fw, err := mw.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(fw, f); err != nil {
		return "", err
	}
	_ = mw.Close()

	req, err := http.NewRequest("POST", baseURL+"/drive/v1/medias/upload_all", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+auth.GetTokenStore().GetAccessToken())
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)

	var wrap struct {
		BaseResponse
		Data struct {
			FileToken string `json:"file_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rb, &wrap); err != nil {
		return "", fmt.Errorf("parse response: %w (body=%s)", err, string(rb))
	}
	if err := wrap.Err(); err != nil {
		return "", err
	}
	return wrap.Data.FileToken, nil
}
