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
	"anttrader/internal/pglisten"
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
	jobs    *repository.JobRepository
	log     *zap.Logger
	pgListen *pglisten.Listener
}

var _ antv1c.JobServiceHandler = (*JobServer)(nil)

func NewJobServer(jobs *repository.JobRepository, log *zap.Logger) *JobServer {
	return &JobServer{jobs: jobs, log: log}
}

func (s *JobServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }

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
		return nil, connect.NewError(connect.CodeInternal, err)
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
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(jobToProto(j)), nil
}

func (s *JobServer) SubscribeJob(ctx context.Context, req *connect.Request[antv1.SubscribeJobRequest], stream *connect.ServerStream[antv1.JobEvent]) error {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	uid := s.userID(ctx)
	afterSeq := req.Msg.AfterSeq

	// Drain any existing events first.
	events, qerr := s.jobs.ListEvents(ctx, uid, jobID, afterSeq, 100)
	if qerr != nil {
		return connect.NewError(connect.CodeInternal, qerr)
	}
	for _, ev := range events {
		if err := stream.Send(&antv1.JobEvent{
			JobId: ev.JobID.String(), Seq: ev.Seq, Type: ev.Type,
			Status: ev.Status, Progress: ev.Progress, Stage: ev.Stage,
			Message: payloadToMsg(ev), Payload: jsonbToStruct(ev.Payload),
			CreatedAt: timestamppb.New(ev.CreatedAt),
		}); err != nil {
			return err
		}
		afterSeq = ev.Seq
	}

	// Check if job is already terminal — no need to listen.
	job, err := s.jobs.GetJob(ctx, uid, jobID)
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}
	if job.Status == "succeeded" || job.Status == "failed" || job.Status == "cancelled" {
		return nil
	}

	// Push-first: PG LISTEN for new events, ticker as fallback.
	var notifCh <-chan string
	var listenCancel func()
	if s.pgListen != nil {
		notifCh, listenCancel, _ = s.pgListen.Listen(ctx, "job_events")
		if listenCancel != nil {
			defer listenCancel()
		}
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-notifCh:
		case <-ticker.C:
		}

		events, qerr := s.jobs.ListEvents(ctx, uid, jobID, afterSeq, 100)
		if qerr != nil {
			s.log.Warn("SubscribeJob: ListEvents failed", zap.Error(qerr))
			continue
		}

		for _, ev := range events {
			if err := stream.Send(&antv1.JobEvent{
				JobId: ev.JobID.String(), Seq: ev.Seq, Type: ev.Type,
				Status: ev.Status, Progress: ev.Progress, Stage: ev.Stage,
				Message: payloadToMsg(ev), Payload: jsonbToStruct(ev.Payload),
				CreatedAt: timestamppb.New(ev.CreatedAt),
			}); err != nil {
				return err
			}
			afterSeq = ev.Seq
		}

		// Check if job reached terminal state.
		if len(events) > 0 {
			last := events[len(events)-1]
			if last.Status == "succeeded" || last.Status == "failed" || last.Status == "cancelled" {
				return nil
			}
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
