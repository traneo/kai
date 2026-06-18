package client

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	kaipb "kaiplatform.com/gen/kaiplatform/v1"
)

type Config struct {
	OrchestratorAddr string
	AgentID          string
	AgentAddr        string
}

type Client struct {
	cfg       Config
	cl        kaipb.OrchestratorClient
	conn      *grpc.ClientConn
	missionID atomic.Value
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Connect(ctx context.Context) error {
	conn, err := grpc.NewClient(c.cfg.OrchestratorAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(50*1024*1024),
			grpc.MaxCallSendMsgSize(50*1024*1024),
		))
	if err != nil {
		return fmt.Errorf("dial orchestrator: %w", err)
	}
	c.conn = conn
	c.cl = kaipb.NewOrchestratorClient(conn)
	log.Printf("connected to orchestrator at %s", c.cfg.OrchestratorAddr)
	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) SetMissionID(id string) {
	c.missionID.Store(id)
}

func (c *Client) ClearMissionID() {
	c.missionID.Store("")
}

func (c *Client) GetMissionID() string {
	v := c.missionID.Load()
	if v == nil {
		return ""
	}
	return v.(string)
}

func (c *Client) HeartbeatLoop(ctx context.Context) error {
	if c.cl == nil {
		return fmt.Errorf("client not connected")
	}

	stream, err := c.cl.Heartbeat(ctx)
	if err != nil {
		return fmt.Errorf("heartbeat stream: %w", err)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_, closeErr := stream.CloseAndRecv()
			if closeErr != nil {
				log.Printf("heartbeat close: %v", closeErr)
			}
			return nil
		case <-ticker.C:
			missionID := c.GetMissionID()
			status := kaipb.MissionStatus_MISSION_STATUS_UNSPECIFIED
			if missionID != "" {
				status = kaipb.MissionStatus_MISSION_STATUS_RUNNING
			}

			req := &kaipb.HeartbeatRequest{
				AgentId:       c.cfg.AgentID,
				AgentAddr:     c.cfg.AgentAddr,
				MissionId:     missionID,
				MissionStatus: status,
			}
			if err := stream.Send(req); err != nil {
				return fmt.Errorf("heartbeat send: %w", err)
			}
		}
	}
}

func (c *Client) ReportLog(ctx context.Context, entry *kaipb.LogEntry) error {
	stream, err := c.cl.ReportLog(ctx)
	if err != nil {
		return fmt.Errorf("report log stream: %w", err)
	}
	if err := stream.Send(entry); err != nil {
		return fmt.Errorf("report log send: %w", err)
	}
	_, err = stream.CloseAndRecv()
	return err
}

func (c *Client) ReportFileChange(ctx context.Context, change *kaipb.FileChange) error {
	stream, err := c.cl.ReportFileChange(ctx)
	if err != nil {
		return fmt.Errorf("report file change stream: %w", err)
	}
	if err := stream.Send(change); err != nil {
		return fmt.Errorf("report file change send: %w", err)
	}
	_, err = stream.CloseAndRecv()
	return err
}

func (c *Client) ReportResult(ctx context.Context, result *kaipb.MissionResult) (*kaipb.ResultAck, error) {
	return c.cl.ReportResult(ctx, result)
}
