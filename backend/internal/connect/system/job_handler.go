package connect

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
)

var errJobNotFound = errors.New("job not found")

// JobServer implements ant.v1.JobServiceHandler.
type JobServer struct {
	pg  *pgxpool.Pool
	log *zap.Logger
}

var _ antv1c.JobServiceHandler = (*JobServer)(nil)

func NewJobServer(pg *pgxpool.Pool, log *zap.Logger) *JobServer { return &JobServer{pg: pg, log: log} }

func (s *JobServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

const jobSelectCols = `id, user_id, kind, status, progress, stage, request_summary, result_ref, result_summary, error_code, error_message, idempotency_key, created_at, started_at, finished_at, expires_at`

func scanJobRow(row pgx.Row) (*antv1.Job, error) {
	var (
		id, userID, kind, status, stage, requestSummary, resultRef, resultSummary, errorCode, errorMessage, idempotencyKey string
		progress                                                                                                           float64
		createdAt                                                                                                          time.Time
		startedAt, finishedAt, expiresAt                                                                                   *time.Time
	)
	err := row.Scan(&id, &userID, &kind, &status, &progress, &stage, &requestSummary, &resultRef, &resultSummary,
		&errorCode, &errorMessage, &idempotencyKey, &createdAt, &startedAt, &finishedAt, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errJobNotFound
		}
		return nil, err
	}
	j := &antv1.Job{
		Id: id, UserId: userID, Kind: kind, Status: status, Progress: progress, Stage: stage,
		RequestSummary: requestSummary, ResultRef: resultRef, ResultSummary: resultSummary,
		ErrorCode: errorCode, ErrorMessage: errorMessage, IdempotencyKey: idempotencyKey,
		CreatedAt: timestamppb.New(createdAt),
	}
	if startedAt != nil {
		j.StartedAt = timestamppb.New(*startedAt)
	}
	if finishedAt != nil {
		j.FinishedAt = timestamppb.New(*finishedAt)
	}
	if expiresAt != nil {
		j.ExpiresAt = timestamppb.New(*expiresAt)
	}
	return j, nil
}

func (s *JobServer) GetJob(ctx context.Context, req *connect.Request[antv1.GetJobRequest]) (*connect.Response[antv1.Job], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	j, err := scanJobRow(s.pg.QueryRow(ctx, "SELECT "+jobSelectCols+" FROM jobs WHERE id = $1 AND user_id = $2", jobID, s.userID(ctx)))
	if err != nil {
		if errors.Is(err, errJobNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	return connect.NewResponse(j), nil
}

func (s *JobServer) CancelJob(ctx context.Context, req *connect.Request[antv1.CancelJobRequest]) (*connect.Response[antv1.Job], error) {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	uid := s.userID(ctx)
	j, err := scanJobRow(s.pg.QueryRow(ctx,
		"UPDATE jobs SET status = 'cancelled', finished_at = NOW() WHERE id = $1 AND user_id = $2 RETURNING "+jobSelectCols,
		jobID, uid))
	if err != nil {
		if errors.Is(err, errJobNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}
	return connect.NewResponse(j), nil
}

func (s *JobServer) SubscribeJob(ctx context.Context, req *connect.Request[antv1.SubscribeJobRequest], stream *connect.ServerStream[antv1.JobEvent]) error {
	jobID, err := uuid.Parse(req.Msg.JobId)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	afterSeq := req.Msg.AfterSeq

	// Send existing events, then poll for new ones.
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		rows, qerr := s.pg.Query(ctx,
			"SELECT job_id, seq, type, status, progress, stage, message, payload, created_at FROM job_events WHERE job_id = $1 AND seq > $2 ORDER BY seq",
			jobID, afterSeq)
		if qerr != nil {
			return qerr
		}

		hasRows := false
		for rows.Next() {
			hasRows = true
			var (
				jid, etype, estatus, estage, msg string
				seq                              int64
				progress                         float64
				payloadBytes                     []byte
				createdAt                        time.Time
			)
			if serr := rows.Scan(&jid, &seq, &etype, &estatus, &progress, &estage, &msg, &payloadBytes, &createdAt); serr != nil {
				rows.Close()
				return serr
			}
			var payload *structpb.Struct
			if len(payloadBytes) > 0 {
				var m map[string]interface{}
				if json.Unmarshal(payloadBytes, &m) == nil {
					payload, _ = structpb.NewStruct(m)
				}
			}
			stream.Send(&antv1.JobEvent{
				JobId: jid, Seq: seq, Type: etype, Status: estatus, Progress: progress, Stage: estage,
				Message: msg, Payload: payload, CreatedAt: timestamppb.New(createdAt),
			})
			afterSeq = seq
		}
		rows.Close()
		if rows.Err() != nil {
			return rows.Err()
		}

		if !hasRows {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
			}
		}
	}
}
