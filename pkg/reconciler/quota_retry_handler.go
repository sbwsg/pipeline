package reconciler

import (
	"time"

	"go.uber.org/zap"
)

type QuotaRetryHandler struct {
	logger   *zap.SugaredLogger
	stopCh   <-chan struct{}
	callback func(interface{})
}

func NewQuotaRetryHandler(stopCh <-chan struct{}, logger *zap.SugaredLogger) *QuotaRetryHandler {
	return &QuotaRetryHandler{
		logger:   logger,
		stopCh:   stopCh,
		callback: nil,
	}
}

func (q *QuotaRetryHandler) SetRetryCallback(callback func(interface{})) {
	q.callback = callback
}

func (q *QuotaRetryHandler) Retry(runObj StatusKey, waitDuration time.Duration) {
	select {
	case <-q.stopCh:
		q.logger.Debugf("Received stop signal, cancelling quota retry for %q", runObj.GetRunKey())
		return
	case <-time.After(waitDuration):
		if q.callback != nil {
			q.logger.Debugf("Retry timer for %q has fired, retrying", runObj.GetRunKey())
			q.callback(runObj)
		} else {
			q.logger.Debugf("Retry timer was fired for %q but no callback was provided; nothing to retry", runObj.GetRunKey())
		}
	}
}
