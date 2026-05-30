package system

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
)

// jsonbToStruct converts JSONB bytes to a proto Struct.
// structpb.NewStruct requires map[string]any — this is the canonical
// conversion point, encapsulated to contain the dynamic type boundary.
func jsonbToStruct(raw []byte) *structpb.Struct {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	s, _ := structpb.NewStruct(m)
	return s
}

// JobServer implements ant.v1.JobServiceHandler.
type JobServer struct {
	jobs *repository.JobRepository
	log  *zap.Logger
}

var _ antv1c.JobServiceHandler = (*JobServer)(nil)

func NewJobServer(jobs *repository.JobRepository, log *zap.Logger) *JobServer {
	return &JobServer{jobs: jobs, log: log}
}

func (s *JobServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func (s *JobServer) GetJob(ctx context.Context, req *connect.Request[antv1.GetJobRequest]) (*connect.Response[antv1.Job], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	j, err := s.jobs.GetJob(ctx, s.userID(ctx), jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	return connect.NewResponse(jobToProto(j)), nil
}

func (s *JobServer) CancelJob(ctx context.Context, req *connect.Request[antv1.CancelJobRequest]) (*connect.Response[antv1.Job], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	j, err := s.jobs.CancelJob(ctx, s.userID(ctx), jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	return connect.NewResponse(jobToProto(j)), nil
}

func (s *JobServer) SubscribeJob(ctx context.Context, req *connect.Request[antv1.SubscribeJobRequest], stream *connect.ServerStream[antv1.JobEvent]) error {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	afterSeq := req.Msg.AfterSeq

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		events, qerr := s.jobs.ListEvents(ctx, jobID, afterSeq, 100)
		if qerr != nil {
			return qerr
		}

		if len(events) == 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
			}
			continue
		}

		for _, ev := range events {
			stream.Send(&antv1.JobEvent{
				JobId: ev.JobID.String(), Seq: ev.Seq, Type: ev.Type,
				Status: ev.Status, Progress: ev.Progress, Stage: ev.Stage,
				Message: payloadToMsg(ev), Payload: jsonbToStruct(ev.Payload),
				CreatedAt: timestamppb.New(ev.CreatedAt),
			})
			afterSeq = ev.Seq
		}
	}
}

func payloadToMsg(ev repository.JobEvent) string {
	return ev.Message
}

func jobToProto(j *repository.Job) *antv1.Job {
	out := &antv1.Job{
		Id: j.ID.String(), UserId: j.UserID.String(), Kind: j.Kind,
		Status: j.Status, Progress: j.Progress, Stage: j.Stage,
		RequestSummary: j.RequestSummary, ResultRef: j.ResultRef,
		ResultSummary: j.ResultSummary, ErrorCode: j.ErrorCode,
		ErrorMessage: j.ErrorMessage, IdempotencyKey: j.IdempotencyKey,
		CreatedAt: timestamppb.New(j.CreatedAt),
	}
	if j.StartedAt != nil {
		out.StartedAt = timestamppb.New(*j.StartedAt)
	}
	if j.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*j.FinishedAt)
	}
	if j.ExpiresAt != nil {
		out.ExpiresAt = timestamppb.New(*j.ExpiresAt)
	}
	return out
}
