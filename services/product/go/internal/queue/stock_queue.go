package queue

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/flash-deals/product/internal/db"
	"github.com/flash-deals/product/internal/repository"
	"github.com/google/uuid"
)

// StockUpdateRequest: 재고 업데이트 요청
type StockUpdateRequest struct {
	ProductID uuid.UUID
	Delta     int32
	Response  chan StockUpdateResponse
}

// StockUpdateResponse: 재고 업데이트 응답
type StockUpdateResponse struct {
	Row *db.UpdateStockRow
	Err error
}

// StockQueue: FIFO 순서 보장을 위한 재고 업데이트 큐
type StockQueue struct {
	repo     repository.ProductRepository
	requests chan StockUpdateRequest
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewStockQueue: 새 StockQueue 생성 및 워커 시작
func NewStockQueue(repo repository.ProductRepository, bufferSize int) *StockQueue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &StockQueue{
		repo:     repo,
		requests: make(chan StockUpdateRequest, bufferSize),
		ctx:      ctx,
		cancel:   cancel,
	}
	q.wg.Add(1)
	go q.worker()
	return q
}

// worker: 채널에서 요청을 순차적으로 처리하는 워커
func (q *StockQueue) worker() {
	defer q.wg.Done()
	log.Println("[StockQueue] Worker started - FIFO order guaranteed")

	for {
		select {
		case <-q.ctx.Done():
			log.Println("[StockQueue] Worker stopped")
			return
		case req, ok := <-q.requests:
			if !ok {
				log.Println("[StockQueue] Channel closed")
				return
			}
			// 순차적으로 재고 업데이트 처리
			row, err := q.repo.UpdateStockWithLock(context.Background(), req.ProductID, req.Delta)
			req.Response <- StockUpdateResponse{Row: row, Err: err}
		}
	}
}

// UpdateStock: 큐에 재고 업데이트 요청 추가 (FIFO 순서 보장)
func (q *StockQueue) UpdateStock(ctx context.Context, productID uuid.UUID, delta int32) (*db.UpdateStockRow, error) {
	responseChan := make(chan StockUpdateResponse, 1)

	req := StockUpdateRequest{
		ProductID: productID,
		Delta:     delta,
		Response:  responseChan,
	}

	// 큐에 요청 추가 (채널에 들어간 순서대로 처리됨)
	select {
	case q.requests <- req:
		// 요청이 큐에 추가됨
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while enqueueing")
	}

	// 응답 대기
	select {
	case resp := <-responseChan:
		return resp.Row, resp.Err
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled while waiting for response")
	}
}

// Close: 큐 종료
func (q *StockQueue) Close() {
	q.cancel()
	close(q.requests)
	q.wg.Wait()
}
