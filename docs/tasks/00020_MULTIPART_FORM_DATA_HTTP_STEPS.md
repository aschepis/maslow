---
id: 20
title: Add multipart/form-data support for HTTP contract steps
status: draft
priority: low
created: 2026-02-20
updated: 2026-02-20
assigned_to: ""
assigned_at: ""
depends_on: [13]
tags: [maslow, verification, contract, http]
---

## Objective

File upload is a core feature of many applications (MaslowTok's entire value prop is video upload) but HTTP contract steps only support string bodies. There's no way to send multipart/form-data requests with file attachments. This means upload endpoints — often the most complex and error-prone part of an API — can't be contract-tested.

## Requirements

- Extend the HTTP step schema to support multipart/form-data requests
- Add a `files` or `multipart` field to HTTP steps:
  - Each part has a field name, file path (or inline content), and optional content type
  - Support mixing file parts with regular form fields
- The runner constructs a proper multipart request with boundaries
- Expectations should work normally on the response (status, json_path, headers, etc.)
- Update CUE schema, embedded copy, Go types, and runner

## Acceptance Criteria

- [ ] Schema accepts `multipart` field on HTTP steps with file and field parts
- [ ] Runner constructs valid multipart/form-data requests
- [ ] File parts read from the specified path relative to the project root
- [ ] Form field parts are sent as text values
- [ ] Content-Type header is automatically set to multipart/form-data with boundary
- [ ] Response expectations work normally after multipart request
- [ ] Missing file paths produce clear error messages
- [ ] Unit tests cover: single file upload, multiple files, mixed files and fields, missing file
- [ ] Update maslow-tok example to include a video upload contract

## Notes

- Example usage:
  ```yaml
  - action: http
    method: POST
    url: "${API_BASE_URL}/v1/videos"
    headers:
      Authorization: "Bearer ${{access_token}}"
    multipart:
      - field: title
        value: "Test Video"
      - field: video
        file: testdata/sample.mp4
        content_type: video/mp4
    expect:
      status: 201
      json_path: "$.id"
  ```
- Depends on task 13 (capture) because upload endpoints almost always require authentication
- Go's `mime/multipart` package handles the heavy lifting
- Consider file size limits for test fixtures — don't want huge files in the repo. Small sample files or generated test data is preferred.
- The RPG game example could use this for asset upload/import testing
