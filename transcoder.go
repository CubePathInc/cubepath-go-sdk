package cubepath

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// TranscoderService handles communication with the Video Transcoder related methods of the
// CubePath API.
//
// A job reads a video from a URL or any S3 compatible bucket and writes the requested outputs
// to your S3 compatible bucket. Jobs run asynchronously: poll GetJob until Status is
// "completed", "failed" or "canceled", or pass a WebhookURL.
type TranscoderService interface {
	CreateJob(ctx context.Context, req *CreateTranscodeJobRequest) (*TranscodeJob, error)
	// CreateBatch submits up to 1000 videos that share the same outputs and destination.
	CreateBatch(ctx context.Context, req *CreateTranscodeBatchRequest) (*TranscodeBatch, error)
	// ListJobs returns one page of jobs, newest first. There is no total: keep paging until a
	// page has fewer than Limit jobs.
	ListJobs(ctx context.Context, opts *TranscodeJobListOptions) (*TranscodeJobList, error)
	GetJob(ctx context.Context, uuid string) (*TranscodeJob, error)
	// GetJobOutputs returns the files a job wrote and their destination.
	GetJobOutputs(ctx context.Context, uuid string) (*TranscodeJobOutputs, error)
	// CancelJob cancels a job that has not finished yet.
	CancelJob(ctx context.Context, uuid string) (*TranscodeJobCancel, error)
}

// Transcode job statuses.
const (
	TranscodeStatusQueued     = "queued"
	TranscodeStatusAnalyzing  = "analyzing"
	TranscodeStatusEncoding   = "encoding"
	TranscodeStatusFinalizing = "finalizing"
	TranscodeStatusCompleted  = "completed"
	TranscodeStatusFailed     = "failed"
	TranscodeStatusCanceled   = "canceled"
)

// TranscodeS3 is an S3 compatible location. Leave Endpoint empty for AWS S3. Path is the
// object key of an input, or the key prefix of an output. The secret key is never returned.
type TranscodeS3 struct {
	Endpoint  string `json:"endpoint,omitempty"`
	Region    string `json:"region,omitempty"`
	Bucket    string `json:"bucket"`
	Path      string `json:"path,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	SecretKey string `json:"secret_key,omitempty"`
}

// TranscodeInput is the source of a job: Source "url" with URL, or "s3" with S3.
type TranscodeInput struct {
	Source string       `json:"source"`
	URL    string       `json:"url,omitempty"`
	S3     *TranscodeS3 `json:"s3,omitempty"`
}

// TranscodeDestination is where a job writes its outputs.
type TranscodeDestination struct {
	S3 TranscodeS3 `json:"s3"`
}

// TranscodeOutputSpec is one requested output. Type is "file", "hls", "thumbnails" or "gif";
// Params holds the format settings (for example "codec", "height", "container") and is sent
// next to "type".
type TranscodeOutputSpec struct {
	Type   string
	Params map[string]interface{}
}

// MarshalJSON sends Params flattened next to "type".
func (o TranscodeOutputSpec) MarshalJSON() ([]byte, error) {
	body := make(map[string]interface{}, len(o.Params)+1)
	for k, v := range o.Params {
		body[k] = v
	}
	body["type"] = o.Type
	return json.Marshal(body)
}

// UnmarshalJSON reads "type" into Type and every other key into Params.
func (o *TranscodeOutputSpec) UnmarshalJSON(data []byte) error {
	var body map[string]interface{}
	if err := json.Unmarshal(data, &body); err != nil {
		return err
	}
	o.Type, _ = body["type"].(string)
	delete(body, "type")
	o.Params = body
	return nil
}

// CreateTranscodeJobRequest represents a request to create a transcoding job. Outputs holds
// 1-20 outputs. A repeated IdempotencyKey returns the job created first instead of a new one.
type CreateTranscodeJobRequest struct {
	Input          TranscodeInput        `json:"input"`
	Output         TranscodeDestination  `json:"output"`
	Outputs        []TranscodeOutputSpec `json:"outputs"`
	WebhookURL     string                `json:"webhook_url,omitempty"`
	IdempotencyKey string                `json:"idempotency_key,omitempty"`
}

// TranscodeBatchInput is one video of a batch: a full S3 location, a URL, or a Path inside
// the batch InputDefaults bucket. OutSubpath is appended to the destination path.
type TranscodeBatchInput struct {
	S3         *TranscodeS3 `json:"s3,omitempty"`
	URL        string       `json:"url,omitempty"`
	Path       string       `json:"path,omitempty"`
	OutSubpath string       `json:"out_subpath,omitempty"`
}

// CreateTranscodeBatchRequest represents a request to create several jobs at once.
type CreateTranscodeBatchRequest struct {
	Output        TranscodeDestination  `json:"output"`
	Outputs       []TranscodeOutputSpec `json:"outputs"`
	InputDefaults *TranscodeInput       `json:"input_defaults,omitempty"`
	Inputs        []TranscodeBatchInput `json:"inputs"`
	WebhookURL    string                `json:"webhook_url,omitempty"`
}

// TranscodeBatch is the result of a batch submission.
type TranscodeBatch struct {
	BatchID string   `json:"batch_id"`
	JobIDs  []string `json:"job_ids"`
	Count   int      `json:"count"`
}

// TranscodeOutputFile is a file written by a job.
type TranscodeOutputFile struct {
	Type   string `json:"type"`
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

// TranscodeJob represents a transcoding job. Outputs lists the files written, once the job
// has finished.
type TranscodeJob struct {
	UUID   string                `json:"uuid"`
	Status string                `json:"status"`
	Input  *TranscodeInput       `json:"input"`
	Output *TranscodeDestination `json:"output"`
	Spec   struct {
		Outputs []TranscodeOutputSpec `json:"outputs"`
	} `json:"spec"`
	Outputs           []TranscodeOutputFile `json:"outputs"`
	Progress          int                   `json:"progress"`
	TotalSegments     *int                  `json:"total_segments"`
	CompletedSegments *int                  `json:"completed_segments"`
	BatchID           *string               `json:"batch_id"`
	Error             *string               `json:"error"`
	CreatedAt         string                `json:"created_at"`
}

// TranscodeJobListOptions pages and filters ListJobs. Limit is 1-500 (default 100).
type TranscodeJobListOptions struct {
	BatchID string
	Limit   int
	Offset  int
}

// TranscodeJobList is one page of jobs.
type TranscodeJobList struct {
	Jobs   []TranscodeJob `json:"jobs"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// TranscodeJobOutputs are the files of a job and the destination they were written to.
type TranscodeJobOutputs struct {
	Outputs     []TranscodeOutputFile `json:"outputs"`
	Destination *TranscodeDestination `json:"destination"`
}

// TranscodeJobCancel is the response of a cancellation.
type TranscodeJobCancel struct {
	Detail string `json:"detail"`
	Status string `json:"status"`
}

type transcoderService struct {
	client *Client
}

func (s *transcoderService) CreateJob(ctx context.Context, req *CreateTranscodeJobRequest) (*TranscodeJob, error) {
	var job TranscodeJob
	if err := s.client.post(ctx, "/transcoder/jobs", req, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *transcoderService) CreateBatch(ctx context.Context, req *CreateTranscodeBatchRequest) (*TranscodeBatch, error) {
	var batch TranscodeBatch
	if err := s.client.post(ctx, "/transcoder/jobs/batch", req, &batch); err != nil {
		return nil, err
	}
	return &batch, nil
}

func (s *transcoderService) ListJobs(ctx context.Context, opts *TranscodeJobListOptions) (*TranscodeJobList, error) {
	path := "/transcoder/jobs"
	if opts != nil {
		v := url.Values{}
		if opts.BatchID != "" {
			v.Set("batch_id", opts.BatchID)
		}
		if opts.Limit > 0 {
			v.Set("limit", strconv.Itoa(opts.Limit))
		}
		if opts.Offset > 0 {
			v.Set("offset", strconv.Itoa(opts.Offset))
		}
		if len(v) > 0 {
			path += "?" + v.Encode()
		}
	}
	var list TranscodeJobList
	if err := s.client.get(ctx, path, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

func (s *transcoderService) GetJob(ctx context.Context, uuid string) (*TranscodeJob, error) {
	var job TranscodeJob
	if err := s.client.get(ctx, "/transcoder/jobs/"+url.PathEscape(uuid), &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *transcoderService) GetJobOutputs(ctx context.Context, uuid string) (*TranscodeJobOutputs, error) {
	var outputs TranscodeJobOutputs
	if err := s.client.get(ctx, "/transcoder/jobs/"+url.PathEscape(uuid)+"/outputs", &outputs); err != nil {
		return nil, err
	}
	return &outputs, nil
}

func (s *transcoderService) CancelJob(ctx context.Context, uuid string) (*TranscodeJobCancel, error) {
	var result TranscodeJobCancel
	if err := s.client.delWithResult(ctx, "/transcoder/jobs/"+url.PathEscape(uuid), &result); err != nil {
		return nil, err
	}
	return &result, nil
}
