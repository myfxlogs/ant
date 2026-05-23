package connect

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/repository"
)

type JobService struct {
	repo *repository.JobRepository
}

func NewJobService(repo *repository.JobRepository) *JobService {
	return &JobService{repo: repo}
}

func (s *JobService) GetJob(ctx context.Context, req *connect.Request[v1.GetJobRequest]) (*connect.Response[v1.Job], error) {
	job, err := s.getOwnedJob(ctx, strings.TrimSpace(req.Msg.GetJobId()))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(jobToProto(job)), nil
}

func (s *JobService) CancelJob(ctx context.Context, req *connect.Request[v1.CancelJobRequest]) (*connect.Response[v1.Job], error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("job service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := parseRequestUUID(req.Msg.GetJobId())
	if err != nil {
		return nil, err
	}
	job, err := s.repo.CancelJob(ctx, userID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	_ = s.repo.AddEvent(ctx, &repository.JobEvent{JobID: job.ID, Type: "cancelled", Status: job.Status, Progress: job.Progress, Stage: job.Stage, Message: "job cancelled"})
	return connect.NewResponse(jobToProto(job)), nil
}

func (s *JobService) SubscribeJob(ctx context.Context, req *connect.Request[v1.SubscribeJobRequest], stream *connect.ServerStream[v1.JobEvent]) error {
	if s == nil || s.repo == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("job service not available"))
	}
	job, err := s.getOwnedJob(ctx, strings.TrimSpace(req.Msg.GetJobId()))
	if err != nil {
		return err
	}
	snapshot := &v1.JobEvent{JobId: job.ID.String(), Seq: req.Msg.GetAfterSeq(), Type: "snapshot", Status: job.Status, Progress: job.Progress, Stage: job.Stage, Message: job.ResultSummary, CreatedAt: timestamppb.Now()}
	if err := stream.Send(snapshot); err != nil {
		return err
	}
	lastSeq, err := s.sendJobEvents(ctx, stream, job.ID, req.Msg.GetAfterSeq())
	if err != nil {
		return err
	}
	for isActiveJobStatus(job.Status) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
		nextSeq, err := s.sendJobEvents(ctx, stream, job.ID, lastSeq)
		if err != nil {
			return err
		}
		job, err = s.getOwnedJob(ctx, job.ID.String())
		if err != nil {
			return err
		}
		if nextSeq == lastSeq {
			ping := newJobPingEvent(job, lastSeq)
			if err := stream.Send(ping); err != nil {
				return err
			}
		}
		lastSeq = nextSeq
	}
	return nil
}

func (s *JobService) sendJobEvents(ctx context.Context, stream *connect.ServerStream[v1.JobEvent], jobID uuid.UUID, afterSeq int64) (int64, error) {
	events, err := s.repo.ListEvents(ctx, jobID, afterSeq, 100)
	if err != nil {
		return afterSeq, connect.NewError(connect.CodeInternal, err)
	}
	lastSeq := afterSeq
	for i := range events {
		if err := stream.Send(jobEventToProto(&events[i])); err != nil {
			return lastSeq, err
		}
		if events[i].Seq > lastSeq {
			lastSeq = events[i].Seq
		}
	}
	return lastSeq, nil
}

func isActiveJobStatus(status string) bool {
	return status == "queued" || status == "running"
}

func newJobPingEvent(job *repository.Job, seq int64) *v1.JobEvent {
	return &v1.JobEvent{JobId: job.ID.String(), Seq: seq, Type: "ping", Status: job.Status, Progress: job.Progress, Stage: job.Stage, CreatedAt: timestamppb.New(time.Now().UTC())}
}

func (s *JobService) getOwnedJob(ctx context.Context, id string) (*repository.Job, error) {
	if s == nil || s.repo == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("job service not available"))
	}
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	jobID, err := parseRequestUUID(id)
	if err != nil {
		return nil, err
	}
	job, err := s.repo.GetJob(ctx, userID, jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return job, nil
}

func jobToProto(job *repository.Job) *v1.Job {
	if job == nil {
		return nil
	}
	out := &v1.Job{Id: job.ID.String(), UserId: job.UserID.String(), Kind: job.Kind, Status: job.Status, Progress: job.Progress, Stage: job.Stage, RequestSummary: job.RequestSummary, ResultRef: job.ResultRef, ResultSummary: job.ResultSummary, ErrorCode: job.ErrorCode, ErrorMessage: job.ErrorMessage, IdempotencyKey: job.IdempotencyKey, CreatedAt: timestamppb.New(job.CreatedAt.UTC())}
	if job.StartedAt != nil {
		out.StartedAt = timestamppb.New(job.StartedAt.UTC())
	}
	if job.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(job.FinishedAt.UTC())
	}
	if job.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(job.ExpiresAt.UTC())
	}
	return out
}

func jobEventToProto(event *repository.JobEvent) *v1.JobEvent {
	if event == nil {
		return nil
	}
	payload := &structpb.Struct{}
	if len(event.Payload) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(event.Payload, &raw); err == nil {
			payload, _ = structpb.NewStruct(raw)
		} else {
			_ = protojson.Unmarshal(event.Payload, payload)
		}
	}
	return &v1.JobEvent{JobId: event.JobID.String(), Seq: event.Seq, Type: event.Type, Status: event.Status, Progress: event.Progress, Stage: event.Stage, Message: event.Message, Payload: payload, CreatedAt: timestamppb.New(event.CreatedAt.UTC())}
}
