package result

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

var DefaultTimeout = time.Second * 10

// RequeueWithDefaultTimeout requeues with default timeout. Usually happens in case of error
func RequeueWithDefaultTimeout() *ctrl.Result {
	return &ctrl.Result{Requeue: true, RequeueAfter: DefaultTimeout}
}

// RequeueNow the reconcile loop
func RequeueNow() *ctrl.Result {
	return &ctrl.Result{Requeue: true}
}

// DontRequeue the loop
func DontRequeue() *ctrl.Result {
	return &ctrl.Result{Requeue: false}
}
